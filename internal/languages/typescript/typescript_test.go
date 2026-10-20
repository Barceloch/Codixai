package typescript_test

import (
	"testing"

	"codixai/internal/domain"
	"codixai/internal/languages/lexical"
	"codixai/internal/languages/typescript"
)

// TestProfile checks the TypeScript profile.
func TestProfile(t *testing.T) {
	profile := typescript.Profile()
	if profile.ID != domain.LangTypeScript {
		t.Errorf("the identifier of the profile is %q, want %q", profile.ID, domain.LangTypeScript)
	}
	if len(profile.Suppressions) == 0 {
		t.Error("the profile has no suppression marker")
	}
	found := false
	for _, marker := range profile.Suppressions {
		if marker.Pattern.MatchString("// @ts-expect-error the API is not typed") {
			found = true
		}
	}
	if !found {
		t.Error("the profile does not recognize the TypeScript compiler directives")
	}
	if !profile.IsTestPath("src/service.test.ts") {
		t.Error("the profile does not recognize a TypeScript test file")
	}
	if profile.IsTestPath("src/service.ts") {
		t.Error("the profile recognizes a source file as a test file")
	}
}

// TestExtractImports checks the extraction of the module specifiers.
func TestExtractImports(t *testing.T) {
	profile := typescript.Profile()
	scanner := lexical.NewScanner(profile)
	texts := []string{
		"import axios from \"axios\";",
		"import { helper } from \"./helper\";",
	}
	var lines []lexical.Line
	segments := map[int][]lexical.Segment{}
	for i, text := range texts {
		number := i + 1
		lines = append(lines, lexical.Line{Number: number, Text: text})
		segments[number] = scanner.Scan(text)
	}
	imports := typescript.ExtractImports(lines, segments)
	if len(imports) != 2 {
		t.Fatalf("the extractor found %d imports, want 2: %+v", len(imports), imports)
	}
	if imports[0].Module != "axios" {
		t.Errorf("the first import is %+v", imports[0])
	}
	if !imports[1].Relative {
		t.Errorf("the second import is %+v, want a relative import", imports[1])
	}
}

// TestReadManifest checks the reader of the package.json file.
func TestReadManifest(t *testing.T) {
	manifest := typescript.ReadManifest(t.TempDir())
	if manifest.Found {
		t.Error("a manifest was found in an empty directory")
	}
}
