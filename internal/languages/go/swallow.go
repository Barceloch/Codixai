package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// BlankErrorAssignment is a statement that assigns the result of a call to the
// blank identifier when the called function is declared to return an error.
type BlankErrorAssignment struct {
	// Line is the line of the assignment in the file.
	Line int
	// Callee is the name of the called function.
	Callee string
}

// FindBlankErrorAssignments returns the assignments to the blank identifier
// whose discarded value is declared as an error.
//
// The analysis needs the declaration of the called function: the result list
// of the declaration states the type, and the type is the only evidence that
// the discarded value is an error. A call whose declaration is not in the
// package directory, such as a method or a function of another package, is not
// reported: the analysis has no type information for it and it does not assume
// that the value is an error.
//
// The siblings are the other files of the package directory, so a function
// declared in one of them is resolved too.
func FindBlankErrorAssignments(filename string, src []byte, siblings []SiblingFile) ([]BlankErrorAssignment, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	packageName := ""
	if file.Name != nil {
		packageName = file.Name.Name
	}

	results := map[string][]ast.Expr{}
	collectResults(file, results)
	for _, sibling := range siblings {
		siblingSet := token.NewFileSet()
		siblingFile, err := parser.ParseFile(siblingSet, sibling.Name, sibling.Src, parser.ParseComments)
		if err != nil {
			continue
		}
		if siblingFile.Name == nil || siblingFile.Name.Name != packageName {
			// Files of another package do not declare visible functions.
			continue
		}
		collectResults(siblingFile, results)
	}

	var assignments []BlankErrorAssignment
	seen := map[int]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			statement, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, assignment := range blankErrorAssignmentsOf(statement, results, fset) {
				if seen[assignment.Line] {
					continue
				}
				seen[assignment.Line] = true
				assignments = append(assignments, assignment)
			}
			return true
		})
	}
	sort.Slice(assignments, func(i, j int) bool { return assignments[i].Line < assignments[j].Line })
	return assignments, nil
}

// blankErrorAssignmentsOf returns the assignments of one statement that
// discard an error.
func blankErrorAssignmentsOf(statement *ast.AssignStmt, results map[string][]ast.Expr, fset *token.FileSet) []BlankErrorAssignment {
	if statement.Tok != token.ASSIGN {
		return nil
	}
	if len(statement.Lhs) == 0 {
		return nil
	}

	// A multiple assignment from a single call, such as "_, _ = read()",
	// spreads the results of the call over the left hand side.
	if len(statement.Rhs) == 1 && len(statement.Lhs) > 1 {
		call, ok := statement.Rhs[0].(*ast.CallExpr)
		if !ok {
			return nil
		}
		callee, declared := calleeResults(call, results)
		if callee == "" || len(declared) != len(statement.Lhs) {
			return nil
		}
		var found []BlankErrorAssignment
		for index, expression := range statement.Lhs {
			if !isBlank(expression) || !isErrorType(declared[index]) {
				continue
			}
			found = append(found, BlankErrorAssignment{
				Line:   fset.Position(statement.Pos()).Line,
				Callee: callee,
			})
			break
		}
		return found
	}

	// One call per element of the right hand side.
	var found []BlankErrorAssignment
	for index, expression := range statement.Lhs {
		if index >= len(statement.Rhs) {
			break
		}
		if !isBlank(expression) {
			continue
		}
		call, ok := statement.Rhs[index].(*ast.CallExpr)
		if !ok {
			continue
		}
		callee, declared := calleeResults(call, results)
		if callee == "" || len(declared) != 1 || !isErrorType(declared[0]) {
			continue
		}
		found = append(found, BlankErrorAssignment{
			Line:   fset.Position(statement.Pos()).Line,
			Callee: callee,
		})
	}
	return found
}

// calleeResults returns the name of the called function and the result types
// that the package directory declares for it.
//
// A selector, such as the call of a method or of a function of another
// package, is not resolved: the analysis would need the type of the receiver
// or the content of the other package.
func calleeResults(call *ast.CallExpr, results map[string][]ast.Expr) (string, []ast.Expr) {
	identifier, ok := call.Fun.(*ast.Ident)
	if !ok || identifier == nil {
		return "", nil
	}
	declared, found := results[identifier.Name]
	if !found {
		return "", nil
	}
	return identifier.Name, declared
}

// collectResults adds the result types of the functions of a file to the given
// map, keyed by the name of the function.
func collectResults(file *ast.File, results map[string][]ast.Expr) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Type == nil || fn.Type.Results == nil {
			continue
		}
		var types []ast.Expr
		for _, field := range fn.Type.Results.List {
			if len(field.Names) == 0 {
				types = append(types, field.Type)
				continue
			}
			// A field with several names declares the same type once per
			// name, as in "func f() (a, b error)".
			for range field.Names {
				types = append(types, field.Type)
			}
		}
		results[fn.Name.Name] = types
	}
}

// isBlank reports whether the expression is the blank identifier.
func isBlank(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier != nil && identifier.Name == "_"
}

// isErrorType reports whether the expression names the predeclared error type.
//
// A named type that implements the error interface is not resolved: the
// analysis reads the syntax, it does not check types.
func isErrorType(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier != nil && identifier.Name == "error"
}
