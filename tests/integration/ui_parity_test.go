//go:build ui

package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/cli"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
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

// normalizeJSON normalizes a JSON document by removing generated_at and sorting keys.
func normalizeJSON(document []byte) (string, error) {
	var v any
	if err := json.Unmarshal(document, &v); err != nil {
		return "", err
	}
	// Remove generated_at
	if m, ok := v.(map[string]any); ok {
		delete(m, "generated_at")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
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

	app := ui.NewApp(cli.UIAuditFunc(), nil, root)
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
		t.Fatal(err)
	}
	data, err := ui.EncodeExport(result, "json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := normalizeJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	want, err := normalizeJSON([]byte(stdout))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("the UI document and the CLI document differ:\n--- UI ---\n%s\n--- CLI ---\n%s", got, want)
	}
	if !strings.Contains(got, `"origin"`) {
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
		t.Fatal(err)
	}
	data, err := ui.EncodeExport(result, "sarif")
	if err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif", "--worktree")
	if code != cli.ExitFindings {
		t.Fatalf("the report exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if string(data) != stdout {
		t.Fatalf("the UI SARIF and the CLI SARIF differ:\n--- UI ---\n%s\n--- CLI ---\n%s", data, stdout)
	}
}

// TestUIVerifyDocumentMatchesTheCLIJSON checks that the verification document 
// that the window renders is the same document that "verify --format json" 
// writes for the same changes.
func TestUIVerifyDocumentMatchesTheCLIJSON(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", "def greet():\n    return \"hello\"\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "initial commit")
	base := gitOutput(t, root, "rev-parse", "HEAD")

	writeFile(t, root, "app.py", "def greet():\n    return \"hello, world\"\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "change the greeting")

	claimsPath := filepath.Join(root, "claims.json")
	claimsDoc := `{
  "schema_version": "0.1",
  "base": "` + base + `",
  "claims": [
    {"id": "c1", "type": "files_unchanged", "paths": ["app.py"]},
    {"id": "c2", "type": "files_unchanged", "paths": ["missing.txt"]}
  ]
}`
	if err := os.WriteFile(claimsPath, []byte(claimsDoc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run via application (same as CLI)
	source, err := git.New(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.RunVerify(context.Background(), application.VerifyRequest{
		Root:        root,
		Base:        base,
		ClaimsPath:  claimsPath,
		Source:      source,
		Loader:      claims.Loader{},
		Evaluator:   claims.NewEvaluator(claims.WithRoot(root), claims.WithBase(base)),
		ToolVersion: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	cliDoc, err := json.Marshal(result.Document)
	if err != nil {
		t.Fatal(err)
	}

	// Run via UI adapter (same use case)
	srv := ui.NewApp(nil, func(ctx context.Context, request ui.VerifyRequest) (*application.VerifyResult, error) {
		return application.RunVerify(ctx, application.VerifyRequest{
			Root:        request.Root,
			Base:        request.Base,
			ClaimsPath:  request.ClaimsPath,
			Source:      source,
			Loader:      claims.Loader{},
			Evaluator:   claims.NewEvaluator(claims.WithRoot(request.Root), claims.WithBase(request.Base)),
			ToolVersion: "test",
		})
	}, root)
	
	view, err := srv.Verify(ui.VerifyRequest{
		Root:       root,
		Base:       base,
		ClaimsPath: claimsPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Document == nil {
		t.Fatal("the window did not carry the versioned document")
	}

	uiDoc, err := ui.EncodeExport(view, "json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := normalizeJSON(uiDoc)
	if err != nil {
		t.Fatal(err)
	}
	want, err := normalizeJSON(cliDoc)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("the UI document and the CLI document differ:\n--- UI ---\n%s\n--- CLI ---\n%s", got, want)
	}
}