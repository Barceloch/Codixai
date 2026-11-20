package index

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

type xlExpectedSymbolRef struct {
	Name      string `json:"name,omitempty"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Language  string `json:"language,omitempty"`
	Container string `json:"container,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type xlCase struct {
	File                   string                `json:"file"`
	Name                   string                `json:"name"`
	Kind                   string                `json:"kind,omitempty"`
	Line                   int                   `json:"line,omitempty"`
	AllowAnyLine           bool                  `json:"allow_any_line,omitempty"`
	ExpectedResolution     string                `json:"expected_resolution"`
	ExpectedTargetFile     string                `json:"expected_target_file,omitempty"`
	ExpectedTargetName     string                `json:"expected_target_name,omitempty"`
	ExpectedTarget         *xlExpectedSymbolRef  `json:"expected_target,omitempty"`
	ExpectedCandidates     []xlExpectedSymbolRef `json:"expected_candidates,omitempty"`
	ExpectedCandidateCount int                   `json:"expected_candidate_count,omitempty"`
	MinCandidates          int                   `json:"min_candidates,omitempty"`
	ExpectedEvidenceKinds  []string              `json:"expected_evidence_kinds,omitempty"`
	RequireEvidence        *bool                 `json:"require_evidence,omitempty"`
	Note                   string                `json:"note,omitempty"`
}

type xlExpect struct {
	Concept       string   `json:"concept"`
	Language      string   `json:"language"`
	Comparability string   `json:"comparability"` // strictly_equivalent|approximately_equivalent|language_specific|non_comparable
	Justification string   `json:"justification,omitempty"`
	Cases         []xlCase `json:"cases"`
}

func TestCrossLanguageReferenceCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "cross-language-reference")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Skipf("cross-language reference corpus not found: %v", err)
	}

	concepts, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read corpus root: %v", err)
	}
	sort.Slice(concepts, func(i, j int) bool { return concepts[i].Name() < concepts[j].Name() })

	type conceptSummary struct {
		concept       string
		strict        int
		approx        int
		langSpecific  int
		nonComparable int
		passed        int
		failed        int
	}
	summaries := map[string]*conceptSummary{}

	for _, ce := range concepts {
		if !ce.IsDir() {
			continue
		}
		concept := ce.Name()
		sum := &conceptSummary{concept: concept}
		summaries[concept] = sum
		langEntries, err := os.ReadDir(filepath.Join(root, concept))
		if err != nil {
			t.Errorf("%s: read concept dir: %v", concept, err)
			continue
		}
		sort.Slice(langEntries, func(i, j int) bool { return langEntries[i].Name() < langEntries[j].Name() })
		for _, le := range langEntries {
			if !le.IsDir() {
				continue
			}
			lang := le.Name()
			caseDir := filepath.Join(root, concept, lang)
			expectPath := filepath.Join(caseDir, "expect.json")
			b, err := os.ReadFile(expectPath)
			if err != nil {
				t.Errorf("%s/%s: missing expect.json: %v", concept, lang, err)
				sum.failed++
				continue
			}
			var ex xlExpect
			if err := json.Unmarshal(b, &ex); err != nil {
				t.Errorf("%s/%s: parse expect.json: %v", concept, lang, err)
				sum.failed++
				continue
			}
			if ex.Concept == "" {
				ex.Concept = concept
			}
			if ex.Language == "" {
				ex.Language = lang
			}
			switch ex.Comparability {
			case "strictly_equivalent":
				sum.strict++
			case "approximately_equivalent":
				sum.approx++
			case "language_specific":
				sum.langSpecific++
			case "non_comparable":
				sum.nonComparable++
			default:
				t.Errorf("%s/%s: unknown comparability %q", concept, lang, ex.Comparability)
				sum.failed++
				continue
			}

			name := concept + "/" + lang + "[" + ex.Comparability + "]"
			ok := t.Run(name, func(t *testing.T) {
				if ex.Comparability == "non_comparable" {
					t.Logf("NON-COMPARABLE %s/%s: %s", concept, lang, ex.Justification)
					// Non-comparable cases document intent; if cases are provided, still validate they are internally coherent as language-specific expectations.
				}
				if ex.Comparability == "language_specific" || ex.Comparability == "non_comparable" {
					t.Logf("comparability=%s concept=%s lang=%s justification=%s", ex.Comparability, concept, lang, ex.Justification)
				}
				runXLLanguageCase(t, caseDir, ex)
			})
			if ok {
				sum.passed++
			} else {
				sum.failed++
			}
		}
	}

	// Deterministic summary.
	keys := make([]string, 0, len(summaries))
	for k := range summaries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s := summaries[k]
		t.Logf("CONCEPT %s: strict=%d approx=%d language_specific=%d non_comparable=%d subtests_passed=%d subtests_failed=%d", s.concept, s.strict, s.approx, s.langSpecific, s.nonComparable, s.passed, s.failed)
	}
}

