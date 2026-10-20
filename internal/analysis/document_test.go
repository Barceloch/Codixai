package analysis_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codixai/internal/analysis"
	"codixai/internal/domain"
	"codixai/internal/integrations/git"
)

// TestRunIsSerializable checks that the run can be exported as JSON and that
// the document holds the fields that the contract requires.
func TestRunIsSerializable(t *testing.T) {
	diffPath := filepath.Join(fixtureRoot(), "CDX-SUP-001", "go", "positive", "nolint.diff")
	file, err := os.Open(diffPath)
	if err != nil {
		t.Fatalf("the diff cannot be opened: %v", err)
	}
	defer file.Close()
	changes, err := git.ParseUnifiedDiff(file)
	if err != nil {
		t.Fatalf("the diff cannot be parsed: %v", err)
	}
	root, err := filepath.Abs(filepath.Dir(diffPath))
	if err != nil {
		t.Fatalf("the directory of the case cannot be resolved: %v", err)
	}
	changes.Root = root
	changes.Origin = domain.OriginBase

	engine := analysis.New(analysis.WithClock(fixedClock()))
	run, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	document, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatalf("the run cannot be serialized: %v", err)
	}
	text := string(document)
	for _, field := range []string{
		`"schema_version": "0.1"`,
		`"tool_version"`,
		`"generated_at": "2026-01-02T03:04:05Z"`,
		`"origin": "base"`,
		`"revision"`,
		`"findings"`,
		`"summary"`,
		`"rules"`,
		`"limitations"`,
		`"rule_id": "CDX-SUP-001"`,
		`"rule_version": "1"`,
		`"severity": "warning"`,
		`"epistemic_status": "observed"`,
		`"language": "go"`,
		`"support"`,
		`"level": "syntactic"`,
		`"recommendation"`,
		`"kind": "diff_hunk"`,
	} {
		if !strings.Contains(text, field) {
			t.Errorf("the document does not hold %s:\n%s", field, text)
		}
	}
}

// TestRunIsDeterministic checks that two runs over the same change set
// produce the same document.
func TestRunIsDeterministic(t *testing.T) {
	diffPath := filepath.Join(fixtureRoot(), "CDX-TST-001", "python", "positive", "skipped_test.diff")
	file, err := os.Open(diffPath)
	if err != nil {
		t.Fatalf("the diff cannot be opened: %v", err)
	}
	defer file.Close()
	changes, err := git.ParseUnifiedDiff(file)
	if err != nil {
		t.Fatalf("the diff cannot be parsed: %v", err)
	}
	engine := analysis.New(analysis.WithClock(fixedClock()))
	first, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	second, err := engine.Analyze(context.Background(), changes)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	firstDocument, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("the run cannot be serialized: %v", err)
	}
	secondDocument, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("the run cannot be serialized: %v", err)
	}
	if string(firstDocument) != string(secondDocument) {
		t.Errorf("two runs produced different documents:\n%s\n%s", firstDocument, secondDocument)
	}
}
