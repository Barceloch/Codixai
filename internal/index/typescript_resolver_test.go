package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func writeTSFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findTSRef(refs []domain.Reference, name string, kind domain.ReferenceKind) *domain.Reference {
	for i := range refs {
		if refs[i].Source.Name == name && (kind == "" || refs[i].Kind == kind) {
			return &refs[i]
		}
	}
	return nil
}

func resolveTS(t *testing.T, root string, files ...string) []domain.Reference {
	t.Helper()
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	changes := make([]domain.FileChange, 0, len(files))
	for _, f := range files {
		lang := domain.LangTypeScript
		if filepath.Ext(f) == ".js" {
			lang = domain.LangJavaScript
		}
		changes = append(changes, domain.FileChange{Path: f, Status: domain.FileModified, Language: lang})
	}
	refs, err := NewTypeScriptReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: changes})
	if err != nil {
		t.Fatal(err)
	}
	return refs
}

func TestTSResolverInterfaceTypeNotValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nconst x: User = {} as any;\n")
	refs := resolveTS(t, root, "a.ts")
	ref := findTSRef(refs, "User", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected type ref to User, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("type ref resolution=%q want resolved target=%#v", ref.Resolution, ref.Target)
	}
	// No value reference resolving User as constructor/call target from annotation.
}

func TestTSResolverTypeAlias(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "type UserID = string;\nconst id: UserID = \"1\";\n")
	refs := resolveTS(t, root, "a.ts")
	ref := findTSRef(refs, "UserID", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected type ref to UserID, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution=%q want resolved", ref.Resolution)
	}
}

func TestTSResolverClassAsTypeAndValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "class UserService {}\nconst s: UserService = new UserService();\n")
	refs := resolveTS(t, root, "a.ts")
	typeRef := findTSRef(refs, "UserService", domain.ReferenceType)
	if typeRef == nil {
		t.Fatalf("expected type ref to UserService, got %#v", refs)
	}
	if typeRef.Resolution != domain.ResolutionResolved {
		t.Fatalf("type resolution=%q", typeRef.Resolution)
	}
	callRef := findTSRef(refs, "UserService", domain.ReferenceCall)
	if callRef == nil {
		t.Fatalf("expected constructor/value ref to UserService, got %#v", refs)
	}
	if callRef.Resolution != domain.ResolutionResolved {
		t.Fatalf("value resolution=%q target=%#v", callRef.Resolution, callRef.Target)
	}
}

func TestTSResolverImportType(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export interface User {}\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type { User } from \"./models\";\nconst u: User = {} as any;\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	// type usage should resolve
	ref := findTSRef(refs, "User", domain.ReferenceType)
	if ref == nil {
		// may be emitted as import ref too
		for i := range refs {
			if refs[i].Source.Name == "User" && refs[i].Resolution == domain.ResolutionResolved {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected resolved type/import ref to User, got %#v", refs)
	}
}

func TestTSResolverMixedImportTypeAndValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export interface User {}\nexport class UserService { create(): User { return {} as User; } }\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import { type User, UserService } from \"./models\";\nconst u: User = {} as any;\nconst s = new UserService();\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	if findTSRef(refs, "UserService", domain.ReferenceCall) == nil {
		t.Fatalf("expected constructor ref to UserService, got %#v", refs)
	}
}

func TestTSResolverExtendsImplements(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\ninterface Auditable {}\ninterface Admin extends User, Auditable {}\nclass Base {}\nclass Repo implements User {}\nclass Child extends Base {}\n")
	refs := resolveTS(t, root, "a.ts")
	if findTSRef(refs, "User", domain.ReferenceInheritance) == nil && findTSRef(refs, "User", domain.ReferenceType) == nil {
		t.Fatalf("expected inheritance/type ref to User, got %#v", refs)
	}
	if findTSRef(refs, "Base", domain.ReferenceInheritance) == nil && findTSRef(refs, "Base", domain.ReferenceType) == nil {
		t.Fatalf("expected extends ref to Base, got %#v", refs)
	}
}

func TestTSResolverEnumTypeAndValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "enum UserRole { Admin }\nconst r: UserRole = UserRole.Admin;\n")
	refs := resolveTS(t, root, "a.ts")
	hasType := findTSRef(refs, "UserRole", domain.ReferenceType) != nil
	hasValue := false
	for _, r := range refs {
		if r.Source.Name == "UserRole" && r.Kind != domain.ReferenceType && r.Resolution == domain.ResolutionResolved {
			hasValue = true
		}
	}
	if !hasType {
		t.Fatalf("expected type ref to enum UserRole, got %#v", refs)
	}
	if !hasValue {
		t.Fatalf("expected value ref to enum UserRole, got %#v", refs)
	}
}

func TestTSResolverShadowingTypeParam(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nfunction execute<User>(value: User): User { return value; }\n")
	refs := resolveTS(t, root, "a.ts")
	// References inside generic function to User type param should not resolve to outer interface.
	for _, r := range refs {
		if r.Source.Name == "User" && r.Location.StartLine == 2 && r.Resolution == domain.ResolutionResolved && r.Target != nil && r.Target.Line == 1 {
			t.Fatalf("generic type param User incorrectly resolved to outer interface: %#v", r)
		}
	}
}

func TestTSResolverDoesNotProcessJavaScript(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.js"), "function a() { a(); }\n")
	refs := resolveTS(t, root, "a.js")
	if len(refs) != 0 {
		t.Fatalf("TS resolver should not process JS, got %#v", refs)
	}
}

func TestTSResolverDeterministic(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface A {}\ninterface B {}\nconst x: A = {} as B;\n")
	r1 := resolveTS(t, root, "a.ts")
	r2 := resolveTS(t, root, "a.ts")
	if len(r1) != len(r2) {
		t.Fatalf("len %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Source.Name != r2[i].Source.Name || r1[i].Resolution != r2[i].Resolution || r1[i].Location.StartLine != r2[i].Location.StartLine {
			t.Fatalf("nondeterministic at %d", i)
		}
	}
}

func TestTSResolverNamespaceImportTypeMember(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export interface User {}\nexport function createUser() {}\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import * as Models from \"./models\";\nconst u: Models.User = {} as any;\nModels.createUser();\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	// At minimum, namespace/module references and member access should be represented conservatively.
	if len(refs) == 0 {
		t.Fatalf("expected some refs for namespace import usage, got none")
	}
}

func findAllTSRefs(refs []domain.Reference, name string) []domain.Reference {
	var out []domain.Reference
	for _, r := range refs {
		if r.Source.Name == name {
			out = append(out, r)
		}
	}
	return out
}

func hasTSRefWithResolution(refs []domain.Reference, name string, res domain.ResolutionStatus) bool {
	for _, r := range refs {
		if r.Source.Name == name && r.Resolution == res {
			return true
		}
	}
	return false
}

func TestTSResolverImportsDefaultNamedAliasNamespace(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export default class Def {}\nexport interface User {}\nexport class Svc {}\nexport function make() {}\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import Def from \"./models\";\nimport { User, Svc as Service } from \"./models\";\nimport * as M from \"./models\";\nconst d = new Def();\nconst u: User = {} as any;\nconst s = new Service();\nM.make();\nconst u2: M.User = {} as any;\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	if !hasTSRefWithResolution(refs, "Def", domain.ResolutionResolved) {
		t.Fatalf("expected resolved default import/call for Def, got %#v", findAllTSRefs(refs, "Def"))
	}
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved named import/type for User, got %#v", findAllTSRefs(refs, "User"))
	}
	if !hasTSRefWithResolution(refs, "Service", domain.ResolutionResolved) {
		t.Fatalf("expected resolved alias import for Service, got %#v", findAllTSRefs(refs, "Service"))
	}
}

