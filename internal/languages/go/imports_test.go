package golang_test

import (
	"testing"

	golang "github.com/barceloch/codixai/internal/languages/go"
)

// TestImportSpecsReadsTheWholeBlock checks that the parser returns every
// import of the file with the line that declares it, including a block whose
// opening line is far from its last entry.
func TestImportSpecsReadsTheWholeBlock(t *testing.T) {
	source := []byte(`package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/unknown/lib"
)

func main() {}
`)
	specs, err := golang.ImportSpecs(source)
	if err != nil {
		t.Fatalf("the analysis returned an error: %v", err)
	}
	want := map[string]int{
		"fmt":                    4,
		"os":                     5,
		"sort":                   6,
		"strings":                7,
		"time":                   8,
		"github.com/unknown/lib": 10,
	}
	if len(specs) != len(want) {
		t.Fatalf("the imports are %+v, want %d", specs, len(want))
	}
	for _, spec := range specs {
		line, ok := want[spec.Module]
		if !ok {
			t.Errorf("unexpected import %q", spec.Module)
			continue
		}
		if spec.Line != line {
			t.Errorf("the import %q is on line %d, want %d", spec.Module, spec.Line, line)
		}
	}
}
