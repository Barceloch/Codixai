package golang

import (
	"go/parser"
	"go/token"
	"strconv"
)

// Import is an import declaration of a Go file.
type Import struct {
	// Path is the import path, without quotes.
	Path string
	// Alias is the local name of the import, when it is not the default one.
	Alias string
	// Line is the line of the import declaration.
	Line int
}

// Comment is a comment of a Go file.
type Comment struct {
	// Text is the raw comment text, including the comment markers.
	Text string
	// Line is the line of the comment.
	Line int
}

// File is the extracted structure of a Go source file.
type File struct {
	// Package is the name of the declared package.
	Package string
	// Imports are the import declarations of the file.
	Imports []Import
	// Comments are the comments of the file.
	Comments []Comment
}

// Parse extracts the imports and the comments of a Go source file.
//
// It uses go/parser with the ParseComments mode. The parser only reads the
// given source bytes: no module is resolved, no build is executed and no
// network access happens.
func Parse(filename string, src []byte) (*File, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	file := &File{Imports: []Import{}, Comments: []Comment{}}
	if parsed.Name != nil {
		file.Package = parsed.Name.Name
	}
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		alias := ""
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		file.Imports = append(file.Imports, Import{
			Path:  path,
			Alias: alias,
			Line:  fset.Position(spec.Pos()).Line,
		})
	}
	for _, group := range parsed.Comments {
		for _, comment := range group.List {
			file.Comments = append(file.Comments, Comment{
				Text: comment.Text,
				Line: fset.Position(comment.Pos()).Line,
			})
		}
	}
	return file, nil
}
