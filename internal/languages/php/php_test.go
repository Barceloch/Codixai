package php_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/languages/lexical"
	"github.com/barceloch/codixai/internal/languages/php"
)

// TestReadManifest checks the reader of the composer.json file.
func TestReadManifest(t *testing.T) {
	directory := t.TempDir()
	content := `{
  "name": "acme/app",
  "require": {
    "acme/lib": "^1.0"
  },
  "require-dev": {
    "phpunit/phpunit": "^10.0"
  },
  "autoload": {
    "psr-4": {
      "App\\": "src/"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(directory, "composer.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("the manifest cannot be written: %v", err)
	}
	manifest := php.ReadManifest(directory)
	if !manifest.Found {
		t.Fatal("the manifest was not found")
	}
	if !manifest.Declares("acme/lib") {
		t.Error("the manifest does not declare acme/lib")
	}
	if !manifest.Declares("phpunit/phpunit") {
		t.Error("the manifest does not declare the development dependencies")
	}
	if manifest.Declares("acme/missing") {
		t.Error("the manifest declares a package that it does not require")
	}
	if !manifest.IsLocal("App\\Service\\Client") {
		t.Error("the manifest does not recognize the namespace of the project")
	}
	if manifest.IsLocal("Other\\Service") {
		t.Error("the manifest recognizes a foreign namespace as local")
	}
}

// TestPackageOf checks the mapping between a namespace and a package.
func TestPackageOf(t *testing.T) {
	packageName, ok := php.PackageOf("Acme\\Library\\Client")
	if !ok {
		t.Fatal("the package of a namespaced class was not found")
	}
	if packageName != "acme/library" {
		t.Errorf("the package is %q, want %q", packageName, "acme/library")
	}
	if _, ok := php.PackageOf("Exception"); ok {
		t.Error("a global class has no package")
	}
}

// TestExtractImports checks the extraction of the PHP use statements.
func TestExtractImports(t *testing.T) {
	profile := php.Profile()
	scanner := lexical.NewScanner(profile)
	texts := []string{
		"use Acme\\Library\\Client;",
		"use function Acme\\Library\\helper;",
		"use App\\{Service, Other};",
	}
	var lines []lexical.Line
	segments := map[int][]lexical.Segment{}
	for i, text := range texts {
		number := i + 1
		lines = append(lines, lexical.Line{Number: number, Text: text})
		segments[number] = scanner.Scan(text)
	}
	imports := php.ExtractImports(lines, segments)
	if len(imports) != 3 {
		t.Fatalf("the extractor found %d imports, want 3: %+v", len(imports), imports)
	}
	if imports[0].Module != "Acme\\Library\\Client" {
		t.Errorf("the first import is %+v", imports[0])
	}
	if imports[2].Module != "App\\Other" {
		t.Errorf("the third import is %+v", imports[2])
	}
}

// TestEmptyHandlers checks the detection of the empty catch blocks.
func TestEmptyHandlers(t *testing.T) {
	profile := php.Profile()
	scanner := lexical.NewScanner(profile)
	texts := []string{
		"try {",
		"    doWork();",
		"} catch (\\Throwable $e) {",
		"}",
		"return true;",
	}
	var lines []lexical.Line
	segments := map[int][]lexical.Segment{}
	for i, text := range texts {
		number := i + 1
		lines = append(lines, lexical.Line{Number: number, Text: text})
		segments[number] = scanner.Scan(text)
	}
	found := php.EmptyHandlers(lines, segments)
	if len(found) != 1 || found[0] != 3 {
		t.Errorf("the empty handlers are %v, want [3]", found)
	}
}
