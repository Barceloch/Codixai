package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/storage"
)

// The tests in this file cover the application-level reference resolution
// orchestration (RunResolveReferences). They were newly authored for the
// Phase 5.2.6 preparation, informed by a historical partial reconstruction
// from the WU3/WU4 recovery. They are not an exact recovery of the deleted
// test file; see the WU4 ODD notes and the preparation report for provenance.

func writeResolveFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunResolveReferencesMixedGoAndPython(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(t.TempDir(), "index.db")

	writeResolveFile(t, filepath.Join(root, "go.mod"), "module example.com/mixed\n\ngo 1.22\n")
	writeResolveFile(t, filepath.Join(root, "util", "util.go"), "package util\n\ntype User struct{ Name string }\n\nfunc Hello() string { return \"hi\" }\n")
	writeResolveFile(t, filepath.Join(root, "main.go"), "package main\n\nimport \"example.com/mixed/util\"\n\nfunc run() string {\n\t_ = util.User{}\n\treturn util.Hello()\n}\n")

	writeResolveFile(t, filepath.Join(root, "pylib", "__init__.py"), "")
	writeResolveFile(t, filepath.Join(root, "pylib", "models.py"), "class User:\n    pass\n\ndef hello():\n    return \"hi\"\n")
	writeResolveFile(t, filepath.Join(root, "app.py"), "from pylib.models import User, hello\n\ndef run():\n    _ = User()\n    return hello()\n")

	changes := &domain.ChangeSet{
		Root:   root,
		Origin: domain.OriginWorktree,
		Files: []domain.FileChange{
			{Path: "main.go", Status: domain.FileModified, Language: domain.LangGo},
			{Path: "app.py", Status: domain.FileModified, Language: domain.LangPython},
		},
	}

	result, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{
		Root:      root,
		IndexPath: indexPath,
		Changes:   changes,
	})
	if err != nil {
		t.Fatalf("RunResolveReferences: %v", err)
	}

	var goRefs, pyRefs int
	for _, ref := range result.References {
		if ref.Language == domain.LangGo {
			goRefs++
		}
		if ref.Language == domain.LangPython {
			pyRefs++
		}
		if ref.Language == "" || ref.Location.Path == "" || ref.Source.Name == "" || ref.Kind == "" || ref.Resolution == "" {
			t.Fatalf("reference missing identity fields: %#v", ref)
		}
		if ref.Resolution == domain.ResolutionAmbiguous && len(ref.Candidates) == 0 {
			t.Fatalf("ambiguous reference without candidates: %#v", ref)
		}
	}
	if goRefs == 0 {
		t.Fatalf("expected Go references in mixed project, got %#v", result.References)
	}
	if pyRefs == 0 {
		t.Fatalf("expected Python references in mixed project, got %#v", result.References)
	}

	// Deterministic order for the same input.
	result2, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{
		Root:      root,
		IndexPath: indexPath,
		Changes:   changes,
	})
	if err != nil {
		t.Fatalf("RunResolveReferences second run: %v", err)
	}
	if len(result.References) != len(result2.References) {
		t.Fatalf("non-deterministic count: %d vs %d", len(result.References), len(result2.References))
	}
	for i := range result.References {
		a, b := result.References[i], result2.References[i]
		if a.Location.Path != b.Location.Path || a.Location.StartLine != b.Location.StartLine || a.Location.StartColumn != b.Location.StartColumn || a.Source.Name != b.Source.Name || a.Kind != b.Kind || a.Resolution != b.Resolution || a.Language != b.Language {
			t.Fatalf("non-deterministic reference at %d: %#v vs %#v", i, a, b)
		}
	}

	// Combined persistence round-trip.
	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	loaded, err := store.LoadReferences(context.Background(), root)
	if err != nil {
		t.Fatalf("LoadReferences: %v", err)
	}
	if len(loaded) == 0 {
		t.Fatal("expected persisted mixed references")
	}
	var loadedGo, loadedPy int
	for _, ref := range loaded {
		if ref.Language == domain.LangGo {
			loadedGo++
		}
		if ref.Language == domain.LangPython {
			loadedPy++
		}
		if ref.Location.Path == "" || ref.Source.Name == "" || ref.Resolution == "" || ref.Kind == "" || ref.Language == "" {
			t.Fatalf("loaded reference missing fields: %#v", ref)
		}
	}
	if loadedGo == 0 || loadedPy == 0 {
		t.Fatalf("expected both languages after load, got go=%d python=%d refs=%#v", loadedGo, loadedPy, loaded)
	}
	if len(loaded) != len(result.References) {
		t.Fatalf("loaded count %d != original %d", len(loaded), len(result.References))
	}
}

