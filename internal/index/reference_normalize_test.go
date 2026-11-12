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

func TestNormalizeReferencesRepeatedNormalizationIsIdempotent(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 2, Language: "go"},
			Evidence: []domain.ReferenceEvidence{
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "C", File: "c.go", Line: 3, Language: "go"},
			Kind:       domain.ReferenceType,
			Resolution: domain.ResolutionUnresolved,
			Language:   "go",
			Location:   domain.Location{Path: "c.go", StartLine: 3, StartColumn: 1},
		},
	}
	first := NormalizeReferences(refs)
	second := NormalizeReferences(first)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("NormalizeReferences not idempotent:\nfirst:  %#v\nsecond: %#v", first, second)
	}
}

func TestNormalizeReferencesInputOrderIndependence(t *testing.T) {
	base := domain.Reference{
		Source:     domain.SymbolRef{Name: "Caller", File: "a.go", Line: 10, Column: 1, Language: "go", Container: "pkg.F"},
		Target:     &domain.SymbolRef{Name: "Callee", File: "b.go", Line: 3, Kind: "function", Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 10, StartColumn: 5},
		Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:10:5", Detail: "call", Method: "ast"}},
	}
	other := domain.Reference{
		Source:     domain.SymbolRef{Name: "X", File: "x.go", Line: 1, Language: "go"},
		Kind:       domain.ReferenceType,
		Resolution: domain.ResolutionUnresolved,
		Language:   "go",
		Location:   domain.Location{Path: "x.go", StartLine: 1, StartColumn: 1},
	}
	third := domain.Reference{
		Source:     domain.SymbolRef{Name: "Y", File: "y.go", Line: 2, Language: "go"},
		Kind:       domain.ReferenceImport,
		Resolution: domain.ResolutionExternal,
		Language:   "go",
		Location:   domain.Location{Path: "y.go", StartLine: 2, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "fmt", Kind: "module", Language: "go"},
	}

	permutations := [][]domain.Reference{
		{base, other, third},
		{base, third, other},
		{other, base, third},
		{other, third, base},
		{third, base, other},
		{third, other, base},
	}

	var reference []domain.Reference
	for i, perm := range permutations {
		got := NormalizeReferences(perm)
		if i == 0 {
			reference = got
			continue
		}
		if !reflect.DeepEqual(reference, got) {
			t.Fatalf("permutation %d produced different result:\nreference: %#v\ngot:       %#v", i, reference, got)
		}
	}
}

func TestNormalizeReferencesPreservesDistinctEvidenceProvenance(t *testing.T) {
	base := domain.Reference{
		Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 2, Language: "go"},
	}
	withHeuristic := base
	withHeuristic.Evidence = []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
		{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:B", Detail: "heuristic", Method: "heuristic"},
	}
	withASTOnly := base
	withASTOnly.Evidence = []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
	}

	// base (no evidence), withASTOnly, withHeuristic are all distinct.
	refs := []domain.Reference{base, withHeuristic, withASTOnly}
	got := NormalizeReferences(refs)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (distinct evidence provenance preserved): %#v", len(got), got)
	}
	var foundHeuristic bool
	for _, ref := range got {
		for _, ev := range ref.Evidence {
			if ev.Kind == domain.EvidenceHeuristicCandidate {
				foundHeuristic = true
			}
		}
	}
	if !foundHeuristic {
		t.Fatalf("expected heuristic evidence preserved, got %#v", got)
	}
}

func TestNormalizeReferencesPreservesDistinctCandidates(t *testing.T) {
	base := domain.Reference{
		Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionAmbiguous,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
	}
	dupCandidates := base
	dupCandidates.Candidates = []domain.SymbolRef{
		{Name: "X", File: "x.go", Line: 1, Language: "go"},
		{Name: "X", File: "x.go", Line: 1, Language: "go"},
	}
	diffCandidates := base
	diffCandidates.Candidates = []domain.SymbolRef{
		{Name: "Y", File: "y.go", Line: 2, Language: "go"},
		{Name: "Z", File: "z.go", Line: 3, Language: "go"},
	}

	refs := []domain.Reference{dupCandidates, diffCandidates}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate sets preserved): %#v", len(got), got)
	}
	// First ref's candidates deduped from 2 to 1; second ref has 2.
	if len(got[0].Candidates) != 1 || len(got[1].Candidates) != 2 {
		t.Fatalf("candidate counts wrong: got %#v and %#v", got[0].Candidates, got[1].Candidates)
	}
}

