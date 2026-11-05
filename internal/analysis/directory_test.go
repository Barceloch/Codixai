package analysis_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/source"
)

// writeFile writes a file below root, creating its directories.
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

// analyzeDirectory discovers and analyzes a directory with the real engine.
func analyzeDirectory(t *testing.T, root string, options ...analysis.Option) (*domain.AnalysisRun, *source.FileSetInfo) {
	t.Helper()
	ds, err := source.NewDirectorySource(root, nil)
	if err != nil {
		t.Fatalf("cannot open the directory source: %v", err)
	}
	set, err := ds.Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	run, err := analysis.NewDirectoryAnalyzer(options...).AnalyzeDirectory(context.Background(), set)
	if err != nil {
		t.Fatalf("the analysis failed: %v", err)
	}
	return run, set
}

// findingsOf returns the findings of a rule.
func findingsOf(run *domain.AnalysisRun, ruleID string) []domain.Finding {
	var found []domain.Finding
	for _, finding := range run.Findings {
		if finding.RuleID == ruleID {
			found = append(found, finding)
		}
	}
	return found
}

// TestAnalyzeDirectoryDeclaresTheNotApplicableRule checks that a rule that
// needs a diff is declared as not applicable and is not published as an
// available rule.
func TestAnalyzeDirectoryDeclaresTheNotApplicableRule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	run, _ := analyzeDirectory(t, root)

	found := false
	for _, rule := range run.MissingRules {
		if rule.ID == "CDX-TST-001" {
			found = true
			if rule.Reason == "" {
				t.Error("the not applicable rule has no reason")
			}
		}
	}
	if !found {
		t.Fatalf("the missing rules are %+v, want CDX-TST-001", run.MissingRules)
	}
	for _, rule := range run.Rules {
		if rule.ID == "CDX-TST-001" {
			t.Error("a rule that needs a diff is published as an available rule")
		}
	}
}

// TestAnalyzeDirectoryCountsFilesNotRules checks that the counters describe
// files, not rule executions.
func TestAnalyzeDirectoryCountsFilesNotRules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, root, "helper.go", "package main\n\nfunc helper() {}\n")

	run, set := analyzeDirectory(t, root)

	if len(set.Files) != 2 {
		t.Fatalf("the file set holds %d files, want 2", len(set.Files))
	}
	if run.Files == nil {
		t.Fatal("the run does not publish the file statistics")
	}
	if run.Files.Analyzed != 2 {
		t.Errorf("analyzed is %d, want 2", run.Files.Analyzed)
	}
	if run.Files.Discovered != 2 {
		t.Errorf("discovered is %d, want 2", run.Files.Discovered)
	}
	if run.Files.Skipped != 0 {
		t.Errorf("skipped is %d, want 0", run.Files.Skipped)
	}
}

// TestAnalyzeDirectoryUsesSourceRangeEvidence checks that a finding of a full
// file analysis does not pretend to belong to a diff hunk.
func TestAnalyzeDirectoryUsesSourceRangeEvidence(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport \"github.com/missing/package\"\n\nfunc main() {}\n")

	run, _ := analyzeDirectory(t, root)

	findings := findingsOf(run, "CDX-IMP-001")
	if len(findings) != 1 {
		t.Fatalf("the findings of CDX-IMP-001 are %+v, want one", findings)
	}
	if len(findings[0].Evidence) == 0 {
		t.Fatal("the finding has no evidence")
	}
	evidence := findings[0].Evidence[0]
	if evidence.Kind != "source_range" {
		t.Errorf("the evidence kind is %q, want %q", evidence.Kind, "source_range")
	}
	if strings.Contains(evidence.Ref, "hunk:") {
		t.Errorf("the evidence of a directory analysis mentions a diff hunk: %q", evidence.Ref)
	}
	if run.Origin != string(domain.OriginDirectory) {
		t.Errorf("the origin is %q, want %q", run.Origin, domain.OriginDirectory)
	}
	if run.Revision != (domain.Revision{}) {
		t.Errorf("the run invents a revision: %+v", run.Revision)
	}
}

// TestAnalyzeDirectoryHonorsTheToolVersion checks that the option of the
// engine reaches the document.
func TestAnalyzeDirectoryHonorsTheToolVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")

	run, _ := analyzeDirectory(t, root, analysis.WithToolVersion("9.9.9"))

	if run.ToolVersion != "9.9.9" {
		t.Errorf("the tool version is %q, want %q", run.ToolVersion, "9.9.9")
	}
}

