package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

func TestGoReferenceResolverSameFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p

func Caller() { Callee() }
func Callee() {}
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root:   root,
		Origin: domain.OriginWorktree,
		Files:  []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Callee", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to Callee, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved", ref.Resolution)
	}
	if ref.Target == nil || ref.Target.File != "a.go" {
		t.Fatalf("target = %#v, want same-file target", ref.Target)
	}
	if len(ref.Evidence) == 0 {
		t.Fatal("expected evidence")
	}
}

func TestGoReferenceResolverSiblingPackage(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func Caller() { Helper() }
`)
	writeTestFile(t, filepath.Join(root, "b.go"), `package p
func Helper() {}
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Helper", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to Helper, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved", ref.Resolution)
	}
	if ref.Target == nil || filepath.Base(ref.Target.File) != "b.go" {
		t.Fatalf("target = %#v, want b.go", ref.Target)
	}
}

func TestGoReferenceResolverTypeInOtherFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func F(x Thing) {}
`)
	writeTestFile(t, filepath.Join(root, "b.go"), `package p
type Thing struct{}
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Thing", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected type ref to Thing, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved", ref.Resolution)
	}
	if ref.Target == nil || ref.Target.Kind != "type" {
		t.Fatalf("target = %#v, want type target", ref.Target)
	}
}

func TestGoReferenceResolverBuiltin(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func F() int { return len("x") }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "len", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to len, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionExternal {
		t.Fatalf("resolution = %q, want external for builtin", ref.Resolution)
	}
}

