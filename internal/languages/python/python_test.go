package python_test

import (
	"os"
	"path/filepath"
	"testing"

	"codixai/internal/languages/lexical"
	"codixai/internal/languages/python"
)

// TestReadRequirements checks the reader of the requirements.txt file.
func TestReadRequirements(t *testing.T) {
	directory := t.TempDir()
	content := `# the dependencies of the project
requests==2.31.0
Flask >= 3.0
httpx[http2]==0.27.0
some_package>=1.0; python_version >= "3.10"
-r other.txt
-e .
`
	if err := os.WriteFile(filepath.Join(directory, "requirements.txt"), []byte(content), 0o600); err != nil {
		t.Fatalf("the manifest cannot be written: %v", err)
	}
	manifest := python.ReadManifest(directory)
	if !manifest.Found {
		t.Fatal("the manifest was not found")
	}
	if !manifest.Declares("requests") {
		t.Error("the manifest does not declare requests")
	}
	if !manifest.Declares("flask") {
		t.Error("the manifest does not declare flask")
	}
	if !manifest.Declares("httpx") {
		t.Error("the manifest does not declare httpx")
	}
	if !manifest.Declares("some-package") {
		t.Error("the manifest does not normalize the name of a distribution")
	}
	if manifest.Declares("missing") {
		t.Error("the manifest declares a distribution that it does not require")
	}
}

// TestReadPyproject checks the reader of the pyproject.toml file.
func TestReadPyproject(t *testing.T) {
	directory := t.TempDir()
	content := `[build-system]
requires = ["setuptools"]

[project]
name = "app"
dependencies = [
    "httpx==0.27.0",
    "requests>=2.31.0",
]

[project.optional-dependencies]
dev = ["pytest==8.0.0"]

[tool.poetry.dependencies]
python = "^3.12"
django = "^5.0"
`
	if err := os.WriteFile(filepath.Join(directory, "pyproject.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("the manifest cannot be written: %v", err)
	}
	manifest := python.ReadManifest(directory)
	if !manifest.Found {
		t.Fatal("the manifest was not found")
	}
	for _, name := range []string{"httpx", "requests", "pytest", "django"} {
		if !manifest.Declares(name) {
			t.Errorf("the manifest does not declare %s", name)
		}
	}
	if manifest.Declares("setuptools") {
		t.Error("the manifest declares a build requirement as a dependency")
	}
	if manifest.Declares("python") {
		t.Error("the manifest declares the python version as a dependency")
	}
}

// TestReadManifestWithoutFile checks the reader when there is no manifest.
func TestReadManifestWithoutFile(t *testing.T) {
	manifest := python.ReadManifest(t.TempDir())
	if manifest.Found {
		t.Error("a manifest was found in an empty directory")
	}
}

// TestNormalize checks the normalization of the distribution names.
func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Requests":       "requests",
		"some_package":   "some-package",
		"some.package":   "some-package",
		"zope.interface": "zope-interface",
	}
	for input, want := range cases {
		if got := python.Normalize(input); got != want {
			t.Errorf("Normalize(%q) is %q, want %q", input, got, want)
		}
	}
}

// TestExtractImports checks the extraction of the Python imports.
func TestExtractImports(t *testing.T) {
	lines := []lexical.Line{
		{Number: 1, Text: "import os"},
		{Number: 2, Text: "import httpx"},
		{Number: 3, Text: "from . import sibling"},
		{Number: 4, Text: "from package.module import value"},
	}
	segments := map[int][]lexical.Segment{}
	imports := python.ExtractImports(lines, segments)
	if len(imports) != 4 {
		t.Fatalf("the extractor found %d imports, want 4: %+v", len(imports), imports)
	}
	if imports[1].Module != "httpx" || imports[1].Relative {
		t.Errorf("the second import is %+v", imports[1])
	}
	if !imports[2].Relative {
		t.Errorf("the third import is %+v, want a relative import", imports[2])
	}
	if imports[3].Module != "package.module" {
		t.Errorf("the fourth import is %+v", imports[3])
	}
}

// TestExtractImportsIgnoresComments checks that an import written in a
// comment is not extracted.
func TestExtractImportsIgnoresComments(t *testing.T) {
	profile := python.Profile()
	scanner := lexical.NewScanner(profile)
	lines := []lexical.Line{{Number: 1, Text: "# import httpx"}}
	segments := map[int][]lexical.Segment{1: scanner.Scan(lines[0].Text)}
	imports := python.ExtractImports(lines, segments)
	if len(imports) != 0 {
		t.Errorf("the extractor found %+v, want no imports", imports)
	}
}
