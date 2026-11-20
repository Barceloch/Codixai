package index

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/lexical"
	"github.com/barceloch/codixai/internal/languages/python"
)

// PythonReferenceResolver resolves Python references conservatively using
// segment-aware lexical analysis and the structural index.
//
// The resolver never executes the analyzed repository. It recognizes imports,
// calls, class/type references, inheritance and selected attribute accesses,
// and keeps unresolved, ambiguous, external or dynamic cases explicit.
type PythonReferenceResolver struct{}

// NewPythonReferenceResolver creates a new Python reference resolver.
func NewPythonReferenceResolver() *PythonReferenceResolver {
	return &PythonReferenceResolver{}
}

// Resolve implements ReferenceResolver for Python.
func (r *PythonReferenceResolver) Resolve(ctx context.Context, idx *Index, changes *domain.ChangeSet) ([]domain.Reference, error) {
	if changes == nil {
		return nil, nil
	}
	root, err := filepath.Abs(changes.Root)
	if err != nil {
		root = changes.Root
	}
	var all []domain.Reference
	for _, fc := range changes.Files {
		if fc.Language != domain.LangPython {
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

type pySymbolBinding struct {
	// module is the dotted module path of the binding.
	module string
	// name is the imported symbol name; empty means the binding is a module.
	name string
	// relative reports whether the binding came from a relative import.
	relative bool
}

type pyScope struct {
	kind   string // "class" or "func"
	name   string
	indent int
	locals map[string]string
}

type pyResolveContext struct {
	root     string
	filePath string

	symbols     []Symbol
	byFile      map[string][]Symbol
	byName      map[string][]Symbol
	moduleFiles map[string]string
	fileModules map[string]string

	bindings map[string]pySymbolBinding
	wildcard []string

	scopes []pyScope
	refs   []domain.Reference
}

func (r *PythonReferenceResolver) resolveFile(root, filePath string, src []byte, idx *Index) ([]domain.Reference, error) {
	rc := &pyResolveContext{
		root:        root,
		filePath:    filePath,
		byFile:      map[string][]Symbol{},
		byName:      map[string][]Symbol{},
		moduleFiles: map[string]string{},
		fileModules: map[string]string{},
		bindings:    map[string]pySymbolBinding{},
	}
	if idx != nil {
		for _, s := range idx.Symbols() {
			if s.Language != domain.LangPython {
				continue
			}
			s.File = relToRoot(root, s.File)
			rc.symbols = append(rc.symbols, s)
			rc.byFile[s.File] = append(rc.byFile[s.File], s)
			rc.byName[s.Name] = append(rc.byName[s.Name], s)
		}
	}
	rc.buildModuleMap()
	rc.scan(filePath, src)
	return rc.refs, nil
}

func (rc *pyResolveContext) buildModuleMap() {
	for _, s := range rc.symbols {
		if _, ok := rc.fileModules[s.File]; ok {
			continue
		}
		module := pyModuleFromPath(s.File)
		rc.fileModules[s.File] = module
		rc.moduleFiles[module] = s.File
	}
	// Always map the file being analyzed, even when the index is empty.
	module := pyModuleFromPath(rc.filePath)
	if _, ok := rc.moduleFiles[module]; !ok {
		rc.moduleFiles[module] = rc.filePath
	}
	if _, ok := rc.fileModules[rc.filePath]; !ok {
		rc.fileModules[rc.filePath] = module
	}
}

var (
	pyClassRe    = regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\((.*)\))?\s*:`)
	pyDefRe      = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	pyImportRe   = regexp.MustCompile(`^import\s+(.+)$`)
	pyFromRe     = regexp.MustCompile(`^from\s*([.]*(?:[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)?|[.]+)\s+import\s+(.+)$`)
	pyCallRe     = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*\(`)
	pyChainRe    = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+`)
	pySelfAttrRe = regexp.MustCompile(`\bself\.([A-Za-z_][A-Za-z0-9_]*)`)
	pyAssignRe   = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+)$`)
	pyAnnotateRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.+)$`)
	pyTypeNameRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*`)
	// pyComprehensionForRe finds the target list of a comprehension/for clause.
	pyComprehensionForRe = regexp.MustCompile(`\bfor\s*\(?\s*([A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*)\s*\)?\s+in\b`)
)

func (rc *pyResolveContext) scan(filename string, src []byte) {
	scanner := lexical.NewScanner(python.Profile())
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	for i, text := range raw {
		line := i + 1
		code := strings.TrimRight(lexical.CodeText(scanner.Scan(text)), " \t")
		trimmed := strings.TrimSpace(code)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(code) - len(strings.TrimLeft(code, " \t"))
		rc.popScopes(indent)
		if strings.HasPrefix(trimmed, "@") {
			continue
		}
		if m := pyClassRe.FindStringSubmatch(trimmed); m != nil {
			rc.handleClass(line, indent, m[1], m[2])
			continue
		}
		if m := pyDefRe.FindStringSubmatch(trimmed); m != nil {
			rc.handleDef(line, indent, m[1], m[2], trimmed)
			continue
		}
		if m := pyFromRe.FindStringSubmatch(trimmed); m != nil {
			rc.handleFromImport(line, m[1], m[2])
			continue
		}
		if m := pyImportRe.FindStringSubmatch(trimmed); m != nil {
			rc.handleImport(line, m[1])
			continue
		}
		rc.handleBodyLine(line, indent, trimmed)
	}
}

