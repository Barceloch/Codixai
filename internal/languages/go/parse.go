package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
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

// Symbol is a named declaration extracted from Go source.
type Symbol struct {
	// Name is the name of the symbol.
	Name string
	// Kind is the kind of symbol: func, type, var, const, package.
	Kind string
	// File is the file path.
	File string
	// Line is the line number.
	Line int
	// Column is the column number.
	Column int
	// Language is the programming language.
	Language string
	// Container is the containing package or type.
	Container string
	// Signature is the signature of the symbol when available.
	Signature string
	// Exported indicates whether the symbol is exported.
	Exported bool
}

// File is the extracted structure of a Go source file.
type File struct {
	// Package is the name of the declared package.
	Package string
	// Imports are the import declarations of the file.
	Imports []Import
	// Comments are the comments of the file.
	Comments []Comment
	// Symbols are the declared symbols in the file.
	Symbols []Symbol
}

// Parse extracts the imports, symbols, and comments of a Go source file.
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
	file := &File{Imports: []Import{}, Comments: []Comment{}, Symbols: []Symbol{}}
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

	ast.Inspect(parsed, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Name == nil {
				return true
			}
			exported := isExported(n.Name.Name)
			sig := formatFuncSig(n)
			container := ""
			if n.Recv != nil && len(n.Recv.List) > 0 {
				if rt := n.Recv.List[0].Type; rt != nil {
					container = receiverTypeName(rt)
				}
			}
			file.Symbols = append(file.Symbols, Symbol{
				Name:      n.Name.Name,
				Kind:      "func",
				File:      filename,
				Line:      fset.Position(n.Pos()).Line,
				Column:    fset.Position(n.Pos()).Column,
				Language:  "go",
				Container: container,
				Signature: sig,
				Exported:  exported,
			})
		case *ast.GenDecl:
			switch n.Tok {
			case token.TYPE:
				for _, spec := range n.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name != nil {
						exported := isExported(ts.Name.Name)
						container := ""
						sig := ts.Name.Name
						if _, ok := ts.Type.(*ast.StructType); ok {
							sig = "struct { ... }"
						} else if _, ok := ts.Type.(*ast.InterfaceType); ok {
							sig = "interface { ... }"
						}
						file.Symbols = append(file.Symbols, Symbol{
							Name:      ts.Name.Name,
							Kind:      "type",
							File:      filename,
							Line:      fset.Position(ts.Pos()).Line,
							Column:    fset.Position(ts.Pos()).Column,
							Language:  "go",
							Container: container,
							Signature: sig,
							Exported:  exported,
						})
					}
				}
			case token.VAR, token.CONST:
				for _, spec := range n.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range vs.Names {
							if name == nil {
								continue
							}
							exported := isExported(name.Name)
							file.Symbols = append(file.Symbols, Symbol{
								Name:      name.Name,
								Kind:      "var",
								File:      filename,
								Line:      fset.Position(name.Pos()).Line,
								Column:    fset.Position(name.Pos()).Column,
								Language:  "go",
								Container: "",
								Signature: "",
								Exported:  exported,
							})
						}
					}
				}
			}
		}
		return true
	})

	return file, nil
}

func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	r := []rune(name)[0]
	return r >= 'A' && r <= 'Z'
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.IndexListExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func formatFuncSig(fn *ast.FuncDecl) string {
	var parts []string
	parts = append(parts, "func")
	if fn.Recv != nil {
		parts = append(parts, "recv")
	}
	if fn.Name != nil {
		parts = append(parts, fn.Name.Name)
	}
	if fn.Type != nil {
		if fn.Type.Params != nil {
			parts = append(parts, "()")
		}
	}
	return strings.Join(parts, " ")
}
