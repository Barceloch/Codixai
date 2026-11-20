package index

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func pyChangeSet(root string, paths ...string) *domain.ChangeSet {
	files := make([]domain.FileChange, 0, len(paths))
	for _, path := range paths {
		files = append(files, domain.FileChange{Path: path, Status: domain.FileModified, Language: domain.LangPython})
	}
	return &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: files}
}

func TestPythonReferenceResolverSameFileCall(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `def caller():
    return callee()

def callee():
    return 1
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "callee", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to callee, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved", ref.Resolution)
	}
	if ref.Target == nil || ref.Target.Name != "callee" || ref.Target.File != "a.py" {
		t.Fatalf("target = %#v, want same-file callee", ref.Target)
	}
}

func TestPythonReferenceResolverAbsoluteFromImport(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "app", "services.py"), `from app.models import User

def make():
    return User()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "app/services.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "User", domain.ReferenceCall)
	if call == nil {
		t.Fatalf("expected call to User, got %#v", refs)
	}
	if call.Resolution != domain.ResolutionResolved {
		t.Fatalf("call resolution = %q, want resolved; target=%#v", call.Resolution, call.Target)
	}
	if call.Target == nil || call.Target.Kind != "type" || call.Target.File != "app/models.py" {
		t.Fatalf("call target = %#v, want app/models.py type", call.Target)
	}
	imp := findRef(refs, "app.models.User", domain.ReferenceImport)
	if imp == nil {
		t.Fatalf("expected import ref, got %#v", refs)
	}
	if imp.Resolution != domain.ResolutionResolved {
		t.Fatalf("import resolution = %q, want resolved", imp.Resolution)
	}
}

func TestPythonReferenceResolverRelativeImport(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "app", "services", "x.py"), `from ..models import User

def make():
    return User()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "app/services/x.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "User", domain.ReferenceCall)
	if call == nil || call.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved relative import call, got %#v", refs)
	}
	if call.Target == nil || call.Target.File != "app/models.py" {
		t.Fatalf("target = %#v, want app/models.py", call.Target)
	}
}

func TestPythonReferenceResolverImportAlias(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "app", "services.py"), `from app.models import User as U

def make():
    return U()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "app/services.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "U", domain.ReferenceCall)
	if call == nil || call.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved alias call, got %#v", refs)
	}
	if call.Target == nil || call.Target.Name != "User" {
		t.Fatalf("target = %#v, want User", call.Target)
	}
}

func TestPythonReferenceResolverStdlibExternal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `import os

def f():
    return os.path.join("a", "b")
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "join", domain.ReferenceCall)
	if call == nil {
		t.Fatalf("expected call to join, got %#v", refs)
	}
	if call.Resolution != domain.ResolutionExternal {
		t.Fatalf("resolution = %q, want external", call.Resolution)
	}
	imp := findRef(refs, "os", domain.ReferenceImport)
	if imp == nil || imp.Resolution != domain.ResolutionExternal {
		t.Fatalf("expected external import ref, got %#v", refs)
	}
}

func TestPythonReferenceResolverThirdPartyExternal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `import requests

def f():
    return requests.get("https://example.test")
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "get", domain.ReferenceCall)
	if call == nil || call.Resolution != domain.ResolutionExternal {
		t.Fatalf("expected external third-party call, got %#v", refs)
	}
}

func TestPythonReferenceResolverSelfMethod(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class User:
    def save(self):
        pass

    def run(self):
        self.save()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "save", domain.ReferenceMethod)
	if call == nil {
		t.Fatalf("expected method call to save, got %#v", refs)
	}
	if call.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved; target=%#v", call.Resolution, call.Target)
	}
	if call.Target == nil || call.Target.Container != "User" {
		t.Fatalf("target = %#v, want User.save", call.Target)
	}
}

func TestPythonReferenceResolverTypedLocalMethod(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class Repo:
    def find(self):
        pass

def run():
    repo = Repo()
    repo.find()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "find", domain.ReferenceMethod)
	if call == nil || call.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved typed local method, got %#v", refs)
	}
	if call.Target == nil || call.Target.Container != "Repo" {
		t.Fatalf("target = %#v, want Repo.find", call.Target)
	}
}