func (rc *pyResolveContext) popScopes(indent int) {
	for len(rc.scopes) > 0 {
		top := rc.scopes[len(rc.scopes)-1]
		if indent > top.indent {
			return
		}
		rc.scopes = rc.scopes[:len(rc.scopes)-1]
	}
}

func (rc *pyResolveContext) pushScope(scope pyScope) {
	if scope.locals == nil {
		scope.locals = map[string]string{}
	}
	rc.scopes = append(rc.scopes, scope)
}

func (rc *pyResolveContext) currentClass() string {
	var names []string
	for _, scope := range rc.scopes {
		if scope.kind == "class" {
			names = append(names, scope.name)
		}
	}
	return strings.Join(names, ".")
}

func (rc *pyResolveContext) lookupLocal(name string) (string, bool) {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if t, ok := rc.scopes[i].locals[name]; ok && t != "" {
			return t, true
		}
	}
	return "", false
}

func (rc *pyResolveContext) hasLocal(name string) bool {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if _, ok := rc.scopes[i].locals[name]; ok {
			return true
		}
	}
	return false
}

func (rc *pyResolveContext) setLocal(name, typ string) {
	if name == "" {
		return
	}
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if rc.scopes[i].kind == "func" {
			rc.scopes[i].locals[name] = typ
			return
		}
	}
	if len(rc.scopes) > 0 {
		rc.scopes[len(rc.scopes)-1].locals[name] = typ
	}
}

func (rc *pyResolveContext) handleClass(line, indent int, name, bases string) {
	rc.pushScope(pyScope{kind: "class", name: name, indent: indent})
	if strings.TrimSpace(bases) == "" {
		return
	}
	for _, base := range pySplitTopLevel(bases, ',') {
		base = strings.TrimSpace(base)
		if base == "" || strings.Contains(base, "=") {
			continue
		}
		rc.emitTypeExpr(base, line, 1, domain.ReferenceInheritance, "Inheritance base ")
	}
}

func (rc *pyResolveContext) handleDef(line, indent int, name, params, trimmed string) {
	locals := map[string]string{}
	for _, rawParam := range pySplitTopLevel(params, ',') {
		param := strings.TrimSpace(rawParam)
		param = strings.TrimPrefix(param, "**")
		param = strings.TrimPrefix(param, "*")
		if param == "" {
			continue
		}
		// Strip default value for name extraction.
		namePart := param
		if eq := strings.Index(namePart, "="); eq >= 0 {
			namePart = strings.TrimSpace(namePart[:eq])
		}
		localName := namePart
		annotation := ""
		if colon := strings.Index(namePart, ":"); colon >= 0 {
			localName = strings.TrimSpace(namePart[:colon])
			annotation = strings.TrimSpace(namePart[colon+1:])
		}
		if localName == "" || localName == "self" || localName == "cls" {
			if localName == "" {
				continue
			}
		}
		typ := ""
		if annotation != "" {
			typ = pySimpleTypeName(annotation)
			rc.emitTypeExpr(annotation, line, 1, domain.ReferenceType, "Type annotation "+localName+": ")
		}
		// Register parameter as local (even without annotation) to preserve shadowing.
		if _, exists := locals[localName]; !exists {
			locals[localName] = typ
		}
	}
	if idx := strings.Index(trimmed, "->"); idx >= 0 {
		ret := trimmed[idx+2:]
		if colon := strings.LastIndex(ret, ":"); colon >= 0 {
			ret = ret[:colon]
		}
		ret = strings.TrimSpace(ret)
		if ret != "" {
			rc.emitTypeExpr(ret, line, 1, domain.ReferenceType, "Return type annotation ")
		}
	}
	rc.pushScope(pyScope{kind: "func", name: name, indent: indent, locals: locals})
}

func (rc *pyResolveContext) handleImport(line int, spec string) {
	for _, part := range pySplitTopLevel(spec, ',') {
		mod, alias := pyParseImportAlias(part)
		if mod == "" {
			continue
		}
		local := alias
		bindModule := mod
		if local == "" {
			local = strings.Split(mod, ".")[0]
			bindModule = local
		}
		rc.bindings[local] = pySymbolBinding{module: bindModule}
		rc.emitImportRef(python.Import{Module: mod, Alias: alias, Line: line})
	}
}

