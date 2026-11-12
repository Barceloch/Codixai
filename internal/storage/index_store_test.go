package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
	_ "modernc.org/sqlite"
)

func TestSaveLoadReferencesRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	refs := []domain.Reference{{
		Source:     domain.SymbolRef{Name: "A", File: filepath.Join(absRoot, "a.go"), Line: 2, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionUnresolved,
		Language:   "go",
		Location:   domain.Location{Path: filepath.Join(absRoot, "a.go"), StartLine: 2, StartColumn: 1},
		Evidence:   []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:2:1", Detail: "test", Method: "ast"}},
	}}
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if got[0].Resolution != domain.ResolutionUnresolved {
		t.Fatalf("resolution=%q", got[0].Resolution)
	}
	if got[0].Source.Name != "A" || len(got[0].Evidence) != 1 {
		t.Fatalf("got %#v", got[0])
	}
}

func TestSaveLoadReferencesAmbiguousAndEvidence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\nfunc B(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(absRoot, "a.go")
	refs := []domain.Reference{{
		Source:     domain.SymbolRef{Name: "X", File: file, Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionAmbiguous,
		Candidates: []domain.SymbolRef{{Name: "A", File: file, Line: 2, Language: "go"}, {Name: "B", File: file, Line: 3, Language: "go"}},
		Language:   "go",
		Location:   domain.Location{Path: file, StartLine: 1, StartColumn: 1},
		Evidence: []domain.ReferenceEvidence{
			{Kind: domain.EvidenceASTCall, Ref: "a.go:1:1", Detail: "call", Method: "ast"},
			{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:A", Detail: "candidate", Method: "heuristic"},
		},
	}}
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Resolution != domain.ResolutionAmbiguous || len(got[0].Candidates) != 2 {
		t.Fatalf("ambiguous not preserved: %#v", got[0])
	}
	if len(got[0].Evidence) != 2 {
		t.Fatalf("evidence not preserved: %#v", got[0].Evidence)
	}
}

func TestSaveReferencesSkipsSourceNotInIndex(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	refs := []domain.Reference{{
		Source:     domain.SymbolRef{Name: "X", File: filepath.Join(absRoot, "missing.go"), Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionUnresolved,
		Language:   "go",
		Location:   domain.Location{Path: filepath.Join(absRoot, "missing.go"), StartLine: 1, StartColumn: 1},
	}}
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 refs for missing source file, got %d", len(got))
	}
}

func TestSaveLoadReferencesAmbiguousDeterministicAndPreserved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\nfunc B(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(absRoot, "a.go")
	refs := index.NormalizeReferences([]domain.Reference{{
		Source:     domain.SymbolRef{Name: "X", File: file, Line: 1, Language: "go", Kind: "type"},
		Kind:       domain.ReferenceType,
		Resolution: domain.ResolutionAmbiguous,
		Language:   "go",
		Location:   domain.Location{Path: file, StartLine: 1, StartColumn: 1},
		Evidence: []domain.ReferenceEvidence{
			{Kind: domain.EvidenceASTTypeReference, Ref: "a.go:1:1", Detail: "type ref X", Method: "ast"},
			{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:A", Detail: "candidate A", Method: "heuristic"},
		},
		Candidates: []domain.SymbolRef{
			{Name: "B", File: file, Line: 3, Language: "go", Kind: "function"},
			{Name: "A", File: file, Line: 2, Language: "go", Kind: "function"},
			{Name: "A", File: file, Line: 2, Language: "go", Kind: "function"},
		},
	}})
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		got, err := store.LoadReferences(context.Background(), absRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("iter %d: len=%d", i, len(got))
		}
		if got[0].Resolution != domain.ResolutionAmbiguous {
			t.Fatalf("iter %d: resolution=%q", i, got[0].Resolution)
		}
		if got[0].Target != nil {
			t.Fatalf("iter %d: ambiguous must have nil target", i)
		}
		if len(got[0].Candidates) != 2 {
			t.Fatalf("iter %d: candidates=%#v", i, got[0].Candidates)
		}
		if got[0].Candidates[0].Name != "A" || got[0].Candidates[1].Name != "B" {
			t.Fatalf("iter %d: candidates not normalized/deterministic order: %#v", i, got[0].Candidates)
		}
		if len(got[0].Evidence) != 2 {
			t.Fatalf("iter %d: evidence not preserved: %#v", i, got[0].Evidence)
		}
		if got[0].Location.Path == "" || got[0].Source.Name != "X" {
			t.Fatalf("iter %d: identity lost: %#v", i, got[0])
		}
	}
}

func TestSaveLoadReferencesRelativePath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	// Resolver emits relative paths; storage should resolve them against root.
	refs := []domain.Reference{{
		Source:     domain.SymbolRef{Name: "A", File: "a.go", Line: 2, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionUnresolved,
		Language:   "go",
		Location:   domain.Location{Path: "a.go", StartLine: 2, StartColumn: 1},
	}}
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1 (relative path should resolve)", len(got))
	}
}

func TestMigrateV1ToV2(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "v1.db")
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	// Minimal v1-like schema (no column/container/signature/exported on symbols, no index_references).
	_, err = db.Exec(`
		CREATE TABLE index_metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE index_files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, language TEXT NOT NULL, hash TEXT NOT NULL, size INTEGER NOT NULL, mod_time INTEGER NOT NULL, indexed_at INTEGER NOT NULL);
		CREATE TABLE index_symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE, name TEXT NOT NULL, kind TEXT NOT NULL, line INTEGER NOT NULL, language TEXT NOT NULL, UNIQUE(file_id,name,kind,line));
		CREATE TABLE index_dependencies (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE, path TEXT NOT NULL, line INTEGER NOT NULL, language TEXT NOT NULL);
		CREATE TABLE index_builds (id INTEGER PRIMARY KEY AUTOINCREMENT, root_path TEXT NOT NULL, tool_version TEXT NOT NULL, schema_version INTEGER NOT NULL, analyzer_version TEXT NOT NULL, started_at INTEGER NOT NULL, completed_at INTEGER, files_total INTEGER NOT NULL DEFAULT 0, files_indexed INTEGER NOT NULL DEFAULT 0, symbols_count INTEGER NOT NULL DEFAULT 0, deps_count INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'running');
		INSERT INTO index_metadata (key, value) VALUES ('schema_version', '1');
	`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Verify v2 schema.
	var version string
	if err := store.db.QueryRow(`SELECT value FROM index_metadata WHERE key='schema_version'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != "2" {
		t.Fatalf("schema_version=%q want 2", version)
	}
	var refsTable string
	if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='index_references'`).Scan(&refsTable); err != nil {
		t.Fatalf("index_references missing after migration: %v", err)
	}
	status, err := store.Status(context.Background(), "/tmp/any")
	if err != nil {
		t.Fatal(err)
	}
	if status.RefCount != 0 {
		t.Fatalf("refcount=%d", status.RefCount)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReferencesDeterministicOrderRecovery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "b.go"), "package p\nfunc B(){}\nfunc B2(){}\n")
	writeFile(t, filepath.Join(root, "a.go"), "package p\nfunc A(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}

	aFile := filepath.Join(absRoot, "a.go")
	bFile := filepath.Join(absRoot, "b.go")
	refs := []domain.Reference{
		{Source: domain.SymbolRef{Name: "Z", File: bFile, Line: 3, Column: 2, Language: "go"}, Kind: domain.ReferenceMethod, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: bFile, StartLine: 3, StartColumn: 2}},
		{Source: domain.SymbolRef{Name: "A", File: bFile, Line: 2, Column: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: bFile, StartLine: 2, StartColumn: 1}},
		{Source: domain.SymbolRef{Name: "M", File: aFile, Line: 2, Column: 5, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionResolved, Language: "go", Location: domain.Location{Path: aFile, StartLine: 2, StartColumn: 5}, Target: &domain.SymbolRef{Name: "A", File: aFile, Line: 2, Language: "go"}},
		{Source: domain.SymbolRef{Name: "M", File: aFile, Line: 2, Column: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionResolved, Language: "go", Location: domain.Location{Path: aFile, StartLine: 2, StartColumn: 1}, Target: &domain.SymbolRef{Name: "A", File: aFile, Line: 2, Language: "go"}},
	}
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("len=%d want 4", len(got))
	}
	// Deterministic logical order recovery: by source file, line, column,
	// referenced/source symbol context, kind, language, resolution, id.
	wantOrder := []struct {
		path string
		line int
		col  int
		kind domain.ReferenceKind
		res  domain.ResolutionStatus
	}{
		{aFile, 2, 1, domain.ReferenceCall, domain.ResolutionResolved},
		{aFile, 2, 5, domain.ReferenceCall, domain.ResolutionResolved},
		{bFile, 2, 1, domain.ReferenceCall, domain.ResolutionUnresolved},
		{bFile, 3, 2, domain.ReferenceMethod, domain.ResolutionUnresolved},
	}
	for i, w := range wantOrder {
		g := got[i]
		if g.Location.Path != w.path || g.Location.StartLine != w.line || g.Location.StartColumn != w.col || g.Kind != w.kind || g.Resolution != w.res {
			t.Fatalf("order[%d]=(%q,%d,%d,%q,%q) want (%q,%d,%d,%q,%q) full=%#v", i, g.Location.Path, g.Location.StartLine, g.Location.StartColumn, g.Kind, g.Resolution, w.path, w.line, w.col, w.kind, w.res, got)
		}
	}
	// Repeated loads must be identical.
	got2, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got2) != len(got) {
		t.Fatalf("len2=%d len1=%d", len(got2), len(got))
	}
	for i := range got {
		a, b := got[i], got2[i]
		if a.ID != b.ID || a.Location != b.Location || a.Source.Name != b.Source.Name || a.Kind != b.Kind || a.Resolution != b.Resolution || a.Language != b.Language {
			t.Fatalf("non-deterministic load at %d: %#v vs %#v", i, a, b)
		}
	}
}

