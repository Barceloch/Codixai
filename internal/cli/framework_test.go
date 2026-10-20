package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes the CLI and returns its exit code and its output.
func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHelpListsTheCommandsAndTheGlobalFlags(t *testing.T) {
	code, stdout, stderr := run(t, "--help")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	for _, expected := range []string{
		"Usage:",
		"Commands:",
		"audit",
		"init",
		"version",
		"Flags:",
		"--root",
		"--format",
		"--no-color",
		"--plain",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("the help does not mention %q:\n%s", expected, stdout)
		}
	}
}

func TestHelpOfACommandListsItsOwnFlags(t *testing.T) {
	code, stdout, stderr := run(t, "audit", "--help")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	for _, expected := range []string{"codixai audit", "--diff", "--base", "--staged", "--worktree", "--rules", "Global flags:"} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("the help of the audit command does not mention %q:\n%s", expected, stdout)
		}
	}
}

func TestUnknownFlagFailsWithTheInvalidArgumentsCode(t *testing.T) {
	code, stdout, stderr := run(t, "audit", "--bogus")
	if code != ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d", code, ExitInvalidArgs)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("the error does not name the unknown flag:\n%s", stderr)
	}
}

func TestInvalidFlagValueFailsWithTheInvalidArgumentsCode(t *testing.T) {
	code, _, stderr := run(t, "--format", "xml", "version")
	if code != ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d", code, ExitInvalidArgs)
	}
	if !strings.Contains(stderr, "xml") {
		t.Errorf("the error does not name the invalid value:\n%s", stderr)
	}
}

func TestUnknownCommandFailsWithTheInvalidArgumentsCode(t *testing.T) {
	code, _, stderr := run(t, "frobnicate")
	if code != ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d", code, ExitInvalidArgs)
	}
	if !strings.Contains(stderr, "frobnicate") {
		t.Errorf("the error does not name the unknown command:\n%s", stderr)
	}
}

func TestGlobalFlagsAreAcceptedBeforeAndAfterTheCommand(t *testing.T) {
	before, _, _ := run(t, "--format", "json", "version")
	after, _, _ := run(t, "version", "--format", "json")
	if before != ExitOK || after != ExitOK {
		t.Fatalf("the exit codes are %d and %d, want %d", before, after, ExitOK)
	}
}

func TestVersionCommandPrintsTheVersionAndTheSchemaVersion(t *testing.T) {
	code, stdout, stderr := run(t, "version")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	if !strings.Contains(stdout, Version) {
		t.Errorf("the output does not hold the version:\n%s", stdout)
	}
	if !strings.Contains(stdout, "0.1") {
		t.Errorf("the output does not hold the schema version:\n%s", stdout)
	}
}

func TestInitCreatesTheConfigurationAndNeverOverwritesIt(t *testing.T) {
	root := t.TempDir()

	code, _, stderr := run(t, "--root", root, "init")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	path := filepath.Join(root, ".codixai.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the configuration file was not created: %v", err)
	}
	if !strings.Contains(string(data), "fail_on: error") {
		t.Errorf("the configuration does not hold the failure threshold:\n%s", data)
	}

	// A second run must not modify the file.
	if err := os.WriteFile(path, []byte("# edited by the user\n"), 0o644); err != nil {
		t.Fatalf("cannot edit the configuration file: %v", err)
	}
	code, stdout, stderr := run(t, "--root", root, "init")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	if !strings.Contains(stdout, "already exists") {
		t.Errorf("the command does not report that the file exists:\n%s", stdout)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the configuration file: %v", err)
	}
	if string(data) != "# edited by the user\n" {
		t.Errorf("the configuration file was modified: %q", data)
	}
}

func TestInitWithForceReplacesTheConfiguration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".codixai.yml")
	if err := os.WriteFile(path, []byte("# edited by the user\n"), 0o644); err != nil {
		t.Fatalf("cannot write the configuration file: %v", err)
	}
	code, _, stderr := run(t, "--root", root, "init", "--force")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the configuration file: %v", err)
	}
	if strings.Contains(string(data), "edited by the user") {
		t.Errorf("the configuration file was not replaced:\n%s", data)
	}
}

func TestJSONOutputHoldsOnlyTheDocument(t *testing.T) {
	code, stdout, stderr := run(t, "--format", "json", "version")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		t.Errorf("the standard output does not hold a JSON document:\n%s", stdout)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Errorf("the standard output carries an escape sequence:\n%q", stdout)
	}
}

func TestNoColorEnvironmentVariableRemovesTheEscapeSequences(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	code, stdout, stderr := run(t, "--help")
	if code != ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, ExitOK, stderr)
	}
	if strings.Contains(stdout, "\x1b[") {
		t.Errorf("the help carries an escape sequence:\n%q", stdout)
	}
}