func TestGoReferenceResolverNonexistent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func F() { Missing() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Missing", domain.ReferenceCall)
	if ref == nil {
		t.Fatalf("expected call to Missing, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionUnresolved {
		t.Fatalf("resolution = %q, want unresolved", ref.Resolution)
	}
}

func TestGoReferenceResolverUnknownReceiverMethod(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func F(x interface{ Do() }) { x.Do() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Do", domain.ReferenceMethod)
	if ref == nil {
		// Conservative: method calls on unresolved receivers may be emitted as method refs.
		for i := range refs {
			if refs[i].Source.Name == "Do" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected method ref to Do, got %#v", refs)
	}
	if ref.Resolution == domain.ResolutionResolved {
		t.Fatalf("unknown receiver method should not be confidently resolved: %#v", ref)
	}
}

func TestGoReferenceResolverDuplicateNamesAmbiguous(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "p2"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "p1", "a.go"), `package p1
type Thing struct{}
`)
	writeTestFile(t, filepath.Join(root, "p2", "a.go"), `package p2
type Thing struct{}
`)
	writeTestFile(t, filepath.Join(root, "main.go"), `package main
func F(x Thing) {}
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "main.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Thing", domain.ReferenceType)
	if ref == nil {
		t.Fatalf("expected type ref to Thing, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionAmbiguous {
		t.Fatalf("resolution = %q, want ambiguous; candidates=%#v target=%#v", ref.Resolution, ref.Candidates, ref.Target)
	}
	if len(ref.Candidates) < 2 {
		t.Fatalf("expected >=2 candidates, got %#v", ref.Candidates)
	}
}

func TestGoReferenceResolverDeterministic(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
func Caller() { A(); B() }
func A() {}
func B() {}
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	cs := &domain.ChangeSet{Root: root, Origin: domain.OriginWorktree, Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}}}
	r1, err := NewGoReferenceResolver().Resolve(context.Background(), idx, cs)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := NewGoReferenceResolver().Resolve(context.Background(), idx, cs)
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

func TestGoReferenceResolverStdlibImportSelector(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
import "fmt"
func F() { fmt.Println("x") }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Println", domain.ReferenceCall)
	if ref == nil {
		for i := range refs {
			if refs[i].Source.Name == "Println" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected ref to Println, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionExternal {
		t.Fatalf("resolution = %q, want external for stdlib", ref.Resolution)
	}
}

func TestGoReferenceResolverImportAlias(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
import f "fmt"
func F() { f.Println("x") }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Println", domain.ReferenceCall)
	if ref == nil {
		for i := range refs {
			if refs[i].Source.Name == "Println" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected ref to Println via alias, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionExternal {
		t.Fatalf("resolution = %q, want external", ref.Resolution)
	}
}

func TestGoReferenceResolverThirdPartyExternal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
import "github.com/example/pkg"
func F() { pkg.Do() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Do", domain.ReferenceCall)
	if ref == nil {
		for i := range refs {
			if refs[i].Source.Name == "Do" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected ref to Do, got %#v", refs)
	}
	// Conservative: unknown third-party is external (not falsely resolved to local symbol).
	if ref.Resolution != domain.ResolutionExternal {
		t.Fatalf("resolution = %q, want external for third-party", ref.Resolution)
	}
}

func TestGoReferenceResolverKnownReceiverParam(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
type User struct{}
func (u User) Save() {}
func F(u User) { u.Save() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Find the method call Save() inside F (not the method declaration context issues; source is call).
	var ref *domain.Reference
	for i := range refs {
		if refs[i].Source.Name == "Save" && refs[i].Kind == domain.ReferenceMethod && refs[i].Location.StartLine == 4 {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		t.Fatalf("expected method call to Save at line 4, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved for known receiver; target=%#v", ref.Resolution, ref.Target)
	}
	if ref.Target == nil || ref.Target.Container != "User" {
		t.Fatalf("target = %#v, want User.Save method", ref.Target)
	}
}

func TestGoReferenceResolverKnownReceiverShortDecl(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), `package p
type Svc struct{}
func (s Svc) Process() {}
func F() { s := Svc{}; s.Process() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ref *domain.Reference
	for i := range refs {
		if refs[i].Source.Name == "Process" && refs[i].Kind == domain.ReferenceMethod {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		t.Fatalf("expected method call to Process, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved", ref.Resolution)
	}
}

func TestGoReferenceResolverLocalImport(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "mypkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "mypkg", "util.go"), `package mypkg
func Help() {}
`)
	writeTestFile(t, filepath.Join(root, "main.go"), `package main
import "example.com/mypkg"
func F() { mypkg.Help() }
`)
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "main.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := findRef(refs, "Help", domain.ReferenceCall)
	if ref == nil {
		for i := range refs {
			if refs[i].Source.Name == "Help" {
				ref = &refs[i]
				break
			}
		}
	}
	if ref == nil {
		t.Fatalf("expected ref to Help, got %#v", refs)
	}
	if ref.Resolution != domain.ResolutionResolved {
		t.Fatalf("resolution = %q, want resolved for local import; target=%#v candidates=%#v", ref.Resolution, ref.Target, ref.Candidates)
	}
}

func findRef(refs []domain.Reference, name string, kind domain.ReferenceKind) *domain.Reference {
	for i := range refs {
		if refs[i].Source.Name == name && (kind == "" || refs[i].Kind == kind) {
			return &refs[i]
		}
	}
	return nil
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGoReferenceResolverLocalShadowing(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), "package p\n\nfunc Helper() {}\n\nfunc F() {\n\tHelper := func() {}\n\tHelper()\n}\n")
	idx := New()
	if err := idx.Build(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	refs, err := NewGoReferenceResolver().Resolve(context.Background(), idx, &domain.ChangeSet{
		Root: root, Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{Path: "a.go", Status: domain.FileModified, Language: domain.LangGo}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range refs {
		if r.Source.Name == "Helper" && r.Kind == domain.ReferenceCall && r.Location.StartLine == 7 {
			if r.Resolution == domain.ResolutionResolved && r.Target != nil && r.Target.Line == 3 {
				t.Fatalf("local shadowed Helper incorrectly resolved to package function: %#v", r)
			}
		}
	}
}
