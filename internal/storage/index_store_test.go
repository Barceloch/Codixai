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
