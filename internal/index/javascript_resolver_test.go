package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func writeJSFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findJSRef(refs []domain.Reference, name string, kind domain.ReferenceKind) *domain.Reference {
	for i := range refs {
		if refs[i].Source.Name == name && (kind == "" || refs[i].Kind == kind) {
			return &refs[i]
		}
	}
	return nil
}

func findJSRefs(refs []domain.Reference, name string, kind domain.ReferenceKind) []domain.Reference {
	var out []domain.Reference
	for _, r := range refs {
		if r.Source.Name == name && (kind == "" || r.Kind == kind) {
			out = append(out, r)
		}
	}
	return out
}

func resolveJS(t *testing.T, root string, files ...string) []domain.Reference {
	t.Helper()
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	var changes []domain.FileChange
	for _, f := range files {
		lang := domain.LangJavaScript
		if ext := filepath.Ext(f); ext == ".ts" || ext == ".tsx" || ext == ".mts" || ext == ".cts" {
			lang = domain.LangTypeScript
		}
		changes = append(changes, domain.FileChange{Path: f, Status: domain.FileModified, Language: lang})
	}
	refs, err := NewJavaScriptReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root:   root,
		Origin: domain.OriginWorktree,
		Files:  changes,
	})
	if err != nil {
		t.Fatal(err)
	}
	return refs
}

func TestJSResolverSameFileFunctionCall(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `function callee() {}
function caller() { callee(); }
`)
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "callee", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to callee, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v evidence=%#v", ref.Resolution, ref.Target, ref.Evidence)
	}
	if ref.Target == nil || ref.Target.File != "a.js" {
		t.Fatalf("target=%#v want a.js", ref.Target)
	}
}

func TestJSResolverSameFileClassConstructor(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `class User {}
const u = new User();
`)
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "User", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected constructor ref to User, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved", ref.Resolution)
	}
}

func TestJSResolverThisMethod(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `class UserService {
  create() {}
  run() { this.create(); }
}
`)
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "create", domain.ReferenceMethod)
	if ref == nil {
		t.Fatalf("expected this.create method ref, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v", ref.Resolution, ref.Target)
	}
}

func TestJSResolverNamedImport(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `export function createUser() {}
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import { createUser } from "./users.js";
createUser();
`)
	refs := resolveJS(t, root, "app.js", "users.js")
	ref := findJSRef(refs, "createUser", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to createUser, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v candidates=%#v evidence=%#v", ref.Resolution, ref.Target, ref.Candidates, ref.Evidence)
	}
	if ref.Target == nil || filepath.Base(ref.Target.File) != "users.js" {
		t.Fatalf("target=%#v want users.js", ref.Target)
	}
}

func TestJSResolverNamedImportAlias(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `export function createUser() {}
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import { createUser as create } from "./users.js";
create();
`)
	refs := resolveJS(t, root, "app.js", "users.js")
	ref := findJSRef(refs, "create", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to aliased create, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v", ref.Resolution, ref.Target)
	}
	if ref.Target == nil || ref.Target.Name != "createUser" {
		t.Fatalf("target=%#v want original createUser", ref.Target)
	}
}

func TestJSResolverDefaultImport(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "service.js"), `export default function UserService() {}
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import UserService from "./service.js";
UserService();
`)
	refs := resolveJS(t, root, "app.js", "service.js")
	ref := findJSRef(refs, "UserService", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to default import, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved", ref.Resolution)
	}
}

func TestJSResolverNamespaceImport(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `export function createUser() {}
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import * as users from "./users.js";
users.createUser();
`)
	refs := resolveJS(t, root, "app.js", "users.js")
	ref := findJSRef(refs, "createUser", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected namespace member call createUser, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v", ref.Resolution, ref.Target)
	}
}

func TestJSResolverNonexistentExport(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `export function existing() {}
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import { missing } from "./users.js";
missing();
`)
	refs := resolveJS(t, root, "app.js", "users.js")
	found := false
	for i := range refs {
		r := &refs[i]
		if r.Source.Name == "missing" {
			found = true
			if r.Resolution == domain.ResolutionResolved {
				t.Fatalf("missing export should not resolve: %#v", r)
			}
		}
	}
	if !found {
		t.Fatalf("expected reference/import record for missing, got %#v", refs)
	}
}

