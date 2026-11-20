package index

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	golang "github.com/barceloch/codixai/internal/languages/go"
)

// GoReferenceResolver resolves Go references conservatively using AST analysis
// and the structural index.
type GoReferenceResolver struct{}

// NewGoReferenceResolver creates a new Go reference resolver.
func NewGoReferenceResolver() *GoReferenceResolver {
	return &GoReferenceResolver{}
}

// Resolve implements ReferenceResolver for Go.
func (r *GoReferenceResolver) Resolve(ctx context.Context, idx *Index, changes *domain.ChangeSet) ([]domain.Reference, error) {
	if changes == nil {
		return nil, nil
	}
	root, err := filepath.Abs(changes.Root)
	if err != nil {
		root = changes.Root
	}
	var all []domain.Reference
	for _, fc := range changes.Files {
		if fc.Language != domain.LangGo {
			continue
		}
		if fc.Status == domain.FileDeleted {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, fc.Path))
		if err != nil {
			continue
		}
		refs, err := r.resolveFile(root, filepath.ToSlash(fc.Path), src, idx)
		if err != nil {
			continue
		}
		all = append(all, refs...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Location.Path != all[j].Location.Path {
			return all[i].Location.Path < all[j].Location.Path
		}
		if all[i].Location.StartLine != all[j].Location.StartLine {
			return all[i].Location.StartLine < all[j].Location.StartLine
		}
		if all[i].Location.StartColumn != all[j].Location.StartColumn {
			return all[i].Location.StartColumn < all[j].Location.StartColumn
		}
		if all[i].Source.Name != all[j].Source.Name {
			return all[i].Source.Name < all[j].Source.Name
		}
		return all[i].Kind < all[j].Kind
	})
	return all, nil
}

type goResolveContext struct {
	root        string
	filePath    string
	pkgDir      string
	fset        *token.FileSet
	file        *ast.File
	importNames map[string]string // local name/alias -> import path
	symbols     []Symbol
	byFile      map[string][]Symbol
	byName      map[string][]Symbol
	pkgFiles    map[string]bool
}

func (r *GoReferenceResolver) resolveFile(root, filePath string, src []byte, idx *Index) ([]domain.Reference, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	rc := &goResolveContext{
		root:        root,
		filePath:    filePath,
		pkgDir:      filepath.Dir(filePath),
		fset:        fset,
		file:        parsed,
		importNames: map[string]string{},
		byFile:      map[string][]Symbol{},
		byName:      map[string][]Symbol{},
		pkgFiles:    map[string]bool{},
	}
	for _, imp := range parsed.Imports {
		path, err := strconvUnquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := defaultImportName(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "_" || name == "." {
			continue
		}
		rc.importNames[name] = path
	}
	if idx != nil {
		for _, s := range idx.Symbols() {
			s.File = relToRoot(root, s.File)
			rc.symbols = append(rc.symbols, s)
			rc.byFile[s.File] = append(rc.byFile[s.File], s)
			rc.byName[s.Name] = append(rc.byName[s.Name], s)
			if filepath.Dir(s.File) == rc.pkgDir && s.Language == domain.LangGo {
				rc.pkgFiles[s.File] = true
			}
		}
	}
	rc.pkgFiles[filePath] = true

	var refs []domain.Reference
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			refs = append(refs, r.resolveCall(rc, node, localsAtPos(parsed, node.Pos()))...)
		case *ast.TypeSpec:
			refs = append(refs, r.resolveTypeExpr(rc, node.Type, domain.ReferenceType)...)
		case *ast.ValueSpec:
			if node.Type != nil {
				refs = append(refs, r.resolveTypeExpr(rc, node.Type, domain.ReferenceType)...)
			}
		case *ast.Field:
			if node.Type != nil {
				refs = append(refs, r.resolveTypeExpr(rc, node.Type, domain.ReferenceType)...)
			}
		}
		return true
	})
	return refs, nil
}

