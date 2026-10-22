package claims

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

// change builds a change set with the given files.
func change(files ...domain.FileChange) *domain.ChangeSet {
	return &domain.ChangeSet{Files: files}
}

// file builds a changed file.
func file(path, status string) domain.FileChange {
	return domain.FileChange{Path: path, Status: status}
}

// evaluate runs the evaluator over one claim and returns the result.
func evaluate(t *testing.T, root string, changes *domain.ChangeSet, claim domain.Claim, head string) domain.ClaimResult {
	t.Helper()
	file := &File{SchemaVersion: SchemaVersion, Claims: []domain.Claim{claim}}
	results, err := NewEvaluator(WithRoot(root)).Evaluate(context.Background(), changes, file, head)
	if err != nil {
		t.Fatalf("Evaluate returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results: got %d, want 1", len(results))
	}
	return results[0]
}

// requireStatus checks the status of a result.
func requireStatus(t *testing.T, result domain.ClaimResult, status string) {
	t.Helper()
	if result.Status != status {
		t.Errorf("status: got %q, want %q (message: %s)", result.Status, status, result.Message)
	}
}

func TestEvaluateFilesUnchanged(t *testing.T) {
	changes := change(file("internal/billing/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeFilesUnchanged,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
	if !strings.Contains(result.Message, "no changed file matches") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateFilesUnchangedIsContradictedByAChange(t *testing.T) {
	changes := change(file("internal/billing/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeFilesUnchanged,
		Paths: []string{"internal/billing/**"},
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
	if len(result.Evidence) == 0 {
		t.Error("expected the result to carry evidence")
	}
}

func TestEvaluateFilesUnchangedIsUnresolvedWithAnEmptyDiff(t *testing.T) {
	result := evaluate(t, "", change(), domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeFilesUnchanged,
		Paths: []string{"internal/billing/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "diff is empty") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateScope(t *testing.T) {
	changes := change(file("internal/billing/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeScope,
		Paths: []string{"internal/**"},
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
}

func TestEvaluateScopeIsContradictedByAChangeOutsideThePaths(t *testing.T) {
	changes := change(
		file("internal/billing/service.go", domain.FileModified),
		file("cmd/codixai/main.go", domain.FileModified),
	)
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeScope,
		Paths: []string{"internal/**"},
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
	if !strings.Contains(result.Message, "cmd/codixai/main.go") {
		t.Errorf("the message does not name the file: %s", result.Message)
	}
}

func TestEvaluateScopeIsUnresolvedWithAnEmptyDiff(t *testing.T) {
	result := evaluate(t, "", change(), domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeScope,
		Paths: []string{"internal/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateTestsAdded(t *testing.T) {
	changes := change(file("internal/payment/payment_test.go", domain.FileAdded))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsAdded,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
}

func TestEvaluateTestsAddedIsContradictedByADeletedTestFile(t *testing.T) {
	changes := change(file("internal/payment/payment_test.go", domain.FileDeleted))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsAdded,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
}

func TestEvaluateTestsAddedIsUnresolvedWithoutAChangedFile(t *testing.T) {
	changes := change(file("internal/billing/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsAdded,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateTestsAddedIsUnresolvedWithoutATestFile(t *testing.T) {
	changes := change(file("internal/payment/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsAdded,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "no test file") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateTestsModified(t *testing.T) {
	changes := change(file("internal/payment/payment_test.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsModified,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
}

func TestEvaluateTestsModifiedIsContradictedByANewTestFile(t *testing.T) {
	changes := change(file("internal/payment/payment_test.go", domain.FileAdded))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsModified,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
}

func TestEvaluateTestsModifiedIsUnresolvedWithoutAChangedFile(t *testing.T) {
	changes := change(file("internal/billing/service.go", domain.FileModified))
	result := evaluate(t, "", changes, domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeTestsModified,
		Paths: []string{"internal/payment/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateDependencyUnchanged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	changes := change(file("internal/payment/service.go", domain.FileModified))
	result := evaluate(t, root, changes, domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeDependencyUnchanged,
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
}

func TestEvaluateDependencyUnchangedIsContradictedByAManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	changes := change(file("go.mod", domain.FileModified))
	result := evaluate(t, root, changes, domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeDependencyUnchanged,
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
}

func TestEvaluateDependencyUnchangedIsUnresolvedWithoutAManifest(t *testing.T) {
	result := evaluate(t, t.TempDir(), change(), domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeDependencyUnchanged,
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateDependencyAdded(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\nrequire github.com/acme/lib v1.2.3\n")
	changes := change(file("go.mod", domain.FileModified))
	result := evaluate(t, root, changes, domain.Claim{
		ID:           "c1",
		Type:         domain.ClaimTypeDependencyAdded,
		Dependencies: []string{"github.com/acme/lib"},
	}, "")
	requireStatus(t, result, domain.ClaimVerified)
}

func TestEvaluateDependencyAddedIsContradictedByAnAbsentDependency(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\nrequire github.com/acme/other v1.2.3\n")
	changes := change(file("go.mod", domain.FileModified))
	result := evaluate(t, root, changes, domain.Claim{
		ID:           "c1",
		Type:         domain.ClaimTypeDependencyAdded,
		Dependencies: []string{"github.com/acme/lib"},
	}, "")
	requireStatus(t, result, domain.ClaimContradicted)
	if !strings.Contains(result.Message, "github.com/acme/lib") {
		t.Errorf("the message does not name the dependency: %s", result.Message)
	}
}

func TestEvaluateDependencyAddedIsUnresolvedWithoutADependency(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	changes := change(file("go.mod", domain.FileModified))
	result := evaluate(t, root, changes, domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeDependencyAdded,
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateDependencyAddedIsUnresolvedWithoutAManifest(t *testing.T) {
	result := evaluate(t, t.TempDir(), change(), domain.Claim{
		ID:           "c1",
		Type:         domain.ClaimTypeDependencyAdded,
		Dependencies: []string{"github.com/acme/lib"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateAPIUnchangedIsAlwaysUnresolved(t *testing.T) {
	result := evaluate(t, "", change(file("internal/payment/service.go", domain.FileModified)), domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeAPIUnchanged,
		Paths: []string{"internal/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "public API") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateBehaviorPreservedIsAlwaysUnresolved(t *testing.T) {
	result := evaluate(t, "", change(file("internal/payment/service.go", domain.FileModified)), domain.Claim{
		ID:    "c1",
		Type:  domain.ClaimTypeBehaviorPreserved,
		Paths: []string{"internal/**"},
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "behavior") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateTestsPassedWithALinkedReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reports", "go-test.json"), goTestSuccess)
	result := evaluate(t, root, change(), domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeTestsPassed,
		// The claim declares the revision of the compared range, so the
		// report is linked even when it declares none.
		Report:         "reports/go-test.json",
		ReportRevision: "9f2c1ab",
	}, "9f2c1ab")
	requireStatus(t, result, domain.ClaimVerified)
	if len(result.Evidence) != 1 {
		t.Fatalf("evidence: got %d, want 1", len(result.Evidence))
	}
	if result.Evidence[0].Kind != "external_report" {
		t.Errorf("evidence kind: got %q, want %q", result.Evidence[0].Kind, "external_report")
	}
	if !strings.Contains(result.Evidence[0].Detail, "outside Codixai") {
		t.Errorf("the evidence does not state the provenance: %s", result.Evidence[0].Detail)
	}
}

func TestEvaluateTestsPassedIsContradictedByAFailingReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reports", "go-test.json"), goTestFailure)
	result := evaluate(t, root, change(), domain.Claim{
		ID:             "c1",
		Type:           domain.ClaimTypeTestsPassed,
		Report:         "reports/go-test.json",
		ReportRevision: "9f2c1ab",
	}, "9f2c1ab")
	requireStatus(t, result, domain.ClaimContradicted)
}

func TestEvaluateTestsPassedIsUnresolvedWithAMissingReport(t *testing.T) {
	root := t.TempDir()
	result := evaluate(t, root, change(), domain.Claim{
		ID:             "c1",
		Type:           domain.ClaimTypeTestsPassed,
		Report:         "reports/go-test.json",
		ReportRevision: "9f2c1ab",
	}, "9f2c1ab")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "cannot be read") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateTestsPassedIsUnresolvedWithAnUnknownFormat(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reports", "report.txt"), "everything is fine")
	result := evaluate(t, root, change(), domain.Claim{
		ID:             "c1",
		Type:           domain.ClaimTypeTestsPassed,
		Report:         "reports/report.txt",
		ReportRevision: "9f2c1ab",
	}, "9f2c1ab")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "not supported") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateTestsPassedIsUnresolvedWithAnUnlinkedReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reports", "go-test.json"), goTestSuccess)
	result := evaluate(t, root, change(), domain.Claim{
		ID:     "c1",
		Type:   domain.ClaimTypeTestsPassed,
		Report: "reports/go-test.json",
	}, "9f2c1ab")
	requireStatus(t, result, domain.ClaimUnresolved)
	if !strings.Contains(result.Message, "cannot be linked") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

func TestEvaluateTestsPassedIsUnresolvedWithoutAReport(t *testing.T) {
	result := evaluate(t, t.TempDir(), change(), domain.Claim{
		ID:   "c1",
		Type: domain.ClaimTypeTestsPassed,
	}, "")
	requireStatus(t, result, domain.ClaimUnresolved)
}

func TestEvaluateSortsTheResultsByIdentifier(t *testing.T) {
	file := &File{
		SchemaVersion: SchemaVersion,
		Claims: []domain.Claim{
			{ID: "c3", Type: domain.ClaimTypeFilesUnchanged, Paths: []string{"internal/**"}},
			{ID: "c1", Type: domain.ClaimTypeFilesUnchanged, Paths: []string{"internal/**"}},
			{ID: "c2", Type: domain.ClaimTypeFilesUnchanged, Paths: []string{"internal/**"}},
		},
	}
	results, err := NewEvaluator().Evaluate(context.Background(), change(), file, "")
	if err != nil {
		t.Fatalf("Evaluate returned an error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results: got %d, want 3", len(results))
	}
	for index, id := range []string{"c1", "c2", "c3"} {
		if results[index].Claim.ID != id {
			t.Errorf("results[%d]: got %q, want %q", index, results[index].Claim.ID, id)
		}
	}
}

func TestEvaluateRejectsAnotherBase(t *testing.T) {
	file := &File{
		SchemaVersion: SchemaVersion,
		Base:          "main",
		Claims: []domain.Claim{
			{ID: "c1", Type: domain.ClaimTypeFilesUnchanged, Paths: []string{"internal/**"}},
		},
	}
	changes := &domain.ChangeSet{Base: "develop"}
	_, err := NewEvaluator().Evaluate(context.Background(), changes, file, "")
	if err == nil {
		t.Fatal("expected an error for another base")
	}
	if !strings.Contains(err.Error(), "another revision") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestEvaluateUsesTheRootOfTheChangeSet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reports", "go-test.json"), goTestSuccess)
	changes := &domain.ChangeSet{Root: root}
	file := &File{
		SchemaVersion: SchemaVersion,
		Claims: []domain.Claim{
			{ID: "c1", Type: domain.ClaimTypeTestsPassed, Report: "reports/go-test.json", ReportRevision: "9f2c1ab"},
		},
	}
	results, err := NewEvaluator().Evaluate(context.Background(), changes, file, "9f2c1ab")
	if err != nil {
		t.Fatalf("Evaluate returned an error: %v", err)
	}
	requireStatus(t, results[0], domain.ClaimVerified)
}

func TestFailed(t *testing.T) {
	verified := domain.ClaimResult{Status: domain.ClaimVerified}
	unresolved := domain.ClaimResult{Status: domain.ClaimUnresolved}
	contradicted := domain.ClaimResult{Status: domain.ClaimContradicted}

	if Failed([]domain.ClaimResult{verified, unresolved}, false) {
		t.Error("expected the results not to fail")
	}
	if !Failed([]domain.ClaimResult{verified, unresolved}, true) {
		t.Error("expected an unresolved claim to fail when the policy says so")
	}
	if !Failed([]domain.ClaimResult{contradicted}, false) {
		t.Error("expected a contradicted claim to fail")
	}
}

// writeFile writes a file of the analyzed repository.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", path, err)
	}
}