func TestJSResolverExternalBuiltin(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "app.js"), `import fs from "node:fs";
fs.readFile();
`)
	refs := resolveJS(t, root, "app.js")
	foundExternal := false
	for _, r := range refs {
		if r.Source.Name == "fs" || r.Source.Name == "readFile" {
			if r.Resolution == domain.ResolutionExternal || r.Resolution == domain.ResolutionDynamic || r.Resolution == domain.ResolutionUnresolved {
				foundExternal = true
			}
		}
	}
	if !foundExternal {
		t.Fatalf("expected external/dynamic handling for builtin module usage, got %#v", refs)
	}
}

func TestJSResolverUnknownReceiverDynamic(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "app.js"), `function run(service) { service.create(); }
`)
	refs := resolveJS(t, root, "app.js")
	ref := findJSRef(refs, "create", domain.ReferenceMethod)
	if ref == nil {
		for i := range refs {
			if refs[i].Source.Name == "create" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected ref to create, got %#v", refs)
	}
	if ref.Resolution == domain.ResolutionResolved {
		t.Fatalf("unknown receiver should not resolve confidently: %#v", ref)
	}
}

func TestJSResolverDeterministic(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `function a() {}
function b() { a(); }
`)
	r1 := resolveJS(t, root, "a.js")
	r2 := resolveJS(t, root, "a.js")
	if len(r1) != len(r2) {
		t.Fatalf("len %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Source.Name != r2[i].Source.Name || r1[i].Resolution != r2[i].Resolution || r1[i].Location.StartLine != r2[i].Location.StartLine {
			t.Fatalf("nondeterministic at %d: %#v vs %#v", i, r1[i], r2[i])
		}
	}
}

func TestJSResolverDoesNotProcessTypeScript(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.ts"), `function a() { a(); }
`)
	refs := resolveJS(t, root, "a.ts")
	if len(refs) != 0 {
		t.Fatalf("JS resolver should not produce TS refs, got %#v", refs)
	}
}

func TestJSResolverShadowing(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `class UserService { create() {} }
const service = new UserService();
function execute() {
  const service = getUnknownService();
  service.create();
}
`)
	refs := resolveJS(t, root, "a.js")
	// Find create() call inside execute (unknown shadowed receiver) - should not resolve to UserService.create.
	for _, r := range refs {
		if r.Source.Name == "create" && r.Kind == domain.ReferenceMethod && r.Location.StartLine >= 5 {
			if r.Resolution == domain.ResolutionResolved && r.Target != nil && r.Target.Container == "UserService" {
				t.Fatalf("shadowed receiver resolved incorrectly to UserService: %#v", r)
			}
		}
	}
}

func TestJSResolverCommonJSRequireDestructure(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `function createUser() {}
module.exports = { createUser };
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `const { createUser } = require("./users.js");
createUser();
`)
	refs := resolveJS(t, root, "app.js", "users.js")
	ref := findJSRef(refs, "createUser", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to createUser, got %#v", refs)
	}
	// Conservative: may resolve or be unresolved depending on export extraction; must not falsely resolve to wrong symbol.
	if ref.Resolution == domain.ResolutionResolved && ref.Target != nil && ref.Target.File != "users.js" && filepath.Base(ref.Target.File) != "users.js" {
		t.Fatalf("resolved to wrong file: %#v", ref.Target)
	}
}

func TestJSResolverReexportNamed(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "users.js"), `export function createUser() {}
`)
	writeJSFile(t, filepath.Join(root, "index.js"), `export { createUser } from "./users.js";
`)
	writeJSFile(t, filepath.Join(root, "app.js"), `import { createUser } from "./index.js";
createUser();
`)
	refs := resolveJS(t, root, "app.js", "index.js", "users.js")
	ref := findJSRef(refs, "createUser", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to reexported createUser, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved target=%#v", ref.Resolution, ref.Target)
	}
}

func TestJSResolverReexportCycle(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `export { b } from "./b.js";
export function a() {}
`)
	writeJSFile(t, filepath.Join(root, "b.js"), `export { a } from "./a.js";
export function b() {}
`)
	refs := resolveJS(t, root, "a.js", "b.js")
	// Must not hang or panic; resolutions may be unresolved/ambiguous.
	_ = refs
}

func TestJSResolverVarAssignedFunction(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `const calculate = () => {};
calculate();
`)
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "calculate", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to calculate, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved", ref.Resolution)
	}
}