func (rc *pyResolveContext) handleFromImport(line int, modulePart, namesPart string) {
	level := 0
	for level < len(modulePart) && modulePart[level] == '.' {
		level++
	}
	mod := strings.TrimPrefix(modulePart, strings.Repeat(".", level))
	target := rc.targetModule(mod, level)
	for _, name := range pyParseFromNames(namesPart) {
		if name.Name == "*" {
			rc.wildcard = append(rc.wildcard, target)
			rc.emitImportRef(python.Import{Module: mod, Name: "*", Relative: level > 0, Level: level, Line: line})
			continue
		}
		local := name.Name
		if name.Alias != "" {
			local = name.Alias
		}
		sub := name.Name
		if target != "" {
			sub = target + "." + name.Name
		}
		if _, ok := rc.moduleFiles[sub]; ok {
			rc.bindings[local] = pySymbolBinding{module: sub}
		} else {
			rc.bindings[local] = pySymbolBinding{module: target, name: name.Name, relative: level > 0}
		}
		rc.emitImportRef(python.Import{Module: mod, Name: name.Name, Alias: name.Alias, Relative: level > 0, Level: level, Line: line})
	}
}

func (rc *pyResolveContext) targetModule(mod string, level int) string {
	if level == 0 {
		return mod
	}
	base := pyRelativeBase(rc.filePath, level)
	if mod == "" {
		return base
	}
	if base == "" {
		return mod
	}
	return base + "." + mod
}

func (rc *pyResolveContext) handleBodyLine(line, indent int, trimmed string) {
	if m := pyAnnotateRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		annotation := strings.TrimSpace(m[2])
		if eq := strings.Index(annotation, "="); eq >= 0 {
			annotation = strings.TrimSpace(annotation[:eq])
		}
		rc.setLocal(name, pySimpleTypeName(annotation))
		rc.emitTypeExpr(annotation, line, 1, domain.ReferenceType, "Type annotation "+name+": ")
		rc.scanCalls(line, trimmed)
		return
	}
	if m := pyAssignRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		rhs := strings.TrimSpace(m[2])
		typ := ""
		if call := pyCallRe.FindStringSubmatch(rhs); call != nil && call[1] != "" {
			typ = call[1]
		}
		rc.setLocal(name, typ)
	}
	rc.scanCalls(line, trimmed)
	rc.scanSelfAttributes(line, trimmed)
	rc.scanModuleAttributes(line, trimmed)
}

// pyComprehensionTargets returns the names bound by comprehension/for clauses
// on a single line and the byte offset of the first "for". Python 3
// comprehension targets are local to the comprehension, so a call through a
// target must not resolve to a module-level symbol of the same name.
func pyComprehensionTargets(line string) (map[string]bool, int) {
	matches := pyComprehensionForRe.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return nil, -1
	}
	out := map[string]bool{}
	for _, m := range matches {
		for _, name := range strings.Split(line[m[2]:m[3]], ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				out[name] = true
			}
		}
	}
	return out, matches[0][0]
}

func (rc *pyResolveContext) scanCalls(line int, trimmed string) {
	compTargets, compStart := pyComprehensionTargets(trimmed)
	for _, loc := range pyCallRe.FindAllStringSubmatchIndex(trimmed, -1) {
		chain := trimmed[loc[2]:loc[3]]
		if chain == "" || pyKeyword(chain) {
			continue
		}
		if pyIsDefOrClassName(trimmed, chain) {
			continue
		}
		if strings.HasPrefix(chain, "__") && strings.HasSuffix(chain, "__") && !strings.Contains(chain, ".") && chain != "__import__" {
			continue
		}
		col := loc[2] + 1
		rc.resolveCall(chain, line, col, compTargets, compStart)
	}
}

func (rc *pyResolveContext) scanSelfAttributes(line int, trimmed string) {
	for _, loc := range pySelfAttrRe.FindAllStringSubmatchIndex(trimmed, -1) {
		attr := trimmed[loc[2]:loc[3]]
		end := loc[1]
		if end < len(trimmed) && strings.HasPrefix(strings.TrimSpace(trimmed[end:]), "(") {
			continue
		}
		rc.resolveSelfAttribute(attr, line, loc[0]+1)
	}
}

func (rc *pyResolveContext) scanModuleAttributes(line int, trimmed string) {
	for _, loc := range pyChainRe.FindAllStringSubmatchIndex(trimmed, -1) {
		chain := trimmed[loc[0]:loc[1]]
		end := loc[1]
		if end < len(trimmed) && strings.HasPrefix(strings.TrimSpace(trimmed[end:]), "(") {
			continue
		}
		parts := strings.Split(chain, ".")
		if rc.hasLocal(parts[0]) {
			if typ, ok := rc.lookupLocal(parts[0]); ok && typ != "" && len(parts) == 2 {
				rc.resolveTypeMember(typ, parts[1], domain.ReferenceField, line, loc[0]+1, "Attribute access on "+typ+"."+parts[1])
			}
			continue
		}
		if binding, ok := rc.bindings[parts[0]]; ok && binding.name == "" {
			rc.resolveModuleChain(binding.module, parts[1:], domain.ReferenceField, line, loc[0]+1)
			continue
		}
	}
}

