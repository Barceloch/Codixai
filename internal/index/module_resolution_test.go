package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func buildAndResolve(t *testing.T, root string) *ModuleGraph {
	t.Helper()
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	g, err := NewJSModuleResolver().ResolveModules(context.Background(), idx, root)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func findModDep(g *ModuleGraph, src, spec string) *ModuleDependency {
	for i := range g.Dependencies {
		d := &g.Dependencies[i]
		if d.SourceFile == src && d.Specifier == spec {
			return d
		}
	}
	return nil
}

func TestModuleResolverRelativeResolution(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "a.js"), "import {b} from './b.js';\nimport {c} from './c';\nimport {d} from '../lib/d';\nimport {i} from './dir';\n")
	mustWrite(t, filepath.Join(root, "src", "b.js"), "export const b=1;\n")
	mustWrite(t, filepath.Join(root, "src", "c.ts"), "export const c=1;\n")
	mustWrite(t, filepath.Join(root, "lib", "d.jsx"), "export const d=1;\n")
	mustWrite(t, filepath.Join(root, "src", "dir", "index.ts"), "export const i=1;\n")

	g := buildAndResolve(t, root)
	cases := []struct{ spec, wantPath string }{
		{"./b.js", "src/b.js"},
		{"./c", "src/c.ts"},
		{"../lib/d", "lib/d.jsx"},
		{"./dir", "src/dir/index.ts"},
	}
	for _, c := range cases {
		dep := findModDep(g, "src/a.js", c.spec)
		if dep == nil {
			t.Fatalf("missing dep %q: %#v", c.spec, g.Dependencies)
		}
		if dep.Status != ModuleResolved || dep.ResolvedPath != c.wantPath {
			t.Fatalf("dep %q status=%q resolved=%q want %q (%s)", c.spec, dep.Status, dep.ResolvedPath, c.wantPath, dep.Evidence)
		}
	}
}

func TestModuleResolverAmbiguity(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.ts"), "import x from './foo';\nimport y from './bar';\n")
	mustWrite(t, filepath.Join(root, "foo.js"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "foo.ts"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "bar.js"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "bar", "index.js"), "export default 1;\n")

	g := buildAndResolve(t, root)
	if dep := findModDep(g, "a.ts", "./foo"); dep == nil || dep.Status != ModuleAmbiguous {
		t.Fatalf("expected ambiguous ./foo, got %#v", dep)
	} else if len(dep.Candidates) < 2 {
		t.Fatalf("expected candidates for ./foo: %#v", dep)
	}
	if dep := findModDep(g, "a.ts", "./bar"); dep == nil || dep.Status != ModuleAmbiguous {
		t.Fatalf("expected ambiguous ./bar (file vs index), got %#v", dep)
	}
}

func TestModuleResolverBuiltinsAndExternal(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.js"), "import fs from 'node:fs';\nimport fsp from 'node:fs/promises';\nimport path from 'path';\nimport express from 'express';\nimport join from 'lodash/join';\n")
	g := buildAndResolve(t, root)
	for _, spec := range []string{"node:fs", "node:fs/promises", "path", "express", "lodash/join"} {
		dep := findModDep(g, "a.js", spec)
		if dep == nil || dep.Status != ModuleExternal {
			t.Fatalf("expected external/builtin %q, got %#v", spec, dep)
		}
	}
}

func TestModuleResolverJSTSCombinations(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.js"), "import b from './b';\n")
	mustWrite(t, filepath.Join(root, "b.ts"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "c.ts"), "import d from './d';\n")
	mustWrite(t, filepath.Join(root, "d.js"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "e.tsx"), "import f from './f';\n")
	mustWrite(t, filepath.Join(root, "f.tsx"), "export default 1;\n")
	g := buildAndResolve(t, root)
	for _, tc := range []struct{ src, spec, want string }{
		{"a.js", "./b", "b.ts"},
		{"c.ts", "./d", "d.js"},
		{"e.tsx", "./f", "f.tsx"},
	} {
		dep := findModDep(g, tc.src, tc.spec)
		if dep == nil || dep.Status != ModuleResolved || dep.ResolvedPath != tc.want {
			t.Fatalf("combo %s -> %s: got %#v", tc.src, tc.spec, dep)
		}
	}
}

func TestModuleResolverESMCommonJSAndTypeOnly(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.mjs"), "import b from './b.mjs';\n")
	mustWrite(t, filepath.Join(root, "b.mjs"), "export default 1;\n")
	mustWrite(t, filepath.Join(root, "c.cjs"), "const d = require('./d.cjs');\nmodule.exports = d;\n")
	mustWrite(t, filepath.Join(root, "d.cjs"), "module.exports = 1;\n")
	mustWrite(t, filepath.Join(root, "t.ts"), "import type {X} from './x';\nimport {Y} from './x';\n")
	mustWrite(t, filepath.Join(root, "x.ts"), "export type X = string; export const Y = 1;\n")
	g := buildAndResolve(t, root)
	if dep := findModDep(g, "a.mjs", "./b.mjs"); dep == nil || dep.Status != ModuleResolved {
		t.Fatalf("esm dep: %#v", dep)
	}
	if dep := findModDep(g, "c.cjs", "./d.cjs"); dep == nil || dep.Status != ModuleResolved {
		t.Fatalf("cjs require dep: %#v", dep)
	}
	// type-only import still creates a dependency in the index; resolution follows normal path rules.
	if dep := findModDep(g, "t.ts", "./x"); dep == nil || (dep.Status != ModuleResolved && dep.Status != ModuleAmbiguous) {
		t.Fatalf("type/value deps to ./x: %#v deps=%#v", dep, g.Dependencies)
	}
}

