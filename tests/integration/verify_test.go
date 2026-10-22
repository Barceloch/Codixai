// Package integration holds the end to end tests of the command line
// interface.
//
// The tests build a real Git repository in a temporary directory, they
// change it and they run the CLI against it. They never modify the
// repository of the project and they never need the network.
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/cli"
)

// gitOutput runs a git command inside the given directory and returns its
// standard output.
func gitOutput(t *testing.T, dir string, args ...string) string {
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
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}

// newVerifiedRepository builds a Git repository with two commits.
//
// The first commit is the base of the comparison and the second one changes
// the file "app.py". The function returns the root of the repository, the
// base revision and the head revision.
func newVerifiedRepository(t *testing.T) (string, string, string) {
	t.Helper()
	root := newRepository(t)
	base := gitOutput(t, root, "rev-parse", "HEAD")
	writeFile(t, root, "app.py", "def greet():\n    return \"hello, world\"\n")
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "change the greeting")
	head := gitOutput(t, root, "rev-parse", "HEAD")
	return root, base, head
}

// writeClaims writes a claim file inside the repository.
func writeClaims(t *testing.T, root, content string) string {
	t.Helper()
	path := filepath.Join(root, "claims.json")
	writeFile(t, root, "claims.json", content)
	return path
}

// claimStates parses the JSON document of a verification and returns the
// state of every claim, keyed by the identifier of the claim.
func claimStates(t *testing.T, stdout string) map[string]string {
	t.Helper()
	var document struct {
		SchemaVersion string `json:"schema_version"`
		Results       []struct {
			Claim struct {
				ID string `json:"id"`
			} `json:"claim"`
			Status string `json:"status"`
		} `json:"results"`
		Summary struct {
			Total    int            `json:"total"`
			ByStatus map[string]int `json:"by_status"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("cannot parse the JSON document: %v\n%s", err, stdout)
	}
	if document.SchemaVersion != "0.1" {
		t.Errorf("the schema version is %q, want 0.1", document.SchemaVersion)
	}
	states := map[string]string{}
	for _, result := range document.Results {
		states[result.Claim.ID] = result.Status
	}
	if document.Summary.Total != len(document.Results) {
		t.Errorf("the summary counts %d claims, want %d", document.Summary.Total, len(document.Results))
	}
	return states
}

func TestVerifyContradictsTheFalseClaimAndVerifiesTheTrueOne(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["app.py"] },
    { "id": "c2", "type": "files_unchanged", "paths": ["docs/**"] }
  ]
}`)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "verify", "--base", base, "--claims", claims)

	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	states := claimStates(t, stdout)
	if states["c1"] != "contradicted" {
		t.Errorf("the state of c1 is %q, want contradicted", states["c1"])
	}
	if states["c2"] != "verified" {
		t.Errorf("the state of c2 is %q, want verified", states["c2"])
	}
}

func TestVerifyJSONOutputIsACleanDocument(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["app.py"] }
  ]
}`)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "verify", "--base", base, "--claims", claims)

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
	for _, key := range []string{"schema_version", "tool_version", "generated_at", "base", "head", "results", "summary", "limitations"} {
		if _, ok := document[key]; !ok {
			t.Errorf("the JSON document does not hold the key %q", key)
		}
	}
}

func TestVerifyExitsWithZeroWhenNoClaimIsContradicted(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["docs/**"] },
    { "id": "c2", "type": "api_unchanged", "paths": ["app.py"] }
  ]
}`)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "verify", "--base", base, "--claims", claims)

	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	states := claimStates(t, stdout)
	if states["c1"] != "verified" {
		t.Errorf("the state of c1 is %q, want verified", states["c1"])
	}
	if states["c2"] != "unresolved" {
		t.Errorf("the state of c2 is %q, want unresolved", states["c2"])
	}
}

func TestVerifyRequiresTheBaseAndTheClaims(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["docs/**"] }
  ]
}`)

	code, _, stderr := execute(t, "--root", root, "verify", "--base", base)
	if code != cli.ExitInvalidArgs {
		t.Errorf("the exit code without --claims is %d, want %d", code, cli.ExitInvalidArgs)
	}
	if !strings.Contains(stderr, "--claims") {
		t.Errorf("the standard error does not mention the missing flag:\n%s", stderr)
	}

	code, _, stderr = execute(t, "--root", root, "verify", "--claims", claims)
	if code != cli.ExitInvalidArgs {
		t.Errorf("the exit code without --base is %d, want %d", code, cli.ExitInvalidArgs)
	}
	if !strings.Contains(stderr, "--base") {
		t.Errorf("the standard error does not mention the missing flag:\n%s", stderr)
	}
}

func TestVerifyLinksAnExternalReportToTheRevision(t *testing.T) {
	root, base, head := newVerifiedRepository(t)
	writeFile(t, root, "reports/go-test.xml", junitReport(head, 0))
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "tests_passed", "report": "reports/go-test.xml" }
  ]
}`)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "verify", "--base", base, "--claims", claims)

	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	states := claimStates(t, stdout)
	if states["c1"] != "verified" {
		t.Errorf("the state of c1 is %q, want verified", states["c1"])
	}
}

func TestVerifyLeavesAnUnlinkedReportUnresolved(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	writeFile(t, root, "reports/go-test.xml", junitReport("0000000000000000000000000000000000000000", 0))
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "tests_passed", "report": "reports/go-test.xml" }
  ]
}`)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "verify", "--base", base, "--claims", claims)

	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	states := claimStates(t, stdout)
	if states["c1"] != "unresolved" {
		t.Errorf("the state of c1 is %q, want unresolved", states["c1"])
	}
}

func TestVerifyDoesNotModifyTheRepository(t *testing.T) {
	root, base, _ := newVerifiedRepository(t)
	claims := writeClaims(t, root, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["app.py"] }
  ]
}`)
	before := gitOutput(t, root, "status", "--porcelain")

	execute(t, "--root", root, "verify", "--base", base, "--claims", claims)

	after := gitOutput(t, root, "status", "--porcelain")
	if before != after {
		t.Errorf("the command changed the repository:\nbefore %q\nafter %q", before, after)
	}
}

// junitReport builds a JUnit XML report that declares a revision.
func junitReport(revision string, failures int) string {
	var document bytes.Buffer
	document.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	document.WriteString("<testsuites>\n")
	document.WriteString("  <testsuite name=\"package\" tests=\"2\" failures=\"" + itoa(failures) + "\" errors=\"0\">\n")
	document.WriteString("    <properties>\n")
	document.WriteString("      <property name=\"revision\" value=\"" + revision + "\"/>\n")
	document.WriteString("    </properties>\n")
	document.WriteString("    <testcase name=\"TestGreet\" classname=\"package\" time=\"0.01\"/>\n")
	document.WriteString("  </testsuite>\n")
	document.WriteString("</testsuites>\n")
	return document.String()
}

// itoa converts an integer into a string.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