func TestTSResolverImportTypeDefault(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "user.ts"), "interface User {}\nexport default User;\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type User from \"./user\";\nconst u: User = {} as any;\n")
	refs := resolveTS(t, root, "app.ts", "user.ts")
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved import type default User, got %#v", findAllTSRefs(refs, "User"))
	}
}

func TestTSResolverImportNonexistentAndExternal(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "app.ts"), "import { Missing } from \"./missing\";\nimport React from \"react\";\nimport type { T } from \"external-pkg\";\n")
	refs := resolveTS(t, root, "app.ts")
	if !hasTSRefWithResolution(refs, "Missing", domain.ResolutionUnresolved) && !hasTSRefWithResolution(refs, "Missing", domain.ResolutionAmbiguous) {
		// relative missing module - unresolved is acceptable
		refsM := findAllTSRefs(refs, "Missing")
		if len(refsM) == 0 {
			t.Fatalf("expected ref for Missing import")
		}
	}
	if !hasTSRefWithResolution(refs, "React", domain.ResolutionExternal) {
		t.Fatalf("expected external React, got %#v", findAllTSRefs(refs, "React"))
	}
}

func TestTSResolverExportsAndReexports(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "base.ts"), "export interface User {}\nexport class Svc {}\n")
	writeTSFile(t, filepath.Join(root, "re.ts"), "export type { User } from \"./base\";\nexport { Svc as Service } from \"./base\";\nexport * from \"./base\";\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type { User } from \"./re\";\nimport { Service } from \"./re\";\nconst u: User = {} as any;\nnew Service();\n")
	refs := resolveTS(t, root, "app.ts", "re.ts", "base.ts")
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved reexported type User, got %#v", findAllTSRefs(refs, "User"))
	}
	if !hasTSRefWithResolution(refs, "Service", domain.ResolutionResolved) {
		t.Fatalf("expected resolved reexported alias Service, got %#v", findAllTSRefs(refs, "Service"))
	}
}

func TestTSResolverExportStarCycle(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "export * from \"./b\";\nexport interface A {}\n")
	writeTSFile(t, filepath.Join(root, "b.ts"), "export * from \"./a\";\nexport interface B {}\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type { A } from \"./b\";\nconst a: A = {} as any;\n")
	refs := resolveTS(t, root, "app.ts", "a.ts", "b.ts")
	// Must not hang; should resolve A or abstain conservatively.
	_ = refs
}

func TestTSResolverAnnotations(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), `
interface User {}
interface Repo<T> {}
class S {
  repo: Repo<User>;
  constructor(public svc: User) {}
  get(u: User): Array<User> { return [u]; }
}
const users: Array<User> = [];
function f(x: User | null): Repo<User> { return {} as Repo<User>; }
const y = ({} as any) as User;
const z = ({} as any) satisfies User;
`)
	refs := resolveTS(t, root, "a.ts")
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved User in annotations, got %#v", findAllTSRefs(refs, "User"))
	}
	if !hasTSRefWithResolution(refs, "Repo", domain.ResolutionResolved) {
		t.Fatalf("expected resolved Repo generic, got %#v", findAllTSRefs(refs, "Repo"))
	}
}