func (r *GoReferenceResolver) resolveCall(rc *goResolveContext, call *ast.CallExpr, localTypes map[string]string) []domain.Reference {
	pos := rc.fset.Position(call.Pos())
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return []domain.Reference{r.resolveIdentRef(rc, fun.Name, domain.ReferenceCall, pos, "Function call to "+fun.Name, localTypes)}
	case *ast.SelectorExpr:
		selPos := rc.fset.Position(fun.Sel.Pos())
		if ident, ok := fun.X.(*ast.Ident); ok {
			if localTypes != nil {
				if _, isLocal := localTypes[ident.Name]; isLocal {
					if typeName := localTypes[ident.Name]; typeName != "" {
						if ref, ok := r.resolveKnownReceiverMethod(rc, typeName, fun.Sel.Name, selPos); ok {
							return []domain.Reference{ref}
						}
					}
					// Local receiver (possibly shadowing import/package name): conservative.
					ref := domain.Reference{
						Kind:       domain.ReferenceMethod,
						Resolution: domain.ResolutionUnresolved,
						Language:   domain.LangGo,
						Location:   locFromPos(rc.filePath, selPos),
						Source: domain.SymbolRef{
							Name:     fun.Sel.Name,
							Kind:     "method",
							File:     rc.filePath,
							Line:     selPos.Line,
							Column:   selPos.Column,
							Language: domain.LangGo,
						},
						Evidence: []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, selPos.Line, "Method call on local receiver: "+fun.Sel.Name)},
					}
					return []domain.Reference{ref}
				}
			}
			if impPath, ok := rc.importNames[ident.Name]; ok {
				return []domain.Reference{r.resolveImportedSelector(rc, impPath, ident.Name, fun.Sel.Name, selPos)}
			}
		}
		// Method call on unknown receiver: conservative unresolved/dynamic evidence.
		ref := domain.Reference{
			Kind:       domain.ReferenceMethod,
			Resolution: domain.ResolutionUnresolved,
			Language:   domain.LangGo,
			Location:   locFromPos(rc.filePath, selPos),
			Source: domain.SymbolRef{
				Name:     fun.Sel.Name,
				Kind:     "method",
				File:     rc.filePath,
				Line:     selPos.Line,
				Column:   selPos.Column,
				Language: domain.LangGo,
			},
			Evidence: []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, selPos.Line, "Method call on unresolved receiver: "+fun.Sel.Name)},
		}
		return []domain.Reference{ref}
	default:
		return nil
	}
}

func (r *GoReferenceResolver) resolveKnownReceiverMethod(rc *goResolveContext, typeName, methodName string, pos token.Position) (domain.Reference, bool) {
	var cands []Symbol
	for _, s := range rc.byName[methodName] {
		if s.Kind != "func" && s.Kind != "method" {
			continue
		}
		if s.Container != typeName {
			continue
		}
		if !rc.pkgFiles[s.File] && filepath.Dir(s.File) != rc.pkgDir {
			continue
		}
		cands = append(cands, s)
	}
	if len(cands) == 0 {
		return domain.Reference{}, false
	}
	ref := domain.Reference{
		Kind:     domain.ReferenceMethod,
		Language: domain.LangGo,
		Location: locFromPos(rc.filePath, pos),
		Source: domain.SymbolRef{
			Name:     methodName,
			Kind:     "method",
			File:     rc.filePath,
			Line:     pos.Line,
			Column:   pos.Column,
			Language: domain.LangGo,
		},
		Evidence: []domain.ReferenceEvidence{NewASTMethodCallEvidence(rc.filePath, pos.Line, pos.Column, "Method call on known local receiver type "+typeName+": "+methodName)},
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].File < cands[j].File || (cands[i].File == cands[j].File && cands[i].Line < cands[j].Line)
	})
	if len(cands) == 1 {
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(cands[0])
		ref.Target = &t
		return ref, true
	}
	ref.Resolution = domain.ResolutionAmbiguous
	for _, c := range cands {
		ref.Candidates = append(ref.Candidates, symbolToRef(c))
	}
	return ref, true
}

