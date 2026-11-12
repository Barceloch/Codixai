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

type expectedSymbolRef struct {
	Name      string `json:"name,omitempty"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Language  string `json:"language,omitempty"`
	Container string `json:"container,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type qualityCase struct {
	File               string `json:"file"`
	Name               string `json:"name"`
	Kind               string `json:"kind,omitempty"`
	Line               int    `json:"line,omitempty"`
	ExpectedResolution string `json:"expected_resolution"`
	ExpectedTargetFile string `json:"expected_target_file,omitempty"`
	ExpectedTargetName string `json:"expected_target_name,omitempty"`
	Note               string `json:"note,omitempty"`
	// AllowAnyLine skips line matching when occurrences may vary.
	AllowAnyLine bool `json:"allow_any_line,omitempty"`
	// Category for reporting.
	Category               string              `json:"category,omitempty"`
	ExpectedCandidates     []expectedSymbolRef `json:"expected_candidates,omitempty"`
	ExpectedCandidateCount int                 `json:"expected_candidate_count,omitempty"`
	MinCandidates          int                 `json:"min_candidates,omitempty"`
	ExpectedTarget         *expectedSymbolRef  `json:"expected_target,omitempty"`
	ExpectedEvidenceKinds  []string            `json:"expected_evidence_kinds,omitempty"`
	RequireEvidence        bool                `json:"require_evidence,omitempty"`
}

type qualityExpect struct {
	Language string        `json:"language"`
	Category string        `json:"category,omitempty"`
	Cases    []qualityCase `json:"cases"`
}

type qualityStats struct {
	tp, fp, fn int
	categories map[string]*qualityStats
}

func newQualityStats() *qualityStats {
	return &qualityStats{categories: map[string]*qualityStats{}}
}

func TestResolverQualityAuditCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "resolver-quality")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("resolver quality corpus not found: %v", err)
	}
	langs := []string{"go", "python", "javascript", "typescript"}
	allStats := map[string]*qualityStats{}
	for _, l := range langs {
		allStats[l] = newQualityStats()
	}
	for _, lang := range langs {

		langDir := filepath.Join(root, lang)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			t.Logf("skip %s: %v", lang, err)
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			caseDir := filepath.Join(langDir, e.Name())
			expectPath := filepath.Join(caseDir, "expect.json")
			b, err := os.ReadFile(expectPath)
			if err != nil {
				continue
			}
			var ex qualityExpect
			if err := json.Unmarshal(b, &ex); err != nil {
				t.Errorf("%s: parse expect.json: %v", caseDir, err)
				continue
			}
			if ex.Language == "" {
				ex.Language = lang
			}
			t.Run(lang+"/"+e.Name(), func(t *testing.T) {
				evalQualityCase(t, caseDir, ex, allStats[lang])
			})
		}
	}
	// Report summary metrics (informational, logged).
	for _, lang := range langs {
		s := allStats[lang]
		t.Logf("SUMMARY %s: TP=%d FP=%d FN=%d precision=%s recall=%s f1=%s", lang, s.tp, s.fp, s.fn, ratio(s.tp, s.tp+s.fp), ratio(s.tp, s.tp+s.fn), f1(s.tp, s.fp, s.fn))
		for cat, cs := range s.categories {
			t.Logf("  %s/%s: TP=%d FP=%d FN=%d", lang, cat, cs.tp, cs.fp, cs.fn)
		}
	}
}

func ratio(num, den int) string {
	if den == 0 {
		return "N/A"
	}
	return strings.TrimRight(strings.TrimRight(fmtFloat(float64(num)/float64(den)), "0"), ".")
}

func f1(tp, fp, fn int) string {
	den := 2*tp + fp + fn
	if den == 0 {
		return "N/A"
	}
	return strings.TrimRight(strings.TrimRight(fmtFloat(2*float64(tp)/float64(den)), "0"), ".")
}

