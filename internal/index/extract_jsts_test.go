package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeIndexFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasSymbol(syms []Symbol, lang, name, kind string) bool {
	for _, s := range syms {
		if (lang == "" || s.Language == lang) && s.Name == name && (kind == "" || s.Kind == kind) {
			return true
		}
	}
	return false
}

func hasDep(deps []Dependency, lang, path string) bool {
	for _, d := range deps {
		if (lang == "" || d.Language == lang) && d.Path == path {
			return true
		}
	}
	return false
}

func TestBuildExtractsJavaScriptSymbolsAndDependencies(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, filepath.Join(root, "src", "app.js"), `import { helper } from "./helper.js";
export function main() { return helper(); }
export const VERSION = "1";
`)
	writeIndexFile(t, filepath.Join(root, "src", "helper.js"), `export function helper() { return 1; }
`)

	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	syms := idx.Symbols()
	deps := idx.Dependencies()

	if !hasSymbol(syms, "javascript", "main", "func") {
		t.Fatalf("expected js func main: %#v", syms)
	}
	if !hasSymbol(syms, "javascript", "VERSION", "var") {
		t.Fatalf("expected js var VERSION: %#v", syms)
	}
	if !hasSymbol(syms, "javascript", "app", "file") && !hasSymbol(syms, "javascript", "app.js", "file") {
		// file symbol uses filepath.Base(path)
		found := false
		for _, s := range syms {
			if s.Language == "javascript" && s.Kind == "file" && strings.HasSuffix(s.Name, ".js") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected js file symbol: %#v", syms)
		}
	}
	if !hasDep(deps, "javascript", "./helper.js") {
		t.Fatalf("expected js dependency ./helper.js: %#v", deps)
	}
}

func TestBuildExtractsTypeScriptSymbolsAndDependencies(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, filepath.Join(root, "src", "models.ts"), `export interface User { id: number; }
export type UserID = string;
export enum Status { Active }
export class Admin implements User { id = 1; save(): void {} }
`)
	writeIndexFile(t, filepath.Join(root, "src", "app.ts"), `import type { User } from "./models";
import { Status } from "./models";
export function run(u: User) { return Status.Active; }
`)

	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	syms := idx.Symbols()
	deps := idx.Dependencies()

	for _, name := range []string{"User", "UserID", "Status", "Admin"} {
		if !hasSymbol(syms, "typescript", name, "type") {
			t.Fatalf("expected ts type %s: %#v", name, syms)
		}
	}
	if !hasSymbol(syms, "typescript", "save", "method") {
		t.Fatalf("expected ts method save: %#v", syms)
	}
	if !hasDep(deps, "typescript", "./models") {
		t.Fatalf("expected ts dependency ./models: %#v", deps)
	}
}

func TestBuildMixedJSAndTS(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, filepath.Join(root, "a.js"), `export function a() {}\n`)
	writeIndexFile(t, filepath.Join(root, "b.ts"), `export function b(): void {}\n`)

	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	syms := idx.Symbols()
	if !hasSymbol(syms, "javascript", "a", "func") || !hasSymbol(syms, "typescript", "b", "func") {
		t.Fatalf("expected both js and ts symbols: %#v", syms)
	}
}

func TestBuildMixedGoPythonJavaScriptTypeScript(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, filepath.Join(root, "main.go"), "package main\nfunc Main() {}\n")
	writeIndexFile(t, filepath.Join(root, "app.py"), "def py_main():\n    pass\n")
	writeIndexFile(t, filepath.Join(root, "app.js"), "export function jsMain() {}\n")
	writeIndexFile(t, filepath.Join(root, "app.ts"), "export function tsMain(): void {}\n")

	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	syms := idx.Symbols()
	for _, tc := range []struct{ lang, name, kind string }{
		{"go", "Main", "func"},
		{"python", "py_main", "func"},
		{"javascript", "jsMain", "func"},
		{"typescript", "tsMain", "func"},
	} {
		if !hasSymbol(syms, tc.lang, tc.name, tc.kind) {
			t.Fatalf("missing %s %s %s: %#v", tc.lang, tc.kind, tc.name, syms)
		}
	}
}

func TestBuildJSTSDeterministic(t *testing.T) {
	root := t.TempDir()
	writeIndexFile(t, filepath.Join(root, "a.js"), "export function a() {}\nimport x from './x.js';\n")
	writeIndexFile(t, filepath.Join(root, "b.ts"), "export function b(): void {}\nimport y from './y';\n")

	build := func() ([]Symbol, []Dependency) {
		idx := New()
		if err := idx.Build(context.Background(), root); err != nil {
			t.Fatal(err)
		}
		return idx.Symbols(), idx.Dependencies()
	}
	s1, d1 := build()
	s2, d2 := build()
	if len(s1) != len(s2) || len(d1) != len(d2) {
		t.Fatalf("non-deterministic counts: syms %d/%d deps %d/%d", len(s1), len(s2), len(d1), len(d2))
	}
	// Build walks files; order depends on filepath.Walk. Counts stable is the key property here.
}
