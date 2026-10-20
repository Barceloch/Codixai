package javascript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Barceloch/Codixai/internal/languages/javascript"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
)

// TestReadManifest checks the reader of the package.json file.
func TestReadManifest(t *testing.T) {
	directory := t.TempDir()
	content := `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "vitest": "^1.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("the manifest cannot be written: %v", err)
	}
	manifest := javascript.ReadManifest(directory)
	if !manifest.Found {
		t.Fatal("the manifest was not found")
	}
	if !manifest.Declares("lodash") {
		t.Error("the manifest does not declare lodash")
	}
	if !manifest.Declares("vitest") {
		t.Error("the manifest does not declare the development dependencies")
	}
	if manifest.Declares("axios") {
		t.Error("the manifest declares a package that it does not require")
	}
}

// TestPackageOf checks the mapping between a specifier and a package.
func TestPackageOf(t *testing.T) {
	cases := map[string]string{
		"lodash":         "lodash",
		"lodash/fp":      "lodash",
		"@scope/pkg":     "@scope/pkg",
		"@scope/pkg/sub": "@scope/pkg",
	}
	for specifier, want := range cases {
		if got := javascript.PackageOf(specifier); got != want {
			t.Errorf("PackageOf(%q) is %q, want %q", specifier, got, want)
		}
	}
}

// TestIsBuiltin checks the recognition of the modules of the runtime.
func TestIsBuiltin(t *testing.T) {
	if !javascript.IsBuiltin("fs") {
		t.Error("fs is not recognized as a module of the runtime")
	}
	if !javascript.IsBuiltin("node:fs") {
		t.Error("node:fs is not recognized as a module of the runtime")
	}
	if javascript.IsBuiltin("lodash") {
		t.Error("lodash is recognized as a module of the runtime")
	}
}

// TestExtractImports checks the extraction of the module specifiers.
func TestExtractImports(t *testing.T) {
	profile := javascript.Profile()
	scanner := lexical.NewScanner(profile)
	texts := []string{
		"import axios from \"axios\";",
		"import \"./side-effect.js\";",
		"const lodash = require(\"lodash\");",
		"const lazy = import(\"lodash/fp\");",
		"// import commented from \"commented\";",
	}
	var lines []lexical.Line
	segments := map[int][]lexical.Segment{}
	for i, text := range texts {
		number := i + 1
		lines = append(lines, lexical.Line{Number: number, Text: text})
		segments[number] = scanner.Scan(text)
	}
	imports := javascript.ExtractImports(lines, segments)
	if len(imports) != 4 {
		t.Fatalf("the extractor found %d imports, want 4: %+v", len(imports), imports)
	}
	if imports[0].Module != "axios" {
		t.Errorf("the first import is %+v", imports[0])
	}
	if !imports[1].Relative {
		t.Errorf("the second import is %+v, want a relative import", imports[1])
	}
	if imports[2].Module != "lodash" {
		t.Errorf("the third import is %+v", imports[2])
	}
	if imports[3].Module != "lodash/fp" {
		t.Errorf("the fourth import is %+v", imports[3])
	}
}

// TestEmptyHandlers checks the detection of the empty catch blocks.
func TestEmptyHandlers(t *testing.T) {
	profile := javascript.Profile()
	scanner := lexical.NewScanner(profile)
	texts := []string{
		"try {",
		"  doWork();",
		"} catch (error) {}",
		"return true;",
	}
	var lines []lexical.Line
	segments := map[int][]lexical.Segment{}
	for i, text := range texts {
		number := i + 1
		lines = append(lines, lexical.Line{Number: number, Text: text})
		segments[number] = scanner.Scan(text)
	}
	found := javascript.EmptyHandlers(lines, segments)
	if len(found) != 1 || found[0] != 3 {
		t.Errorf("the empty handlers are %v, want [3]", found)
	}
}
