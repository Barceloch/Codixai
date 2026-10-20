package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Barceloch/Codixai/internal/domain"
)

// mutatingGitVerbs lists the git subcommands that can modify the repository.
// None of them may ever be part of an invocation of this package.
var mutatingGitVerbs = []string{
	"add", "am", "apply", "branch", "checkout", "cherry-pick", "clean", "commit",
	"commit-tree", "config", "fetch", "format-patch", "gc", "merge", "mv",
	"pull", "push", "rebase", "repack", "reset", "restore", "revert", "rm",
	"stash", "switch", "symbolic-ref", "tag", "update-index", "update-ref",
	"worktree", "write-tree",
}

func TestBaseArgsNeutralizeRepositoryConfiguration(t *testing.T) {
	source := &Source{root: "/tmp", gitPath: "git"}
	args := source.baseArgs()

	required := []string{
		"--no-pager",
		"core.pager=cat",
		"core.fsmonitor=false",
	}
	for _, needle := range required {
		if !containsArg(args, needle) {
			t.Errorf("base arguments %v do not contain %q", args, needle)
		}
	}
}

func TestDiffArgsAreReadOnly(t *testing.T) {
	source := &Source{root: "/tmp", gitPath: "git"}
	origins := []domain.ChangeOrigin{domain.OriginBase, domain.OriginStaged, domain.OriginWorktree}
	for _, origin := range origins {
		args, err := source.diffArgs(origin, "main")
		if err != nil {
			t.Fatalf("diffArgs(%q) returned an error: %v", origin, err)
		}
		joined := strings.Join(args, " ")
		for _, needle := range []string{"--no-ext-diff", "--no-textconv", "--find-renames", "--no-color", "--"} {
			if !containsArg(args, needle) {
				t.Errorf("%s arguments %v do not contain %q", origin, args, needle)
			}
		}
		if origin == domain.OriginStaged && !containsArg(args, "--cached") {
			t.Errorf("staged arguments %v do not contain --cached", args)
		}
		if origin == domain.OriginBase && !strings.Contains(joined, "main HEAD") {
			t.Errorf("base arguments %v do not compare main with HEAD", args)
		}
		for _, verb := range mutatingGitVerbs {
			if containsArg(args, verb) {
				t.Errorf("%s arguments %v contain the mutating verb %q", origin, args, verb)
			}
		}
	}
}

func TestDiffArgsRejectInvalidBaseRef(t *testing.T) {
	source := &Source{root: "/tmp", gitPath: "git"}
	invalid := []string{
		"--upload-pack=evil",
		"-x",
		"main branch",
		"../escape",
		"bad..ref",
	}
	for _, ref := range invalid {
		if _, err := source.diffArgs(domain.OriginBase, ref); err == nil {
			t.Errorf("diffArgs accepted the invalid ref %q", ref)
		}
	}
	if _, err := source.diffArgs(domain.OriginBase, ""); err == nil {
		t.Error("diffArgs accepted an empty base ref")
	}
	if _, err := source.diffArgs(domain.ChangeOrigin("unknown"), ""); err == nil {
		t.Error("diffArgs accepted an unknown origin")
	}
}

func TestGitEnvRemovesRedirectingVariables(t *testing.T) {
	t.Setenv("GIT_DIR", "/tmp/other")
	t.Setenv("GIT_EXTERNAL_DIFF", "evil")
	t.Setenv("GIT_CONFIG_COUNT", "1")

	env := gitEnv()
	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_DIR=") ||
			strings.HasPrefix(entry, "GIT_EXTERNAL_DIFF=") ||
			strings.HasPrefix(entry, "GIT_CONFIG_COUNT=") {
			t.Errorf("the environment still contains %q", entry)
		}
	}
}

func TestSourceChangesAgainstBase(t *testing.T) {
	repo := newTestRepository(t)
	writeFile(t, repo, "main.go", "package main\n\nfunc main() {}\n")
	commitAll(t, repo, "add main")
	writeFile(t, repo, "main.go", "package main\n\nfunc main() { println(\"hi\") }\n")
	commitAll(t, repo, "change main")

	source := newTestSource(t, repo)
	changes, err := source.Changes(context.Background(), domain.OriginBase, "HEAD~1")
	if err != nil {
		t.Fatalf("Changes returned an error: %v", err)
	}
	if changes.Origin != domain.OriginBase {
		t.Errorf("origin: got %q", changes.Origin)
	}
	if changes.Base != "HEAD~1" {
		t.Errorf("base: got %q", changes.Base)
	}
	if changes.Head == "" {
		t.Error("expected the head revision to be recorded")
	}
	if len(changes.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(changes.Files))
	}
	file := changes.Files[0]
	if file.Path != "main.go" || file.Status != domain.FileModified {
		t.Errorf("file: %+v", file)
	}
	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}
}

func TestSourceChangesStagedAndWorktree(t *testing.T) {
	repo := newTestRepository(t)
	writeFile(t, repo, "main.go", "package main\n")
	commitAll(t, repo, "add main")

	writeFile(t, repo, "main.go", "package main\n\n// staged\n")
	runGitIn(t, repo, "add", "main.go")
	writeFile(t, repo, "main.go", "package main\n\n// staged\n// worktree\n")

	source := newTestSource(t, repo)

	staged, err := source.Changes(context.Background(), domain.OriginStaged, "")
	if err != nil {
		t.Fatalf("Changes(staged) returned an error: %v", err)
	}
	if len(staged.Files) != 1 {
		t.Fatalf("expected 1 staged file, got %d", len(staged.Files))
	}
	if staged.Origin != domain.OriginStaged {
		t.Errorf("origin: got %q", staged.Origin)
	}

	worktree, err := source.Changes(context.Background(), domain.OriginWorktree, "")
	if err != nil {
		t.Fatalf("Changes(worktree) returned an error: %v", err)
	}
	if len(worktree.Files) != 1 {
		t.Fatalf("expected 1 worktree file, got %d", len(worktree.Files))
	}
	if worktree.Origin != domain.OriginWorktree {
		t.Errorf("origin: got %q", worktree.Origin)
	}
}

