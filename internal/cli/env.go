package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/barceloch/codixai/internal/cli/ui"
	"github.com/barceloch/codixai/internal/policies"
	"github.com/barceloch/codixai/internal/report"
)

// Version is the version of the tool.
//
// The value is injected at build time with the linker flag
// "-X main.version=<version>". It stays "dev" when the binary is built
// without it.
var Version = "dev"

// Env holds the environment of one run of the CLI.
//
// The environment carries the global options of the invocation and the
// writers of the command. The commands reach it through the context, so
// their handlers keep a single argument.
type Env struct {
	// Stdout receives the document that the command produces.
	Stdout io.Writer
	// Stderr receives the diagnostics, the warnings and the errors.
	Stderr io.Writer
	// Root is the root of the analyzed repository. It is empty when the
	// command runs in the current directory.
	Root string
	// Format is the output format of the commands that produce a
	// document: text or json, or sarif for the report command.
	Format string
	// FormatSet reports whether the invocation asked for a format
	// explicitly.
	//
	// A command with a format of its own has to tell the default of the
	// tool apart from an explicit choice, so that "codixai report" writes
	// its own format and "codixai report --format text" is reported as an
	// invalid combination instead of being silently ignored.
	FormatSet bool
	// NoColor disables the color output.
	NoColor bool
	// Plain disables every decoration of the output.
	Plain bool
	// ShowVersion asks for the version of the tool.
	ShowVersion bool

	renderer *ui.Renderer
}

// Renderer returns the renderer of the environment.
//
// The renderer is built once, with the options of the invocation and with
// the state of the terminal.
func (e *Env) Renderer() *ui.Renderer {
	if e.renderer == nil {
		e.renderer = ui.NewRenderer(ui.OptionsFor(e.Stdout, e.NoColor, e.Plain, os.Getenv))
	}
	return e.renderer
}

// JSON reports whether the command has to produce a JSON document.
func (e *Env) JSON() bool {
	return strings.EqualFold(strings.TrimSpace(e.Format), policies.FormatJSON)
}

// validate checks the global options of the invocation.
//
// The target is the name of the command that is about to run. The SARIF
// format belongs to the report command, so a command that cannot write it
// reports the invalid format instead of silently writing another one: the
// standard output would hold a document that nobody asked for.
func (e *Env) validate(target string) error {
	format := strings.ToLower(strings.TrimSpace(e.Format))
	switch format {
	case policies.FormatText, policies.FormatJSON:
		return nil
	case report.Format:
		if target == "report" {
			return nil
		}
		if target == "" {
			target = "codixai"
		}
		return Usagef("the format %q is not available for %q: it is the format of the report command, "+
			"use 'codixai report --format %s', or one of %s, %s",
			e.Format, target, report.Format, policies.FormatText, policies.FormatJSON)
	default:
		return Usagef("invalid output format %q, expected one of %s, %s, %s",
			e.Format, policies.FormatText, policies.FormatJSON, report.Format)
	}
}

// envKey is the key of the environment inside the context.
type envKey struct{}

// withEnv stores the environment in the context.
func withEnv(ctx context.Context, env *Env) context.Context {
	return context.WithValue(ctx, envKey{}, env)
}

// envFrom returns the environment of the context.
func envFrom(ctx context.Context) *Env {
	if ctx == nil {
		return nil
	}
	env, _ := ctx.Value(envKey{}).(*Env)
	return env
}

// resolveRoot returns the absolute path of the analyzed repository.
//
// The default is the current directory. The path is validated, so that the
// failure of a command never depends on the state of the working tree.
func resolveRoot(value string) (string, error) {
	root := strings.TrimSpace(value)
	if root == "" {
		working, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine the current directory: %w", err)
		}
		root = working
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("cannot resolve the repository root %q: %w", value, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("cannot open the repository root %s: %w", displayPath(absolute), err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("the repository root %s is not a directory", displayPath(absolute))
	}
	return absolute, nil
}

// displayPath returns the shortest form of an absolute path that identifies
// it.
//
// A path inside the current directory is shown relative to it, so that the
// output of the tool never exposes the home directory of the user. A path
// outside of the current directory is shown relative to it only when it
// lives in the home directory: the rest of the paths are shown as they are,
// because a relative path would be longer and less readable.
func displayPath(path string) string {
	working, err := os.Getwd()
	if err != nil {
		return path
	}
	relative, err := filepath.Rel(working, path)
	if err != nil {
		return path
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return relative
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+string(filepath.Separator)) {
		return relative
	}
	return path
}

// checkGitRepository checks that the root of the repository holds a Git
// repository.
//
// The check only looks at the file system: it never runs a Git command and
// it never writes anything. It gives an actionable message instead of the
// usage of Git, which is what the Git integration reports when the
// directory is not a repository.
func checkGitRepository(root string) error {
	path := filepath.Join(root, ".git")
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("the root %s is not a Git repository: run the command inside a repository "+
			"or point it to one with --root <path>", displayPath(root))
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("the root %s is not a Git repository: the .git entry is not a directory",
			displayPath(root))
	}
	return nil
}

// sanitizeError replaces the absolute path of the repository root with its
// short form in the message of an error.
//
// The messages of the providers often name the files they could not read.
// The tool never has to show the absolute path of the home directory of the
// user when a relative path identifies the file.
func sanitizeError(err error, root string) error {
	if err == nil || root == "" {
		return err
	}
	message := err.Error()
	replaced := strings.ReplaceAll(message, root, displayPath(root))
	if replaced == message {
		return err
	}
	return fmt.Errorf("%s", replaced)
}