func (rc *pyResolveContext) resolveCall(chain string, line, col int, compTargets map[string]bool, compStart int) {
	if pyDynamicCall(chain) {
		ref := rc.baseRef(chain, domain.ReferenceCall, line, col)
		ref.Resolution = domain.ResolutionDynamic
		ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "Dynamic call: "+chain)}
		rc.refs = append(rc.refs, ref)
		return
	}
	parts := strings.Split(chain, ".")
	// Comprehension targets are visible in the whole comprehension expression,
	// including the output clause that precedes the "for". The line-oriented
	// model applies the target to the whole line; a preceding unrelated call on
	// the same line would only be over-abstained, never falsely resolved.
	if compTargets != nil && compTargets[parts[0]] {
		if len(parts) == 1 {
			ref := rc.baseRef(chain, domain.ReferenceCall, line, col)
			ref.Resolution = domain.ResolutionDynamic
			ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "Call through comprehension variable: "+chain)}
			rc.refs = append(rc.refs, ref)
			return
		}
		rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "Method call on comprehension variable "+parts[0])
		return
	}
	if len(parts) == 1 {
		if rc.hasLocal(parts[0]) {
			ref := rc.baseRef(chain, domain.ReferenceCall, line, col)
			ref.Resolution = domain.ResolutionDynamic
			ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "Call through local variable: "+chain)}
			rc.refs = append(rc.refs, ref)
			return
		}
		if binding, ok := rc.bindings[parts[0]]; ok {
			if binding.name == "" {
				ref := rc.baseRef(chain, domain.ReferenceCall, line, col)
				ref.Resolution = domain.ResolutionDynamic
				ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "Call through module object: "+chain)}
				rc.refs = append(rc.refs, ref)
				return
			}
			rc.resolveModuleSymbol(binding.module, binding.name, chain, domain.ReferenceCall, line, col, chain, binding.relative)
			return
		}
		rc.resolveSimpleName(parts[0], domain.ReferenceCall, line, col, "Function call "+parts[0])
		return
	}

	receiver := parts[0]
	if (receiver == "self" || receiver == "cls") && rc.currentClass() != "" {
		if len(parts) == 2 {
			rc.resolveClassMember(rc.currentClass(), parts[1], domain.ReferenceMethod, line, col, "Method call self."+parts[1])
			return
		}
		rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "Method call through unknown attribute chain "+chain)
		return
	}
	if rc.hasLocal(receiver) {
		if typ, ok := rc.lookupLocal(receiver); ok && typ != "" {
			if len(parts) == 2 {
				rc.resolveTypeMember(typ, parts[1], domain.ReferenceMethod, line, col, "Method call on "+typ+"."+parts[1])
				return
			}
			rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "Method call through unknown attribute chain "+chain)
			return
		}
		rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "Method call on local variable "+receiver)
		return
	}
	if binding, ok := rc.bindings[receiver]; ok && binding.name == "" {
		rc.resolveModuleChain(binding.module, parts[1:], domain.ReferenceCall, line, col)
		return
	}
	if len(parts) == 2 {
		if typeSym, ok := rc.resolveTypeSymbol(receiver); ok {
			rc.resolveTypeMemberSymbol(typeSym, parts[1], domain.ReferenceMethod, line, col, "Method call on "+receiver+"."+parts[1])
			return
		}
	}
	rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "Method call on unresolved receiver "+receiver+" ("+chain+")")
}

func (rc *pyResolveContext) resolveSelfAttribute(attr string, line, col int) {
	class := rc.currentClass()
	if class == "" {
		rc.emitDynamic(attr, domain.ReferenceField, line, col, "Attribute access self."+attr+" outside a class")
		return
	}
	rc.resolveClassMember(class, attr, domain.ReferenceField, line, col, "Attribute access self."+attr)
}

func (rc *pyResolveContext) resolveModuleChain(modulePath string, rest []string, kind domain.ReferenceKind, line, col int) {
	if len(rest) == 0 {
		return
	}
	if !rc.moduleKnown(modulePath) {
		// Unknown module root (stdlib or third-party): classify the full chain as external.
		name := rest[len(rest)-1]
		container := modulePath
		if len(rest) > 1 {
			container = modulePath + "." + strings.Join(rest[:len(rest)-1], ".")
		}
		ref := rc.baseRef(name, kind, line, col)
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "symbol", Language: domain.LangPython, Container: container}
		ref.Evidence = []domain.ReferenceEvidence{NewModuleResolutionEvidence("External module chain " + modulePath + "." + strings.Join(rest, "."))}
		rc.refs = append(rc.refs, ref)
		return
	}
	cur := modulePath
	for i := 0; i < len(rest); i++ {
		candidate := cur + "." + rest[i]
		if _, ok := rc.moduleFiles[candidate]; ok {
			cur = candidate
			continue
		}
		cands := rc.moduleSymbols(cur, rest[i])
		if len(cands) == 0 {
			if !rc.moduleKnown(cur) {
				name := rest[len(rest)-1]
				ref := rc.baseRef(name, kind, line, col)
				ref.Resolution = domain.ResolutionExternal
				ref.Target = &domain.SymbolRef{Name: name, Kind: "symbol", Language: domain.LangPython, Container: cur}
				ref.Evidence = []domain.ReferenceEvidence{NewModuleResolutionEvidence("External module chain " + cur + "." + strings.Join(rest[i:], "."))}
				rc.refs = append(rc.refs, ref)
				return
			}
			rc.resolveModuleSymbol(cur, rest[i], rest[i], kind, line, col, strings.Join(rest, "."), false)
			return
		}
		if i+1 < len(rest) {
			// Attribute access on a symbol: only a class member can be resolved.
			if len(cands) == 1 && cands[0].Kind == "type" {
				rc.resolveTypeMemberSymbol(&cands[0], rest[i+1], kind, line, col, "Member access on "+cands[0].Name+"."+rest[i+1])
				return
			}
			rc.emitDynamic(rest[len(rest)-1], kind, line, col, "Attribute access on unresolved symbol chain "+strings.Join(rest, "."))
			return
		}
		rc.resolveModuleSymbol(cur, rest[i], rest[i], kind, line, col, strings.Join(rest, "."), false)
		return
	}
	// The chain resolves to the module itself.
	ref := rc.baseRef(modulePath, domain.ReferenceImport, line, col)
	if file, ok := rc.moduleFiles[modulePath]; ok {
		ref.Resolution = domain.ResolutionResolved
		ref.Target = &domain.SymbolRef{Name: pyModuleFromPath(file), Kind: "module", File: file, Language: domain.LangPython}
	} else {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: modulePath, Kind: "module", Language: domain.LangPython, Container: modulePath}
	}
	ref.Evidence = []domain.ReferenceEvidence{NewModuleResolutionEvidence("Module chain resolved to " + modulePath)}
	rc.refs = append(rc.refs, ref)
}