func TestPythonReferenceResolverUnknownReceiverDynamic(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `def f(obj):
    obj.save()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "save", domain.ReferenceMethod)
	if call == nil {
		t.Fatalf("expected method ref to save, got %#v", refs)
	}
	if call.Resolution == domain.ResolutionResolved {
		t.Fatalf("unknown receiver must not be resolved: %#v", call)
	}
	if call.Resolution != domain.ResolutionDynamic {
		t.Fatalf("resolution = %q, want dynamic", call.Resolution)
	}
}

func TestPythonReferenceResolverInheritance(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class Base:
    pass

class Admin(Base):
    pass
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Base", domain.ReferenceInheritance)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved inheritance base, got %#v", refs)
	}
	if ref.Target == nil || ref.Target.Kind != "type" {
		t.Fatalf("target = %#v, want type", ref.Target)
	}
}

func TestPythonReferenceResolverTypeAnnotation(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class User:
    pass

def f(user: User) -> User:
    return user
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	var resolved int
	for _, ref := range refs {
		if ref.Source.Name == "User" && ref.Kind == domain.ReferenceType && ref.Resolution == domain.ResolutionResolved {
			resolved++
		}
	}
	if resolved < 2 {
		t.Fatalf("expected parameter and return annotations resolved, got %#v", refs)
	}
}

func TestPythonReferenceResolverWildcardDynamic(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "app", "services.py"), `from app.models import *

def make():
    return User()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "app/services.py"))
	if err != nil {
		t.Fatal(err)
	}
	call := findRef(refs, "User", domain.ReferenceCall)
	if call == nil || call.Resolution != domain.ResolutionDynamic {
		t.Fatalf("expected dynamic call through wildcard import, got %#v", refs)
	}
	imp := findRef(refs, "app.models.*", domain.ReferenceImport)
	if imp == nil || imp.Resolution != domain.ResolutionDynamic {
		t.Fatalf("expected dynamic wildcard import, got %#v", refs)
	}
}

func TestPythonReferenceResolverGetattrDynamic(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `def f(obj):
    return getattr(obj, "name")
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "getattr", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionDynamic {
		t.Fatalf("expected dynamic getattr, got %#v", refs)
	}
}

func TestPythonReferenceResolverModuleAttribute(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "config.py"), `DEBUG = True
`)
	writeTestFile(t, filepath.Join(root, "a.py"), `import app.config as cfg

def f():
    return cfg.DEBUG
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "DEBUG", domain.ReferenceField)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved module attribute, got %#v", refs)
	}
	if ref.Target == nil || ref.Target.File != "app/config.py" {
		t.Fatalf("target = %#v, want app/config.py", ref.Target)
	}
}

func TestPythonReferenceResolverSelfField(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class User:
    name = ""

    def run(self):
        return self.name
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "name", domain.ReferenceField)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved self field, got %#v", refs)
	}
	if ref.Target == nil || ref.Target.Container != "User" {
		t.Fatalf("target = %#v, want User.name", ref.Target)
	}
}

func TestPythonReferenceResolverDottedClassReference(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "a.py"), `import app.models

def f():
    return app.models.User
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "User", domain.ReferenceType)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved dotted class reference, got %#v", refs)
	}
	if ref.Target == nil || ref.Target.File != "app/models.py" {
		t.Fatalf("target = %#v, want app/models.py", ref.Target)
	}
}

func TestPythonReferenceResolverDottedInheritance(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "models.py"), `class User:
    pass
`)
	writeTestFile(t, filepath.Join(root, "a.py"), `import app.models

class Admin(app.models.User):
    pass
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "User", domain.ReferenceInheritance)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved dotted inheritance, got %#v", refs)
	}
}

func TestPythonReferenceResolverSelfInstanceField(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `class User:
    def __init__(self, email):
        self.email = email

    def contact(self):
        return self.email
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "email", domain.ReferenceField)
	if ref == nil || ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("expected resolved self instance field, got %#v", refs)
	}
	if ref.Target == nil || ref.Target.Container != "User" {
		t.Fatalf("target = %#v, want User.email", ref.Target)
	}
}

func TestPythonReferenceResolverMissingLocalImportUnresolved(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "app", "__init__.py"), "")
	writeTestFile(t, filepath.Join(root, "a.py"), `from app.missing import X
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	imp := findRef(refs, "app.missing.X", domain.ReferenceImport)
	if imp == nil {
		t.Fatalf("expected import ref, got %#v", refs)
	}
	if imp.Resolution != domain.ResolutionUnresolved {
		t.Fatalf("resolution = %q, want unresolved for a missing local module", imp.Resolution)
	}
}