func TestSourceHead(t *testing.T) {
	repo := newTestRepository(t)
	source := newTestSource(t, repo)
	head, err := source.Head(context.Background())
	if err != nil {
		t.Fatalf("Head returned an error: %v", err)
	}
	if len(head) != 40 {
		t.Errorf("expected a full revision, got %q", head)
	}
}

func TestSourceMissingBaseRefGivesActionableError(t *testing.T) {
	repo := newTestRepository(t)
	source := newTestSource(t, repo)
	_, err := source.Changes(context.Background(), domain.OriginBase, "does-not-exist")
	if err == nil {
		t.Fatal("expected an error for a missing base ref")
	}
	message := err.Error()
	if !strings.Contains(message, "does-not-exist") {
		t.Errorf("the error does not name the base ref: %s", message)
	}
	if !strings.Contains(message, "does not exist") {
		t.Errorf("the error is not actionable: %s", message)
	}
}

// TestSourceInvocationsAreReadOnly proves two things: no mutating git
// subcommand is ever used and the repository is not modified on disk.
func TestSourceInvocationsAreReadOnly(t *testing.T) {
	repo := newTestRepository(t)
	writeFile(t, repo, "main.go", "package main\n")
	commitAll(t, repo, "add main")

	source := newTestSource(t, repo)
	var recorded [][]string
	original := source.exec
	source.exec = func(ctx context.Context, dir string, args []string) ([]byte, error) {
		recorded = append(recorded, append([]string(nil), args...))
		return original(ctx, dir, args)
	}

	indexBefore := fileStamp(t, filepath.Join(repo, ".git", "index"))
	treeBefore := listTree(t, repo)

	if _, err := source.Changes(context.Background(), domain.OriginBase, "HEAD"); err != nil {
		t.Fatalf("Changes returned an error: %v", err)
	}
	if _, err := source.Changes(context.Background(), domain.OriginStaged, ""); err != nil {
		t.Fatalf("Changes returned an error: %v", err)
	}
	if _, err := source.Changes(context.Background(), domain.OriginWorktree, ""); err != nil {
		t.Fatalf("Changes returned an error: %v", err)
	}
	if _, err := source.Head(context.Background()); err != nil {
		t.Fatalf("Head returned an error: %v", err)
	}

	if len(recorded) == 0 {
		t.Fatal("no git invocation was recorded")
	}
	for _, args := range recorded {
		for _, verb := range mutatingGitVerbs {
			if containsArg(args, verb) {
				t.Errorf("the invocation %v contains the mutating verb %q", args, verb)
			}
		}
	}

	indexAfter := fileStamp(t, filepath.Join(repo, ".git", "index"))
	if indexBefore != indexAfter {
		t.Errorf("the git index was modified: %q -> %q", indexBefore, indexAfter)
	}
	treeAfter := listTree(t, repo)
	if treeBefore != treeAfter {
		t.Errorf("the repository content changed:\n%s\n%s", treeBefore, treeAfter)
	}
}

// TestSourceDoesNotRunRepositoryCommands proves that the executable
// configuration of a repository (external diff drivers and textconv filters)
// is never run.
func TestSourceDoesNotRunRepositoryCommands(t *testing.T) {
	repo := newTestRepository(t)
	marker := filepath.Join(t.TempDir(), "executed")
	runGitIn(t, repo, "config", "diff.evil.command", "touch "+marker)
	runGitIn(t, repo, "config", "diff.evil.textconv", "touch "+marker)
	writeFile(t, repo, ".gitattributes", "*.txt diff=evil\n")
	writeFile(t, repo, "secret.txt", "first\n")
	commitAll(t, repo, "attributes")
	writeFile(t, repo, "secret.txt", "second\n")

	source := newTestSource(t, repo)
	if _, err := source.Changes(context.Background(), domain.OriginWorktree, ""); err != nil {
		t.Fatalf("Changes returned an error: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the external diff driver of the repository was executed")
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// containsArg reports whether the argument list holds the given value.
func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

// newTestRepository creates a repository with a single empty commit.
func newTestRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available in this environment")
	}
	repo := t.TempDir()
	runGitIn(t, repo, "init", "-q", ".")
	runGitIn(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "--allow-empty", "-m", "init")
	return repo
}

// newTestSource creates a Source for a test repository.
func newTestSource(t *testing.T, repo string) *Source {
	t.Helper()
	source, err := New(repo)
	if err != nil {
		t.Fatalf("cannot create the git source: %v", err)
	}
	return source
}

// runGitIn runs a git command inside a directory and fails the test on error.
func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, out)
	}
}

// writeFile writes a file inside a directory.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", name, err)
	}
}

// commitAll stages and commits every change of a repository.
func commitAll(t *testing.T, repo, message string) {
	t.Helper()
	runGitIn(t, repo, "add", "-A")
	runGitIn(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "-m", message)
}

// fileStamp returns a cheap fingerprint of a file.
func fileStamp(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	return fmt.Sprintf("%d:%s", info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano))
}

// listTree returns a fingerprint of the content of a directory tree.
func listTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		b.WriteString(rel)
		b.WriteString(":")
		b.WriteString(string(data))
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatalf("cannot list %s: %v", root, err)
	}
	return b.String()
}