// --- P5252-G: Reference Persistence Fidelity and Recovery Integrity ---

func boolPtr(v bool) *bool { return &v }

func setupRefStore(t *testing.T) (*IndexStore, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewIndexStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package p\ntype T struct{}\nfunc A(){}\nfunc B(){}\n")
	writeFile(t, filepath.Join(root, "b.go"), "package p\nfunc C(){}\n")
	idx := index.New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	absRoot, _ := filepath.Abs(root)
	if err := store.Save(context.Background(), idx, absRoot, "test", "test"); err != nil {
		t.Fatal(err)
	}
	return store, absRoot
}

func assertEvidenceEqual(t *testing.T, got, want []domain.ReferenceEvidence) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("evidence len=%d want %d: got=%#v want=%#v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("evidence[%d]=%#v want %#v", i, got[i], want[i])
		}
	}
}

func assertCandidatesEqual(t *testing.T, got, want []domain.SymbolRef) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("candidates len=%d want %d: got=%#v want=%#v", len(got), len(want), got, want)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Name != w.Name || g.Kind != w.Kind || g.File != w.File || g.Line != w.Line || g.Column != w.Column || g.Language != w.Language || g.Container != w.Container || g.Signature != w.Signature || g.SignatureHash != w.SignatureHash {
			t.Fatalf("candidate[%d]=%#v want %#v", i, g, w)
		}
		ge, we := g.Exported != nil && *g.Exported, w.Exported != nil && *w.Exported
		gn, wn := g.Exported != nil, w.Exported != nil
		if gn != wn || ge != we {
			t.Fatalf("candidate[%d].Exported got=%v want=%v", i, g.Exported, w.Exported)
		}
	}
}