// TestAnalyzeDirectoryDetectsASingleLineImport checks that a one line import
// of a dependency that no manifest declares is reported.
func TestAnalyzeDirectoryDetectsASingleLineImport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport \"github.com/missing/package\"\n\nfunc main() {}\n")

	run, _ := analyzeDirectory(t, root)

	findings := findingsOf(run, "CDX-IMP-001")
	if len(findings) != 1 {
		t.Fatalf("the findings of CDX-IMP-001 are %+v, want one", findings)
	}
	if findings[0].Location.StartLine != 3 {
		t.Errorf("the finding is on line %d, want 3", findings[0].Location.StartLine)
	}
	if findings[0].EpistemicStatus != domain.EpistemicObserved {
		t.Errorf("the epistemic status is %q, want %q", findings[0].EpistemicStatus, domain.EpistemicObserved)
	}
}

// TestAnalyzeDirectoryDetectsAnImportBlock checks the same rule over an import
// block.
func TestAnalyzeDirectoryDetectsAnImportBlock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport (\n\t\"fmt\"\n\t\"github.com/missing/package\"\n)\n\nfunc main() { fmt.Println(\"x\") }\n")

	run, _ := analyzeDirectory(t, root)

	findings := findingsOf(run, "CDX-IMP-001")
	if len(findings) != 1 {
		t.Fatalf("the findings of CDX-IMP-001 are %+v, want one", findings)
	}
	if findings[0].Location.StartLine != 5 {
		t.Errorf("the finding is on line %d, want 5", findings[0].Location.StartLine)
	}
}

// TestAnalyzeDirectoryIgnoresDeclaredAndStandardImports checks the negative
// cases: a declared dependency and a standard library package are not
// findings.
func TestAnalyzeDirectoryIgnoresDeclaredAndStandardImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n\nrequire github.com/declared/package v1.0.0\n")
	writeFile(t, root, "main.go", "package main\n\nimport (\n\t\"fmt\"\n\t\"github.com/declared/package\"\n)\n\nfunc main() { fmt.Println(\"x\") }\n")

	run, _ := analyzeDirectory(t, root)

	if findings := findingsOf(run, "CDX-IMP-001"); len(findings) != 0 {
		t.Errorf("a valid import produced findings: %+v", findings)
	}
}

// TestAnalyzeDirectoryIgnoresUnsupportedExtensions checks that a file whose
// language no rule supports is not analyzed and is counted as skipped.
func TestAnalyzeDirectoryIgnoresUnsupportedExtensions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "README.md", "# title\n")

	run, _ := analyzeDirectory(t, root)

	if run.Files.Discovered != 2 {
		t.Errorf("discovered is %d, want 2", run.Files.Discovered)
	}
	if run.Files.Skipped != 1 {
		t.Errorf("skipped is %d, want 1", run.Files.Skipped)
	}
	if run.Files.Analyzed != 1 {
		t.Errorf("analyzed is %d, want 1", run.Files.Analyzed)
	}
}

// TestAnalyzeDirectoryRulesDoNotPublishTheDiffOnlyRule checks that an
// analyzer implements its Rules method with the applicable catalog.
func TestAnalyzeDirectoryRulesDoNotPublishTheDiffOnlyRule(t *testing.T) {
	analyzer := analysis.NewDirectoryAnalyzer()
	rules := analyzer.Rules()
	if len(rules) == 0 {
		t.Fatal("the analyzer publishes no rule")
	}
	for _, rule := range rules {
		if rule.ID == "CDX-TST-001" {
			t.Error("the catalog publishes a rule that needs a diff")
		}
	}
}

// containsLimitation reports whether a limitation contains the fragment.
func containsLimitation(limitations []string, fragment string) bool {
	for _, limitation := range limitations {
		if strings.Contains(limitation, fragment) {
			return true
		}
	}
	return false
}

// TestAnalyzeDirectoryDeclaresItsLimitations checks that the run publishes the
// limitations of the mode, including the ones that a reader cannot infer from
// the findings: the ignored .gitignore, the linked directories and the
// time-of-check to time-of-use race.
func TestAnalyzeDirectoryDeclaresItsLimitations(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")

	run, _ := analyzeDirectory(t, root)

	required := map[string]string{
		"no execution":     "does not execute any code",
		"absence":          "does not prove that the directory is correct",
		"not applicable":   "CDX-TST-001",
		"gitignore":        "respect_gitignore",
		"linked directory": "Symbolic links to directories are not traversed",
		"time of check":    "time-of-check to time-of-use",
	}
	for name, fragment := range required {
		if !containsLimitation(run.Limitations, fragment) {
			t.Errorf("the limitation %q is missing: %v", name, run.Limitations)
		}
	}
}