func TestModuleResolverDynamicLiteral(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.js"), "import('./b.js');\n")
	mustWrite(t, filepath.Join(root, "b.js"), "export default 1;\n")
	g := buildAndResolve(t, root)
	dep := findModDep(g, "a.js", "./b.js")
	if dep == nil || dep.Status != ModuleResolved {
		t.Fatalf("dynamic literal import recorded by extractor should resolve: %#v", dep)
	}
}

func TestModuleResolverPathTraversalAndSecurity(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "a.js"), "import x from '../../etc/passwd';\nimport y from './missing';\n")
	mustWrite(t, filepath.Join(root, "etc", "passwd"), "nope\n") // inside root but traversal path as written from src; actual resolved escapes? ../../ from src => root/etc/passwd
	g := buildAndResolve(t, root)
	// ../../etc/passwd from src resolves to etc/passwd inside root in this temp layout; the security check is "escapes project root".
	// Better explicit escape:
	root2 := t.TempDir()
	sub := filepath.Join(root2, "proj")
	mustWrite(t, filepath.Join(sub, "a.js"), "import x from '../../../outside';\n")
	g2 := buildAndResolve(t, sub)
	dep := findModDep(g2, "a.js", "../../../outside")
	if dep == nil || dep.Status != ModuleUnresolved {
		t.Fatalf("expected unresolved escape, got %#v", dep)
	}
	_ = g
	if dep := findModDep(g, "src/a.js", "./missing"); dep == nil || dep.Status != ModuleUnresolved {
		t.Fatalf("expected unresolved missing, got %#v", dep)
	}
}

func TestModuleResolverDeterminismAndCycles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.js"), "import './b.js';\n")
	mustWrite(t, filepath.Join(root, "b.js"), "import './a.js';\n")
	g1 := buildAndResolve(t, root)
	g2 := buildAndResolve(t, root)
	if len(g1.Edges) != len(g2.Edges) || len(g1.Dependencies) != len(g2.Dependencies) || len(g1.Nodes) != len(g2.Nodes) {
		t.Fatalf("non-deterministic graph: %#v vs %#v", g1, g2)
	}
	for i := range g1.Dependencies {
		a, b := g1.Dependencies[i], g2.Dependencies[i]
		if a.SourceFile != b.SourceFile || a.Specifier != b.Specifier || a.Status != b.Status || a.ResolvedPath != b.ResolvedPath {
			t.Fatalf("non-deterministic dep %d: %#v vs %#v", i, a, b)
		}
	}
	if len(g1.Edges) != 2 {
		t.Fatalf("expected 2 cycle edges, got %#v", g1.Edges)
	}
}

func TestModuleResolverRegressionMultilanguage(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\nfunc Main() {}\n")
	mustWrite(t, filepath.Join(root, "app.py"), "def py_main():\n    pass\n")
	mustWrite(t, filepath.Join(root, "a.js"), "import b from './b';\nimport path from 'path';\n")
	mustWrite(t, filepath.Join(root, "b.ts"), "export default 1;\n")

	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	// Regression: Go/Python extraction still present.
	if !hasSymbol(idx.Symbols(), "go", "Main", "func") || !hasSymbol(idx.Symbols(), "python", "py_main", "func") {
		t.Fatalf("regression: missing Go/Python symbols: %#v", idx.Symbols())
	}
	g, err := NewJSModuleResolver().ResolveModules(context.Background(), idx, root)
	if err != nil {
		t.Fatal(err)
	}
	if dep := findModDep(g, "a.js", "./b"); dep == nil || dep.Status != ModuleResolved {
		t.Fatalf("expected resolved js dep: %#v", dep)
	}
	for _, d := range g.Dependencies {
		if d.SourceLanguage != "javascript" && d.SourceLanguage != "typescript" {
			t.Fatalf("resolver produced non-JS/TS dep: %#v", d)
		}
	}
}

func TestModuleResolverSymlinkSkipped(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.js"), "import b from './b';\n")
	target := filepath.Join(root, "target.js")
	mustWrite(t, target, "export default 1;\n")
	link := filepath.Join(root, "b.js")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	g := buildAndResolve(t, root)
	dep := findModDep(g, "a.js", "./b")
	if dep == nil {
		t.Fatalf("missing dep: %#v", g.Dependencies)
	}
	// b.js exists as symlink and is skipped; ./b without ext may also find candidates via extension expansion to b.js symlink (skipped) — expect unresolved or resolved only if non-symlink candidate exists.
	if dep.Status == ModuleResolved && dep.ResolvedPath == "b.js" {
		t.Fatalf("symlink target should not be resolved as local file: %#v", dep)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