func runXLLanguageCase(t *testing.T, caseDir string, ex xlExpect) {
	t.Helper()
	if len(ex.Cases) == 0 {
		if ex.Comparability == "non_comparable" || ex.Comparability == "language_specific" {
			return
		}
		t.Fatalf("%s/%s: expect.json must contain at least one case", ex.Concept, ex.Language)
	}
	idx := New()
	if err := idx.Build(context.Background(), caseDir); err != nil {
		t.Fatalf("build index: %v", err)
	}

	files := xlCollectFiles(caseDir, ex.Language)
	cs := &domain.ChangeSet{Root: caseDir, Origin: domain.OriginWorktree, Files: files}
	refs, err := xlResolveByLanguage(context.Background(), idx, cs, ex.Language)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	for _, c := range ex.Cases {
		got := xlFindMatch(refs, c)
		xlEvaluate(t, ex, c, got, refs)
	}
}

func xlCollectFiles(caseDir, lang string) []domain.FileChange {
	var files []domain.FileChange
	_ = filepath.WalkDir(caseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(caseDir, path)
		rel = filepath.ToSlash(rel)
		if rel == "expect.json" {
			return nil
		}
		fLang := xlLangForPath(path, lang)
		if fLang == "" {
			return nil
		}
		files = append(files, domain.FileChange{Path: rel, Status: domain.FileModified, Language: fLang})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func xlLangForPath(path, contextLang string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx", ".mts", ".cts":
		return "typescript"
	default:
		return ""
	}
}

func xlResolveByLanguage(ctx context.Context, idx *Index, cs *domain.ChangeSet, lang string) ([]domain.Reference, error) {
	switch lang {
	case "go":
		return NewGoReferenceResolver().Resolve(ctx, idx, cs)
	case "python":
		return NewPythonReferenceResolver().Resolve(ctx, idx, cs)
	case "javascript":
		return NewJavaScriptReferenceResolver().Resolve(ctx, idx, cs)
	case "typescript":
		return NewTypeScriptReferenceResolver().Resolve(ctx, idx, cs)
	default:
		return nil, nil
	}
}

func xlFindMatch(refs []domain.Reference, c xlCase) *domain.Reference {
	for i := range refs {
		r := &refs[i]
		if filepath.ToSlash(r.Location.Path) != filepath.ToSlash(c.File) {
			continue
		}
		if r.Source.Name != c.Name {
			continue
		}
		if c.Kind != "" && string(r.Kind) != c.Kind {
			continue
		}
		if c.Line > 0 && !c.AllowAnyLine && r.Location.StartLine != c.Line {
			continue
		}
		return r
	}
	return nil
}

func xlEvaluate(t *testing.T, ex xlExpect, c xlCase, got *domain.Reference, refs []domain.Reference) {
	t.Helper()
	wantRes := domain.ResolutionStatus(c.ExpectedResolution)
	label := ex.Concept + "/" + ex.Language + " " + c.File + ":" + itoa(c.Line) + " " + c.Name
	if got == nil {
		_ = refs
		t.Fatalf("missing reference for %s want=%s (%s) [%s] justification=%s", label, c.ExpectedResolution, c.Note, ex.Comparability, ex.Justification)
	}
	if got.Resolution != wantRes {
		t.Fatalf("resolution mismatch for %s got=%s want=%s target=%#v candidates=%d evidence=%d (%s) [%s]", label, got.Resolution, wantRes, got.Target, len(got.Candidates), len(got.Evidence), c.Note, ex.Comparability)
	}

	// Shared invariants by state.
	switch wantRes {
	case domain.ResolutionResolved:
		if got.Target == nil {
			t.Fatalf("resolved without target for %s (%s)", label, c.Note)
		}
		if len(got.Candidates) != 0 {
			t.Fatalf("resolved must not have candidates for %s: %#v (%s)", label, got.Candidates, c.Note)
		}
		if c.ExpectedTargetName != "" && got.Target.Name != c.ExpectedTargetName {
			t.Fatalf("wrong target name for %s got=%q want=%q (%s)", label, got.Target.Name, c.ExpectedTargetName, c.Note)
		}
		if c.ExpectedTargetFile != "" && filepath.ToSlash(got.Target.File) != filepath.ToSlash(c.ExpectedTargetFile) {
			t.Fatalf("wrong target file for %s got=%q want=%q (%s)", label, got.Target.File, c.ExpectedTargetFile, c.Note)
		}
		if c.ExpectedTarget != nil && !xlMatchesSymbolRef(*got.Target, *c.ExpectedTarget) {
			t.Fatalf("wrong target ref for %s got=%#v want=%#v (%s)", label, got.Target, c.ExpectedTarget, c.Note)
		}
	case domain.ResolutionAmbiguous:
		if got.Target != nil {
			t.Fatalf("ambiguous must not have target for %s: %#v (%s)", label, got.Target, c.Note)
		}
		minC := c.MinCandidates
		if minC == 0 && c.ExpectedCandidateCount == 0 && len(c.ExpectedCandidates) == 0 {
			// Module-level or representation-specific ambiguity may not expose SymbolRef candidates.
			minC = 0
		} else if minC < 2 {
			minC = 2
		}
		if minC > 0 && len(got.Candidates) < minC {
			t.Fatalf("insufficient candidates for %s got=%d want>=%d (%s)", label, len(got.Candidates), minC, c.Note)
		}
		if c.ExpectedCandidateCount > 0 && len(got.Candidates) != c.ExpectedCandidateCount {
			t.Fatalf("wrong candidate count for %s got=%d want=%d (%s)", label, len(got.Candidates), c.ExpectedCandidateCount, c.Note)
		}
		if len(c.ExpectedCandidates) > 0 {
			if len(got.Candidates) < 2 {
				t.Fatalf("ambiguous must have >=2 candidates when candidate set is asserted for %s: %#v (%s)", label, got.Candidates, c.Note)
			}
			if !xlContainsCandidates(got.Candidates, c.ExpectedCandidates) {
				t.Fatalf("missing expected candidates for %s got=%#v want subset=%#v (%s)", label, got.Candidates, c.ExpectedCandidates, c.Note)
			}
		}
		if len(got.Candidates) > 0 && !xlCandidatesDeterministic(got.Candidates) {
			t.Fatalf("nondeterministic candidates for %s: %#v (%s)", label, got.Candidates, c.Note)
		}
	case domain.ResolutionUnresolved, domain.ResolutionExternal, domain.ResolutionDynamic:
		// Shared invariant: evidence must justify non-speculative abstention/classification.
	}

	requireEvidence := true
	if c.RequireEvidence != nil {
		requireEvidence = *c.RequireEvidence
	}
	// Shared invariant: all compared references carry auditable evidence.
	if requireEvidence && len(got.Evidence) == 0 {
		t.Fatalf("missing evidence for %s state=%s (%s)", label, got.Resolution, c.Note)
	}
	if len(c.ExpectedEvidenceKinds) > 0 && !xlHasEvidenceKinds(got.Evidence, c.ExpectedEvidenceKinds) {
		t.Fatalf("missing evidence kinds for %s got=%#v want=%#v (%s)", label, got.Evidence, c.ExpectedEvidenceKinds, c.Note)
	}
}

func xlMatchesSymbolRef(got domain.SymbolRef, want xlExpectedSymbolRef) bool {
	if want.Name != "" && got.Name != want.Name {
		return false
	}
	if want.File != "" && filepath.ToSlash(got.File) != filepath.ToSlash(want.File) {
		return false
	}
	if want.Line > 0 && got.Line != want.Line {
		return false
	}
	if want.Column > 0 && got.Column != want.Column {
		return false
	}
	if want.Kind != "" && got.Kind != want.Kind {
		return false
	}
	if want.Language != "" && got.Language != want.Language {
		return false
	}
	if want.Container != "" && got.Container != want.Container {
		return false
	}
	if want.Signature != "" && got.Signature != want.Signature {
		return false
	}
	return true
}

func xlContainsCandidates(got []domain.SymbolRef, want []xlExpectedSymbolRef) bool {
	for _, w := range want {
		found := false
		for _, g := range got {
			if xlMatchesSymbolRef(g, w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func xlCandidatesDeterministic(cands []domain.SymbolRef) bool {
	norm := NormalizeCandidates(cands)
	if len(norm) != len(cands) {
		return false
	}
	for i := range norm {
		if !symbolRefEqual(norm[i], cands[i]) {
			return false
		}
	}
	return true
}

func xlHasEvidenceKinds(evidence []domain.ReferenceEvidence, kinds []string) bool {
	for _, k := range kinds {
		found := false
		for _, ev := range evidence {
			if string(ev.Kind) == k {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
