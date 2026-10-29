package golang_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/languages/lexical"
	golang "github.com/barceloch/codixai/internal/languages/go"
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
	references, complete, err := golang.FindUndefinedReferences("main.go", []byte(source), nil, nil)
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
	references, complete, err := golang.FindUndefinedReferences("main.go", []byte(source), siblings, nil)
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

// TestExtractImportsIgnoresErrorMessages checks that error messages
// containing quoted paths are not mistaken for import declarations.
func TestExtractImportsIgnoresErrorMessages(t *testing.T) {
	lines := []lexical.Line{
		{Number: 1, Text: `return nil, fmt.Errorf("the visualization needs the root of the repository: write it in the repository field")`},
		{Number: 2, Text: `return nil, fmt.Errorf("failed to build index: %w", err)`},
	}
	imports := golang.ExtractImportsRaw(lines)
	if len(imports) > 0 {
		t.Errorf("expected no imports from error messages, got: %v", imports)
	}
}

// TestExtractImportsRawWithContext checks that the raw extraction correctly
// identifies imports in multiline blocks when context lines are present.
func TestExtractImportsRawWithContext(t *testing.T) {
	// Test case 1: error message added (should produce no imports)
	lines1 := []lexical.Line{
		{Number: 1, Text: `return nil, fmt.Errorf("the visualization needs the root of the repository: write it in the repository field")`},
	}
	imports1 := golang.ExtractImportsRaw(lines1)
	if len(imports1) > 0 {
		t.Errorf("expected no imports from error message, got: %v", imports1)
	}

	// Test case 2: multiline import block with added line
	lines2 := []lexical.Line{
		{Number: 8, Text: `import (`},
		{Number: 9, Text: `	"fmt"`},
		{Number: 10, Text: `	"github.com/new/pkg"`},
		{Number: 11, Text: `)`},
	}
	imports2 := golang.ExtractImportsRaw(lines2)
	if len(imports2) != 2 {
		t.Errorf("expected two imports from multiline block, got: %v", imports2)
	}
	// The first import should be on line 9, second on line 10
	found := map[string]bool{}
	for _, imp := range imports2 {
		found[imp.Module] = true
	}
	if !found["fmt"] || !found["github.com/new/pkg"] {
		t.Errorf("expected imports fmt and github.com/new/pkg, got: %v", imports2)
	}
}

// TestExtractImportsRawFalsePositives checks that various false positive
// scenarios are handled correctly.
func TestExtractImportsRawFalsePositives(t *testing.T) {
	// Test case 1: Comment with import-like text
	lines1 := []lexical.Line{
		{Number: 1, Text: `// import "github.com/fake/pkg"`},
	}
	imports1 := golang.ExtractImportsRaw(lines1)
	if len(imports1) > 0 {
		t.Errorf("expected no imports from comment, got: %v", imports1)
	}

	// Test case 2: String with import-like text (not in import block)
	lines2 := []lexical.Line{
		{Number: 1, Text: `fmt.Println("import \"github.com/fake/pkg\"")`},
	}
	imports2 := golang.ExtractImportsRaw(lines2)
	if len(imports2) > 0 {
		t.Errorf("expected no imports from string, got: %v", imports2)
	}

	// Test case 3: Multiline string with import-like text
	lines3 := []lexical.Line{
		{Number: 1, Text: `const msg = ` + "`" + `import "github.com/fake/pkg"` + "`"},
	}
	imports3 := golang.ExtractImportsRaw(lines3)
	if len(imports3) > 0 {
		t.Errorf("expected no imports from multiline string, got: %v", imports3)
	}

	// Test case 4: Single-line import (not in block)
	lines4 := []lexical.Line{
		{Number: 1, Text: `import "fmt"`},
	}
	imports4 := golang.ExtractImportsRaw(lines4)
	if len(imports4) != 1 || imports4[0].Module != "fmt" {
		t.Errorf("expected import fmt, got: %v", imports4)
	}

	// Test case 5: Import with alias
	lines5 := []lexical.Line{
		{Number: 1, Text: `import myfmt "fmt"`},
	}
	imports5 := golang.ExtractImportsRaw(lines5)
	if len(imports5) != 1 || imports5[0].Module != "fmt" {
		t.Errorf("expected import fmt with alias, got: %v", imports5)
	}

	// Test case 6: Import block with only context (no added lines)
	lines6 := []lexical.Line{
		{Number: 1, Text: `import (`},
		{Number: 2, Text: `	"fmt"`},
		{Number: 3, Text: `)`},
	}
	imports6 := golang.ExtractImportsRaw(lines6)
	if len(imports6) != 1 {
		t.Errorf("expected one import from context block, got: %v", imports6)
	}
}

// TestExtractImportsRawUIErrorMessages tests the original false positive
// from ui/ui.go where error messages containing quoted paths were
// incorrectly flagged as imports.
func TestExtractImportsRawUIErrorMessages(t *testing.T) {
	// These are the exact error messages from ui/ui.go that were causing false positives
	lines := []lexical.Line{
		{Number: 612, Text: `return nil, fmt.Errorf("the visualization needs the root of the repository: write it in the repository field")`},
		{Number: 613, Text: `return nil, fmt.Errorf("failed to build index: %w", err)`},
	}
	imports := golang.ExtractImportsRaw(lines)
	if len(imports) > 0 {
		t.Errorf("expected no imports from ui.go error messages, got: %v", imports)
	}
}

// TestFindUndefinedReferencesHandlesControlFlow checks that the analyzer
// does not crash on the control flow statements whose label is optional.
//
// A break or a continue without a label has no identifier, and a function
// declaration without a body has no block. Both are valid Go, and both
// used to reach a nil identifier.
func TestFindUndefinedReferencesHandlesControlFlow(t *testing.T) {
	source := `package main

func declared() {
}

func withoutBody()

func main() {
	for i := 0; i < 3; i++ {
		if i == 1 {
			break
		}
		continue
	}
outer:
	for i := 0; i < 3; i++ {
		if i == 1 {
			break outer
		}
	}
	declared()
}
`
	references, complete, err := golang.FindUndefinedReferences("main.go", []byte(source), nil, nil)
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
