package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// moduleRoot returns the root of the module of the project.
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("cannot resolve the root of the module: %v", err)
	}
	return root
}

// buildBinary builds the command line interface in a temporary directory.
//
// The test is skipped when the go binary is not available.
func buildBinary(t *testing.T) string {
	t.Helper()
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go binary is not available: the test cannot build the command")
	}
	binary := filepath.Join(t.TempDir(), "codixai")
	command := exec.Command(goBinary, "build", "-o", binary, "./cmd/codixai")
	command.Dir = moduleRoot(t)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cannot build the command: %v: %s", err, output)
	}
	return binary
}

func TestBinaryReportsTheFindingsAndTheExitCode(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	binary := buildBinary(t)
	command := exec.Command(binary, "--root", root, "--format", "json", "audit", "--diff", "--worktree")
	command.Dir = root
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()

	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("the command failed: %v", err)
		}
		exitCode = exitError.ExitCode()
	}
	if exitCode != 3 {
		t.Fatalf("the exit code is %d, want 3: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CDX-SUP-001") {
		t.Errorf("the document does not report the suppression:\n%s", stdout.String())
	}
	if ansiSequence(stdout.String()) {
		t.Errorf("the document carries an escape sequence:\n%q", stdout.String())
	}
}

// TestBinaryDoesNotWriteOutsideOfTheRepository checks that the binary
// leaves no file behind in the current directory.
func TestBinaryDoesNotWriteOutsideOfTheRepository(t *testing.T) {
	root := newRepository(t)
	binary := buildBinary(t)

	command := exec.Command(binary, "--root", root, "init")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("the command failed: %v: %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(root, ".codixai.yml")); err != nil {
		t.Fatalf("the configuration file was not created: %v", err)
	}
}
