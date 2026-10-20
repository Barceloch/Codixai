package golang_test

import (
	"os"
	"path/filepath"
	"testing"

	golang "codixai/internal/languages/go"
)

// TestParseGoMod checks the reader of the go.mod file.
func TestParseGoMod(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "go.mod")
	content := `module example.com/app

go 1.27.0

require github.com/spf13/cobra v1.8.0

require (
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/sync v0.7.0
)

replace example.com/old => example.com/new v1.0.0
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("the manifest cannot be written: %v", err)
	}
	manifest := golang.ReadManifest(directory)
	if !manifest.Found {
		t.Fatal("the manifest was not found")
	}
	if manifest.Module != "example.com/app" {
		t.Errorf("the module is %q, want %q", manifest.Module, "example.com/app")
	}
	if !manifest.Declares("github.com/spf13/cobra") {
		t.Error("the manifest does not declare github.com/spf13/cobra")
	}
	if !manifest.Declares("github.com/spf13/cobra/cmd") {
		t.Error("the manifest does not declare the sub package of a required module")
	}
	if manifest.Declares("github.com/acme/missing") {
		t.Error("the manifest declares a module that it does not require")
	}
	if !manifest.IsLocal("example.com/app/internal/payment") {
		t.Error("the manifest does not recognize a local package")
	}
	if manifest.IsLocal("example.com/other/pkg") {
		t.Error("the manifest recognizes a foreign package as local")
	}
}

// TestReadManifestWithoutFile checks the reader when there is no manifest.
func TestReadManifestWithoutFile(t *testing.T) {
	manifest := golang.ReadManifest(t.TempDir())
	if manifest.Found {
		t.Error("a manifest was found in an empty directory")
	}
	if len(manifest.Requires) != 0 {
		t.Errorf("the manifest requires %v, want nothing", manifest.Requires)
	}
}

// TestIsThirdPartyShape checks the shape of the import paths.
func TestIsThirdPartyShape(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"github.com/acme/missing", true},
		{"golang.org/x/sync", true},
		{"fmt", false},
		{"net/http", false},
		{"example.com/app/internal/payment", true},
	}
	for _, testCase := range cases {
		if got := golang.IsThirdPartyShape(testCase.path); got != testCase.want {
			t.Errorf("IsThirdPartyShape(%q) is %v, want %v", testCase.path, got, testCase.want)
		}
	}
}

// TestParseExtractsImportsAndComments checks the extraction with go/parser.
func TestParseExtractsImportsAndComments(t *testing.T) {
	source := `package main

import (
	"fmt"

	"github.com/acme/missing/pkg"
)

//nolint:errcheck
func main() {
	fmt.Println(pkg.New())
}
`
	file, err := golang.Parse("main.go", []byte(source))
	if err != nil {
		t.Fatalf("the file cannot be parsed: %v", err)
	}
	if file.Package != "main" {
		t.Errorf("the package is %q, want %q", file.Package, "main")
	}
	if len(file.Imports) != 2 {
		t.Fatalf("the file declares %d imports, want 2", len(file.Imports))
	}
	if file.Imports[1].Path != "github.com/acme/missing/pkg" {
		t.Errorf("the second import is %q", file.Imports[1].Path)
	}
	found := false
	for _, comment := range file.Comments {
		if comment.Text == "//nolint:errcheck" {
			found = true
		}
	}
	if !found {
		t.Error("the parser did not find the nolint directive")
	}
}

// TestFindUndefinedReferences checks the resolution of the local symbols.
func TestFindUndefinedReferences(t *testing.T) {
	source := `package main

func main() {
	missingHelper()
}
`
	references, complete, err := golang.FindUndefinedReferences("main.go", []byte(source), nil)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if !complete {
		t.Fatal("the analysis is not complete")
	}
	if len(references) != 1 || references[0].Name != "missingHelper" {
		t.Fatalf("the references are %+v, want missingHelper", references)
	}
	if references[0].Line != 4 {
		t.Errorf("the reference is in the line %d, want 4", references[0].Line)
	}
}

// TestFindUndefinedReferencesIgnoresDeclaredSymbols checks that the symbols
// that the package declares are not reported.
func TestFindUndefinedReferencesIgnoresDeclaredSymbols(t *testing.T) {
	source := `package main

func helper() {
}

func main() {
	helper()
	local := 1
	_ = local
}
`
	siblings := []golang.SiblingFile{{Name: "other.go", Src: []byte("package main\n\nfunc other() {\n}\n")}}
	references, complete, err := golang.FindUndefinedReferences("main.go", []byte(source), siblings)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if !complete {
		t.Fatal("the analysis is not complete")
	}
	if len(references) != 0 {
		t.Errorf("the references are %+v, want none", references)
	}
}
