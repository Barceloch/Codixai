package application

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// forbiddenImports holds the packages that the application layer must never
// import.
//
// The specification requires, from the first phase, that the application
// layer depends on neither the command line nor the user interface, and
// that an architecture test verifies it. The user interface is a Wails
// adapter that lives behind a build tag, and the command line is the
// presentation of the results.
var forbiddenImports = []string{
	"github.com/barceloch/codixai/internal/cli",
	"github.com/barceloch/codixai/internal/ui",
	"github.com/barceloch/codixai/cmd",
	"github.com/wailsapp/wails",
}

// TestApplicationDoesNotDependOnThePresentation checks that the application
// layer does not import the command line, the user interface or a desktop
// toolkit.
//
// The direction of the dependencies is `ui -> application -> domain`. A
// presentation package imported here would make the use cases depend on a
// specific interface, and the future user interface would have to work
// around it.
func TestApplicationDoesNotDependOnThePresentation(t *testing.T) {
	files, err := packageFiles(".")
	if err != nil {
		t.Fatalf("the package cannot be read: %v", err)
	}
	for _, file := range files {
		for _, imported := range importsOf(t, file) {
			for _, forbidden := range forbiddenImports {
				if imported == forbidden || strings.HasPrefix(imported, forbidden+"/") {
					t.Errorf("%s imports %s: the application layer must not depend on the presentation", file, imported)
				}
			}
		}
	}
}

// TestDomainImportsNothingOfTheProject checks that the domain models do
// not depend on any package of the project.
//
// The domain is the stable contract of the tool. A dependency on the
// application, on the providers or on a framework would make the contract
// depend on the implementation.
func TestDomainImportsNothingOfTheProject(t *testing.T) {
	files, err := packageFiles("../domain")
	if err != nil {
		t.Fatalf("the package cannot be read: %v", err)
	}
	for _, file := range files {
		for _, imported := range importsOf(t, file) {
			if strings.HasPrefix(imported, "github.com/barceloch/codixai/") {
				t.Errorf("%s imports %s: the domain must not import any package of the project", file, imported)
			}
		}
	}
}

// packageFiles returns the Go files of a directory, without the test files.
func packageFiles(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(directory, name))
	}
	return files, nil
}

// importsOf returns the import paths of a Go file.
func importsOf(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s cannot be parsed: %v", path, err)
	}
	var imports []string
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("%s holds an import that cannot be read: %v", path, err)
		}
		imports = append(imports, value)
	}
	return imports
}
