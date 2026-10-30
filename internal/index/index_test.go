package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/languages/registry"
)

func TestIndexBuildAndFind(t *testing.T) {
	// Create a temporary directory with some Go files
	tmpDir := t.TempDir()

	// Write a simple Go file
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello")
}
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Write a Python file
	pyFile := filepath.Join(tmpDir, "script.py")
	err = os.WriteFile(pyFile, []byte(`def hello():
    print("hello")

if __name__ == "__main__":
    hello()
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Build index
	idx := New()
	err = idx.Build(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check symbols
	symbols := idx.Symbols()
	if len(symbols) == 0 {
		t.Fatal("expected symbols, got none")
	}

	// Check Go package symbol
	foundPackage := false
	for _, sym := range symbols {
		if sym.Kind == "package" && sym.Name == "main" {
			foundPackage = true
			break
		}
	}
	if !foundPackage {
		t.Errorf("expected package symbol 'main', not found. Got: %v", symbols)
	}

	// Check imports in dependencies
	deps := idx.Dependencies()
	foundImport := false
	for _, dep := range deps {
		if dep.Path == "fmt" {
			foundImport = true
			break
		}
	}
	if !foundImport {
		t.Errorf("expected import 'fmt' in dependencies, not found. Got: %v", deps)
	}

	// Check Python file symbol
	foundPy := false
	for _, sym := range symbols {
		if sym.Language == "python" && sym.Kind == "file" {
			foundPy = true
			break
		}
	}
	if !foundPy {
		t.Errorf("expected python file symbol")
	}

	// Test Find - symbols only (package, not imports)
	results := idx.Find("main")
	if len(results) == 0 {
		t.Errorf("Find('main') returned no results")
	}
	found := false
	for _, r := range results {
		if r.Name == "main" && r.Kind == "package" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Find('main') did not return package symbol")
	}
}

func TestIndexDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte(`package main

import (
	"fmt"
	"github.com/foo/bar"
)

func main() {
	fmt.Println(bar.Baz())
}
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	idx := New()
	err = idx.Build(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	deps := idx.Dependencies()
	if len(deps) < 2 {
		t.Errorf("expected at least 2 dependencies, got %d", len(deps))
	}

	foundFmt := false
	foundBar := false
	for _, dep := range deps {
		if dep.Path == "fmt" {
			foundFmt = true
		}
		if dep.Path == "github.com/foo/bar" {
			foundBar = true
		}
	}
	if !foundFmt {
		t.Errorf("expected dependency 'fmt'")
	}
	if !foundBar {
		t.Errorf("expected dependency 'github.com/foo/bar'")
	}
}

func TestRegistryDetect(t *testing.T) {
	tests := []struct {
		path string
		lang string
		ok   bool
	}{
		{"main.go", "go", true},
		{"script.py", "python", true},
		{"app.js", "javascript", true},
		{"app.ts", "typescript", true},
		{"index.php", "php", true},
		{"unknown.xyz", "", false},
	}

	for _, tc := range tests {
		lang, ok := registry.Detect(tc.path)
		if ok != tc.ok {
			t.Errorf("Detect(%q): ok=%v, want %v", tc.path, ok, tc.ok)
		}
		if ok && lang != tc.lang {
			t.Errorf("Detect(%q): lang=%q, want %q", tc.path, lang, tc.lang)
		}
	}
}

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{".git", true},
		{"vendor", true},
		{"node_modules", true},
		{"vendor_test.go", false},
		{".gitignore", false},
		{"node_modules.js", false},
		{"src", false},
		{"pkg", false},
		{"test", false},
	}

	for _, tc := range tests {
		got := shouldSkipDir(tc.name)
		if got != tc.want {
			t.Errorf("shouldSkipDir(%q): got=%v, want=%v", tc.name, got, tc.want)
		}
	}
}

func TestIsInsideSkippedDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(tmpDir, ".git", "config"), true},
		{filepath.Join(tmpDir, "vendor", "foo.go"), true},
		{filepath.Join(tmpDir, "node_modules", "index.js"), true},
		{filepath.Join(tmpDir, "src", "main.go"), false},
		{filepath.Join(tmpDir, "vendor_test.go"), false},
		{filepath.Join(tmpDir, ".gitignore"), false},
		{filepath.Join(tmpDir, "node_modules.js"), false},
		{filepath.Join(tmpDir, "pkg", "vendor", "foo.go"), true}, // nested vendor
	}

	for _, tc := range tests {
		got := isInsideSkippedDir(tmpDir, tc.path)
		if got != tc.want {
			t.Errorf("isInsideSkippedDir(%q): got=%v, want=%v", tc.path, got, tc.want)
		}
	}
}

func TestIndexSkipsDotGitVendorNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	t.Logf("Temp dir: %s", tmpDir)
	// Create files that should be skipped
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte(""), 0o644)

	os.MkdirAll(filepath.Join(tmpDir, "vendor", "github.com", "foo"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "vendor", "github.com", "foo", "bar.go"), []byte(`package foo

func Bar() {}
`), 0o644)

	os.MkdirAll(filepath.Join(tmpDir, "node_modules", "foo"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "node_modules", "foo", "index.js"), []byte(`module.exports = {};`), 0o644)

	// Create files that should NOT be skipped (false positive cases)
	os.WriteFile(filepath.Join(tmpDir, "vendor_test.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Println("test")
}
`), 0o644)

	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.tmp\n"), 0o644)

	os.WriteFile(filepath.Join(tmpDir, "node_modules.js"), []byte(`console.log("test");`), 0o644)

	// Create a normal file that should be indexed
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`), 0o644)

	// Debug: list all files in temp dir
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		t.Logf("Walk: %s (isDir=%v)", path, info.IsDir())
		return nil
	})

	idx := New()
	err := idx.Build(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	symbols := idx.Symbols()

	// Should have indexed main.go
	foundMain := false
	for _, sym := range symbols {
		if sym.Kind == "package" && sym.Name == "main" {
			foundMain = true
			break
		}
		// Debug output
		t.Logf("Symbol: %s (kind=%s, file=%s)", sym.Name, sym.Kind, sym.File)
	}
	if !foundMain {
		t.Errorf("expected package symbol 'main' from main.go, not found")
	}

	// Should NOT have indexed vendor/github.com/foo/bar.go
	foundVendor := false
	for _, sym := range symbols {
		if isInsidePath(sym.File, "vendor") && sym.Kind == "package" {
			foundVendor = true
			break
		}
	}
	if foundVendor {
		t.Errorf("expected vendor/ to be skipped, but found symbols from it")
	}

	// Should NOT have indexed node_modules/foo/index.js
	foundNodeModules := false
	for _, sym := range symbols {
		if isInsidePath(sym.File, "node_modules") {
			foundNodeModules = true
			break
		}
	}
	if foundNodeModules {
		t.Errorf("expected node_modules/ to be skipped, but found symbols from it")
	}

	// Should NOT have indexed .git/config
	foundDotGit := false
	for _, sym := range symbols {
		if strings.Contains(sym.File, ".git") {
			foundDotGit = true
			break
		}
	}
	if foundDotGit {
		t.Errorf("expected .git/ to be skipped, but found symbols from it")
	}

	// vendor_test.go should be indexed (not skipped)
	foundVendorTest := false
	for _, sym := range symbols {
		if strings.Contains(sym.File, "vendor_test.go") {
			foundVendorTest = true
			break
		}
	}
	if !foundVendorTest {
		t.Errorf("expected vendor_test.go to be indexed, but it was skipped")
	}
}

func isInsidePath(path, dir string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		if part == dir && i < len(parts)-1 {
			return true
		}
	}
	return false
}

func TestIndexContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many files to make indexing take some time
	for i := 0; i < 100; i++ {
		os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i)), []byte(fmt.Sprintf(`package main

func Func%d() {}
`, i)), 0o644)
	}

	idx := New()

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := idx.Build(ctx, tmpDir)
	if err == nil {
		t.Errorf("expected context cancellation error, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestIndexSymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.go")
	os.WriteFile(realFile, []byte(`package main

func Real() {}
`), 0o644)

	// Create a symlink to the real file
	linkFile := filepath.Join(tmpDir, "link.go")
	err := os.Symlink(realFile, linkFile)
	if err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	idx := New()
	err = idx.Build(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should have indexed real.go but NOT the symlink
	symbols := idx.Symbols()
	count := 0
	for _, sym := range symbols {
		if sym.Kind == "package" && sym.Name == "main" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 package symbol (from real.go), got %d (symlink should be skipped)", count)
	}
}

func TestIndexRootEscape(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file in temp dir
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func Main() {}
`), 0o644)

	// Create a subdirectory and try to escape with ..
	subDir := filepath.Join(tmpDir, "sub")
	os.MkdirAll(subDir, 0o755)

	idx := New()
	err := idx.Build(context.Background(), subDir)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should not find the parent directory's main.go
	symbols := idx.Symbols()
	found := false
	for _, sym := range symbols {
		if sym.Kind == "package" && sym.Name == "main" {
			found = true
			break
		}
	}
	if found {
		t.Errorf("expected no symbols from parent directory (root escape prevented)")
	}
}