func (r *GoReferenceResolver) resolveImportedSelector(rc *goResolveContext, impPath, alias, name string, pos token.Position) domain.Reference {
	ref := domain.Reference{
		Kind:     domain.ReferenceCall,
		Language: domain.LangGo,
		Location: locFromPos(rc.filePath, pos),
		Source: domain.SymbolRef{
			Name:     name,
			Kind:     "symbol",
			File:     rc.filePath,
			Line:     pos.Line,
			Column:   pos.Column,
			Language: domain.LangGo,
		},
		Evidence: []domain.ReferenceEvidence{
			NewImportDeclarationEvidence(rc.filePath, 0, impPath),
			NewASTCallEvidence(rc.filePath, pos.Line, pos.Column, "Selector reference "+alias+"."+name),
		},
	}
	if golang.IsStdlib(impPath) || golang.IsBuiltin(name) {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "symbol", Language: domain.LangGo, Container: impPath}
		return ref
	}
	cands := r.findImportedCandidates(rc, impPath, name)
	switch len(cands) {
	case 0:
		if r.isKnownLocalImportPackage(rc, impPath) {
			ref.Resolution = domain.ResolutionUnresolved
			ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(name, "Symbol not found in known imported package: "+impPath))
		} else {
			ref.Resolution = domain.ResolutionExternal
			ref.Target = &domain.SymbolRef{Name: name, Kind: "symbol", Language: domain.LangGo, Container: impPath}
		}
	case 1:
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(cands[0])
		ref.Target = &t
	default:
		ref.Resolution = domain.ResolutionAmbiguous
		for _, c := range cands {
			ref.Candidates = append(ref.Candidates, symbolToRef(c))
		}
	}
	return ref
}

func (r *GoReferenceResolver) isKnownLocalImportPackage(rc *goResolveContext, impPath string) bool {
	impSlash := filepath.ToSlash(impPath)
	pkgName := defaultImportName(impPath)
	for _, s := range rc.symbols {
		dirSlash := filepath.ToSlash(filepath.Dir(s.File))
		if dirSlash == impSlash || strings.HasSuffix(dirSlash, "/"+impSlash) || strings.HasSuffix(dirSlash, "/"+pkgName) || (filepath.Base(dirSlash) == pkgName && strings.Contains(impSlash, "/")) {
			return true
		}
	}
	// Also treat package dir with indexed files in same project as known when path matches.
	for f := range rc.byFile {
		dirSlash := filepath.ToSlash(filepath.Dir(f))
		if dirSlash == impSlash || strings.HasSuffix(dirSlash, "/"+impSlash) || strings.HasSuffix(dirSlash, "/"+pkgName) {
			return true
		}
	}
	return false
}

func (r *GoReferenceResolver) findImportedCandidates(rc *goResolveContext, impPath, name string) []Symbol {
	var out []Symbol
	impSlash := filepath.ToSlash(impPath)
	pkgName := defaultImportName(impPath)
	for _, s := range rc.byName[name] {
		dirSlash := filepath.ToSlash(filepath.Dir(s.File))
		if dirSlash == impSlash || strings.HasSuffix(dirSlash, "/"+impSlash) || strings.HasSuffix(dirSlash, "/"+pkgName) || filepath.Base(dirSlash) == pkgName && strings.Contains(impSlash, "/") {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].File < out[j].File || (out[i].File == out[j].File && out[i].Line < out[j].Line)
	})
	return out
}

func (r *GoReferenceResolver) resolveTypeExpr(rc *goResolveContext, expr ast.Expr, kind domain.ReferenceKind) []domain.Reference {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "_" {
			return nil
		}
		pos := rc.fset.Position(t.Pos())
		return []domain.Reference{r.resolveIdentRef(rc, t.Name, kind, pos, "Type reference to "+t.Name)}
	case *ast.SelectorExpr:
		selPos := rc.fset.Position(t.Sel.Pos())
		if ident, ok := t.X.(*ast.Ident); ok {
			if impPath, ok := rc.importNames[ident.Name]; ok {
				ref := r.resolveImportedSelector(rc, impPath, ident.Name, t.Sel.Name, selPos)
				ref.Kind = kind
				ref.Source.Kind = "type"
				ref.Evidence = []domain.ReferenceEvidence{NewASTTypeReferenceEvidence(rc.filePath, selPos.Line, selPos.Column, "Type reference "+ident.Name+"."+t.Sel.Name)}
				return []domain.Reference{ref}
			}
		}
		return r.resolveTypeExpr(rc, t.X, kind)
	case *ast.StarExpr:
		return r.resolveTypeExpr(rc, t.X, kind)
	case *ast.ArrayType:
		return r.resolveTypeExpr(rc, t.Elt, kind)
	case *ast.MapType:
		return append(r.resolveTypeExpr(rc, t.Key, kind), r.resolveTypeExpr(rc, t.Value, kind)...)
	case *ast.ChanType:
		return r.resolveTypeExpr(rc, t.Value, kind)
	case *ast.FuncType:
		var refs []domain.Reference
		if t.Params != nil {
			for _, f := range t.Params.List {
				refs = append(refs, r.resolveTypeExpr(rc, f.Type, kind)...)
			}
		}
		if t.Results != nil {
			for _, f := range t.Results.List {
				refs = append(refs, r.resolveTypeExpr(rc, f.Type, kind)...)
			}
		}
		return refs
	case *ast.StructType, *ast.InterfaceType:
		return nil
	default:
		return nil
	}
}