func fmtFloat(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func evalQualityCase(t *testing.T, caseDir string, ex qualityExpect, st *qualityStats) {
	t.Helper()
	idx := New()
	if err := idx.Build(context.Background(), caseDir); err != nil {
		t.Fatalf("build index: %v", err)
	}
	// Build changeset from all source files matching language.
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
		lang := langForExt(filepath.Ext(path), ex.Language)
		if lang == "" {
			return nil
		}
		files = append(files, domain.FileChange{Path: rel, Status: domain.FileModified, Language: lang})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	cs := &domain.ChangeSet{Root: caseDir, Origin: domain.OriginWorktree, Files: files}
	var refs []domain.Reference
	var err error
	switch ex.Language {
	case "go":
		refs, err = NewGoReferenceResolver().Resolve(context.Background(), idx, cs)
	case "python":
		refs, err = NewPythonReferenceResolver().Resolve(context.Background(), idx, cs)
	case "javascript":
		refs, err = NewJavaScriptReferenceResolver().Resolve(context.Background(), idx, cs)
	case "typescript":
		refs, err = NewTypeScriptReferenceResolver().Resolve(context.Background(), idx, cs)
	default:
		t.Fatalf("unknown language %q", ex.Language)
	}
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, c := range ex.Cases {
		cat := c.Category
		if cat == "" {
			cat = ex.Category
		}
		if cat == "" {
			cat = "uncategorized"
		}
		cs2 := st.categories[cat]
		if cs2 == nil {
			cs2 = &qualityStats{categories: map[string]*qualityStats{}}
			st.categories[cat] = cs2
		}
		match := findQualityMatch(refs, c)
		evaluateExpectation(t, c, match, st, cs2)
	}
}

func langForExt(ext, fallback string) string {
	switch strings.ToLower(ext) {
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

func findQualityMatch(refs []domain.Reference, c qualityCase) *domain.Reference {
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

func evaluateExpectation(t *testing.T, c qualityCase, got *domain.Reference, st, cs *qualityStats) {
	t.Helper()
	wantRes := domain.ResolutionStatus(c.ExpectedResolution)
	if got == nil {
		// Missing reference.
		if wantRes == domain.ResolutionDynamic || wantRes == domain.ResolutionExternal || wantRes == domain.ResolutionUnresolved || wantRes == domain.ResolutionAmbiguous {
			// For some states, "no reference emitted" may be acceptable if the resolver abstains from emitting; count as FN for strict detection but note.
			t.Logf("MISS %s:%d %q want=%s (%s)", c.File, c.Line, c.Name, c.ExpectedResolution, c.Note)
			st.fn++
			cs.fn++
			return
		}
		if wantRes == domain.ResolutionResolved {
			t.Errorf("FN %s:%d %q kind=%s want resolved (%s)", c.File, c.Line, c.Name, c.Kind, c.Note)
			st.fn++
			cs.fn++
			return
		}
		st.fn++
		cs.fn++
		return
	}
	if got.Resolution != wantRes {
		// Special: a correctly conservative abstention (dynamic/unresolved) when we expected resolved may be FN; wrong resolved is FP.
		if got.Resolution == domain.ResolutionResolved && wantRes != domain.ResolutionResolved {
			t.Errorf("FP %s:%d %q got resolved target=%#v want=%s (%s)", c.File, got.Location.StartLine, c.Name, got.Target, c.ExpectedResolution, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if wantRes == domain.ResolutionResolved {
			t.Errorf("FN/WRONG %s:%d %q got=%s want=resolved (%s)", c.File, got.Location.StartLine, c.Name, got.Resolution, c.Note)
			st.fn++
			cs.fn++
			return
		}
		// Different non-resolved states - log but count based on severity.
		t.Logf("STATE %s:%d %q got=%s want=%s (%s)", c.File, got.Location.StartLine, c.Name, got.Resolution, c.ExpectedResolution, c.Note)
		if wantRes == domain.ResolutionDynamic && (got.Resolution == domain.ResolutionUnresolved || got.Resolution == domain.ResolutionAmbiguous) {
			// conservative difference
			st.tp++ // acceptable classification
			cs.tp++
			return
		}
		st.fp++
		cs.fp++
		return
	}
	if wantRes == domain.ResolutionResolved {
		if got.Target == nil {
			t.Errorf("FP missing target %s:%d %q got resolved without target (%s)", c.File, got.Location.StartLine, c.Name, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if len(got.Candidates) != 0 {
			t.Errorf("FP resolved with candidates %s:%d %q candidates=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Candidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if c.ExpectedTargetName != "" && got.Target.Name != c.ExpectedTargetName {
			t.Errorf("FP wrong target %s:%d %q got=%#v want target name=%q (%s)", c.File, got.Location.StartLine, c.Name, got.Target, c.ExpectedTargetName, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if c.ExpectedTargetFile != "" && filepath.ToSlash(got.Target.File) != filepath.ToSlash(c.ExpectedTargetFile) {
			t.Errorf("FP wrong target file %s:%d %q got=%q want=%q (%s)", c.File, got.Location.StartLine, c.Name, got.Target.File, c.ExpectedTargetFile, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if c.ExpectedTarget != nil && !matchesExpectedSymbolRef(*got.Target, *c.ExpectedTarget) {
			t.Errorf("FP wrong target ref %s:%d %q got=%#v want=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Target, c.ExpectedTarget, c.Note)
			st.fp++
			cs.fp++
			return
		}
	}
	if wantRes == domain.ResolutionAmbiguous {
		if got.Target != nil {
			t.Errorf("FP ambiguous with target %s:%d %q target=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Target, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if len(got.Candidates) < 2 {
			t.Errorf("FP ambiguous without >=2 candidates %s:%d %q candidates=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Candidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if c.ExpectedCandidateCount > 0 && len(got.Candidates) != c.ExpectedCandidateCount {
			t.Errorf("FP wrong candidate count %s:%d %q got=%d want=%d candidates=%#v (%s)", c.File, got.Location.StartLine, c.Name, len(got.Candidates), c.ExpectedCandidateCount, got.Candidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
		minC := c.MinCandidates
		if minC < 2 {
			minC = 2
		}
		if len(got.Candidates) < minC {
			t.Errorf("FP insufficient candidates %s:%d %q got=%d want >=%d candidates=%#v (%s)", c.File, got.Location.StartLine, c.Name, len(got.Candidates), minC, got.Candidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if len(c.ExpectedCandidates) > 0 && !containsExpectedCandidates(got.Candidates, c.ExpectedCandidates) {
			t.Errorf("FP missing expected candidates %s:%d %q got=%#v want subset=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Candidates, c.ExpectedCandidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
		if !candidatesDeterministic(got.Candidates) {
			t.Errorf("FP nondeterministic candidates %s:%d %q candidates=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Candidates, c.Note)
			st.fp++
			cs.fp++
			return
		}
	}
	if c.RequireEvidence || wantRes == domain.ResolutionResolved || wantRes == domain.ResolutionAmbiguous || wantRes == domain.ResolutionExternal || wantRes == domain.ResolutionDynamic || wantRes == domain.ResolutionUnresolved {
		if len(got.Evidence) == 0 {
			t.Errorf("FP missing evidence %s:%d %q state=%s (%s)", c.File, got.Location.StartLine, c.Name, got.Resolution, c.Note)
			st.fp++
			cs.fp++
			return
		}
	}
	if len(c.ExpectedEvidenceKinds) > 0 && !hasExpectedEvidenceKinds(got.Evidence, c.ExpectedEvidenceKinds) {
		t.Errorf("FP missing evidence kinds %s:%d %q got=%#v want kinds=%#v (%s)", c.File, got.Location.StartLine, c.Name, got.Evidence, c.ExpectedEvidenceKinds, c.Note)
		st.fp++
		cs.fp++
		return
	}
	st.tp++
	cs.tp++
}

func matchesExpectedSymbolRef(got domain.SymbolRef, want expectedSymbolRef) bool {
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

func containsExpectedCandidates(got []domain.SymbolRef, want []expectedSymbolRef) bool {
	for _, w := range want {
		found := false
		for _, g := range got {
			if matchesExpectedSymbolRef(g, w) {
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

func candidatesDeterministic(cands []domain.SymbolRef) bool {
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

func hasExpectedEvidenceKinds(evidence []domain.ReferenceEvidence, kinds []string) bool {
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
