//go:build ui

package integration

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/cli"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/ui"
)

// generatedAt matches the timestamp that makes two runs of the same change
// differ. The conventions of the project fix the rest of the document as
// deterministic.
var generatedAt = regexp.MustCompile(`"generated_at": "[^"]*"`)

// normalizeGeneratedAt replaces the timestamp of a document.
func normalizeGeneratedAt(document string) string {
	return generatedAt.ReplaceAllString(document, `"generated_at": "<timestamp>"`)
}

// TestUIAuditDocumentMatchesTheCLIJSON checks that the document that the
// window renders is the same document that "audit --format json" writes for
// the same changes.
//
// The test runs the exact composition of the window (cli.UIAuditFunc) and
// the exact encoder of the export (ui.EncodeExport); it does not copy
// either of them.
func TestUIAuditDocumentMatchesTheCLIJSON(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	code, stdout, stderr := execute(t, "--root", root, "--format", "json", "audit", "--diff", "--worktree")
	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}

	app := ui.NewApp(cli.UIAuditFunc(), root)
	view, err := app.Audit(ui.AuditRequest{Root: root, Origin: string(domain.OriginWorktree), RunID: "run-parity"})
	if err != nil {
		t.Fatalf("the window audit failed: %v", err)
	}
	if view.Document == nil {
		t.Fatal("the window did not carry the versioned document")
	}

	result, err := cli.UIAuditFunc()(context.Background(), ui.AuditRunRequest{
		Root:   root,
		Origin: domain.OriginWorktree,
	})
	if err != nil {
		t.Fatalf("the audit use case failed: %v", err)
	}
	data, err := ui.EncodeExport(result, "json")
	if err != nil {
		t.Fatalf("the JSON export failed: %v", err)
	}

	got := normalizeGeneratedAt(string(data))
	want := normalizeGeneratedAt(stdout)
	if got != want {
		t.Fatalf("the UI document and the CLI document differ:\n--- UI ---\n%s\n--- CLI ---\n%s", got, want)
	}
	if !strings.Contains(got, `"origin": "worktree"`) {
		t.Errorf("the document does not declare the origin:\n%s", got)
	}
}

// TestUIExportSARIFMatchesTheCLIReport checks that the SARIF export of the
// window is the same document that "report --format sarif" writes for the
// same changes.
func TestUIExportSARIFMatchesTheCLIReport(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	result, err := cli.UIAuditFunc()(context.Background(), ui.AuditRunRequest{
		Root:   root,
		Origin: domain.OriginWorktree,
	})
	if err != nil {
		t.Fatalf("the audit use case failed: %v", err)
	}
	data, err := ui.EncodeExport(result, "sarif")
	if err != nil {
		t.Fatalf("the SARIF export failed: %v", err)
	}

	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif", "--worktree")
	if code != cli.ExitFindings {
		t.Fatalf("the report exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if string(data) != stdout {
		t.Fatalf("the UI SARIF and the CLI SARIF differ:\n--- UI ---\n%s\n--- CLI ---\n%s", data, stdout)
	}
}