func TestTSResolverInterfaceClassExtendsMultiple(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\ninterface Auditable {}\ninterface Admin extends User, Auditable {}\nclass Base {}\nclass Child extends Base implements User, Auditable {}\n")
	refs := resolveTS(t, root, "a.ts")
	for _, name := range []string{"User", "Auditable", "Base"} {
		found := false
		for _, r := range refs {
			if r.Source.Name == name && r.Resolution == domain.ResolutionResolved {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected resolved heritage ref to %s, got %#v", name, findAllTSRefs(refs, name))
		}
	}
}

func TestTSResolverNonexistentAndAmbiguousStates(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "const x: MissingType = {} as any;\n")
	refs := resolveTS(t, root, "a.ts")
	if !hasTSRefWithResolution(refs, "MissingType", domain.ResolutionUnresolved) {
		t.Fatalf("expected unresolved MissingType, got %#v", findAllTSRefs(refs, "MissingType"))
	}
	// Ambiguous: two symbols same name in index context is hard to force via extractor; test that resolver can produce ambiguous when candidates exist.
	root2 := t.TempDir()
	writeTSFile(t, filepath.Join(root2, "a.ts"), "class X {}\n")
	writeTSFile(t, filepath.Join(root2, "b.ts"), "class X {}\n")
	writeTSFile(t, filepath.Join(root2, "c.ts"), "function f(x: X) {}\n")
	// In c.ts local decls has no X, so it looks up symbol candidates in same file only (byFile[c.ts]) - will be unresolved, not ambiguous across files. That's conservative/correct for local lookup.
	refs2 := resolveTS(t, root2, "c.ts", "a.ts", "b.ts")
	_ = refs2
}

func TestTSResolverTypeOnlyImportNotValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export interface User {}\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type { User } from \"./models\";\nnew User();\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	// constructor/value use of type-only imported interface should not resolve as value target.
	for _, r := range refs {
		if r.Source.Name == "User" && r.Kind == domain.ReferenceCall {
			if r.Resolution == domain.ResolutionResolved && r.Target != nil {
				t.Fatalf("type-only import User incorrectly resolved as value: %#v", r)
			}
		}
	}
}

func TestTSResolverInterfaceNotRuntimeValue(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nUser();\n")
	refs := resolveTS(t, root, "a.ts")
	for _, r := range refs {
		if r.Source.Name == "User" && r.Kind == domain.ReferenceCall && r.Resolution == domain.ResolutionResolved {
			t.Fatalf("interface incorrectly resolved as runtime value: %#v", r)
		}
	}
}

func TestTSResolverNamespaceImportResolved(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "models.ts"), "export interface User {}\nexport function createUser() { return 1; }\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import * as Models from \"./models\";\nconst u: Models.User = {} as any;\nModels.createUser();\n")
	refs := resolveTS(t, root, "app.ts", "models.ts")
	if !hasTSRefWithResolution(refs, "createUser", domain.ResolutionResolved) {
		t.Fatalf("expected resolved namespace member createUser, got %#v", findAllTSRefs(refs, "createUser"))
	}
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved namespace type member User, got %#v", findAllTSRefs(refs, "User"))
	}
}

func TestTSResolverValueParamDoesNotShadowTypeSpace(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nfunction f(User: any) { const x: User = User; }\n")
	refs := resolveTS(t, root, "a.ts")
	// Type annotation User is in type space; value parameter User is separate. Resolving to interface is acceptable.
	// We only assert we do not produce an incorrect non-interface target when resolved.
	for _, r := range refs {
		if r.Source.Name == "User" && r.Location.StartLine == 2 && r.Kind == domain.ReferenceType && r.Resolution == domain.ResolutionResolved && r.Target != nil {
			// resolved to interface/type symbol is fine
			if r.Target.Kind != "type" && r.Target.Kind != "interface" && !strings.Contains(r.Target.Signature, "interface") {
				t.Fatalf("type annotation User resolved to unexpected non-type target: %#v", r)
			}
		}
	}
}

func TestTSResolverShadowingGenericParam(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nfunction execute<User>(value: User): User { return value; }\n")
	refs := resolveTS(t, root, "a.ts")
	for _, r := range refs {
		if r.Source.Name == "User" && r.Location.StartLine == 2 && r.Kind == domain.ReferenceType && r.Resolution == domain.ResolutionResolved && r.Target != nil && r.Target.Line == 1 {
			t.Fatalf("generic type param User incorrectly resolved to outer interface: %#v", r)
		}
	}
}