func TestNormalizeReferencesPreservesSameNameDifferentLocation(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "Helper", File: "a.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Helper", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "Helper", File: "a.go", Line: 10, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 10, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Helper", File: "c.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (same name, different location/target preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSameNameDifferentKind(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "Do", File: "a.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Do", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "Do", File: "a.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceMethod,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Do", File: "c.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (same name/location, different kind preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesDistinctResolution(t *testing.T) {
	base := domain.Reference{
		Source:   domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
		Kind:     domain.ReferenceCall,
		Language: "go",
		Location: domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
	}
	resolved := base
	resolved.Resolution = domain.ResolutionResolved
	resolved.Target = &domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"}
	ambiguous := base
	ambiguous.Resolution = domain.ResolutionAmbiguous
	ambiguous.Candidates = []domain.SymbolRef{
		{Name: "A", File: "a.go", Line: 1, Language: "go"},
		{Name: "A", File: "b.go", Line: 1, Language: "go"},
	}

	refs := []domain.Reference{resolved, ambiguous}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different resolution preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesMultiResolverCombinationDeterministic(t *testing.T) {
	goRef := domain.Reference{
		Source:     domain.SymbolRef{Name: "GoFunc", File: "main.go", Line: 5, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "main.go", StartLine: 5, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "GoFunc", File: "util.go", Line: 1, Language: "go"},
	}
	pyRef := domain.Reference{
		Source:     domain.SymbolRef{Name: "py_func", File: "app.py", Line: 3, Language: "python"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "python",
		Location:   domain.Location{Path: "app.py", StartLine: 3, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "py_func", File: "util.py", Line: 1, Language: "python"},
	}
	jsRef := domain.Reference{
		Source:     domain.SymbolRef{Name: "jsFunc", File: "app.js", Line: 2, Language: "javascript"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "javascript",
		Location:   domain.Location{Path: "app.js", StartLine: 2, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "jsFunc", File: "util.js", Line: 1, Language: "javascript"},
	}
	tsRef := domain.Reference{
		Source:     domain.SymbolRef{Name: "tsFunc", File: "app.ts", Line: 2, Language: "typescript"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "typescript",
		Location:   domain.Location{Path: "app.ts", StartLine: 2, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "tsFunc", File: "util.ts", Line: 1, Language: "typescript"},
	}

	first := NormalizeReferences([]domain.Reference{goRef, pyRef, jsRef, tsRef})
	second := NormalizeReferences([]domain.Reference{tsRef, jsRef, pyRef, goRef})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("multi-resolver combination not deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first) != 4 {
		t.Fatalf("len = %d, want 4: %#v", len(first), first)
	}
}

func TestNormalizeReferencesCanonicalRepresentationStable(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "B", File: "b.go", Line: 2, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "b.go", StartLine: 2, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 2, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "b.go:2:1", Detail: "call", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"}},
		},
	}
	first := NormalizeReferences(refs)
	for i := 0; i < 10; i++ {
		shuffled := []domain.Reference{refs[1], refs[0]}
		if i%2 == 0 {
			shuffled = []domain.Reference{refs[0], refs[1]}
		}
		got := NormalizeReferences(shuffled)
		if !reflect.DeepEqual(first, got) {
			t.Fatalf("iteration %d: canonical representation not stable:\nfirst: %#v\ngot:   %#v", i, first, got)
		}
	}
}

func TestNormalizeReferencesDoesNotMergeHomonymsFromDifferentModules(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "Thing", File: "main.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceType,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "main.go", StartLine: 5, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "Thing", File: "p1/thing.go", Line: 1, Kind: "type", Language: "go", Container: "p1"},
				{Name: "Thing", File: "p2/thing.go", Line: 1, Kind: "type", Language: "go", Container: "p2"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if len(got[0].Candidates) != 2 {
		t.Fatalf("expected 2 candidates preserved, got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesEmptyAndNil(t *testing.T) {
	if got := NormalizeReferences(nil); got != nil {
		t.Fatalf("NormalizeReferences(nil) = %#v, want nil", got)
	}
	if got := NormalizeReferences([]domain.Reference{}); len(got) != 0 {
		t.Fatalf("NormalizeReferences([]) = %#v, want empty", got)
	}
}

func TestNormalizeReferencesDeduplicatesExactDuplicatesFromMultipleEmissionPaths(t *testing.T) {
	ref := domain.Reference{
		Source:     domain.SymbolRef{Name: "Helper", File: "a.go", Line: 5, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "Helper", File: "b.go", Line: 1, Language: "go"},
		Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:5:1", Detail: "call", Method: "ast"}},
	}
	refs := []domain.Reference{ref, ref, ref}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (exact duplicates from multiple emission paths deduped): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesRefsAtSameLocationWithDifferentSourceNames(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "B", File: "a.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "c.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source names at same location preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesSortOrderIsCanonical(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "Z", File: "z.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "z.go", StartLine: 1, StartColumn: 1},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 10, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 10, StartColumn: 1},
		},
		{
			Source:     domain.SymbolRef{Name: "M", File: "m.go", Line: 5, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "m.go", StartLine: 5, StartColumn: 1},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %#v", len(got), got)
	}
	if got[0].Location.Path != "a.go" || got[1].Location.Path != "m.go" || got[2].Location.Path != "z.go" {
		t.Fatalf("sort order not canonical: got paths %q, %q, %q", got[0].Location.Path, got[1].Location.Path, got[2].Location.Path)
	}
}

func TestNormalizeReferencesDoesNotMergeDifferentLanguages(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "Func", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Func", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "Func", File: "a.py", Line: 1, Language: "python"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "python",
			Location:   domain.Location{Path: "a.py", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Func", File: "b.py", Line: 1, Language: "python"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different languages preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesDoesNotMergeDifferentTargets(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "X", File: "x.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "Y", File: "y.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different targets preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesDeduplicatesIdenticalEvidenceButPreservesDistinct(t *testing.T) {
	base := domain.Reference{
		Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionResolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
		Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
	}
	withDupEvidence := base
	withDupEvidence.Evidence = []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
		{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
	}
	refs := []domain.Reference{withDupEvidence}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if len(got[0].Evidence) != 1 {
		t.Fatalf("expected evidence deduped to 1, got %#v", got[0].Evidence)
	}
}

func TestNormalizeReferencesPreservesSourceContainerDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Container: "pkg1"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Container: "pkg2"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "c.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source containers preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetContainerDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "X", File: "x.go", Line: 1, Language: "go", Container: "pkg1"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "X", File: "x.go", Line: 1, Language: "go", Container: "pkg2"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target containers preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSignatureDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Signature: "func(int) string"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Signature: "func(string) int"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different signatures preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesExportedDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(true)},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(false)},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different exported status preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesLocationEndDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 5},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 10},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different end locations preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceLineDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 2, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 5, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source lines preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetLineDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 2, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target lines preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetKindDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Kind: "function", Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Kind: "method", Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target kinds preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetLanguageDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "python"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target languages preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetSignatureHashDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", SignatureHash: "abc"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", SignatureHash: "def"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different signature hashes preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceSignatureDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Signature: "func(int)"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Signature: "func(string)"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source signatures preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceKindDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Kind: "function"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Kind: "method"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source kinds preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceLanguageDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "python"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source languages preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceSignatureHashDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", SignatureHash: "abc"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", SignatureHash: "def"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source signature hashes preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceExportedDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Exported: boolPtr(true)},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go", Exported: boolPtr(false)},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source exported status preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetExportedDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(true)},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(false)},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target exported status preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetFileDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "c.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target files preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesTargetColumnDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Column: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Column: 2, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different target columns preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceColumnDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Column: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Column: 2, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source columns preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSourceFileDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "c.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "A", File: "b.go", Line: 1, Language: "go"},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different source files preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesCandidateFileDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go"},
				{Name: "A", File: "c.go", Line: 1, Language: "go"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "d.go", Line: 1, Language: "go"},
				{Name: "A", File: "e.go", Line: 1, Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate sets preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesCandidateContainerDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Container: "pkg1"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Container: "pkg2"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate containers preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesCandidateSignatureDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Signature: "func(int)"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Signature: "func(string)"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate signatures preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesCandidateSignatureHashDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", SignatureHash: "abc"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", SignatureHash: "def"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate signature hashes preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesCandidateExportedDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(true)},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "b.go", Line: 1, Language: "go", Exported: boolPtr(false)},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different candidate exported status preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesEvidenceMethodDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "lexical_pattern"}},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different evidence methods preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesEvidenceDetailDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call A", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call B", Method: "ast"}},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different evidence details preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesEvidenceRefDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:2", Detail: "call", Method: "ast"}},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different evidence refs preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesEvidenceKindDifference(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceLexicalCallPattern, Ref: "a.go:1:1", Detail: "call", Method: "lexical_pattern"}},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different evidence kinds preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesMultipleEvidenceRecords(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence: []domain.ReferenceEvidence{
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
				{Kind: domain.EvidenceModuleResolution, Ref: "module_resolution", Detail: "resolved", Method: "module_resolution"},
			},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence: []domain.ReferenceEvidence{
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
				{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:B", Detail: "heuristic", Method: "heuristic"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (different evidence sets preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesSingleEvidenceVsMultiple(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"}},
		},
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence: []domain.ReferenceEvidence{
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
				{Kind: domain.EvidenceModuleResolution, Ref: "module_resolution", Detail: "resolved", Method: "module_resolution"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (single vs multiple evidence preserved): %#v", len(got), got)
	}
}

