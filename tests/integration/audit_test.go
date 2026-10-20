// Package integration holds the end to end tests of the command line
// interface.
//
// The tests build a real Git repository in a temporary directory, they
// change it and they run the CLI against it. They never modify the
// repository of the project and they never need the network.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Barceloch/Codixai/internal/cli"
)

// ansiSequence detects an ANSI escape sequence in a text.
func ansiSequence(text string) bool {
	return strings.Contains(text, "\x1b[")
}

// requireGit skips the test when the git binary is not available.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("the git binary is not available: the end to end test needs it")
	}
}

// runGit runs a git command inside the given directory.
//
// The identity of the author and the signature configuration are given on
// the command line, so that the test does not depend on the configuration
// of the machine that runs it.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := []string{
		"-c", "user.email=test@example.com",
		"-c", "user.name=Codixai Test",
		"-c", "commit.gpgsign=false",
		"-c", "init.defaultBranch=main",
	}
	full = append(full, args...)
	command := exec.Command("git", full...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, output)
	}
}

// writeFile writes a file of the repository.
func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", name, err)
	}
}

// newRepository builds a Git repository with one committed Python file.
//
// It returns the root of the repository.
func newRepository(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	writeFile(t, root, "app.py", "def greet():\n    return \"hello\"\n")
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "initial commit")
	return root
}

// failingChange is a change that adds a suppression and a placeholder.
const failingChange = "def greet():\n    # TODO: return the greeting of the user\n    return \"hello\"  # noqa\n"

// benignChange is a change that no rule of the catalog reports.
const benignChange = "def greet():\n    return \"hello\"\n"

// writePolicy writes a configuration that fails the audit on a warning.
func writePolicy(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, ".codixai.yml", "version: 1\naudit:\n  fail_on: warning\n")
}

// execute runs the CLI and returns its exit code and its output.
func execute(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cli.Execute(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestAuditReportsTheSuppressionAndThePlaceholder(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	code, stdout, stderr := execute(t, "--root", root, "audit", "--diff", "--worktree")

	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	for _, rule := range []string{"CDX-SUP-001", "CDX-PLH-001"} {
		if !strings.Contains(stdout, rule) {
			t.Errorf("the report does not mention %s:\n%s", rule, stdout)
		}
	}
	if !strings.Contains(stdout, "worktree") {
		t.Errorf("the report does not declare the analyzed origin:\n%s", stdout)
	}
	if !strings.Contains(stderr, "fail_on") {
		t.Errorf("the standard error does not explain the failure:\n%s", stderr)
	}
}

func TestAuditJSONOutputIsACleanDocument(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "audit", "--diff", "--worktree")
	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if ansiSequence(stdout) {
		t.Errorf("the JSON document carries an escape sequence:\n%q", stdout)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("the standard output is not a JSON document:\n%s", stdout)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("cannot parse the JSON document: %v", err)
	}
	if document["schema_version"] != "0.1" {
		t.Errorf("the schema version is %v, want 0.1", document["schema_version"])
	}
	for _, key := range []string{"tool_version", "generated_at", "origin", "revision", "findings", "summary", "rules", "limitations"} {
		if _, ok := document[key]; !ok {
			t.Errorf("the JSON document does not hold the key %q", key)
		}
	}
	if document["origin"] != "worktree" {
		t.Errorf("the analyzed origin is %v, want worktree", document["origin"])
	}
	findings, ok := document["findings"].([]any)
	if !ok {
		t.Fatalf("the findings of the document are not a list: %v", document["findings"])
	}
	rules := map[string]bool{}
	for _, finding := range findings {
		entry, ok := finding.(map[string]any)
		if !ok {
			t.Fatalf("a finding is not an object: %v", finding)
		}
		rule, _ := entry["rule_id"].(string)
		rules[rule] = true
	}
	for _, rule := range []string{"CDX-SUP-001", "CDX-PLH-001"} {
		if !rules[rule] {
			t.Errorf("the document does not report %s, got %v", rule, rules)
		}
	}
}

func TestAuditWithoutFindingsSucceeds(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", benignChange)

	code, stdout, stderr := execute(t, "--root", root, "audit", "--diff", "--worktree")
	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if strings.Contains(stdout, "CDX-SUP-001") || strings.Contains(stdout, "CDX-PLH-001") {
		t.Errorf("the report mentions a rule that did not fire:\n%s", stdout)
	}
}

func TestAuditWithoutOriginFailsWithTheInvalidArgumentsCode(t *testing.T) {
	root := newRepository(t)

	code, _, stderr := execute(t, "--root", root, "audit", "--diff")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if !strings.Contains(stderr, "origin") {
		t.Errorf("the error does not explain the missing origin:\n%s", stderr)
	}
}

func TestAuditDoesNotModifyTheRepository(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	before := treeState(t, root)
	execute(t, "--root", root, "audit", "--diff", "--worktree")
	after := treeState(t, root)

	if before != after {
		t.Errorf("the audit changed the repository:\nbefore %s\nafter  %s", before, after)
	}
}

// treeState returns the state of the working tree of the repository.
//
// The test uses the porcelain output of git status, which reports the
// changes of the index and of the working tree.
func treeState(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "status", "--porcelain")
	command.Dir = root
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
	)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("cannot read the state of the repository: %v", err)
	}
	return string(output)
}