func TestPythonReferenceResolverDeterministic(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `def caller():
    a()
    b()

def a():
    pass

def b():
    pass
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	cs := pyChangeSet(root, "a.py")
	r1, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, cs)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, cs)
	if err != nil {
		t.Fatal(err)
	}
	if len(r1) != len(r2) {
		t.Fatalf("len %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Source.Name != r2[i].Source.Name || r1[i].Resolution != r2[i].Resolution || r1[i].Location.StartLine != r2[i].Location.StartLine {
			t.Fatalf("nondeterministic at %d: %#v vs %#v", i, r1[i], r2[i])
		}
	}
}

func TestPythonReferenceResolverRelativeMissingModuleNotExternal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pkg", "__init__.py"), "")
	writeTestFile(t, filepath.Join(root, "a.py"), `from .missing import symbol

def f():
    symbol()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "symbol", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to symbol, got %#v", refs)
	}
	if ref.Resolution == domain.ResolutionExternal {
		t.Fatalf("relative import of a missing local module must not be external: %#v", ref)
	}
	if ref.Resolution != domain.ResolutionUnresolved {
		t.Fatalf("resolution = %q, want unresolved (local-missing)", ref.Resolution)
	}
	if len(ref.Evidence) == 0 {
		t.Fatalf("expected evidence for unresolved relative import, got %#v", ref)
	}
}

func TestPythonReferenceResolverComprehensionTargetNotModule(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), `def item():
    pass

def helper():
    pass

def outer():
    _ = [item() for item in items]
    _ = [helper() for item in items]
    item()
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	// The comprehension target shadows the module-level function inside the comprehension.
	target := findRefAtLine(refs, "item", domain.ReferenceCall, 8)
	if target == nil {
		t.Fatalf("expected call to item inside comprehension, got %#v", refs)
	}
	if target.Resolution != domain.ResolutionDynamic {
		t.Fatalf("comprehension target call resolution = %q, want dynamic", target.Resolution)
	}
	if target.Target != nil {
		t.Fatalf("comprehension target call must not resolve to the module function: %#v", target.Target)
	}
	// A module-level function referenced inside the comprehension still resolves.
	helper := findRefAtLine(refs, "helper", domain.ReferenceCall, 9)
	if helper == nil || helper.Resolution != domain.ResolutionResolved || helper.Target == nil || helper.Target.Name != "helper" {
		t.Fatalf("outer function inside comprehension should resolve: %#v", helper)
	}
	// The comprehension target does not leak: the later call resolves to the module function.
	later := findRefAtLine(refs, "item", domain.ReferenceCall, 10)
	if later == nil || later.Resolution != domain.ResolutionResolved || later.Target == nil || later.Target.Name != "item" {
		t.Fatalf("call after comprehension should resolve to module item: %#v", later)
	}
}

func findRefAtLine(refs []domain.Reference, name string, kind domain.ReferenceKind, line int) *domain.Reference {
	for i := range refs {
		if refs[i].Source.Name == name && refs[i].Kind == kind && refs[i].Location.StartLine == line {
			return &refs[i]
		}
	}
	return nil
}

func TestPythonReferenceResolverWildcardImportConservative(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pkg", "__init__.py"), "")
	writeTestFile(t, filepath.Join(root, "pkg", "mod.py"), "def run():\n    pass\n")
	writeTestFile(t, filepath.Join(root, "a.py"), "from pkg.mod import *\n\ndef f():\n    run()\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "run", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to run, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionDynamic {
		t.Fatalf("wildcard-imported call resolution = %q, want dynamic (conservative)", ref.Resolution)
	}
	if ref.Target != nil {
		t.Fatalf("wildcard-imported call must not claim a target: %#v", ref.Target)
	}
}

func TestPythonReferenceResolverRelativeAliasResolves(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pkg", "__init__.py"), "")
	writeTestFile(t, filepath.Join(root, "pkg", "mod.py"), "def run():\n    pass\n")
	writeTestFile(t, filepath.Join(root, "pkg", "app.py"), "from .mod import run as r\n\ndef f():\n    r()\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "pkg/app.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "r", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionResolved || ref.Target == nil {
		t.Fatalf("aliased relative import should resolve: %#v", ref)
	}
	if ref.Target.File != "pkg/mod.py" || ref.Target.Name != "run" {
		t.Fatalf("unexpected target: %#v", ref.Target)
	}
}

func TestPythonReferenceResolverAbsoluteExternalStaysExternal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.py"), "import requests\n\ndef f():\n    requests.get()\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewPythonReferenceResolver().Resolve(context.Background(), idx, pyChangeSet(root, "a.py"))
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "get", domain.ReferenceCall)
	if ref == nil || ref.Resolution != domain.ResolutionExternal {
		t.Fatalf("absolute unknown module call should be external: %#v", ref)
	}
}