func (rc *pyResolveContext) resolveSimpleName(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{pyEvidenceForKind(kind, rc.filePath, line, col, detail)}
	if pyBuiltin(name) {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "builtin", Language: domain.LangPython}
		rc.refs = append(rc.refs, ref)
		return
	}
	if binding, ok := rc.bindings[name]; ok {
		if binding.name != "" {
			rc.resolveModuleSymbol(binding.module, binding.name, name, kind, line, col, detail, binding.relative)
			return
		}
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: binding.module, Kind: "module", Language: domain.LangPython, Container: binding.module}
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.sameFileSymbols(name, kind)
	switch len(cands) {
	case 0:
		if len(rc.wildcard) > 0 {
			ref.Resolution = domain.ResolutionDynamic
			ref.Evidence = append(ref.Evidence, NewDynamicDispatchEvidence(rc.filePath, line, "Name may come from a wildcard import"))
		} else {
			ref.Resolution = domain.ResolutionUnresolved
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
	rc.refs = append(rc.refs, ref)
}

func (rc *pyResolveContext) resolveModuleSymbol(modulePath, name, sourceName string, kind domain.ReferenceKind, line, col int, detail string, relativeImport bool) {
	if sourceName == "" {
		sourceName = name
	}
	ref := rc.baseRef(sourceName, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{
		NewModuleResolutionEvidence("Resolve " + name + " in module " + modulePath),
		pyEvidenceForKind(kind, rc.filePath, line, col, detail),
	}
	file, ok := rc.moduleFiles[modulePath]
	if !ok {
		// A relative import that does not resolve is a missing local module, not
		// an external dependency: absence of the module must not imply externality.
		if rc.moduleKnown(modulePath) || rc.moduleKnown(pyRootModule(modulePath)) || relativeImport {
			ref.Resolution = domain.ResolutionUnresolved
			reason := "Module " + modulePath + " was not found"
			if relativeImport {
				reason += " (relative import)"
			} else {
				reason += " is a local package but the symbol " + name + " could not be located"
			}
			ref.Evidence = append(ref.Evidence, NewModuleResolutionEvidence(reason))
		} else {
			ref.Resolution = domain.ResolutionExternal
			ref.Target = &domain.SymbolRef{Name: name, Kind: "symbol", Language: domain.LangPython, Container: modulePath}
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.moduleSymbols(modulePath, name)
	filtered := pyFilterKind(cands, kind)
	if len(filtered) == 0 && kind == domain.ReferenceField {
		// A module attribute without a call may be a class reference.
		filtered = cands
		if len(filtered) == 1 && filtered[0].Kind == "type" {
			ref.Kind = domain.ReferenceType
			ref.Source.Kind = string(domain.ReferenceType)
		}
	}
	cands = filtered
	switch len(cands) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewModuleResolutionEvidence("Module "+modulePath+" found at "+file+" but symbol "+name+" was not found"))
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
	rc.refs = append(rc.refs, ref)
}

func (rc *pyResolveContext) moduleKnown(modulePath string) bool {
	if _, ok := rc.moduleFiles[modulePath]; ok {
		return true
	}
	prefix := modulePath + "."
	for module := range rc.moduleFiles {
		if strings.HasPrefix(module, prefix) {
			return true
		}
	}
	return false
}

func (rc *pyResolveContext) moduleSymbols(modulePath, name string) []Symbol {
	file, ok := rc.moduleFiles[modulePath]
	if !ok {
		return nil
	}
	var out []Symbol
	for _, s := range rc.byFile[file] {
		if s.Name != name {
			continue
		}
		if s.Kind == "file" || s.Kind == "module" {
			continue
		}
		out = append(out, s)
	}
	pySortSymbols(out)
	return out
}

func (rc *pyResolveContext) sameFileSymbols(name string, kind domain.ReferenceKind) []Symbol {
	var out []Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Name != name {
			continue
		}
		if s.Kind == "file" || s.Kind == "module" {
			continue
		}
		if s.Container != "" && kind != domain.ReferenceMethod && kind != domain.ReferenceField {
			continue
		}
		out = append(out, s)
	}
	out = pyFilterKind(out, kind)
	pySortSymbols(out)
	return out
}

func (rc *pyResolveContext) resolveClassMember(class, member string, kind domain.ReferenceKind, line, col int, detail string) {
	var classSym *Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Kind == "type" && s.Name == class {
			cp := s
			classSym = &cp
			break
		}
	}
	if classSym == nil {
		rc.emitDynamic(member, kind, line, col, "Class "+class+"."+member+" is not declared in this file")
		return
	}
	rc.resolveTypeMemberSymbol(classSym, member, kind, line, col, detail)
}

func (rc *pyResolveContext) resolveTypeMember(typ, member string, kind domain.ReferenceKind, line, col int, detail string) {
	if sym, ok := rc.resolveTypeSymbol(typ); ok {
		rc.resolveTypeMemberSymbol(sym, member, kind, line, col, detail)
		return
	}
	rc.emitDynamic(member, kind, line, col, "Receiver type "+typ+" could not be resolved ("+typ+"."+member+")")
}

func (rc *pyResolveContext) resolveTypeMemberSymbol(typeSym *Symbol, member string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(member, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{pyEvidenceForKind(kind, rc.filePath, line, col, detail)}
	var cands []Symbol
	for _, s := range rc.byFile[typeSym.File] {
		if s.Name != member {
			continue
		}
		if s.Container != typeSym.Name {
			continue
		}
		if kind == domain.ReferenceMethod && s.Kind != "method" && s.Kind != "func" {
			continue
		}
		if kind == domain.ReferenceField && s.Kind != "field" && s.Kind != "var" {
			continue
		}
		cands = append(cands, s)
	}
	pySortSymbols(cands)
	switch len(cands) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(member, "No member "+member+" declared in "+typeSym.Name))
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
	rc.refs = append(rc.refs, ref)
}

func (rc *pyResolveContext) resolveTypeSymbol(typ string) (*Symbol, bool) {
	typ = pySimpleTypeName(typ)
	if typ == "" {
		return nil, false
	}
	if strings.Contains(typ, ".") {
		parts := strings.Split(typ, ".")
		binding, ok := rc.bindings[parts[0]]
		if !ok || binding.name != "" {
			return nil, false
		}
		rest := parts[1:]
		cur := binding.module
		for i := 0; i < len(rest); i++ {
			candidate := cur + "." + rest[i]
			if _, ok := rc.moduleFiles[candidate]; ok {
				cur = candidate
				continue
			}
			cands := rc.moduleSymbols(cur, rest[i])
			if len(cands) == 1 && cands[0].Kind == "type" {
				cp := cands[0]
				return &cp, true
			}
			return nil, false
		}
		return nil, false
	}
	if binding, ok := rc.bindings[typ]; ok && binding.name != "" {
		cands := rc.moduleSymbols(binding.module, binding.name)
		if len(cands) == 1 && cands[0].Kind == "type" {
			cp := cands[0]
			return &cp, true
		}
		return nil, false
	}
	for _, s := range rc.byFile[rc.filePath] {
		if s.Kind == "type" && s.Name == typ {
			cp := s
			return &cp, true
		}
	}
	return nil, false
}

func (rc *pyResolveContext) emitTypeExpr(expr string, line, col int, kind domain.ReferenceKind, detail string) {
	for _, name := range pyTypeNames(expr) {
		if pyBuiltin(name) || pyTypingName(name) || name == "None" || name == "True" || name == "False" {
			continue
		}
		if strings.Contains(name, ".") {
			parts := strings.Split(name, ".")
			if binding, ok := rc.bindings[parts[0]]; ok && binding.name == "" {
				rc.resolveModuleChain(binding.module, parts[1:], kind, line, col)
				continue
			}
		}
		rc.resolveSimpleName(name, kind, line, col, detail+name)
	}
}

func (rc *pyResolveContext) emitImportRef(imp python.Import) {
	module := imp.Module
	if imp.Relative {
		module = rc.targetModule(imp.Module, imp.Level)
	}
	ref := domain.Reference{
		Kind:     domain.ReferenceImport,
		Language: domain.LangPython,
		Location: pyLoc(rc.filePath, imp.Line, 1),
		Source: domain.SymbolRef{
			Name:     pyImportSourceName(imp),
			Kind:     "import",
			File:     rc.filePath,
			Line:     imp.Line,
			Column:   1,
			Language: domain.LangPython,
		},
		Evidence: []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, imp.Line, module)},
	}
	if imp.Name == "*" {
		ref.Resolution = domain.ResolutionDynamic
		ref.Evidence = append(ref.Evidence, NewDynamicDispatchEvidence(rc.filePath, imp.Line, "Wildcard import from "+module))
		rc.refs = append(rc.refs, ref)
		return
	}
	targetModule := module
	if imp.Name != "" {
		if _, ok := rc.moduleFiles[module+"."+imp.Name]; ok {
			targetModule = module + "." + imp.Name
			imp.Name = ""
		}
	}
	if file, ok := rc.moduleFiles[targetModule]; ok {
		if imp.Name == "" {
			ref.Resolution = domain.ResolutionResolved
			ref.Target = &domain.SymbolRef{Name: pyModuleFromPath(file), Kind: "module", File: file, Language: domain.LangPython}
			ref.Evidence = append(ref.Evidence, NewModuleResolutionEvidence("Import resolved to "+file))
		} else {
			cands := pyFilterKind(rc.moduleSymbols(targetModule, imp.Name), domain.ReferenceImport)
			if len(cands) == 0 {
				cands = rc.moduleSymbols(targetModule, imp.Name)
			}
			switch len(cands) {
			case 0:
				ref.Resolution = domain.ResolutionUnresolved
				ref.Evidence = append(ref.Evidence, NewModuleResolutionEvidence("Module "+targetModule+" found at "+file+" but symbol "+imp.Name+" was not found"))
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
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	if imp.Relative || rc.moduleKnown(targetModule) || rc.moduleKnown(pyRootModule(targetModule)) {
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewModuleResolutionEvidence("Import target "+targetModule+" was not found in the project"))
	} else {
		ref.Resolution = domain.ResolutionExternal
		targetName := targetModule
		targetKind := "module"
		if imp.Name != "" {
			targetName = imp.Name
			targetKind = "symbol"
		}
		ref.Target = &domain.SymbolRef{Name: targetName, Kind: targetKind, Language: domain.LangPython, Container: targetModule}
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *pyResolveContext) baseRef(name string, kind domain.ReferenceKind, line, col int) domain.Reference {
	return domain.Reference{
		Kind:     kind,
		Language: domain.LangPython,
		Location: pyLoc(rc.filePath, line, col),
		Source: domain.SymbolRef{
			Name:     name,
			Kind:     string(kind),
			File:     rc.filePath,
			Line:     line,
			Column:   col,
			Language: domain.LangPython,
		},
	}
}

func (rc *pyResolveContext) emitDynamic(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Resolution = domain.ResolutionDynamic
	ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, detail)}
	rc.refs = append(rc.refs, ref)
}

