package source_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/source"
)

// writeFile writes a file below root, creating its directories.
func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", name, err)
	}
}

// openSource opens a directory source over root.
func openSource(t *testing.T, root string) *source.DirectorySource {
	t.Helper()
	ds, err := source.NewDirectorySource(root, nil)
	if err != nil {
		t.Fatalf("cannot open the directory source: %v", err)
	}
	return ds
}

// filePaths returns the analyzed paths of a file set, sorted by the walk.
func filePaths(set *source.FileSetInfo) []string {
	paths := make([]string, 0, len(set.Files))
	for _, file := range set.Files {
		paths = append(paths, file.Path)
	}
	return paths
}

// diagnosticsContain reports whether a diagnostic contains the fragment.
func diagnosticsContain(set *source.FileSetInfo, fragment string) bool {
	for _, diagnostic := range set.Diagnostics {
		if strings.Contains(diagnostic, fragment) {
			return true
		}
	}
	return false
}

// TestDirectorySourceCountsTheDiscoveredFiles checks the counters of a
// discovery: every walked file is discovered, an unsupported language is
// skipped and an analyzed file belongs to the file set.
func TestDirectorySourceCountsTheDiscoveredFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "sub/lib.py", "print('x')\n")
	writeFile(t, root, "notes.txt", "hello\n")
	writeFile(t, root, "go.mod", "module example.com/app\n")

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	if set.Discovered != 4 {
		t.Errorf("discovered is %d, want 4", set.Discovered)
	}
	if set.Excluded != 0 {
		t.Errorf("excluded is %d, want 0", set.Excluded)
	}
	if set.Skipped != 2 {
		t.Errorf("skipped is %d, want 2 (notes.txt and go.mod)", set.Skipped)
	}
	if len(set.Files) != 2 {
		t.Fatalf("the analyzed files are %v, want two", filePaths(set))
	}
	if set.Files[0].Path != "main.go" || set.Files[1].Path != "sub/lib.py" {
		t.Errorf("the analyzed files are %v, want main.go and sub/lib.py", filePaths(set))
	}
}

// TestDirectorySourceRejectsASymlinkOutsideTheRoot checks that a link that
// points outside the analyzed root is never read.
func TestDirectorySourceRejectsASymlinkOutsideTheRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, outside, "secret.go", "package secret\n")

	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(root, "leak.go")); err != nil {
		t.Skipf("the platform cannot create symbolic links: %v", err)
	}

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	for _, path := range filePaths(set) {
		if path == "leak.go" {
			t.Fatal("the source read a link that points outside the root")
		}
	}
	if !diagnosticsContain(set, "access denied") {
		t.Errorf("the discovery does not explain the rejected link: %v", set.Diagnostics)
	}
}

// TestDirectorySourceRejectsAPrefixSibling checks that a directory whose name
// shares a prefix with the root is not mistaken for a part of the root.
func TestDirectorySourceRejectsAPrefixSibling(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	sibling := filepath.Join(parent, "project-external")
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, sibling, "trap.go", "package trap\n")

	if err := os.Symlink(filepath.Join(sibling, "trap.go"), filepath.Join(root, "trap.go")); err != nil {
		t.Skipf("the platform cannot create symbolic links: %v", err)
	}

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	for _, path := range filePaths(set) {
		if path == "trap.go" {
			t.Fatal("the source read a link to a sibling directory with a shared prefix")
		}
	}
}

// TestDirectorySourceReadsAnInternalSymlink checks that a link that points
// inside the root is followed.
func TestDirectorySourceReadsAnInternalSymlink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")

	if err := os.Symlink(filepath.Join(root, "main.go"), filepath.Join(root, "alias.go")); err != nil {
		t.Skipf("the platform cannot create symbolic links: %v", err)
	}

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	found := false
	for _, path := range filePaths(set) {
		if path == "alias.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("the source did not read the internal link: %v", filePaths(set))
	}
}

