package golang

import (
	"go/parser"
	"go/token"
	"sort"
	"strconv"

	"github.com/barceloch/codixai/internal/languages/lexical"
)

// ImportSpecs returns the imports that the given source declares, with the
// line of every one of them.
//
// The source is parsed with go/parser and go/ast, which only read the text:
// no module is downloaded, no package manager is invoked and no code is
// executed. The line is the line of the import specification, so a caller can
// attribute the import to a line of a diff.
func ImportSpecs(src []byte) ([]lexical.ImportSpec, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	specs := make([]lexical.ImportSpec, 0, len(file.Imports))
	for _, spec := range file.Imports {
		if spec.Path == nil {
			continue
		}
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		specs = append(specs, lexical.ImportSpec{
			Module: path,
			Line:   fset.Position(spec.Pos()).Line,
		})
	}
	sort.SliceStable(specs, func(i, j int) bool {
		if specs[i].Line != specs[j].Line {
			return specs[i].Line < specs[j].Line
		}
		return specs[i].Module < specs[j].Module
	})
	return specs, nil
}