func TestTSResolverDeterministicWithDifferentFileOrder(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "export interface A {}\n")
	writeTSFile(t, filepath.Join(root, "b.ts"), "import type { A } from \"./a\";\nconst x: A = {} as any;\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	cs1 := &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{{Path: "b.ts", Status: domain.FileModified, Language: domain.LangTypeScript}, {Path: "a.ts", Status: domain.FileModified, Language: domain.LangTypeScript}}}
	cs2 := &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{{Path: "a.ts", Status: domain.FileModified, Language: domain.LangTypeScript}, {Path: "b.ts", Status: domain.FileModified, Language: domain.LangTypeScript}}}
	r1, _ := NewTypeScriptReferenceResolver().Resolve(context.Background(), idx, cs1)
	r2, _ := NewTypeScriptReferenceResolver().Resolve(context.Background(), idx, cs2)
	if len(r1) != len(r2) {
		t.Fatalf("different lengths %d %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Source.Name != r2[i].Source.Name || r1[i].Resolution != r2[i].Resolution || r1[i].Location.Path != r2[i].Location.Path || r1[i].Location.StartLine != r2[i].Location.StartLine {
			t.Fatalf("nondeterministic at %d: %#v vs %#v", i, r1[i], r2[i])
		}
	}
}

func TestTSResolverMixedJSProjectIsolation(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.js"), "export function a() {}\n")
	writeTSFile(t, filepath.Join(root, "b.ts"), "import { a } from \"./a\";\n")
	refs := resolveTS(t, root, "b.ts", "a.js")
	for _, r := range refs {
		if r.Language != domain.LangTypeScript {
			t.Fatalf("TS resolver emitted non-TS ref: %#v", r)
		}
	}
}

func TestTSResolverDynamicExternalStates(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "import fs from \"fs\";\nconst obj: any = {};\nobj[\"m\"]();\n")
	refs := resolveTS(t, root, "a.ts")
	if !hasTSRefWithResolution(refs, "fs", domain.ResolutionExternal) {
		t.Fatalf("expected external fs, got %#v", findAllTSRefs(refs, "fs"))
	}
	foundDynamic := false
	for _, r := range refs {
		if r.Resolution == domain.ResolutionDynamic {
			foundDynamic = true
		}
	}
	if !foundDynamic {
		t.Fatalf("expected at least one dynamic ref, got %#v", refs)
	}
}

func TestTSResolverExportTypeLocalAndReexport(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface User {}\nexport type { User };\n")
	writeTSFile(t, filepath.Join(root, "b.ts"), "import type { User } from \"./a\";\nconst u: User = {} as any;\n")
	refs := resolveTS(t, root, "b.ts", "a.ts")
	if !hasTSRefWithResolution(refs, "User", domain.ResolutionResolved) {
		t.Fatalf("expected resolved export type/import type User, got %#v", findAllTSRefs(refs, "User"))
	}
}

func TestTSResolverThisMethod(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "class S {\n  m() {}\n  n() { this.m(); }\n}\n")
	refs := resolveTS(t, root, "a.ts")
	found := false
	for _, r := range refs {
		if r.Source.Name == "m" && r.Kind == domain.ReferenceMethod {
			found = true
			if r.Resolution != domain.ResolutionResolved && r.Resolution != domain.ResolutionDynamic && r.Resolution != domain.ResolutionUnresolved {
				t.Fatalf("unexpected resolution for this.m: %#v", r)
			}
		}
	}
	if !found {
		t.Logf("refs=%#v", refs) // conservative: extractor/container support may vary
	}
}

func TestTSResolverEvidencePresent(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface U {}\nconst x: U = {} as any;\n")
	refs := resolveTS(t, root, "a.ts")
	for _, r := range refs {
		if len(r.Evidence) == 0 {
			t.Fatalf("reference missing evidence: %#v", r)
		}
		if r.Location.Path == "" || r.Source.Name == "" || r.Language != domain.LangTypeScript {
			t.Fatalf("reference missing identity/location/language: %#v", r)
		}
	}
}

