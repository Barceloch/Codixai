// Package git implements a read-only integration with the Git version control
// system.
//
// The package is designed around the security rules of the product
// specification (section 13):
//
//   - git is always invoked with an argument slice, never through a shell;
//   - every invocation neutralizes the executable configuration a repository
//     can control (external diff drivers, textconv filters, fsmonitor hooks,
//     pagers and external transport helpers);
//   - no git hook is ever executed and no code of the analyzed repository is
//     ever run;
//   - only read-only commands are used, so the working tree, the index and the
//     object database are never modified.
//
// The package does not depend on the application layer: it only imports the
// domain models it has to produce.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"codixai/internal/application"
	"codixai/internal/domain"
)

// Source implements the application.ChangeSource port on top of the git
// command line. It is safe for concurrent use as long as it is not
// reconfigured.
type Source struct {
	root    string
	gitPath string
	opts    options
	exec    runner
}

// The integration satisfies the port of the application layer.
var _ application.ChangeSource = (*Source)(nil)

// options holds the capabilities of the detected git binary.
type options struct {
	// noOptionalLocks is true when git understands --no-optional-locks
	// (git >= 2.15). It prevents read-only commands from refreshing the
	// index, which keeps the repository byte-for-byte untouched.
	noOptionalLocks bool
	// endOfOptions is true when git understands --end-of-options
	// (git >= 2.24). It stops option parsing so a ref can never be
	// interpreted as an option.
	endOfOptions bool
}

// runner executes git with an argument slice and returns its standard output.
type runner func(ctx context.Context, dir string, args []string) ([]byte, error)

var versionRe = regexp.MustCompile(`git version (\d+)\.(\d+)(?:[.\-+].*)?$`)

// New creates a Source for the repository located at root.
func New(root string) (*Source, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git executable not found in PATH: %w", err)
	}
	return NewWithGit(root, gitPath)
}

// NewWithGit creates a Source using an explicit git executable. It is the
// entry point used by tests and by callers that manage the binary themselves.
func NewWithGit(root, gitPath string) (*Source, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("repository root is required")
	}
	if strings.TrimSpace(gitPath) == "" {
		return nil, errors.New("git executable is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve repository root %q: %w", root, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("cannot open repository root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repository root %q is not a directory", root)
	}

	s := &Source{
		root:    abs,
		gitPath: gitPath,
		exec: func(ctx context.Context, dir string, args []string) ([]byte, error) {
			return runGit(ctx, dir, gitPath, args)
		},
	}
	if err := s.detectOptions(); err != nil {
		return nil, err
	}
	return s, nil
}

// Root returns the absolute path of the repository the source points at.
func (s *Source) Root() string { return s.root }

// detectOptions probes the git version to know which safety flags are
// supported by the installed binary.
func (s *Source) detectOptions() error {
	out, err := s.exec(context.Background(), s.root, []string{"--version"})
	if err != nil {
		return fmt.Errorf("cannot determine the git version: %w", err)
	}
	m := versionRe.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		// An unknown version is treated as an old one: only the flags
		// supported by every git are used.
		return nil
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	s.opts.noOptionalLocks = major > 2 || (major == 2 && minor >= 15)
	s.opts.endOfOptions = major > 2 || (major == 2 && minor >= 24)
	return nil
}

// baseArgs returns the arguments shared by every git invocation.
//
// The configuration is neutralized on purpose: a repository can define
// external diff drivers, textconv filters, fsmonitor hooks, a pager or
// alternative transports through its own configuration files, and those
// settings would run code of the analyzed repository.
func (s *Source) baseArgs() []string {
	args := []string{"--no-pager"}
	if s.opts.noOptionalLocks {
		// Read-only commands must not refresh the index on disk.
		args = append(args, "--no-optional-locks")
	}
	args = append(args,
		"-c", "core.pager=cat",
		"-c", "core.fsmonitor=false",
		"-c", "core.quotePath=true",
		"-c", "diff.relative=false",
		"-c", "diff.noprefix=false",
		"-c", "diff.mnemonicPrefix=false",
		"-c", "diff.external=",
		"-c", "diff.renames=true",
		"-c", "diff.context=3",
		"-c", "diff.suppressBlankEmpty=false",
		"-c", "protocol.ext.allow=never",
	)
	return args
}

// diffArgs builds the argument list of the diff command for an origin.
func (s *Source) diffArgs(origin domain.ChangeOrigin, base string) ([]string, error) {
	args := []string{
		"diff",
		"--no-ext-diff",
		"--no-textconv",
		"--no-color",
		"--find-renames",
		"--unified=3",
	}
	switch origin {
	case domain.OriginBase:
		if strings.TrimSpace(base) == "" {
			return nil, errors.New("a base ref is required to compare the repository with a base")
		}
		if err := validateRef(base); err != nil {
			return nil, err
		}
		args = append(args, base, "HEAD")
	case domain.OriginStaged:
		args = append(args, "--cached")
	case domain.OriginWorktree:
		// No revision: git compares the index with the working tree.
	default:
		return nil, fmt.Errorf("unsupported change origin %q", origin)
	}
	// "--" ends the revision list, so a path can never be taken for a ref.
	args = append(args, "--")
	return args, nil
}