func pyEvidenceForKind(kind domain.ReferenceKind, file string, line, col int, detail string) domain.ReferenceEvidence {
	switch kind {
	case domain.ReferenceMethod:
		return NewLexicalMethodCallEvidence(file, line, detail)
	case domain.ReferenceType, domain.ReferenceInheritance:
		return NewLexicalTypeReferenceEvidence(file, line, detail)
	default:
		return NewLexicalCallPatternEvidence(file, line, detail)
	}
}

func pyLoc(file string, line, col int) domain.Location {
	return domain.Location{Path: file, StartLine: line, StartColumn: col}
}

func pyRootModule(module string) string {
	if idx := strings.Index(module, "."); idx >= 0 {
		return module[:idx]
	}
	return module
}

func pyModuleFromPath(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, ".py")
	parts := strings.Split(rel, "/")
	if len(parts) > 0 && parts[len(parts)-1] == "__init__" {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, ".")
}

func pyRelativeBase(fileRel string, level int) string {
	dir := filepath.ToSlash(filepath.Dir(fileRel))
	if dir == "." {
		dir = ""
	}
	var parts []string
	if dir != "" {
		parts = strings.Split(dir, "/")
	}
	up := level - 1
	if up < 0 {
		up = 0
	}
	if up > len(parts) {
		up = len(parts)
	}
	parts = parts[:len(parts)-up]
	return strings.Join(parts, ".")
}