func TestSaveLoadReferencesRoundTripByState(t *testing.T) {
	store, absRoot := setupRefStore(t)
	defer store.Close()
	aFile := filepath.Join(absRoot, "a.go")
	bFile := filepath.Join(absRoot, "b.go")

	resolvedTarget := domain.SymbolRef{Name: "C", Kind: "function", File: bFile, Line: 2, Column: 6, Language: "go", Container: "p", Signature: "func C()", Exported: boolPtr(true), SignatureHash: "sh-c"}
	ambCandidates := []domain.SymbolRef{
		{Name: "A", Kind: "function", File: aFile, Line: 3, Column: 6, Language: "go", Container: "p", Signature: "func A()", Exported: boolPtr(true), SignatureHash: "sh-a"},
		{Name: "B", Kind: "function", File: aFile, Line: 4, Column: 6, Language: "go", Container: "p", Signature: "func B()", Exported: boolPtr(true), SignatureHash: "sh-b"},
	}
	evAST := []domain.ReferenceEvidence{{Kind: domain.EvidenceASTCall, Ref: "a.go:3:1", Detail: "call", Method: "ast"}}
	evMulti := []domain.ReferenceEvidence{
		{Kind: domain.EvidenceASTCall, Ref: "a.go:3:1", Detail: "call", Method: "ast"},
		{Kind: domain.EvidenceModuleResolution, Ref: "module_resolution", Detail: "resolved import", Method: "module_resolution"},
	}
	evDyn := []domain.ReferenceEvidence{{Kind: domain.EvidenceDynamicDispatch, Ref: "a.go:3:1", Detail: "dynamic", Method: "dynamic"}}
	evExt := []domain.ReferenceEvidence{{Kind: domain.EvidenceImportDeclaration, Ref: "import:fmt", Detail: "external", Method: "import"}}
	evHeur := []domain.ReferenceEvidence{{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:A", Detail: "candidate A", Method: "heuristic"}}

	cases := []struct {
		name string
		ref  domain.Reference
	}{
		{name: "resolved", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerR", Kind: "function", File: aFile, Line: 3, Column: 1, Language: "go", Container: "p", Signature: "func CallerR()", Exported: boolPtr(false), SignatureHash: "src-r"}, Target: &resolvedTarget, Kind: domain.ReferenceCall, Resolution: domain.ResolutionResolved, Evidence: evAST, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 1}}},
		{name: "ambiguous", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerA", Kind: "function", File: aFile, Line: 3, Column: 2, Language: "go", Container: "p", Signature: "func CallerA()", Exported: boolPtr(false), SignatureHash: "src-a"}, Target: nil, Kind: domain.ReferenceCall, Resolution: domain.ResolutionAmbiguous, Candidates: ambCandidates, Evidence: append(append([]domain.ReferenceEvidence(nil), evAST...), evHeur...), Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 2}}},
		{name: "unresolved", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerU", Kind: "function", File: aFile, Line: 3, Column: 3, Language: "go", Container: "p"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Evidence: evAST, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 3}}},
		{name: "external", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerE", Kind: "function", File: aFile, Line: 3, Column: 4, Language: "go", Container: "p"}, Target: &domain.SymbolRef{Name: "Println", Kind: "function", File: "", Language: "go", Container: "fmt"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionExternal, Evidence: evExt, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 4}}},
		{name: "dynamic", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerD", Kind: "function", File: aFile, Line: 3, Column: 5, Language: "go", Container: "p"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionDynamic, Evidence: evDyn, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 5}}},
		{name: "resolved_multi_evidence", ref: domain.Reference{Source: domain.SymbolRef{Name: "CallerME", Kind: "function", File: aFile, Line: 4, Column: 1, Language: "go"}, Target: &domain.SymbolRef{Name: "C", Kind: "function", File: bFile, Line: 2, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionResolved, Evidence: evMulti, Language: "go", Location: domain.Location{Path: aFile, StartLine: 4, StartColumn: 1}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := index.NormalizeReferences([]domain.Reference{tc.ref})
			if len(normalized) != 1 {
				t.Fatalf("normalized len=%d", len(normalized))
			}
			in := normalized[0]
			if err := store.SaveReferences(context.Background(), normalized, absRoot); err != nil {
				t.Fatal(err)
			}
			got, err := store.LoadReferences(context.Background(), absRoot)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Fatalf("len=%d want 1: %#v", len(got), got)
			}
			out := got[0]

			// Preserved logical fields (contractual for persisted refs).
			if out.Kind != in.Kind || out.Language != in.Language || out.Resolution != in.Resolution {
				t.Fatalf("kind/lang/resolution mismatch: got (%q,%q,%q) want (%q,%q,%q)", out.Kind, out.Language, out.Resolution, in.Kind, in.Language, in.Resolution)
			}
			if out.Location.Path == "" || out.Location.StartLine != in.Location.StartLine || out.Location.StartColumn != in.Location.StartColumn {
				t.Fatalf("location mismatch: got %#v want line=%d col=%d", out.Location, in.Location.StartLine, in.Location.StartColumn)
			}
			if out.Source.Name == "" {
				t.Fatalf("source name lost: %#v", out.Source)
			}
			// Source reconstructed minimally: Name/File/Line(=ref line)/Column(=ref col)/Language preserved.
			if out.Source.File == "" || out.Source.Language != in.Language {
				t.Fatalf("source file/language mismatch: %#v", out.Source)
			}
			// Target reconstruction: LoadReferences only reconstructs target when target_file_id or target_symbol_id is linked.
			// External/dynamic/unresolved refs may carry target metadata in memory without indexed linkage; such target SymbolRef
			// is not part of the persisted reference-row contract and may load as nil.
			if in.Target == nil {
				if out.Target != nil {
					t.Fatalf("expected nil target, got %#v", out.Target)
				}
			} else if in.Resolution == domain.ResolutionResolved || in.Resolution == domain.ResolutionAmbiguous {
				if out.Target == nil {
					t.Fatalf("expected target, got nil")
				}
				if out.Target.Name == "" {
					t.Fatalf("target name lost: %#v", out.Target)
				}
			} else {
				// external/dynamic: target may or may not be reconstructable depending on indexed linkage; not asserted as preserved.
			}
			// Candidates/evidence fully preserved via JSON.
			assertCandidatesEqual(t, out.Candidates, in.Candidates)
			assertEvidenceEqual(t, out.Evidence, in.Evidence)

			// Explicit: not reconstructed as part of current persisted contract for Source/Target SymbolRef.
			// Source.Kind/Container/Signature/Exported/SignatureHash are not loaded from reference rows.
			// Target.Kind/Line/Column/Container/Signature/Exported/SignatureHash are not loaded.
			// Location.EndLine/EndColumn are not persisted. Reference.ID is DB-row derived.
			if out.Source.Kind != "" || out.Source.Container != "" || out.Source.Signature != "" || out.Source.SignatureHash != "" || out.Source.Exported != nil {
				t.Fatalf("unexpected source metadata reconstruction (contract states not reconstructed): %#v", out.Source)
			}
			if out.Target != nil {
				if out.Target.Kind != "" || out.Target.Line != 0 || out.Target.Column != 0 || out.Target.Container != "" || out.Target.Signature != "" || out.Target.SignatureHash != "" || out.Target.Exported != nil {
					t.Fatalf("unexpected target metadata reconstruction (contract states not reconstructed): %#v", out.Target)
				}
			}
			if out.Location.EndLine != 0 || out.Location.EndColumn != 0 {
				t.Fatalf("unexpected location end reconstruction: %#v", out.Location)
			}
			if out.ID == "" {
				t.Fatalf("expected DB-derived ID on load")
			}
		})
	}
}

func TestSaveLoadReferencesSymbolKindsPreservedInCandidates(t *testing.T) {
	store, absRoot := setupRefStore(t)
	defer store.Close()
	aFile := filepath.Join(absRoot, "a.go")
	refs := index.NormalizeReferences([]domain.Reference{{
		Source:     domain.SymbolRef{Name: "X", File: aFile, Line: 1, Language: "go"},
		Kind:       domain.ReferenceCall,
		Resolution: domain.ResolutionAmbiguous,
		Language:   "go",
		Location:   domain.Location{Path: aFile, StartLine: 1, StartColumn: 1},
		Candidates: []domain.SymbolRef{
			{Name: "T", Kind: "type", File: aFile, Line: 2, Language: "go", Container: "p"},
			{Name: "A", Kind: "function", File: aFile, Line: 3, Language: "go", Container: "p", Signature: "func A()", Exported: boolPtr(true)},
		},
		Evidence: []domain.ReferenceEvidence{{Kind: domain.EvidenceHeuristicCandidate, Ref: "heuristic:T", Detail: "cand", Method: "heuristic"}},
	}})
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Candidates) != 2 {
		t.Fatalf("got %#v", got)
	}
	assertCandidatesEqual(t, got[0].Candidates, refs[0].Candidates)
}

func TestSaveLoadReferencesDeterministicAcrossStates(t *testing.T) {
	store, absRoot := setupRefStore(t)
	defer store.Close()
	aFile := filepath.Join(absRoot, "a.go")
	bFile := filepath.Join(absRoot, "b.go")
	refs := index.NormalizeReferences([]domain.Reference{
		{Source: domain.SymbolRef{Name: "U", File: bFile, Line: 2, Column: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: bFile, StartLine: 2, StartColumn: 1}},
		{Source: domain.SymbolRef{Name: "D", File: aFile, Line: 4, Column: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionDynamic, Language: "go", Location: domain.Location{Path: aFile, StartLine: 4, StartColumn: 1}, Evidence: []domain.ReferenceEvidence{{Kind: domain.EvidenceDynamicDispatch, Ref: "a.go:4:1", Detail: "dyn", Method: "dynamic"}}},
		{Source: domain.SymbolRef{Name: "R", File: aFile, Line: 3, Column: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionResolved, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 1}, Target: &domain.SymbolRef{Name: "A", File: aFile, Line: 3, Language: "go"}},
		{Source: domain.SymbolRef{Name: "E", File: aFile, Line: 3, Column: 5, Language: "go"}, Kind: domain.ReferenceImport, Resolution: domain.ResolutionExternal, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 5}, Target: &domain.SymbolRef{Name: "fmt", Language: "go"}},
	})
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	first, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		got, err := store.LoadReferences(context.Background(), absRoot)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(first) {
			t.Fatalf("iter %d len=%d want %d", i, len(got), len(first))
		}
		for j := range first {
			a, b := first[j], got[j]
			if a.Location != b.Location || a.Source.Name != b.Source.Name || a.Kind != b.Kind || a.Resolution != b.Resolution || a.Language != b.Language {
				t.Fatalf("iter %d idx %d mismatch: %#v vs %#v", i, j, a, b)
			}
		}
	}
}

func TestSaveLoadReferencesSourceFileNotIndexableStillSkipped(t *testing.T) {
	store, absRoot := setupRefStore(t)
	defer store.Close()
	aFile := filepath.Join(absRoot, "a.go")
	missing := filepath.Join(absRoot, "missing.go")
	refs := index.NormalizeReferences([]domain.Reference{
		{Source: domain.SymbolRef{Name: "OK", File: aFile, Line: 3, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 1}},
		{Source: domain.SymbolRef{Name: "SKIP", File: missing, Line: 1, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: missing, StartLine: 1, StartColumn: 1}},
	})
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	// P5252-G-E: ref whose source file cannot map to index_files is silently omitted (FK/integrity constraint behavior).
	if len(got) != 1 || got[0].Source.Name != "OK" {
		t.Fatalf("expected only indexable-source ref persisted, got %#v", got)
	}
}

func TestLoadReferencesIDDerivedFromRowAndUnstableAcrossResave(t *testing.T) {
	store, absRoot := setupRefStore(t)
	defer store.Close()
	aFile := filepath.Join(absRoot, "a.go")
	refs := index.NormalizeReferences([]domain.Reference{{
		Source: domain.SymbolRef{Name: "X", File: aFile, Line: 3, Language: "go"}, Kind: domain.ReferenceCall, Resolution: domain.ResolutionUnresolved, Language: "go", Location: domain.Location{Path: aFile, StartLine: 3, StartColumn: 1},
	}})
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	first, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].ID == "" {
		t.Fatalf("expected DB-derived ID, got %#v", first)
	}
	// Resave (delete+insert) produces a new row ID; ID is not a stable logical identity.
	if err := store.SaveReferences(context.Background(), refs, absRoot); err != nil {
		t.Fatal(err)
	}
	second, err := store.LoadReferences(context.Background(), absRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || second[0].ID == "" {
		t.Fatalf("expected DB-derived ID after resave, got %#v", second)
	}
	// Logical content preserved despite ID change.
	if first[0].Location != second[0].Location || first[0].Source.Name != second[0].Source.Name || first[0].Kind != second[0].Kind || first[0].Resolution != second[0].Resolution {
		t.Fatalf("logical content changed across resave: %#v vs %#v", first[0], second[0])
	}
	// ID may differ due to autoincrement/delete+insert behavior; we assert it is row-derived, not semantically stable.
	_ = first[0].ID != second[0].ID
}
