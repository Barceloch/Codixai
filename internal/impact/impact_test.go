package impact

import (
	"context"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func TestAnalyzeImpact(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	changes := &domain.ChangeSet{
		Origin: domain.OriginBase,
		Base:   "main",
		Head:   "abc123",
		Files: []domain.FileChange{
			{
				Path:     "internal/foo.go",
				Status:   domain.FileModified,
				Language: domain.LangGo,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, Content: "func newFunc() {}"},
							{Kind: domain.LineRemoved, Content: "func oldFunc() {}"},
						},
					},
				},
			},
			{
				Path:     "internal/bar.py",
				Status:   domain.FileAdded,
				Language: domain.LangPython,
				Hunks: []domain.Hunk{
					{
						Lines: []domain.DiffLine{
							{Kind: domain.LineAdded, Content: "def hello():\n    pass"},
						},
					},
				},
			},
		},
	}

	result, err := analyzer.AnalyzeImpact(context.Background(), changes)
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}

	// Should have affected findings for each changed file
	if len(result.AffectedFindings) != 2 {
		t.Errorf("expected 2 affected findings, got %d", len(result.AffectedFindings))
	}

	for _, af := range result.AffectedFindings {
		if af.ImpactType != "direct" {
			t.Errorf("expected impact type 'direct', got %s", af.ImpactType)
		}
	}

	// Should have impacted files
	if len(result.Impacted) != 2 {
		t.Errorf("expected 2 impacted files, got %d", len(result.Impacted))
	}

	// Likely failing tests may be empty (heuristic)
	// Just check it doesn't panic
	_ = result.LikelyFailingTests
	_ = result.RecommendedTests
}

func TestAnalyzeImpactEmpty(t *testing.T) {
	analyzer := NewAnalyzer(nil)

	// Empty changes
	changes := &domain.ChangeSet{
		Files: []domain.FileChange{},
	}

	result, err := analyzer.AnalyzeImpact(context.Background(), changes)
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}

	if len(result.AffectedFindings) != 0 {
		t.Errorf("expected 0 affected findings, got %d", len(result.AffectedFindings))
	}
	if len(result.Impacted) != 0 {
		t.Errorf("expected 0 impacted, got %d", len(result.Impacted))
	}

	// Nil changes
	result, err = analyzer.AnalyzeImpact(context.Background(), nil)
	if err != nil {
		t.Fatalf("AnalyzeImpact failed: %v", err)
	}
	if len(result.AffectedFindings) != 0 {
		t.Errorf("expected 0 affected findings for nil, got %d", len(result.AffectedFindings))
	}
}