func TestJSResolverParamShadowing(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `function create() {}
function run(create) { create(); }
`)
	refs := resolveJS(t, root, "a.js")
	// The call inside run uses parameter create - should not resolve to top-level create.
	for _, r := range refs {
		if r.Source.Name == "create" && r.Kind == domain.ReferenceCall && r.Location.StartLine == 2 {
			if r.Resolution == domain.ResolutionResolved && r.Target != nil && r.Target.Line == 1 {
				t.Fatalf("param-shadowed call incorrectly resolved to top-level: %#v", r)
			}
		}
	}
}

func TestJSResolverComputedPropertyDynamic(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `function run(obj, name) { obj[name](); }
`)
	refs := resolveJS(t, root, "a.js")
	for _, r := range refs {
		if r.Resolution == domain.ResolutionResolved && r.Source.Name == "name" {
			t.Fatalf("computed access should not be confidently resolved as symbol call target: %#v", r)
		}
	}
}

func TestJSResolverUnknownMethodNotResolved(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `function run(svc) { svc.missing(); }
`)
	refs := resolveJS(t, root, "a.js")
	for _, r := range refs {
		if r.Source.Name == "missing" && r.Resolution == domain.ResolutionResolved {
			t.Fatalf("unknown receiver method should not resolve: %#v", r)
		}
	}
}

func TestJSResolverExtendsRef(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), `class Base {}
class Child extends Base {}
`)
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "Base", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected extends type ref to Base, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved", ref.Resolution)
	}
}

func TestJSResolverRegressionWithGoPython(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "main.go"), "package main\nfunc Main() {}\n")
	writeJSFile(t, filepath.Join(root, "app.py"), "def py_main():\n    pass\n")
	writeJSFile(t, filepath.Join(root, "a.js"), "function jsMain() {}\njsMain();\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	changes := &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{{Path: "a.js", Status: domain.FileModified, Language: domain.LangJavaScript}}}
	jsRefs, err := NewJavaScriptReferenceResolver().Resolve(context.Background(), idx, changes)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range jsRefs {
		if r.Language != domain.LangJavaScript {
			t.Fatalf("JS resolver produced non-JS ref: %#v", r)
		}
	}
	// Ensure Go/Python symbols still present.
	syms := idx.Symbols()
	if !hasSymbol(syms, "go", "Main", "func") || !hasSymbol(syms, "python", "py_main", "func") {
		t.Fatalf("regression: missing Go/Python symbols: %#v", syms)
	}
}

func TestJSResolverTDZCallConservativeAmbiguity(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), "function target() {}\nfunction outer() {\n  target();\n  let target = () => {};\n}\n")
	refs := resolveJS(t, root, "a.js")
	ref := findJSRef(refs, "target", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call reference, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionAmbiguous {
		t.Fatalf("TDZ call resolution = %q, want ambiguous (conservative)", ref.Resolution)
	}
	if ref.Target != nil || len(ref.Candidates) < 2 {
		t.Fatalf("TDZ call must keep candidates and no target: %#v", ref)
	}
}

func TestJSResolverReexportMissingTargetUnresolved(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "barrel.js"), "export * from \"./missing.js\";\n")
	writeJSFile(t, filepath.Join(root, "app.js"), "import { thing } from \"./barrel.js\";\nfunction f() { thing(); }\n")
	refs := resolveJS(t, root, "app.js", "barrel.js")
	ref := findJSRef(refs, "thing", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionUnresolved {
		t.Fatalf("missing reexport target should stay unresolved: %#v", ref)
	}
}

func TestJSResolverAliasedReexportResolves(t *testing.T) {
	root := t.TempDir()
	writeJSFile(t, filepath.Join(root, "a.js"), "export function foo() {}\n")
	writeJSFile(t, filepath.Join(root, "barrel.js"), "export { foo as bar } from \"./a.js\";\n")
	writeJSFile(t, filepath.Join(root, "app.js"), "import { bar } from \"./barrel.js\";\nfunction f() { bar(); }\n")
	refs := resolveJS(t, root, "app.js", "barrel.js", "a.js")
	ref := findJSRef(refs, "bar", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionResolved || ref.Target == nil {
		t.Fatalf("aliased reexport should resolve to the original: %#v", ref)
	}
	if ref.Target.File != "a.js" || ref.Target.Name != "foo" {
		t.Fatalf("unexpected reexport target: %#v", ref.Target)
	}
}