// TestDirectorySourceSkipsABrokenSymlink checks that a broken link is skipped
// with a diagnostic instead of failing the discovery.
func TestDirectorySourceSkipsABrokenSymlink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")

	if err := os.Symlink(filepath.Join(root, "missing.go"), filepath.Join(root, "broken.go")); err != nil {
		t.Skipf("the platform cannot create symbolic links: %v", err)
	}

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("a broken link failed the discovery: %v", err)
	}
	if !diagnosticsContain(set, "broken.go") {
		t.Errorf("the discovery does not report the broken link: %v", set.Diagnostics)
	}
	if set.Skipped != 1 {
		t.Errorf("skipped is %d, want 1", set.Skipped)
	}
}

// TestDirectorySourceAnalyzesASymlinkedRoot checks that a root that is itself
// a symbolic link is resolved and analyzed.
func TestDirectorySourceAnalyzesASymlinkedRoot(t *testing.T) {
	real := t.TempDir()
	writeFile(t, real, "main.go", "package main\n")

	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("the platform cannot create symbolic links: %v", err)
	}

	set, err := openSource(t, link).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	if len(set.Files) != 1 || set.Files[0].Path != "main.go" {
		t.Errorf("the analyzed files are %v, want main.go", filePaths(set))
	}
}

// TestDirectorySourceSkipsAnOversizedFile checks that a file larger than the
// limit is not read and is counted as skipped.
func TestDirectorySourceSkipsAnOversizedFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")

	path := filepath.Join(root, "big.go")
	content := make([]byte, source.MaxFileSize+1)
	for index := range content {
		content[index] = 'x'
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("cannot write the oversized file: %v", err)
	}

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	if len(set.Files) != 1 || set.Files[0].Path != "main.go" {
		t.Errorf("the analyzed files are %v, want main.go", filePaths(set))
	}
	if set.Skipped != 1 {
		t.Errorf("skipped is %d, want 1", set.Skipped)
	}
	if !diagnosticsContain(set, "too large") {
		t.Errorf("the discovery does not report the size limit: %v", set.Diagnostics)
	}
}

// TestDirectorySourceSkipsABinaryFile checks that a file with a source
// extension and binary content is not analyzed.
func TestDirectorySourceSkipsABinaryFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "blob.go", "package main\x00\x01\x02binary")

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	if len(set.Files) != 1 || set.Files[0].Path != "main.go" {
		t.Errorf("the analyzed files are %v, want main.go", filePaths(set))
	}
	if set.Skipped != 1 {
		t.Errorf("skipped is %d, want 1", set.Skipped)
	}
}

// TestDirectorySourceExcludesConfiguredPaths checks that the excluded paths of
// the configuration are not read.
func TestDirectorySourceExcludesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	writeFile(t, root, "node_modules/pkg/index.js", "export const x = 1;\n")

	set, err := openSource(t, root).Files(context.Background())
	if err != nil {
		t.Fatalf("the discovery failed: %v", err)
	}
	if len(set.Files) != 1 || set.Files[0].Path != "main.go" {
		t.Errorf("the analyzed files are %v, want main.go", filePaths(set))
	}
	if set.Excluded != 0 {
		// The excluded path is a directory: it is reported in the excluded
		// paths, not counted as an excluded file.
		t.Errorf("excluded is %d, want 0", set.Excluded)
	}
	found := false
	for _, path := range set.ExcludedPaths {
		if path == "node_modules" {
			found = true
		}
	}
	if !found {
		t.Errorf("the excluded paths are %v, want node_modules", set.ExcludedPaths)
	}
}

// TestDirectorySourceRejectsAnUnknownRoot checks the error of a root that does
// not exist.
func TestDirectorySourceRejectsAnUnknownRoot(t *testing.T) {
	if _, err := source.NewDirectorySource(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("a missing root was accepted")
	}
}

// TestDirectorySourceRejectsAFileRoot checks the error of a root that is a
// file.
func TestDirectorySourceRejectsAFileRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n")
	if _, err := source.NewDirectorySource(filepath.Join(root, "main.go"), nil); err == nil {
		t.Fatal("a file was accepted as a directory root")
	}
}