func pySimpleTypeName(expr string) string {
	expr = strings.TrimSpace(expr)
	expr = strings.Trim(expr, "\"'")
	if idx := strings.Index(expr, "["); idx >= 0 {
		expr = expr[:idx]
	}
	if idx := strings.Index(expr, "|"); idx >= 0 {
		expr = expr[:idx]
	}
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	if pyBuiltin(expr) || pyTypingName(expr) {
		return ""
	}
	return expr
}

func pyTypeNames(expr string) []string {
	var out []string
	seen := map[string]bool{}
	for _, name := range pyTypeNameRe.FindAllString(expr, -1) {
		if seen[name] || pyKeyword(name) {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func pyFilterKind(syms []Symbol, kind domain.ReferenceKind) []Symbol {
	if len(syms) == 0 {
		return nil
	}
	var out []Symbol
	for _, s := range syms {
		switch kind {
		case domain.ReferenceCall:
			if s.Kind == "func" || s.Kind == "method" || s.Kind == "type" {
				out = append(out, s)
			}
		case domain.ReferenceType, domain.ReferenceInheritance:
			if s.Kind == "type" {
				out = append(out, s)
			}
		case domain.ReferenceMethod:
			if s.Kind == "method" || s.Kind == "func" {
				out = append(out, s)
			}
		case domain.ReferenceField:
			if s.Kind == "field" || s.Kind == "var" {
				out = append(out, s)
			}
		default:
			out = append(out, s)
		}
	}
	return out
}

func pySortSymbols(syms []Symbol) {
	sort.Slice(syms, func(i, j int) bool {
		if syms[i].File != syms[j].File {
			return syms[i].File < syms[j].File
		}
		if syms[i].Line != syms[j].Line {
			return syms[i].Line < syms[j].Line
		}
		return syms[i].Name < syms[j].Name
	})
}

func pySplitTopLevel(text string, sep byte) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				out = append(out, text[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, text[start:])
	return out
}

func pyParseImportAlias(part string) (string, string) {
	part = strings.TrimSpace(part)
	if idx := strings.Index(part, "#"); idx >= 0 {
		part = strings.TrimSpace(part[:idx])
	}
	fields := strings.Fields(part)
	if len(fields) == 0 {
		return "", ""
	}
	mod := fields[0]
	alias := ""
	if len(fields) >= 3 && fields[1] == "as" {
		alias = fields[2]
	}
	return mod, alias
}

type pyImportName struct {
	Name  string
	Alias string
}

func pyParseFromNames(namesPart string) []pyImportName {
	if idx := strings.Index(namesPart, "#"); idx >= 0 {
		namesPart = namesPart[:idx]
	}
	namesPart = strings.TrimSpace(namesPart)
	if strings.HasPrefix(namesPart, "(") {
		namesPart = strings.TrimSuffix(strings.TrimPrefix(namesPart, "("), ")")
	}
	if namesPart == "*" {
		return []pyImportName{{Name: "*"}}
	}
	var out []pyImportName
	for _, part := range strings.Split(namesPart, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		alias := ""
		if len(fields) >= 3 && fields[1] == "as" {
			alias = fields[2]
		}
		if name != "" {
			out = append(out, pyImportName{Name: name, Alias: alias})
		}
	}
	return out
}

func pyImportSourceName(imp python.Import) string {
	module := imp.Module
	if imp.Relative {
		module = strings.Repeat(".", imp.Level) + module
	}
	if imp.Name != "" && imp.Name != "*" {
		return module + "." + imp.Name
	}
	if imp.Name == "*" {
		return module + ".*"
	}
	return module
}

func pyIsDefOrClassName(trimmed, chain string) bool {
	if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") || strings.HasPrefix(trimmed, "class ") {
		if idx := strings.Index(trimmed, chain); idx >= 0 {
			return true
		}
	}
	return false
}

func pyDynamicCall(chain string) bool {
	switch chain {
	case "getattr", "setattr", "delattr", "eval", "exec", "compile", "__import__", "globals", "locals", "vars":
		return true
	}
	return chain == "importlib.import_module" || strings.HasSuffix(chain, ".import_module")
}

func pyKeyword(name string) bool {
	switch name {
	case "False", "None", "True", "and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield", "match", "case":
		return true
	}
	return false
}

var pyBuiltins = map[string]bool{
	"abs": true, "all": true, "any": true, "ascii": true, "bin": true, "bool": true,
	"breakpoint": true, "bytearray": true, "bytes": true, "callable": true, "chr": true,
	"classmethod": true, "compile": true, "complex": true, "delattr": true, "dict": true,
	"dir": true, "divmod": true, "enumerate": true, "eval": true, "exec": true,
	"filter": true, "float": true, "format": true, "frozenset": true, "getattr": true,
	"globals": true, "hasattr": true, "hash": true, "help": true, "hex": true, "id": true,
	"input": true, "int": true, "isinstance": true, "issubclass": true, "iter": true,
	"len": true, "list": true, "locals": true, "map": true, "max": true, "memoryview": true,
	"min": true, "next": true, "object": true, "oct": true, "open": true, "ord": true,
	"pow": true, "print": true, "property": true, "range": true, "repr": true,
	"reversed": true, "round": true, "set": true, "setattr": true, "slice": true,
	"sorted": true, "staticmethod": true, "str": true, "sum": true, "super": true,
	"tuple": true, "type": true, "vars": true, "zip": true, "__import__": true,
	"Exception": true, "BaseException": true, "ValueError": true, "TypeError": true,
	"KeyError": true, "IndexError": true, "RuntimeError": true, "OSError": true,
	"NotImplementedError": true, "AttributeError": true, "ImportError": true,
}

func pyBuiltin(name string) bool {
	return pyBuiltins[name]
}

var pyTypingNames = map[string]bool{
	"Any": true, "Optional": true, "Union": true, "List": true, "Dict": true, "Set": true,
	"Tuple": true, "FrozenSet": true, "Callable": true, "Iterable": true, "Iterator": true,
	"Sequence": true, "Mapping": true, "MutableMapping": true, "MutableSequence": true,
	"Type": true, "ClassVar": true, "Final": true, "Literal": true, "Protocol": true,
	"Generic": true, "TypeVar": true, "Annotated": true, "Awaitable": true,
	"Coroutine": true, "AsyncIterator": true, "AsyncIterable": true, "NamedTuple": true,
	"TypedDict": true, "Self": true, "Never": true, "NoReturn": true, "overload": true,
}

func pyTypingName(name string) bool {
	return pyTypingNames[name]
}