func TestRunResolveReferencesSingleLanguageIsolation(t *testing.T) {
	rootGo := t.TempDir()
	writeResolveFile(t, filepath.Join(rootGo, "go.mod"), "module example.com/onlygo\n\ngo 1.22\n")
	writeResolveFile(t, filepath.Join(rootGo, "a.go"), "package main\nfunc a() {}\nfunc b() { a() }\n")
	goRes, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{
		Root:      rootGo,
		IndexPath: filepath.Join(t.TempDir(), "go.db"),
		Changes: &domain.ChangeSet{Root: rootGo, Origin: domain.OriginWorktree, Files: []domain.FileChange{
			{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range goRes.References {
		if ref.Language == domain.LangPython {
			t.Fatalf("Go-only project produced Python reference: %#v", ref)
		}
	}

	rootPy := t.TempDir()
	writeResolveFile(t, filepath.Join(rootPy, "a.py"), "def a():\n    pass\n\ndef b():\n    a()\n")
	pyRes, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{
		Root:      rootPy,
		IndexPath: filepath.Join(t.TempDir(), "py.db"),
		Changes: &domain.ChangeSet{Root: rootPy, Origin: domain.OriginWorktree, Files: []domain.FileChange{
			{Path: "a.py", Status: domain.FileModified, Language: domain.LangPython},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range pyRes.References {
		if ref.Language == domain.LangGo {
			t.Fatalf("Python-only project produced Go reference: %#v", ref)
		}
	}
}

func TestRunResolveReferencesTSNotProcessedByJSResolver(t *testing.T) {
	root := t.TempDir()
	writeResolveFile(t, filepath.Join(root, "a.ts"), "function a() { a(); }\n")
	res, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{
		Root: root, IndexPath: filepath.Join(t.TempDir(), "i.db"),
		Changes: &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{
			{Path: "a.ts", Status: domain.FileModified, Language: domain.LangTypeScript},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range res.References {
		if ref.Language == domain.LangJavaScript {
			t.Fatalf("JS resolver produced ref for TS file: %#v", ref)
		}
	}
}

func TestRunResolveReferencesTypeScript(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(t.TempDir(), "index.db")
	writeResolveFile(t, filepath.Join(root, "models.ts"), "export interface User {}\nexport class UserService { create(): User { return {} as User; } }\n")
	writeResolveFile(t, filepath.Join(root, "app.ts"), "import type { User } from \"./models\";\nimport { UserService } from \"./models\";\nconst u: User = {} as any;\nnew UserService();\n")
	changes := &domain.ChangeSet{
		Root:   root,
		Origin: domain.OriginWorktree,
		Files: []domain.FileChange{
			{Path: "app.ts", Status: domain.FileModified, Language: domain.LangTypeScript},
			{Path: "models.ts", Status: domain.FileModified, Language: domain.LangTypeScript},
		},
	}
	result, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{Root: root, IndexPath: indexPath, Changes: changes})
	if err != nil {
		t.Fatalf("RunResolveReferences: %v", err)
	}
	var tsResolved bool
	for _, ref := range result.References {
		if ref.Language != domain.LangTypeScript && ref.Language != "" {
			t.Fatalf("unexpected language in TS-only project: %#v", ref)
		}
		if ref.Source.Name == "UserService" && ref.Resolution == domain.ResolutionResolved {
			tsResolved = true
		}
	}
	if !tsResolved {
		t.Fatalf("expected resolved TS reference, got %#v", result.References)
	}
	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	loaded, err := store.LoadReferences(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) == 0 {
		t.Fatal("expected persisted TS references")
	}
}

func TestRunResolveReferencesMixedGoPythonJSTypeScript(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(t.TempDir(), "index.db")
	writeResolveFile(t, filepath.Join(root, "go.mod"), "module example.com/mixed4\n\ngo 1.22\n")
	writeResolveFile(t, filepath.Join(root, "main.go"), "package main\nfunc A() {}\nfunc B() { A() }\n")
	writeResolveFile(t, filepath.Join(root, "a.py"), "def a():\n    pass\n\ndef b():\n    a()\n")
	writeResolveFile(t, filepath.Join(root, "a.js"), "export function a() {}\n")
	writeResolveFile(t, filepath.Join(root, "b.js"), "import { a } from \"./a.js\";\na();\n")
	writeResolveFile(t, filepath.Join(root, "a.ts"), "export interface U {}\n")
	writeResolveFile(t, filepath.Join(root, "b.ts"), "import type { U } from \"./a\";\nconst x: U = {} as any;\n")
	changes := &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{
		{Path: "main.go", Status: domain.FileModified, Language: domain.LangGo},
		{Path: "a.py", Status: domain.FileModified, Language: domain.LangPython},
		{Path: "b.js", Status: domain.FileModified, Language: domain.LangJavaScript},
		{Path: "b.ts", Status: domain.FileModified, Language: domain.LangTypeScript},
	}}
	result, err := RunResolveReferences(context.Background(), ResolveReferencesRequest{Root: root, IndexPath: indexPath, Changes: changes})
	if err != nil {
		t.Fatalf("RunResolveReferences: %v", err)
	}
	langs := map[string]bool{}
	for _, ref := range result.References {
		langs[ref.Language] = true
	}
	for _, want := range []string{domain.LangGo, domain.LangPython, domain.LangJavaScript, domain.LangTypeScript} {
		if !langs[want] {
			t.Fatalf("expected %s refs in mixed project, got langs=%#v refs=%#v", want, langs, result.References)
		}
	}
	for _, ref := range result.References {
		if ref.Language == domain.LangJavaScript && filepath.Ext(ref.Location.Path) == ".ts" {
			t.Fatalf("JS resolver produced ref for TS path: %#v", ref)
		}
		if ref.Language == domain.LangTypeScript && filepath.Ext(ref.Location.Path) == ".js" {
			t.Fatalf("TS resolver produced ref for JS path: %#v", ref)
		}
	}
}