func (r *GoReferenceResolver) resolveIdentRef(rc *goResolveContext, name string, kind domain.ReferenceKind, pos token.Position, detail string, localTypes ...map[string]string) domain.Reference {
	ref := domain.Reference{
		Kind:     kind,
		Language: domain.LangGo,
		Location: locFromPos(rc.filePath, pos),
		Source: domain.SymbolRef{
			Name:     name,
			Kind:     string(kind),
			File:     rc.filePath,
			Line:     pos.Line,
			Column:   pos.Column,
			Language: domain.LangGo,
		},
		Evidence: []domain.ReferenceEvidence{evidenceForKind(kind, rc.filePath, pos.Line, pos.Column, detail)},
	}
	if golang.IsBuiltin(name) {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "builtin", Language: domain.LangGo}
		return ref
	}
	if len(localTypes) > 0 && localTypes[0] != nil {
		if _, shadowed := localTypes[0][name]; shadowed && kind == domain.ReferenceCall {
			ref.Resolution = domain.ResolutionDynamic
			ref.Evidence = append(ref.Evidence, NewDynamicDispatchEvidence(rc.filePath, pos.Line, "Call to locally shadowed name "+name))
			return ref
		}
	}
	cands := rc.byName[name]
	// Prefer same-file, then same-package.
	var sameFile, samePkg []Symbol
	for _, c := range cands {
		if c.File == rc.filePath {
			sameFile = append(sameFile, c)
			continue
		}
		if rc.pkgFiles[c.File] {
			samePkg = append(samePkg, c)
		}
	}
	filtered := sameFile
	if len(filtered) == 0 {
		filtered = samePkg
	}
	if len(filtered) == 0 {
		// Could be local/imported in ways we don't model; be conservative.
		filtered = cands
	}
	// For type refs, prefer type symbols.
	if kind == domain.ReferenceType {
		typed := filtered[:0]
		for _, c := range filtered {
			if c.Kind == "type" {
				typed = append(typed, c)
			}
		}
		if len(typed) > 0 {
			filtered = typed
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].File < filtered[j].File || (filtered[i].File == filtered[j].File && filtered[i].Line < filtered[j].Line)
	})
	switch len(filtered) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
	case 1:
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(filtered[0])
		ref.Target = &t
	default:
		ref.Resolution = domain.ResolutionAmbiguous
		for _, c := range filtered {
			ref.Candidates = append(ref.Candidates, symbolToRef(c))
		}
	}
	return ref
}

func evidenceForKind(kind domain.ReferenceKind, file string, line, col int, detail string) domain.ReferenceEvidence {
	switch kind {
	case domain.ReferenceType:
		return NewASTTypeReferenceEvidence(file, line, col, detail)
	case domain.ReferenceMethod:
		return NewASTMethodCallEvidence(file, line, col, detail)
	default:
		return NewASTCallEvidence(file, line, col, detail)
	}
}

func locFromPos(file string, pos token.Position) domain.Location {
	return domain.Location{Path: file, StartLine: pos.Line, StartColumn: pos.Column}
}

func symbolToRef(s Symbol) domain.SymbolRef {
	return domain.SymbolRef{
		Name:      s.Name,
		Kind:      s.Kind,
		File:      s.File,
		Line:      s.Line,
		Column:    s.Column,
		Language:  s.Language,
		Container: s.Container,
		Signature: s.Signature,
	}
}