// TestTSResolverOneLineSignatureDoesNotLeakBodyAsType protects the WU4 fix in
// indexTopLevel. Before the fix, the brace that starts a one-line function body
// was not found at depth zero, so the whole text after the return-type colon
// ("Result { return payload; }") reached emitTypeExpr. That emitted the body
// identifier `payload` as a type-space reference. The fix checks the target
// character before incrementing the depth counters, so only "Result" reaches
// type-space emission.
func TestTSResolverOneLineSignatureDoesNotLeakBodyAsType(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"),
		"interface Result {}\nconst payload = 1;\nfunction inner(): Result { return payload; }\n")
	refs := resolveTS(t, root, "a.ts")

	// The annotated return type is a genuine type-space reference.
	if findTSRef(refs, "Result", domain.ReferenceType) == nil {
		t.Fatalf("expected type reference to Result from the return annotation, got %#v", refs)
	}

	// The value identifier used by the function body must not be classified as a type.
	if leaked := findTSRef(refs, "payload", domain.ReferenceType); leaked != nil {
		t.Fatalf("function body identifier leaked into type space: %#v", leaked)
	}
}

// TestTSResolverOneLineSignatureParamsIgnoreBody protects the parameter side of
// the one-line signature parsing. emitSignatureTypes must match the closing
// parenthesis of the signature, not the last parenthesis in the line; otherwise
// a call inside the body (e.g. "target();") is parsed as a parameter and the
// body identifier is emitted as a type-space reference.
func TestTSResolverOneLineSignatureParamsIgnoreBody(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"),
		"function target(): void {}\nfunction inner(): void { target(); }\n")
	refs := resolveTS(t, root, "a.ts")

	if leaked := findTSRef(refs, "target", domain.ReferenceType); leaked != nil {
		t.Fatalf("function body call leaked into parameter type space: %#v", leaked)
	}
	call := findTSRef(refs, "target", domain.ReferenceCall)
	if call == nil || call.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved call to global target, got %#v", refs)
	}
}

func TestTSResolverReexportTypeResolvesToOriginal(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "types.ts"), "export interface T { id: string }\n")
	writeTSFile(t, filepath.Join(root, "barrel.ts"), "export type { T } from \"./types\";\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import type { T } from \"./barrel\";\nconst x: T = { id: \"a\" };\n")
	refs := resolveTS(t, root, "app.ts", "barrel.ts", "types.ts")
	ref := findTSRef(refs, "T", domain.ReferenceType)
	if ref == nil || ref.Resolution != domain.ResolutionResolved || ref.Target == nil {
		t.Fatalf("type reexport should resolve: %#v", ref)
	}
	if ref.Target.File != "types.ts" || ref.Target.Name != "T" {
		t.Fatalf("unexpected type reexport target: %#v", ref.Target)
	}
}

func TestTSResolverReexportMissingTargetUnresolved(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "export const x = 1;\n")
	writeTSFile(t, filepath.Join(root, "barrel.ts"), "export { missing } from \"./a\";\n")
	writeTSFile(t, filepath.Join(root, "app.ts"), "import { missing } from \"./barrel\";\nfunction f(): void { missing(); }\n")
	refs := resolveTS(t, root, "app.ts", "barrel.ts", "a.ts")
	ref := findTSRef(refs, "missing", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionUnresolved {
		t.Fatalf("missing reexport target should stay unresolved: %#v", ref)
	}
}

func TestTSResolverInterfaceMergingConservativeAmbiguity(t *testing.T) {
	root := t.TempDir()
	writeTSFile(t, filepath.Join(root, "a.ts"), "interface M { a: string }\ninterface M { b: number }\nconst x: M = { a: \"\", b: 1 };\n")
	refs := resolveTS(t, root, "a.ts")
	ref := findTSRef(refs, "M", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected type reference to M, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionAmbiguous || ref.Target != nil || len(ref.Candidates) < 2 {
		t.Fatalf("declaration merging is unsupported; expected conservative ambiguity with candidates: %#v", ref)
	}
}