// validateRef rejects refs that could be interpreted as an option or that
// contain characters git does not accept.
func validateRef(ref string) error {
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("invalid base ref %q: a ref cannot start with '-'", ref)
	}
	if strings.ContainsAny(ref, " \t\n\r\x00") {
		return fmt.Errorf("invalid base ref %q: a ref cannot contain whitespace or control characters", ref)
	}
	if strings.Contains(ref, "..") {
		return fmt.Errorf("invalid base ref %q: a ref cannot contain '..'", ref)
	}
	if !validRefRe.MatchString(ref) {
		return fmt.Errorf("invalid base ref %q: it contains characters that are not valid in a revision name", ref)
	}
	return nil
}

var validRefRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/@:+^~-]*$`)

// Changes collects the changes of the given origin and normalizes them into a
// domain.ChangeSet.
func (s *Source) Changes(ctx context.Context, origin domain.ChangeOrigin, base string) (*domain.ChangeSet, error) {
	args, err := s.diffArgs(origin, base)
	if err != nil {
		return nil, err
	}
	if origin == domain.OriginBase {
		if err := s.verifyRef(ctx, base); err != nil {
			return nil, err
		}
	}
	out, err := s.exec(ctx, s.root, args)
	if err != nil {
		return nil, err
	}
	changes, err := ParseUnifiedDiff(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("cannot parse the git diff output: %w", err)
	}
	changes.Origin = origin
	if origin == domain.OriginBase {
		changes.Base = base
	}
	if head, err := s.Head(ctx); err == nil {
		changes.Head = head
	} else {
		changes.Diagnostics = append(changes.Diagnostics, domain.Diagnostic{
			Path:     s.root,
			Message:  fmt.Sprintf("cannot determine the HEAD revision: %v", err),
			Severity: domain.SeverityWarning,
		})
	}
	return changes, nil
}

// verifyRef checks that a base ref exists in the repository and gives an
// actionable error when it does not.
func (s *Source) verifyRef(ctx context.Context, ref string) error {
	args := []string{"rev-parse", "--verify", "--quiet"}
	if s.opts.endOfOptions {
		args = append(args, "--end-of-options")
	}
	args = append(args, ref+"^{commit}")
	out, err := s.exec(ctx, s.root, args)
	if err != nil {
		return fmt.Errorf("base ref %q does not exist in the repository: it must be a reachable commit (for example 'main', 'origin/main' or a commit hash)", ref)
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("base ref %q does not exist in the repository: it must be a reachable commit (for example 'main', 'origin/main' or a commit hash)", ref)
	}
	return nil
}

// Head returns the revision of the current HEAD commit.
func (s *Source) Head(ctx context.Context) (string, error) {
	args := []string{"rev-parse", "--verify", "--quiet"}
	if s.opts.endOfOptions {
		args = append(args, "--end-of-options")
	}
	args = append(args, "HEAD")
	out, err := s.exec(ctx, s.root, args)
	if err != nil {
		return "", fmt.Errorf("cannot resolve HEAD: the repository has no commit yet or it is not a git repository: %w", err)
	}
	head := strings.TrimSpace(string(out))
	if head == "" {
		return "", errors.New("cannot resolve HEAD: the repository has no commit yet")
	}
	return head, nil
}

// runGit executes git with an argument slice. No shell is involved, so the
// arguments are never interpreted by a shell.
func runGit(ctx context.Context, dir, bin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return stdout.Bytes(), fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
		return stdout.Bytes(), fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, message)
	}
	return stdout.Bytes(), nil
}

// gitEnv builds the environment of the git process. Variables that could
// redirect git to another repository, inject configuration or run an external
// diff driver are removed.
func gitEnv() []string {
	blocked := map[string]bool{
		"GIT_DIR":                          true,
		"GIT_WORK_TREE":                    true,
		"GIT_INDEX_FILE":                   true,
		"GIT_OBJECT_DIRECTORY":             true,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
		"GIT_CEILING_DIRECTORIES":          true,
		"GIT_EXTERNAL_DIFF":                true,
		"GIT_DIFF_OPTS":                    true,
		"GIT_CONFIG_COUNT":                 true,
		"GIT_CONFIG_PARAMETERS":            true,
		"GIT_PAGER":                        true,
		"GIT_EDITOR":                       true,
	}
	env := make([]string, 0, len(os.Environ())+6)
	for _, entry := range os.Environ() {
		name := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			name = entry[:idx]
		}
		if blocked[name] {
			continue
		}
		if strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env,
		// Deterministic, non interactive and non spawning behaviour.
		"LC_ALL=C",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
		// System wide attributes can define diff drivers, so they are
		// disabled as well.
		"GIT_ATTR_NOSYSTEM=1",
		// Never ask a credential helper or a remote for anything.
		"GIT_ASKPASS=",
		"GIT_SSH_COMMAND=",
	)
	return env
}
