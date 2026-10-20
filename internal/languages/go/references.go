package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// SiblingFile is a Go file of the same directory as the analyzed file.
type SiblingFile struct {
	// Name is the file name.
	Name string
	// Src is the content of the file.
	Src []byte
}

// UndefinedReference is a symbol that is referenced by a file and that is
// not declared in the package directory.
type UndefinedReference struct {
	// Name is the referenced symbol.
	Name string
	// Line is the line of the reference.
	Line int
	// Column is the column of the reference.
	Column int
}

// FindUndefinedReferences reports the symbols that the given file references
// and that cannot be resolved inside its package directory.
//
// The analysis is conservative. It resolves a reference only when it is
// certain that the symbol does not exist: the symbol must be absent from the
// declarations of the directory, absent from the builtins, absent from the
// imported package names and absent from the local names of the file.
//
// The second return value reports whether the analysis could be completed.
// It is false when one of the sibling files could not be parsed, because in
// that case a declaration may be missing and a false positive would be
// reported.
func FindUndefinedReferences(filename string, src []byte, siblings []SiblingFile) ([]UndefinedReference, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	packageName := file.Name.Name

	declared := map[string]bool{}
	collectDeclarations(file, declared)

	complete := true
	for _, sibling := range siblings {
		siblingSet := token.NewFileSet()
		siblingFile, err := parser.ParseFile(siblingSet, sibling.Name, sibling.Src, parser.ParseComments)
		if err != nil {
			complete = false
			continue
		}
		if siblingFile.Name.Name != packageName {
			// Files of another package do not declare visible symbols.
			continue
		}
		collectDeclarations(siblingFile, declared)
	}

	imports := map[string]bool{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if spec.Name != nil {
			imports[spec.Name.Name] = true
			continue
		}
		imports[defaultPackageName(path)] = true
	}

	ignored := ignoredPositions(file)

	var references []UndefinedReference
	seen := map[string]bool{}
	for _, decl := range file.Decls {
		locals := map[string]bool{}
		if fn, ok := decl.(*ast.FuncDecl); ok {
			locals = collectLocals(fn)
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			if ignored[ident.Pos()] {
				return false
			}
			name := ident.Name
			if name == "" || name == "_" {
				return false
			}
			if builtins[name] || imports[name] || declared[name] || locals[name] {
				return false
			}
			position := fset.Position(ident.Pos())
			key := name + ":" + strconv.Itoa(position.Line) + ":" + strconv.Itoa(position.Column)
			if seen[key] {
				return false
			}
			seen[key] = true
			references = append(references, UndefinedReference{
				Name:   name,
				Line:   position.Line,
				Column: position.Column,
			})
			return false
		})
	}
	return references, complete, nil
}

// collectDeclarations adds the package level declarations of a file to the
// given set.
func collectDeclarations(file *ast.File, declared map[string]bool) {
	for _, decl := range file.Decls {
		generic, ok := decl.(*ast.GenDecl)
		if !ok {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				declared[fn.Name.Name] = true
			}
			continue
		}
		for _, spec := range generic.Specs {
			switch typed := spec.(type) {
			case *ast.ValueSpec:
				for _, name := range typed.Names {
					declared[name.Name] = true
				}
			case *ast.TypeSpec:
				declared[typed.Name.Name] = true
			}
		}
	}
}

// collectLocals returns the names that are local to a function: its
// parameters, its receiver, its results and every name that it declares.
func collectLocals(fn *ast.FuncDecl) map[string]bool {
	locals := map[string]bool{}
	if fn.Recv != nil {
		for _, field := range fn.Recv.List {
			for _, name := range field.Names {
				locals[name.Name] = true
			}
		}
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				locals[name.Name] = true
			}
		}
	}
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			for _, name := range field.Names {
				locals[name.Name] = true
			}
		}
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if typed.Tok != token.DEFINE {
				return true
			}
			for _, expression := range typed.Lhs {
				if ident, ok := expression.(*ast.Ident); ok {
					locals[ident.Name] = true
				}
			}
		case *ast.DeclStmt:
			generic, ok := typed.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range generic.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range value.Names {
						locals[name.Name] = true
					}
				}
			}
		case *ast.RangeStmt:
			for _, expression := range []ast.Expr{typed.Key, typed.Value} {
				if ident, ok := expression.(*ast.Ident); ok {
					locals[ident.Name] = true
				}
			}
		case *ast.FuncLit:
			if typed.Type.Params != nil {
				for _, field := range typed.Type.Params.List {
					for _, name := range field.Names {
						locals[name.Name] = true
					}
				}
			}
		}
		return true
	})
	return locals
}

// ignoredPositions returns the positions of the identifiers that are not
// symbol references: declaration names, field names, labels, import aliases
// and the selected side of a selector expression.
func ignoredPositions(file *ast.File) map[token.Pos]bool {
	ignored := map[token.Pos]bool{}
	ignored[file.Name.Pos()] = true
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			ignored[typed.Name.Pos()] = true
		case *ast.FuncLit:
			// The parameters and the results are declarations.
		case *ast.ValueSpec:
			for _, name := range typed.Names {
				ignored[name.Pos()] = true
			}
		case *ast.TypeSpec:
			ignored[typed.Name.Pos()] = true
		case *ast.Field:
			for _, name := range typed.Names {
				ignored[name.Pos()] = true
			}
		case *ast.ImportSpec:
			if typed.Name != nil {
				ignored[typed.Name.Pos()] = true
			}
		case *ast.LabeledStmt:
			ignored[typed.Label.Pos()] = true
		case *ast.BranchStmt:
			ignored[typed.Label.Pos()] = true
		case *ast.AssignStmt:
			if typed.Tok != token.DEFINE {
				return true
			}
			for _, expression := range typed.Lhs {
				if ident, ok := expression.(*ast.Ident); ok {
					ignored[ident.Pos()] = true
				}
			}
		case *ast.RangeStmt:
			for _, expression := range []ast.Expr{typed.Key, typed.Value} {
				if ident, ok := expression.(*ast.Ident); ok {
					ignored[ident.Pos()] = true
				}
			}
		case *ast.SelectorExpr:
			// The selected symbol is a field or a method, not a reference
			// to a package level symbol. The base of the selector is
			// resolved as a normal reference: it is skipped when it is an
			// imported package name.
			ignored[typed.Sel.Pos()] = true
		case *ast.KeyValueExpr:
			if ident, ok := typed.Key.(*ast.Ident); ok {
				ignored[ident.Pos()] = true
			}
		}
		return true
	})
	return ignored
}

// defaultPackageName returns the name that an import path binds by default.
func defaultPackageName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if idx := strings.Index(name, "."); idx > 0 {
		name = name[:idx]
	}
	return name
}

// builtins holds the names of the Go builtins.
var builtins = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "complex": true,
	"copy": true, "delete": true, "imag": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true,
}
