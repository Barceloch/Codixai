package golang_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	golang "github.com/barceloch/codixai/internal/languages/go"
)

// TestFindUndefinedReferencesResolvesPackageQualifiers covers the resolution
// of the qualifier of a selector expression.
//
// A qualifier is the name that an import binds, and it is never a symbol that
// the package directory has to declare. The name of an external package is
// taken from the last element of its import path, except when that element is
// a major version suffix, which the semantic import versioning of Go appends
// after the name of the module.
func TestFindUndefinedReferencesResolvesPackageQualifiers(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		packageNames map[string]string
		want         []string
	}{
		{
			name: "external package with a major version suffix",
			source: `package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func main() {
	client := resty.New()
	fmt.Println(client)
}
`,
			want: nil,
		},
		{
			name: "explicit import alias",
			source: `package main

import (
	"fmt"

	r "github.com/go-resty/resty/v2"
)

func main() {
	fmt.Println(r.New())
}
`,
			want: nil,
		},
		{
			name: "standard library package",
			source: `package main

import (
	"net/http"
)

func fetch() {
	http.Get("http://example.invalid")
}
`,
			want: nil,
		},
		{
			name: "external package without a suffix",
			source: `package main

import (
	"example.com/dependency"
)

func run() {
	dependency.Do()
}
`,
			want: nil,
		},
		{
			name: "unused import is not a reference",
			source: `package main

import (
	_ "github.com/go-resty/resty/v2"
)

func run() {
}
`,
			want: nil,
		},
		{
			name: "package of the module whose name differs from its directory",
			source: `package main

import (
	"example.com/app/util"
)

func run() {
	helpers.Do()
}
`,
			packageNames: map[string]string{"example.com/app/util": "helpers"},
			want:         nil,
		},
		{
			name: "symbol that the package does not declare",
			source: `package main

import (
	"fmt"
)

func main() {
	fmt.Println(NonExistentFunction())
}
`,
			want: []string{"NonExistentFunction"},
		},
		{
			name: "qualifier that no import binds",
			source: `package main

import (
	"fmt"
)

func main() {
	fmt.Println(unknown.Do())
}
`,
			want: []string{"unknown"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			references, complete, err := golang.FindUndefinedReferences(
				"main.go", []byte(test.source), nil, test.packageNames)
			if err != nil {
				t.Fatalf("the analysis returned an error: %v", err)
			}
			if !complete {
				t.Fatal("the analysis is not complete")
			}
			got := make([]string, 0, len(references))
			for _, reference := range references {
				got = append(got, reference.Name)
			}
			sort.Strings(got)
			want := append([]string(nil), test.want...)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("the references are %v, want %v", got, want)
			}
			for index := range want {
				if got[index] != want[index] {
					t.Fatalf("the references are %v, want %v", got, want)
				}
			}
		})
	}
}

// TestImportPackageNamesReadsThePackageClauseOfTheModule checks that an import
// of the module of the repository is resolved with the package clause of its
// directory, so a package whose name differs from the directory name is not
// mistaken for a missing symbol.
func TestImportPackageNamesReadsThePackageClauseOfTheModule(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("cannot write the go.mod: %v", err)
	}
	util := filepath.Join(root, "util")
	if err := os.MkdirAll(util, 0o755); err != nil {
		t.Fatalf("cannot create the directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(util, "helpers.go"), []byte("package helpers\n\nfunc Do() {}\n"), 0o644); err != nil {
		t.Fatalf("cannot write the file: %v", err)
	}
	source := []byte(`package main

import (
	"example.com/app/util"
)

func run() {
	helpers.Do()
}
`)
	names, err := golang.ImportPackageNames(root, "example.com/app", source)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if got := names["example.com/app/util"]; got != "helpers" {
		t.Fatalf("the package name is %q, want %q", got, "helpers")
	}
	references, complete, err := golang.FindUndefinedReferences("main.go", source, nil, names)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	if !complete {
		t.Fatal("the analysis is not complete")
	}
	if len(references) != 0 {
		t.Fatalf("the references are %+v, want none", references)
	}
}

// TestIsStdlibCoversTheWholeStandardLibrary checks that the packages of the
// standard library that the analysis may meet are recognized.
//
// A package of the standard library that is missing from the list is reported
// as an undeclared dependency, which is a false positive that blocks code that
// is correct.
func TestIsStdlibCoversTheWholeStandardLibrary(t *testing.T) {
	packages := []string{
		"fmt",
		"net/http",
		"net/http/httputil",
		"encoding/json",
		"os/exec",
		"text/template",
		"go/ast",
		"sync/atomic",
	}
	for _, path := range packages {
		if !golang.IsStdlib(path) {
			t.Errorf("the package %q is not recognized as a package of the standard library", path)
		}
	}
	if golang.IsStdlib("github.com/go-resty/resty/v2") {
		t.Error("an external module is recognized as a package of the standard library")
	}
}
