package index

import (
	"reflect"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func boolPtr(v bool) *bool { return &v }

func TestNormalizeEvidenceSortsAndDedupsConservatively(t *testing.T) {
	in := []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:10:1", Detail: "call b", Method: "ast"},
		{Kind: domain.EvidenceASTCall, Ref: "a.go:10:1", Detail: "call b", Method: "ast"},
		{Kind: domain.EvidenceASTCall, Ref: "a.go:10:1", Detail: "different detail", Method: "ast"},
		{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:X", Detail: "cand", Method: "heuristic"},
	}
	got := NormalizeEvidence(in)
	want := []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:10:1", Detail: "call b", Method: "ast"},
		{Kind: domain.EvidenceASTCall, Ref: "a.go:10:1", Detail: "different detail", Method: "ast"},
		{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:X", Detail: "cand", Method: "heuristic"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeEvidence() = %#v, want %#v", got, want)
	}
}

func TestNormalizeCandidatesSortsAndDedups(t *testing.T) {
	in := []domain.SymbolRef{
		{Name: "B", File: "b.go", Line: 2, Language: "go"},
		{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go"},
		{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go"},
		{Name: "A", File: "a.go", Line: 5, Column: 2, Kind: "function", Language: "go"},
	}
	got := NormalizeCandidates(in)
	want := []domain.SymbolRef{
		{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go"},
		{Name: "A", File: "a.go", Line: 5, Column: 2, Kind: "function", Language: "go"},
		{Name: "B", File: "b.go", Line: 2, Language: "go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeCandidates() = %#v, want %#v", got, want)
	}
}

func TestNormalizeReferencesDedupsSemanticDuplicatesButPreservesDistinct(t *testing.T) {
	base := domain.Reference{
		Source:     domain.SymbolRef{Name: "Caller", File: "a.go", Line: 10, Column: 1, Language: "go", Container: "pkg.F"},
		Target:     &domain.SymbolRef{Name: "Callee", File: "b.go", Line: 3, Kind: "function", Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 10, StartColumn: 5},
		Evidence: []domain.ReferenceEvidence{
			{Kind: domain.EvidenceASTCall, Ref: "a.go:10:5", Detail: "call", Method: "ast"},
			{Kind: domain.EvidenceASTCall, Ref: "a.go:10:5", Detail: "call", Method: "ast"},
		},
	}
	dup := base
	dup.Evidence = []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:10:5", Detail: "call", Method: "ast"}}

	diffLoc := base
	diffLoc.Location.StartColumn = 9
	diffLoc.Source.Column = 9
	diffLoc.Evidence = []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:10:9", Detail: "call", Method: "ast"}}

	diffKind := base
	diffKind.Kind = domain.ReferenceMethod

	diffTarget := base
	diffTarget.Target = &domain.SymbolRef{Name: "Other", File: "c.go", Line: 1, Kind: "function", Language: "go"}

	diffCandidates := base
	diffCandidates.Resolution = domain.ResolutionAmbiguous
	diffCandidates.Target = nil
	diffCandidates.Candidates = []domain.SymbolRef{{Name: "Callee", File: "b.go", Line: 3, Kind: "function", Language: "go"}}

	diffCandidates2 := base
	diffCandidates2.Resolution = domain.ResolutionAmbiguous
	diffCandidates2.Target = nil
	diffCandidates2.Candidates = []domain.SymbolRef{{Name: "Callee2", File: "d.go", Line: 1, Kind: "function", Language: "go"}}

	refs := []domain.Reference{diffLoc, base, diffCandidates2, diffKind, dup, diffTarget, diffCandidates}
	got := NormalizeReferences(refs)
	if len(got) != 6 {
		t.Fatalf("len(NormalizeReferences()) = %d, want 6 (semantic duplicate removed only): %#v", len(got), got)
	}
	// must be deterministically sorted
	sorted := SortReferences(got)
	if !reflect.DeepEqual(got, sorted) {
		t.Fatalf("NormalizeReferences not sorted: got %#v want %#v", got, sorted)
	}
	// evidence normalized inside retained refs
	for _, ref := range got {
		if ref.Location.Path == base.Location.Path && ref.Location.StartColumn == base.Location.StartColumn && ref.Kind == base.Kind && ref.Target != nil && ref.Target.Name == "Callee" {
			if len(ref.Evidence) != 1 {
				t.Fatalf("expected normalized evidence on retained duplicate, got %#v", ref.Evidence)
			}
		}
	}
}