func strconvUnquote(s string) (string, error) {
	// minimal unquote for import paths
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1], nil
	}
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1], nil
	}
	return "", fmt.Errorf("invalid quoted string")
}

func defaultImportName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	// Handle versioned module paths like v2, v3.
	if len(name) >= 2 && name[0] == 'v' && name[1] >= '0' && name[1] <= '9' && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return name
}

func relToRoot(root, path string) string {
	if path == "" {
		return path
	}
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func localsAtPos(file *ast.File, pos token.Pos) map[string]string {
	// Collect every enclosing function scope (FuncDecl and FuncLit) that contains
	// pos. A name declared in an enclosing scope (a captured local) shadows
	// package-level identifiers just like a local of the innermost scope; merging
	// the scopes prevents a captured name from resolving to an unrelated package
	// symbol. Declaration order inside a function body is not modeled, so the
	// result is deliberately conservative.
	type scope struct {
		body  *ast.BlockStmt
		ftype *ast.FuncType
		recv  *ast.FieldList
	}
	var scopes []scope
	ast.Inspect(file, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil && pos >= fn.Body.Pos() && pos <= fn.Body.End() {
				scopes = append(scopes, scope{body: fn.Body, ftype: fn.Type, recv: fn.Recv})
			}
		case *ast.FuncLit:
			if fn.Body != nil && pos >= fn.Body.Pos() && pos <= fn.Body.End() {
				scopes = append(scopes, scope{body: fn.Body, ftype: fn.Type})
			}
		}
		return true
	})
	if len(scopes) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	// Innermost scope first so an inner declaration wins over an outer one.
	for i := len(scopes) - 1; i >= 0; i-- {
		sc := scopes[i]
		names := map[string]string{}
		add := func(name, typ string) {
			if name == "" || name == "_" {
				return
			}
			if _, exists := names[name]; !exists {
				names[name] = typ
			} else if typ != "" && names[name] == "" {
				names[name] = typ
			}
		}
		addFields := func(fl *ast.FieldList) {
			if fl == nil {
				return
			}
			for _, f := range fl.List {
				tn := simpleLocalTypeName(f.Type)
				for _, n := range f.Names {
					if n == nil {
						continue
					}
					add(n.Name, tn)
				}
			}
		}
		addFields(sc.recv)
		if sc.ftype != nil {
			addFields(sc.ftype.Params)
			addFields(sc.ftype.Results)
		}
		ast.Inspect(sc.body, func(n ast.Node) bool {
			switch st := n.(type) {
			case *ast.FuncLit:
				// Declarations inside a nested function belong to its own scope.
				if st != nil && st.Body != sc.body {
					return false
				}
			case *ast.DeclStmt:
				gd, ok := st.Decl.(*ast.GenDecl)
				if !ok {
					return true
				}
				for _, sp := range gd.Specs {
					vs, ok := sp.(*ast.ValueSpec)
					if !ok {
						continue
					}
					tn := ""
					if vs.Type != nil {
						tn = simpleLocalTypeName(vs.Type)
					}
					for _, n := range vs.Names {
						if n != nil {
							add(n.Name, tn)
						}
					}
				}
			case *ast.AssignStmt:
				if st.Tok != token.DEFINE {
					return true
				}
				for i, lhs := range st.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					tn := ""
					if i < len(st.Rhs) {
						tn = typeNameFromRHS(st.Rhs[i])
					}
					add(id.Name, tn)
				}
			}
			return true
		})
		for name, typ := range names {
			if _, exists := out[name]; !exists {
				out[name] = typ
			}
		}
	}
	return out
}

func simpleLocalTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "_" {
			return ""
		}
		return t.Name
	case *ast.StarExpr:
		return simpleLocalTypeName(t.X)
	default:
		return ""
	}
}

func typeNameFromRHS(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.CompositeLit:
		return simpleLocalTypeName(t.Type)
	case *ast.UnaryExpr:
		if t.Op.String() == "&" {
			if cl, ok := t.X.(*ast.CompositeLit); ok {
				return simpleLocalTypeName(cl.Type)
			}
		}
	}
	return ""
}
