package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/cli"
)

// directoryDocument parses the JSON document of a directory audit.
func directoryDocument(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("cannot parse the document: %v\n%s", err, stdout)
	}
	return document
}

// TestAuditDirectoryWithoutGitProducesTheDocument checks the end to end
// contract of a directory audit: no Git repository is needed, the counters
// describe files, the not applicable rule is declared and the evidence is a
// source range.
func TestAuditDirectoryWithoutGitProducesTheDocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport \"github.com/missing/package\"\n\nfunc main() {}\n")
	writeFile(t, root, "notes.txt", "hello\n")

	code, stdout, stderr := execute(t, "--format", "json", "audit", root)
	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}

	document := directoryDocument(t, stdout)
	if document["origin"] != "directory" {
		t.Errorf("the origin is %v, want directory", document["origin"])
	}
	revision, ok := document["revision"].(map[string]any)
	if !ok {
		t.Fatalf("the revision is %v, want an object", document["revision"])
	}
	if len(revision) != 0 {
		t.Errorf("the revision invents a value: %+v", revision)
	}

	files, ok := document["files"].(map[string]any)
	if !ok {
		t.Fatalf("the document does not publish the file statistics: %v", document["files"])
	}
	if files["discovered"] != float64(3) {
		t.Errorf("discovered is %v, want 3", files["discovered"])
	}
	if files["analyzed"] != float64(1) {
		t.Errorf("analyzed is %v, want 1", files["analyzed"])
	}
	if files["skipped"] != float64(2) {
		t.Errorf("skipped is %v, want 2", files["skipped"])
	}

	missing, ok := document["missing_rules"].([]any)
	if !ok || len(missing) == 0 {
		t.Fatalf("the document does not declare the not applicable rules: %v", document["missing_rules"])
	}
	declared := false
	for _, item := range missing {
		rule, _ := item.(map[string]any)
		if rule["id"] == "CDX-TST-001" {
			declared = true
		}
	}
	if !declared {
		t.Errorf("the missing rules are %v, want CDX-TST-001", missing)
	}

	findings, ok := document["findings"].([]any)
	if !ok || len(findings) != 1 {
		t.Fatalf("the findings are %v, want one", document["findings"])
	}
	finding, _ := findings[0].(map[string]any)
	if finding["rule_id"] != "CDX-IMP-001" {
		t.Errorf("the rule is %v, want CDX-IMP-001", finding["rule_id"])
	}
	evidence, ok := finding["evidence"].([]any)
	if !ok || len(evidence) == 0 {
		t.Fatalf("the finding has no evidence: %v", finding["evidence"])
	}
	item, _ := evidence[0].(map[string]any)
	if item["kind"] != "source_range" {
		t.Errorf("the evidence kind is %v, want source_range", item["kind"])
	}
	if ref, _ := item["ref"].(string); strings.Contains(ref, "hunk:") {
		t.Errorf("the evidence mentions a diff hunk: %q", ref)
	}
}

// TestAuditDirectoryTextReportDeclaresTheNotApplicableRule checks the text
// report of a directory audit.
func TestAuditDirectoryTextReportDeclaresTheNotApplicableRule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	code, stdout, stderr := execute(t, "audit", root)
	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stdout, "directory") {
		t.Errorf("the report does not declare the origin:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Rules not applicable") || !strings.Contains(stdout, "CDX-TST-001") {
		t.Errorf("the report does not declare the not applicable rule:\n%s", stdout)
	}
	if strings.Contains(stdout, "hunk:") {
		t.Errorf("the report mentions a diff hunk:\n%s", stdout)
	}
}

// TestAuditDirectoryRejectsAPositionalArgumentWithDiff checks that the two
// modes are never mixed silently.
func TestAuditDirectoryRejectsAPositionalArgumentWithDiff(t *testing.T) {
	root := t.TempDir()
	code, stdout, stderr := execute(t, "audit", "--diff", "--worktree", root)
	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
}

// TestAuditDirectoryJSONKeepsTheDiffModeIntact checks that the directory mode
// did not change the document of the diff mode.
func TestAuditDirectoryJSONKeepsTheDiffModeIntact(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "audit", "--diff", "--worktree")
	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	document := directoryDocument(t, stdout)
	if document["origin"] != "worktree" {
		t.Errorf("the origin is %v, want worktree", document["origin"])
	}
	if _, ok := document["missing_rules"]; ok {
		t.Error("the diff document declares missing rules")
	}
	if _, ok := document["files"]; ok {
		t.Error("the diff document publishes directory file statistics")
	}
}

// TestAuditDirectoryDeclaresItsLimitationsInBothFormats checks that the
// limitations of the mode reach the user in the text report and in the JSON
// document, so a limitation is never only a source comment.
func TestAuditDirectoryDeclaresItsLimitationsInBothFormats(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	fragments := []string{"respect_gitignore", "Symbolic links to directories", "time-of-check to time-of-use"}

	code, stdout, stderr := execute(t, "audit", root)
	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	for _, fragment := range fragments {
		if !strings.Contains(stdout, fragment) {
			t.Errorf("the text report does not declare %q:\n%s", fragment, stdout)
		}
	}

	code, stdout, stderr = execute(t, "--format", "json", "audit", root)
	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	document := directoryDocument(t, stdout)
	limitations, ok := document["limitations"].([]any)
	if !ok || len(limitations) == 0 {
		t.Fatalf("the document publishes no limitations: %v", document["limitations"])
	}
	for _, fragment := range fragments {
		found := false
		for _, item := range limitations {
			text, _ := item.(string)
			if strings.Contains(text, fragment) {
				found = true
			}
		}
		if !found {
			t.Errorf("the JSON document does not declare %q: %v", fragment, limitations)
		}
	}
}
