package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/languages/lexical"
)

// writeGoFile writes a Go file below root.
func writeGoFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", name, err)
	}
}

// importFile is a Go file with an import block whose opening line is line 3
// and whose last import is line 10.
const importFile = `package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/unknown/lib"
)

func main() {}
`

// TestGoImportSpecsFromDiskReadsTheBlockThatTheHunkCannotSee checks that the
// imports of the file on disk are recovered when the hunk does not contain the
// line that opens the block.
func TestGoImportSpecsFromDiskReadsTheBlockThatTheHunkCannotSee(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "main.go", importFile)
	ctx := &Context{
		Root: root,
		View: &View{
			Path:  "main.go",
			Added: []lexical.Line{{Number: 10, Text: "\t\"github.com/unknown/lib\""}},
		},
	}
	specs := goImportSpecsFromDisk(ctx)
	if len(specs) != 1 {
		t.Fatalf("the imports are %+v, want the one of the added line", specs)
	}
	if specs[0].Module != "github.com/unknown/lib" || specs[0].Line != 10 {
		t.Fatalf("the import is %+v, want github.com/unknown/lib on line 10", specs[0])
	}
}

// TestGoImportSpecsFromDiskNeverInventsAnImport checks that an import of the
// file on disk is not accepted when the line that the change adds declares
// something else, which is the case of a working tree that differs from the
// analyzed revision.
func TestGoImportSpecsFromDiskNeverInventsAnImport(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "main.go", importFile)
	ctx := &Context{
		Root: root,
		View: &View{
			Path:  "main.go",
			Added: []lexical.Line{{Number: 10, Text: "\t\"example.com/other/module\""}},
		},
	}
	if specs := goImportSpecsFromDisk(ctx); len(specs) != 0 {
		t.Fatalf("the imports are %+v, want none: the added line declares another module", specs)
	}
}

// TestGoImportSpecsFromDiskRequiresAnAddedLine checks that an import of the
// file on disk is not reported when the change does not add its line.
func TestGoImportSpecsFromDiskRequiresAnAddedLine(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "main.go", importFile)
	ctx := &Context{Root: root, View: &View{Path: "main.go"}}
	if specs := goImportSpecsFromDisk(ctx); len(specs) != 0 {
		t.Fatalf("the imports are %+v, want none: the change adds no line", specs)
	}
}

// TestMergeImportSpecsDropsDuplicates checks that the two sources of imports
// do not report the same import twice.
func TestMergeImportSpecsDropsDuplicates(t *testing.T) {
	spec := lexical.ImportSpec{Module: "github.com/unknown/lib", Line: 10}
	merged := mergeImportSpecs(
		[]lexical.ImportSpec{spec},
		[]lexical.ImportSpec{spec, {Module: "github.com/other/lib", Line: 11}},
	)
	if len(merged) != 2 {
		t.Fatalf("the merged imports are %+v, want two", merged)
	}
	if merged[0].Line != 10 || merged[1].Line != 11 {
		t.Fatalf("the merged imports are not sorted by line: %+v", merged)
	}
}