func TestNormalizeReferencesPreservesEvidenceOrder(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionResolved,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Target:     &domain.SymbolRef{Name: "B", File: "b.go", Line: 1, Language: "go"},
			Evidence: []domain.ReferenceEvidence{
				{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:B", Detail: "heuristic", Method: "heuristic"},
				{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if len(got[0].Evidence) != 2 {
		t.Fatalf("expected 2 evidence records, got %#v", got[0].Evidence)
	}
	// AST call should sort before heuristic
	if got[0].Evidence[0].Kind != domain.EvidenceASTCall {
		t.Fatalf("evidence not sorted: got %#v", got[0].Evidence)
	}
}

func TestNormalizeReferencesPreservesCandidateOrder(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "Z", File: "z.go", Line: 1, Language: "go"},
				{Name: "A", File: "a.go", Line: 1, Language: "go"},
				{Name: "M", File: "m.go", Line: 1, Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if len(got[0].Candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %#v", got[0].Candidates)
	}
	if got[0].Candidates[0].Name != "A" || got[0].Candidates[1].Name != "M" || got[0].Candidates[2].Name != "Z" {
		t.Fatalf("candidates not sorted: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFile(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 10, Language: "go"},
				{Name: "A", File: "a.go", Line: 2, Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Line != 2 || got[0].Candidates[1].Line != 5 || got[0].Candidates[2].Line != 10 {
		t.Fatalf("candidates not sorted by line: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileAndLine(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 10, Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 2, Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 5, Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Column != 2 || got[0].Candidates[1].Column != 5 || got[0].Candidates[2].Column != 10 {
		t.Fatalf("candidates not sorted by column: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumn(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "B", File: "a.go", Line: 5, Column: 1, Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Language: "go"},
				{Name: "C", File: "a.go", Line: 5, Column: 1, Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Name != "A" || got[0].Candidates[1].Name != "B" || got[0].Candidates[2].Name != "C" {
		t.Fatalf("candidates not sorted by name: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumnName(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "method", Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "type", Language: "go"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Kind != "function" || got[0].Candidates[1].Kind != "method" || got[0].Candidates[2].Kind != "type" {
		t.Fatalf("candidates not sorted by kind: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumnKindName(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "python"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "typescript"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Language != "go" || got[0].Candidates[1].Language != "python" || got[0].Candidates[2].Language != "typescript" {
		t.Fatalf("candidates not sorted by language: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumnKindLanguageName(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg2"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg3"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Container != "pkg1" || got[0].Candidates[1].Container != "pkg2" || got[0].Candidates[2].Container != "pkg3" {
		t.Fatalf("candidates not sorted by container: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumnKindLanguageNameContainer(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(int)"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(string)"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(bool)"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].Signature != "func(bool)" || got[0].Candidates[1].Signature != "func(int)" || got[0].Candidates[2].Signature != "func(string)" {
		t.Fatalf("candidates not sorted by signature: got %#v", got[0].Candidates)
	}
}

func TestNormalizeReferencesPreservesCandidateOrderWithSameFileLineColumnKindLanguageNameContainerSignature(t *testing.T) {
	refs := []domain.Reference{
		{
			Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 1, Language: "go"},
			Kind:       domain.ReferenceCall,
			Resolution: domain.ResolutionAmbiguous,
			Language:   "go",
			Location:   domain.Location{Path: "a.go", StartLine: 1, StartColumn: 1},
			Candidates: []domain.SymbolRef{
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(int)", SignatureHash: "def"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(int)", SignatureHash: "abc"},
				{Name: "A", File: "a.go", Line: 5, Column: 1, Kind: "function", Language: "go", Container: "pkg1", Signature: "func(int)", SignatureHash: "ghi"},
			},
		},
	}
	got := NormalizeReferences(refs)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].Candidates[0].SignatureHash != "abc" || got[0].Candidates[1].SignatureHash != "def" || got[0].Candidates[2].SignatureHash != "ghi" {
		t.Fatalf("candidates not sorted by signature hash: got %#v", got[0].Candidates)
	}
}
