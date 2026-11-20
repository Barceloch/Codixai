package index

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/javascript"
	"github.com/barceloch/codixai/internal/languages/lexical"
)

// JavaScriptReferenceResolver resolves JavaScript references conservatively
// using segment-aware lexical analysis, the structural index, and the JS/TS
// module graph.
//
// The resolver never executes the analyzed repository. It resolves imports,
// local declarations, selected inter-module bindings, calls/constructors, and
// conservative member accesses. Unresolved, ambiguous, external, and dynamic
// cases remain explicit.
type JavaScriptReferenceResolver struct{}

// NewJavaScriptReferenceResolver creates a new JavaScript reference resolver.
func NewJavaScriptReferenceResolver() *JavaScriptReferenceResolver {
	return &JavaScriptReferenceResolver{}
}

// Resolve implements ReferenceResolver for JavaScript.
func (r *JavaScriptReferenceResolver) Resolve(ctx context.Context, idx *Index, changes *domain.ChangeSet) ([]domain.Reference, error) {
	if changes == nil {
		return nil, nil
	}
	root, err := filepath.Abs(changes.Root)
	if err != nil {
		root = changes.Root
	}
	root = filepath.Clean(root)

	var modGraph *ModuleGraph
	if idx != nil {
		modGraph, _ = NewJSModuleResolver().ResolveModules(ctx, idx, root)
	}
	if modGraph == nil {
		modGraph = &ModuleGraph{Root: filepath.ToSlash(root)}
	}

	var all []domain.Reference
	for _, fc := range changes.Files {
		if fc.Language != domain.LangJavaScript {
			continue
		}
		if fc.Status == domain.FileDeleted {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src, err := os.ReadFile(filepath.Join(root, fc.Path))
		if err != nil {
			continue
		}
		refs, err := r.resolveFile(ctx, root, filepath.ToSlash(fc.Path), src, idx, modGraph)
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

type jsBinding struct {
	local        string
	moduleSpec   string
	importedName string // exported name in target module; "*" for namespace; "default" for default
	kind         string // "named", "default", "namespace", "require", "require_destructured"
	line         int
}

type jsExportInfo struct {
	defaultSymbol string            // local/exported symbol name for default export, or ""
	named         map[string]string // exported name -> local/source name
	reexports     []jsReexport
	cjs           bool
}

type jsReexport struct {
	exportedName string // "*" for export * from
	localName    string
	moduleSpec   string
	line         int
}

type jsScopeKind string

const (
	jsScopeFile  jsScopeKind = "file"
	jsScopeFunc  jsScopeKind = "func"
	jsScopeClass jsScopeKind = "class"
	jsScopeBlock jsScopeKind = "block"
)

type jsScope struct {
	kind   jsScopeKind
	name   string // class/function name
	indent int
	locals map[string]bool
	types  map[string]string // local variable -> known constructor/type name
}

type jsResolveContext struct {
	root     string
	filePath string

	idx      *Index
	modGraph *ModuleGraph

	symbols     []Symbol
	byFile      map[string][]Symbol
	byName      map[string][]Symbol
	fileModules map[string]bool

	bindings map[string]jsBinding
	exports  *jsExportInfo

	scopes []jsScope
	refs   []domain.Reference

	reexportStack map[string]bool
	exportCache   map[string]*jsExportInfo
}

func (r *JavaScriptReferenceResolver) resolveFile(ctx context.Context, root, filePath string, src []byte, idx *Index, modGraph *ModuleGraph) ([]domain.Reference, error) {
	_ = ctx
	rc := &jsResolveContext{
		root:          root,
		filePath:      filePath,
		idx:           idx,
		modGraph:      modGraph,
		byFile:        map[string][]Symbol{},
		byName:        map[string][]Symbol{},
		fileModules:   map[string]bool{},
		bindings:      map[string]jsBinding{},
		exportCache:   map[string]*jsExportInfo{},
		reexportStack: map[string]bool{},
	}
	if idx != nil {
		for _, s := range idx.Symbols() {
			cp := s
			cp.File = relToRoot(root, s.File)
			rc.symbols = append(rc.symbols, cp)
			rc.byFile[cp.File] = append(rc.byFile[cp.File], cp)
			rc.byName[cp.Name] = append(rc.byName[cp.Name], cp)
		}
	}
	rc.fileModules[filePath] = true
	rc.pushScope(jsScope{kind: jsScopeFile, indent: -1})

	scanner := lexical.NewScanner(javascript.Profile())
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	codeLines := make([]string, len(raw))
	segLines := make([][]lexical.Segment, len(raw))
	for i, text := range raw {
		segs := scanner.Scan(text)
		segLines[i] = segs
		codeLines[i] = lexical.CodeText(segs)
	}
	rc.exports = rc.parseModuleBindingsAndExports(codeLines, segLines, filePath)
	rc.scanOccurrences(codeLines, segLines)
	return rc.refs, nil
}

func (rc *jsResolveContext) pushScope(s jsScope) {
	if s.locals == nil {
		s.locals = map[string]bool{}
	}
	if s.types == nil {
		s.types = map[string]string{}
	}
	rc.scopes = append(rc.scopes, s)
}

func (rc *jsResolveContext) popScopes(indent int, trimmed string) {
	for len(rc.scopes) > 1 {
		top := rc.scopes[len(rc.scopes)-1]
		if indent > top.indent {
			return
		}
		if strings.HasPrefix(trimmed, "}") || trimmed == "" {
			rc.scopes = rc.scopes[:len(rc.scopes)-1]
			continue
		}
		// Keep scope when line starts a sibling at same indent only in limited cases.
		if indent == top.indent && (strings.HasPrefix(trimmed, "else") || strings.HasPrefix(trimmed, "catch") || strings.HasPrefix(trimmed, "finally")) {
			return
		}
		if indent < top.indent || (indent == top.indent && top.kind != jsScopeFile) {
			rc.scopes = rc.scopes[:len(rc.scopes)-1]
			continue
		}
		return
	}
}

func (rc *jsResolveContext) currentClass() string {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if rc.scopes[i].kind == jsScopeClass && rc.scopes[i].name != "" {
			return rc.scopes[i].name
		}
	}
	return ""
}

func (rc *jsResolveContext) hasLocal(name string) bool {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if rc.scopes[i].locals[name] {
			return true
		}
	}
	return false
}

func (rc *jsResolveContext) lookupLocalType(name string) (string, bool) {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if t, ok := rc.scopes[i].types[name]; ok && t != "" {
			return t, true
		}
	}
	return "", false
}

func (rc *jsResolveContext) setLocal(name string) {
	if name == "" || len(rc.scopes) == 0 {
		return
	}
	// Top-level declarations are resolved via index symbols; only inner-scope
	// declarations create locals that can shadow references.
	if rc.scopes[len(rc.scopes)-1].kind == jsScopeFile {
		return
	}
	rc.scopes[len(rc.scopes)-1].locals[name] = true
}

func (rc *jsResolveContext) setLocalType(name, typ string) {
	if name == "" || typ == "" || len(rc.scopes) == 0 {
		return
	}
	if rc.scopes[len(rc.scopes)-1].kind == jsScopeFile {
		return
	}
	rc.scopes[len(rc.scopes)-1].locals[name] = true
	rc.scopes[len(rc.scopes)-1].types[name] = typ
}

func (rc *jsResolveContext) localDeclaredBefore(name string) bool {
	// Conservative: top-level declarations and binding names are visible unless shadowed in inner scopes.
	if rc.hasLocal(name) {
		return true
	}
	return false
}

var (
	jsImportLineRe     = regexp.MustCompile(`^\s*import\s+(.+?)\s+from\s+["'\x60]([^"'\x60]+)["'\x60]`)
	jsImportSideRe     = regexp.MustCompile(`^\s*import\s+["'\x60]([^"'\x60]+)["'\x60]`)
	jsExportFromRe     = regexp.MustCompile(`^\s*export\s+(?:type\s+)?(\*|\{[^}]*\})\s*(?:as\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*)?from\s+["'\x60]([^"'\x60]+)["'\x60]`)
	jsExportNamedRe    = regexp.MustCompile(`^\s*export\s+(?:type\s+)?\{([^}]*)\}\s*;?\s*$`)
	jsExportDeclRe     = regexp.MustCompile(`^\s*export\s+(?:default\s+)?(?:async\s+)?(?:function\s*\*?\s*|class\s+|const\s+|let\s+|var\s+)([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsExportDefFuncRe  = regexp.MustCompile(`^\s*export\s+default\s+(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsExportDefClassRe = regexp.MustCompile(`^\s*export\s+default\s+class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsExportDefIdentRe = regexp.MustCompile(`^\s*export\s+default\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*;?\s*$`)
	jsRequireBindRe    = regexp.MustCompile(`^\s*(?:const|let|var)\s+(.+?)\s*=\s*require\s*\(\s*["'\x60]([^"'\x60]+)["'\x60]\s*\)`)
	jsClassDeclRe      = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsFuncDeclRe       = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsVarDeclRe        = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?::[^=]+)?=\s*(.+)$`)
	jsNewAssignRe      = regexp.MustCompile(`^new\s+([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*)\s*\(`)
	jsParamIdentRe     = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)
)

func (rc *jsResolveContext) moduleFromSegments(segs []lexical.Segment) string {
	var mod string
	for _, seg := range segs {
		if seg.Kind != lexical.KindString {
			continue
		}
		text := strings.Trim(seg.Text, "\"'`")
		if text == "" {
			continue
		}
		mod = text
	}
	return mod
}

func (rc *jsResolveContext) parseModuleBindingsAndExports(codeLines []string, segLines [][]lexical.Segment, filePath string) *jsExportInfo {
	info := &jsExportInfo{named: map[string]string{}}
	for i, code := range codeLines {
		line := i + 1
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		segs := segLines[i]
		if strings.HasPrefix(trimmed, "export") && strings.Contains(trimmed, "from") {
			spec := rc.moduleFromSegments(segs)
			if spec != "" {
				if strings.Contains(trimmed, "export *") {
					info.reexports = append(info.reexports, jsReexport{exportedName: "*", localName: "*", moduleSpec: spec, line: line})
					continue
				}
				if l := strings.Index(trimmed, "{"); l >= 0 {
					if r := strings.LastIndex(trimmed, "}"); r > l {
						for _, nb := range parseJSNamedBindings(trimmed[l : r+1]) {
							exported := nb.exported
							if exported == "" {
								exported = nb.local
							}
							info.named[exported] = nb.local
							info.reexports = append(info.reexports, jsReexport{exportedName: exported, localName: nb.local, moduleSpec: spec, line: line})
						}
					}
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, " from ") {
			// side-effect import
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			spec := rc.moduleFromSegments(segs)
			if spec != "" {
				clause := rc.importClauseFromCode(trimmed)
				rc.parseImportClause(clause, spec, line)
			}
			continue
		}
		if strings.Contains(trimmed, "require(") && (strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ")) {
			spec := rc.moduleFromSegments(segs)
			if spec != "" {
				// extract LHS: const/let/var <lhs> = require...
				eq := strings.Index(trimmed, "=")
				if eq > 0 {
					lhs := strings.TrimSpace(trimmed[strings.Index(trimmed, " ")+1 : eq])
					rc.parseRequireBinding(lhs, spec, line)
				}
				continue
			}
		}
		if m := jsExportNamedRe.FindStringSubmatch(trimmed); m != nil {
			for _, nb := range parseJSNamedBindings(m[1]) {
				exported := nb.exported
				if exported == "" {
					exported = nb.local
				}
				if nb.local != "" {
					info.named[exported] = nb.local
				}
			}
			continue
		}
		if m := jsExportDefFuncRe.FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			continue
		}
		if m := jsExportDefClassRe.FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			continue
		}
		if m := jsExportDefIdentRe.FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			continue
		}
		if strings.HasPrefix(trimmed, "export default ") {
			info.defaultSymbol = ""
			continue
		}
		if m := jsExportDeclRe.FindStringSubmatch(trimmed); m != nil {
			info.named[m[1]] = m[1]
			continue
		}
		// CommonJS export markers
		if strings.HasPrefix(trimmed, "module.exports") || strings.HasPrefix(trimmed, "exports.") || strings.HasPrefix(trimmed, "exports[") {
			info.cjs = true
			// module.exports = { a, b } or exports.a = ...
			if strings.HasPrefix(trimmed, "exports.") {
				name := strings.TrimPrefix(trimmed, "exports.")
				if idx := strings.IndexAny(name, " =;(["); idx > 0 {
					n := name[:idx]
					if jsIdentRe.MatchString(n) {
						info.named[n] = n
					}
				}
			}
			if strings.HasPrefix(trimmed, "module.exports") && strings.Contains(trimmed, "{") {
				if l := strings.Index(trimmed, "{"); l >= 0 {
					if r := strings.LastIndex(trimmed, "}"); r > l {
						for _, nb := range parseJSNamedBindings(trimmed[l : r+1]) {
							n := nb.exported
							if n == "" {
								n = nb.local
							}
							if n != "" {
								info.named[n] = nb.local
								if info.named[n] == "" {
									info.named[n] = n
								}
							}
						}
					}
				}
			}
		}
	}
	return info
}

func (rc *jsResolveContext) importClauseFromCode(trimmed string) string {
	// import <clause> from ...  (specifier removed from code text)
	s := strings.TrimPrefix(trimmed, "import")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "type ")
	if idx := strings.Index(s, " from "); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, " from"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

type jsNamedBinding struct{ local, exported string }

func parseJSNamedBindings(clause string) []jsNamedBinding {
	clause = strings.TrimSpace(clause)
	clause = strings.TrimPrefix(clause, "{")
	clause = strings.TrimSuffix(clause, "}")
	var out []jsNamedBinding
	for _, part := range strings.Split(clause, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "type ")
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		local := fields[0]
		exported := local
		for i := 0; i+2 <= len(fields); i++ {
			if fields[i] == "as" && i+1 < len(fields) {
				exported = fields[i+1]
				break
			}
		}
		if local != "" {
			out = append(out, jsNamedBinding{local: local, exported: exported})
		}
	}
	return out
}

func (rc *jsResolveContext) parseImportClause(clause, spec string, line int) {
	clause = strings.TrimSpace(clause)
	// namespace: * as ns
	if m := regexp.MustCompile(`^\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)$`).FindStringSubmatch(clause); m != nil {
		rc.bindings[m[1]] = jsBinding{local: m[1], moduleSpec: spec, importedName: "*", kind: "namespace", line: line}
		return
	}
	// default + named/namespace
	if strings.HasPrefix(clause, "{") {
		for _, nb := range parseJSNamedBindings(clause) {
			local := nb.exported
			if local == "" {
				local = nb.local
			}
			// In `import {x as y}`, local binding is y, imported is x.
			if nb.exported != "" && nb.exported != nb.local {
				local = nb.exported
			} else {
				local = nb.local
			}
			rc.bindings[local] = jsBinding{local: local, moduleSpec: spec, importedName: nb.local, kind: "named", line: line}
		}
		return
	}
	// default, possibly with { ... } or * as ns
	if idx := strings.Index(clause, ","); idx >= 0 {
		def := strings.TrimSpace(clause[:idx])
		rest := strings.TrimSpace(clause[idx+1:])
		if def != "" && !strings.HasPrefix(def, "{") && !strings.HasPrefix(def, "*") {
			rc.bindings[def] = jsBinding{local: def, moduleSpec: spec, importedName: "default", kind: "default", line: line}
		}
		rc.parseImportClause(rest, spec, line)
		return
	}
	if !strings.HasPrefix(clause, "*") && clause != "" {
		rc.bindings[clause] = jsBinding{local: clause, moduleSpec: spec, importedName: "default", kind: "default", line: line}
	}
}

func (rc *jsResolveContext) parseRequireBinding(lhs, spec string, line int) {
	lhs = strings.TrimSpace(lhs)
	if strings.HasPrefix(lhs, "{") {
		for _, nb := range parseJSNamedBindings(lhs) {
			local := nb.exported
			if local == "" || local == nb.local {
				local = nb.local
			}
			rc.bindings[local] = jsBinding{local: local, moduleSpec: spec, importedName: nb.local, kind: "require_destructured", line: line}
		}
		return
	}
	if jsIdentRe.MatchString(lhs) {
		rc.bindings[lhs] = jsBinding{local: lhs, moduleSpec: spec, importedName: "*", kind: "require", line: line}
	}
}

var jsIdentRe = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func (rc *jsResolveContext) scanOccurrences(codeLines []string, segLines [][]lexical.Segment) {
	for i, code := range codeLines {
		line := i + 1
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		indent := len(code) - len(strings.TrimLeft(code, " \t"))
		rc.popScopes(indent, trimmed)
		segs := segLines[i]

		// Import/export declaration references.
		if strings.HasPrefix(trimmed, "import ") {
			spec := rc.moduleFromSegments(segs)
			if spec != "" {
				clause := rc.importClauseFromCode(trimmed)
				rc.resolveImportClause(clause, spec, line)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "export ") {
			spec := rc.moduleFromSegments(segs)
			if spec != "" && strings.Contains(trimmed, "from") {
				rc.emitExportRefs(trimmed, spec, line)
			}
			// Continue to also pick up exported declarations as local refs/decls.
		}

		// Declarations and scopes.
		if m := jsClassDeclRe.FindStringSubmatch(trimmed); m != nil {
			rc.setLocal(m[1])
			rc.pushScope(jsScope{kind: jsScopeClass, name: m[1], indent: indent})
			if strings.Contains(trimmed, " extends ") {
				rc.emitExtendsRef(trimmed, line)
			}
		}
		if m := jsFuncDeclRe.FindStringSubmatch(trimmed); m != nil {
			rc.setLocal(m[1])
			rc.pushScope(jsScope{kind: jsScopeFunc, name: m[1], indent: indent})
			rc.declareParams(trimmed)
		}
		if m := jsVarDeclRe.FindStringSubmatch(trimmed); m != nil {
			rhs := strings.TrimSpace(m[2])
			if nm := jsNewAssignRe.FindStringSubmatch(rhs); nm != nil {
				rc.setLocalType(m[1], nm[1])
			} else {
				rc.setLocal(m[1])
			}
		}
		// Method definitions inside class: simple heuristic.
		if rc.currentClass() != "" && indent > 0 && !strings.HasPrefix(trimmed, "class ") && !strings.HasPrefix(trimmed, "export ") {
			if mm := regexp.MustCompile(`^(?:async\s+|static\s+|get\s+|set\s+|\*\s*)*([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)\s*\{?`).FindStringSubmatch(trimmed); mm != nil {
				if !jsControlWord(mm[1]) && !strings.Contains(trimmed, "=") {
					rc.pushScope(jsScope{kind: jsScopeFunc, name: mm[1], indent: indent})
					rc.declareParams(trimmed)
				}
			}
		}
		// Block scopes for if/for/while/try/catch/finally - conservative.
		if strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "for ") || strings.HasPrefix(trimmed, "while ") || strings.HasPrefix(trimmed, "try") || strings.HasPrefix(trimmed, "catch") || strings.HasPrefix(trimmed, "switch ") {
			if strings.Contains(trimmed, "{") {
				rc.pushScope(jsScope{kind: jsScopeBlock, indent: indent})
			}
		}

		rc.scanCallsAndTypes(code, trimmed, line)
	}
}

func (rc *jsResolveContext) declareParams(line string) {
	l := strings.Index(line, "(")
	r := strings.Index(line, ")")
	if l < 0 || r <= l {
		return
	}
	for _, m := range jsParamIdentRe.FindAllString(line[l+1:r], -1) {
		if !jsControlWord(m) {
			rc.setLocal(m)
		}
	}
}

func (rc *jsResolveContext) emitExtendsRef(trimmed string, line int) {
	idx := strings.Index(trimmed, " extends ")
	if idx < 0 {
		return
	}
	rest := strings.TrimSpace(trimmed[idx+9:])
	for _, cut := range []string{" implements ", " {", "{"} {
		if c := strings.Index(rest, cut); c >= 0 {
			rest = rest[:c]
		}
	}
	name := strings.TrimSpace(rest)
	if jsIdentRe.MatchString(name) {
		rc.resolveName(name, domain.ReferenceType, line, strings.Index(trimmed, name)+1, "extends "+name)
	}
}

func (rc *jsResolveContext) emitImportRefs(trimmed string, line int) {
	if m := jsImportLineRe.FindStringSubmatch(trimmed); m != nil {
		clause := strings.TrimSpace(m[1])
		spec := m[2]
		rc.resolveImportClause(clause, spec, line)
	}
}

func (rc *jsResolveContext) resolveImportClause(clause, spec string, line int) {
	dep := rc.findModuleDep(spec, line)
	if strings.HasPrefix(clause, "{") {
		for _, nb := range parseJSNamedBindings(clause) {
			local := nb.exported
			if local == "" || local == nb.local {
				local = nb.local
			}
			rc.resolveImportedBinding(local, nb.local, spec, dep, line)
		}
		return
	}
	if m := regexp.MustCompile(`^\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)$`).FindStringSubmatch(clause); m != nil {
		rc.emitModuleImportRef(m[1], spec, dep, line, "namespace import")
		return
	}
	if idx := strings.Index(clause, ","); idx >= 0 {
		def := strings.TrimSpace(clause[:idx])
		rest := strings.TrimSpace(clause[idx+1:])
		if def != "" && jsIdentRe.MatchString(def) {
			rc.resolveImportedBinding(def, "default", spec, dep, line)
		}
		rc.resolveImportClause(rest, spec, line)
		return
	}
	if jsIdentRe.MatchString(clause) {
		rc.resolveImportedBinding(clause, "default", spec, dep, line)
	}
}

func (rc *jsResolveContext) emitExportRefs(trimmed, spec string, line int) {
	dep := rc.findModuleDep(spec, line)
	if strings.Contains(trimmed, "export *") {
		return
	}
	if l := strings.Index(trimmed, "{"); l >= 0 {
		if r := strings.LastIndex(trimmed, "}"); r > l {
			for _, nb := range parseJSNamedBindings(trimmed[l : r+1]) {
				rc.resolveImportedBinding(nb.local, nb.local, spec, dep, line)
			}
		}
	}
}

func (rc *jsResolveContext) emitModuleImportRef(name, spec string, dep *ModuleDependency, line int, detail string) {
	ref := rc.baseRef(name, domain.ReferenceImport, line, 1)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, line, spec), NewModuleResolutionEvidence(detail + " " + spec)}
	if dep == nil {
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	}
	switch dep.Status {
	case ModuleExternal:
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: spec, Kind: "module", Language: domain.LangJavaScript, Container: spec}
	case ModuleAmbiguous:
		ref.Resolution = domain.ResolutionAmbiguous
	case ModuleDynamic:
		ref.Resolution = domain.ResolutionDynamic
	case ModuleResolved:
		ref.Resolution = domain.ResolutionResolved
		ref.Target = &domain.SymbolRef{Name: filepath.Base(dep.ResolvedPath), Kind: "module", File: dep.ResolvedPath, Language: dep.ResolvedLanguage}
	default:
		ref.Resolution = domain.ResolutionUnresolved
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *jsResolveContext) findModuleDep(spec string, line int) *ModuleDependency {
	if rc.modGraph == nil {
		return nil
	}
	for i := range rc.modGraph.Dependencies {
		d := &rc.modGraph.Dependencies[i]
		if d.SourceFile == rc.filePath && d.Specifier == spec && (d.Line == line || d.Line == 0 || line == 0) {
			return d
		}
	}
	for i := range rc.modGraph.Dependencies {
		d := &rc.modGraph.Dependencies[i]
		if d.SourceFile == rc.filePath && d.Specifier == spec {
			return d
		}
	}
	return nil
}

func (rc *jsResolveContext) resolveImportedBinding(localName, importedName, spec string, dep *ModuleDependency, line int) {
	ref := rc.baseRef(localName, domain.ReferenceImport, line, 1)
	ref.Evidence = []domain.ReferenceEvidence{
		NewImportDeclarationEvidence(rc.filePath, line, spec),
		NewModuleResolutionEvidence("binding " + localName + " <- " + importedName + " from " + spec),
	}
	if dep == nil {
		// Classify by specifier.
		if javascript.IsBuiltin(spec) || (!javascript.IsRelative(spec) && !strings.HasPrefix(spec, "/")) {
			ref.Resolution = domain.ResolutionExternal
			ref.Target = &domain.SymbolRef{Name: importedName, Kind: "symbol", Language: domain.LangJavaScript, Container: spec}
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	switch dep.Status {
	case ModuleExternal:
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: importedName, Kind: "symbol", Language: domain.LangJavaScript, Container: spec}
		rc.refs = append(rc.refs, ref)
		return
	case ModuleAmbiguous:
		ref.Resolution = domain.ResolutionAmbiguous
		ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(localName, "Module specifier resolves ambiguously"))
		rc.refs = append(rc.refs, ref)
		return
	case ModuleDynamic:
		ref.Resolution = domain.ResolutionDynamic
		rc.refs = append(rc.refs, ref)
		return
	case ModuleUnresolved:
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	case ModuleResolved:
		// continue to symbol resolution
	default:
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	}
	targetFile := dep.ResolvedPath
	if targetFile == "" {
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	}
	if importedName == "*" {
		ref.Resolution = domain.ResolutionResolved
		ref.Target = &domain.SymbolRef{Name: filepath.Base(targetFile), Kind: "module", File: targetFile, Language: dep.ResolvedLanguage}
		rc.refs = append(rc.refs, ref)
		return
	}
	syms := rc.resolveExportedSymbol(targetFile, importedName, dep.ResolvedLanguage)
	switch len(syms) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(importedName, "Exported symbol not found in "+targetFile))
	case 1:
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(syms[0])
		ref.Target = &t
	default:
		ref.Resolution = domain.ResolutionAmbiguous
		for _, s := range syms {
			ref.Candidates = append(ref.Candidates, symbolToRef(s))
		}
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *jsResolveContext) resolveExportedSymbol(file, exportedName, lang string) []Symbol {
	file = filepath.ToSlash(file)
	exports := rc.getExports(file)
	localName := exportedName
	mapped := false
	if exportedName == "default" {
		if exports.defaultSymbol != "" {
			localName = exports.defaultSymbol
			mapped = true
		} else {
			// Check reexports for default? uncommon; try symbol named default won't exist.
			// Fall through to find by local name heuristics below only if explicitly mapped.
		}
	} else if ln, ok := exports.named[exportedName]; ok && ln != "" {
		localName = ln
		mapped = true
	}
	// Check reexports
	if exportedName != "default" {
		for _, re := range exports.reexports {
			if re.exportedName == exportedName {
				dep := rc.findModuleDepFromFile(file, re.moduleSpec)
				if dep != nil && dep.Status == ModuleResolved && dep.ResolvedPath != "" {
					name := re.localName
					if name == "" || name == "*" {
						name = exportedName
					}
					res := rc.resolveExportedSymbolWithStack(dep.ResolvedPath, name, dep.ResolvedLanguage)
					if len(res) > 0 {
						return res
					}
				}
			}
		}
		// export * from - try after direct/named reexports
		if !mapped {
			for _, re := range exports.reexports {
				if re.exportedName == "*" {
					dep := rc.findModuleDepFromFile(file, re.moduleSpec)
					if dep != nil && dep.Status == ModuleResolved && dep.ResolvedPath != "" {
						res := rc.resolveExportedSymbolWithStack(dep.ResolvedPath, exportedName, dep.ResolvedLanguage)
						if len(res) > 0 {
							return res
						}
					}
				}
			}
		}
	}
	if exportedName == "default" && !mapped {
		// Try to find a default-like symbol? Conservatively return nil.
		return nil
	}
	var out []Symbol
	for _, s := range rc.byFile[file] {
		if s.Name != localName {
			continue
		}
		if s.Kind == "file" || s.Kind == "module" {
			continue
		}
		if lang != "" && s.Language != "" && s.Language != lang && !isJSTSLangPair(s.Language, lang) {
			continue
		}
		out = append(out, s)
	}
	// For CommonJS modules, allow conservative fallback when a same-file symbol
	// matches and the export relationship is not explicitly mapped as ESM.
	if len(out) == 0 && !mapped && exportedName != "default" && exports.cjs {
		for _, s := range rc.byFile[file] {
			if s.Name == exportedName && s.Kind != "file" && s.Kind != "module" {
				if lang != "" && s.Language != "" && s.Language != lang && !isJSTSLangPair(s.Language, lang) {
					continue
				}
				out = append(out, s)
			}
		}
	}
	if len(out) > 1 {
		var exported []Symbol
		for _, s := range out {
			if s.Exported != nil && *s.Exported {
				exported = append(exported, s)
			}
		}
		if len(exported) == 1 {
			return exported
		}
		if len(exported) > 1 {
			out = exported
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Line < out[j].Line || (out[i].Line == out[j].Line && out[i].Kind < out[j].Kind)
	})
	return out
}

func (rc *jsResolveContext) resolveExportedSymbolWithStack(file, exportedName, lang string) []Symbol {
	key := filepath.ToSlash(file) + "\x00" + exportedName
	if rc.reexportStack[key] {
		return nil
	}
	rc.reexportStack[key] = true
	defer delete(rc.reexportStack, key)
	return rc.resolveExportedSymbol(file, exportedName, lang)
}

func isJSTSLangPair(a, b string) bool {
	js := a == domain.LangJavaScript || b == domain.LangJavaScript
	ts := a == domain.LangTypeScript || b == domain.LangTypeScript
	return js && ts
}

func (rc *jsResolveContext) findModuleDepFromFile(srcFile, spec string) *ModuleDependency {
	if rc.modGraph == nil {
		return nil
	}
	srcFile = filepath.ToSlash(srcFile)
	for i := range rc.modGraph.Dependencies {
		d := &rc.modGraph.Dependencies[i]
		if d.SourceFile == srcFile && d.Specifier == spec {
			return d
		}
	}
	return nil
}

func (rc *jsResolveContext) getExports(file string) *jsExportInfo {
	file = filepath.ToSlash(file)
	if info, ok := rc.exportCache[file]; ok {
		return info
	}
	abs := filepath.Join(rc.root, filepath.FromSlash(file))
	src, err := os.ReadFile(abs)
	if err != nil {
		info := &jsExportInfo{named: map[string]string{}}
		rc.exportCache[file] = info
		return info
	}
	scanner := lexical.NewScanner(javascript.Profile())
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	codeLines := make([]string, len(raw))
	segLines := make([][]lexical.Segment, len(raw))
	for i, text := range raw {
		segs := scanner.Scan(text)
		segLines[i] = segs
		codeLines[i] = lexical.CodeText(segs)
	}
	tmp := &jsResolveContext{bindings: map[string]jsBinding{}, exportCache: map[string]*jsExportInfo{}}
	info := tmp.parseModuleBindingsAndExports(codeLines, segLines, file)
	if info.named == nil {
		info.named = map[string]string{}
	}
	rc.exportCache[file] = info
	return info
}

func (rc *jsResolveContext) scanCallsAndTypes(code, trimmed string, line int) {
	// new ClassName(...)
	for _, loc := range regexp.MustCompile(`\bnew\s+([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*)\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		name := code[loc[2]:loc[3]]
		col := loc[2] + len("new ") + 1
		if idx := strings.Index(code[loc[0]:loc[1]], name); idx >= 0 {
			col = loc[0] + idx + 1
		}
		rc.resolveNew(name, line, col)
	}
	// this.method(...)
	for _, loc := range regexp.MustCompile(`\bthis\.([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		method := code[loc[2]:loc[3]]
		col := loc[2] + 1
		rc.resolveThisMethod(method, line, col)
	}
	// obj.method(...) and simple calls ident(...)
	for _, loc := range regexp.MustCompile(`([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*)\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		chain := code[loc[2]:loc[3]]
		col := loc[2] + 1
		// Skip keywords/control constructs and "new X" already handled.
		if jsControlWord(chain) || strings.HasPrefix(strings.TrimSpace(code[max(0, loc[0]-4):loc[0]]), "new") {
			continue
		}
		if strings.Contains(chain, ".") {
			parts := strings.Split(chain, ".")
			if parts[0] == "this" {
				continue // handled above
			}
			rc.resolveMemberCall(parts, line, col, code)
		} else {
			rc.resolveCallIdent(chain, line, col, trimmed)
		}
	}
	// require(...) as expression in calls - dependency exists; reference to module handled by binding/import refs where possible.
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (rc *jsResolveContext) resolveNew(chain string, line, col int) {
	parts := strings.Split(chain, ".")
	if len(parts) == 1 {
		rc.resolveName(chain, domain.ReferenceCall, line, col, "constructor call "+chain)
		return
	}
	// Namespace or require binding: ns.Class or req.Class
	if binding, ok := rc.bindings[parts[0]]; ok && len(parts) == 2 {
		if binding.kind == "namespace" || binding.kind == "require" || (binding.importedName == "*" && (binding.kind == "require" || binding.kind == "namespace")) {
			rc.resolveNamespaceMember(binding, parts[1], domain.ReferenceCall, line, col, "constructor "+chain)
			return
		}
	}
	rc.emitDynamic(parts[len(parts)-1], domain.ReferenceMethod, line, col, "constructor call on unresolved chain "+chain)
}

func (rc *jsResolveContext) resolveThisMethod(method string, line, col int) {
	class := rc.currentClass()
	if class == "" {
		rc.emitDynamic(method, domain.ReferenceMethod, line, col, "this."+method+" outside class")
		return
	}
	cands := rc.classMembers(class, method)
	ref := rc.baseRef(method, domain.ReferenceMethod, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewLexicalMethodCallEvidence(rc.filePath, line, "this."+method+" in class "+class)}
	switch len(cands) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
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

func (rc *jsResolveContext) classMembers(class, member string) []Symbol {
	// Find class symbol in same file.
	var out []Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Container == class && s.Name == member && (s.Kind == "method" || s.Kind == "func" || s.Kind == "field") {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

func (rc *jsResolveContext) resolveMemberCall(parts []string, line, col int, code string) {
	if len(parts) < 2 {
		return
	}
	receiver := parts[0]
	member := parts[len(parts)-1]
	if !rc.hasLocal(receiver) {
		if binding, ok := rc.bindings[receiver]; ok {
			if binding.kind == "namespace" || binding.kind == "require" || binding.importedName == "*" {
				if len(parts) == 2 {
					rc.resolveNamespaceMember(binding, member, domain.ReferenceCall, line, col, "member call "+strings.Join(parts, "."))
					return
				}
			}
		}
	}
	if typ, ok := rc.lookupLocalType(receiver); ok && len(parts) == 2 {
		rc.resolveTypedMember(receiver, typ, member, line, col)
		return
	}
	if rc.hasLocal(receiver) {
		// Computed or dynamic-looking access.
		if strings.Contains(code, "[") && strings.Contains(code, "]") {
			rc.emitDynamic(member, domain.ReferenceMethod, line, col, "computed/dynamic member access")
			return
		}
		rc.emitDynamic(member, domain.ReferenceMethod, line, col, "method call on local receiver "+receiver)
		return
	}
	// Computed or dynamic-looking access.
	if strings.Contains(code, "[") && strings.Contains(code, "]") {
		rc.emitDynamic(member, domain.ReferenceMethod, line, col, "computed/dynamic member access")
		return
	}
	rc.emitDynamic(member, domain.ReferenceMethod, line, col, "method call on unresolved receiver "+receiver)
}

func (rc *jsResolveContext) resolveTypedMember(receiver, typeName, member string, line, col int) {
	// typeName may be "Class" or "ns.Class" - handle local class.
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if binding, ok := rc.bindings[parts[0]]; ok && len(parts) == 2 {
			rc.resolveNamespaceMember(binding, parts[1], domain.ReferenceType, line, col, "receiver type "+typeName)
			// Member on external/type from namespace: dynamic/unresolved conservatively.
		}
		rc.emitDynamic(member, domain.ReferenceMethod, line, col, "member call on typed receiver "+receiver+" : "+typeName)
		return
	}
	// Local or imported class?
	cands := rc.localSymbolsByName(typeName, domain.ReferenceType)
	if len(cands) == 1 {
		classSym := cands[0]
		members := rc.classMembersInFile(classSym.File, classSym.Name, member)
		ref := rc.baseRef(member, domain.ReferenceMethod, line, col)
		ref.Evidence = []domain.ReferenceEvidence{NewLexicalMethodCallEvidence(rc.filePath, line, "method call on local instance of "+typeName)}
		switch len(members) {
		case 0:
			ref.Resolution = domain.ResolutionUnresolved
		case 1:
			ref.Resolution = domain.ResolutionResolved
			t := symbolToRef(members[0])
			ref.Target = &t
		default:
			ref.Resolution = domain.ResolutionAmbiguous
			for _, m := range members {
				ref.Candidates = append(ref.Candidates, symbolToRef(m))
			}
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	if binding, ok := rc.bindings[typeName]; ok {
		dep := rc.findModuleDep(binding.moduleSpec, binding.line)
		if dep != nil && dep.Status == ModuleResolved {
			// Resolve class symbol in target module, then method - conservative.
			classSyms := rc.resolveExportedSymbol(dep.ResolvedPath, binding.importedName, dep.ResolvedLanguage)
			if len(classSyms) == 1 {
				members := rc.classMembersInFile(classSyms[0].File, classSyms[0].Name, member)
				ref := rc.baseRef(member, domain.ReferenceMethod, line, col)
				ref.Evidence = []domain.ReferenceEvidence{NewLexicalMethodCallEvidence(rc.filePath, line, "method call on instance of imported "+typeName)}
				switch len(members) {
				case 0:
					ref.Resolution = domain.ResolutionUnresolved
				case 1:
					ref.Resolution = domain.ResolutionResolved
					t := symbolToRef(members[0])
					ref.Target = &t
				default:
					ref.Resolution = domain.ResolutionAmbiguous
					for _, m := range members {
						ref.Candidates = append(ref.Candidates, symbolToRef(m))
					}
				}
				rc.refs = append(rc.refs, ref)
				return
			}
		}
	}
	rc.emitDynamic(member, domain.ReferenceMethod, line, col, "method call on typed receiver "+receiver+" : "+typeName)
}

func (rc *jsResolveContext) classMembersInFile(file, class, member string) []Symbol {
	file = filepath.ToSlash(relToRoot(rc.root, filepath.Join(rc.root, filepath.FromSlash(file))))
	var out []Symbol
	for _, s := range rc.byFile[file] {
		if s.Container == class && s.Name == member {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		// try with file as-is (already rel)
		for _, s := range rc.byFile[filepath.ToSlash(file)] {
			if s.Container == class && s.Name == member {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

func (rc *jsResolveContext) resolveNamespaceMember(binding jsBinding, member string, kind domain.ReferenceKind, line, col int, detail string) {
	dep := rc.findModuleDep(binding.moduleSpec, binding.line)
	ref := rc.baseRef(member, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, binding.line, binding.moduleSpec), NewLexicalCallPatternEvidence(rc.filePath, line, detail)}
	if dep == nil || dep.Status == ModuleExternal {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: member, Kind: "symbol", Language: domain.LangJavaScript, Container: binding.moduleSpec}
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status == ModuleAmbiguous {
		ref.Resolution = domain.ResolutionAmbiguous
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status != ModuleResolved || dep.ResolvedPath == "" {
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	}
	syms := rc.resolveExportedSymbol(dep.ResolvedPath, member, dep.ResolvedLanguage)
	switch len(syms) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
	case 1:
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(syms[0])
		ref.Target = &t
	default:
		ref.Resolution = domain.ResolutionAmbiguous
		for _, s := range syms {
			ref.Candidates = append(ref.Candidates, symbolToRef(s))
		}
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *jsResolveContext) resolveCallIdent(name string, line, col int, trimmed string) {
	if jsControlWord(name) || jsBuiltinValue(name) {
		return
	}
	if rc.isDeclarationName(trimmed, name) {
		return
	}
	if rc.hasLocal(name) {
		// Local/shadowed variable - could be dynamic call target. Don't resolve to module-level symbol.
		if _, hasType := rc.lookupLocalType(name); hasType {
			// constructor/typed local used as call target without new - rare; treat conservatively.
		}
		return
	}
	if binding, ok := rc.bindings[name]; ok {
		rc.resolveBindingUse(binding, domain.ReferenceCall, line, col)
		return
	}
	rc.resolveName(name, domain.ReferenceCall, line, col, "call "+name)
}

func (rc *jsResolveContext) isDeclarationName(trimmed, name string) bool {
	prefixes := []string{"function " + name, "class " + name, "const " + name, "let " + name, "var " + name, "export function " + name, "export class " + name, "export default function " + name, "export default class " + name, "export const " + name, "export let " + name, "export var " + name, "async function " + name, "export async function " + name}
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func (rc *jsResolveContext) resolveBindingUse(binding jsBinding, kind domain.ReferenceKind, line, col int) {
	dep := rc.findModuleDep(binding.moduleSpec, binding.line)
	ref := rc.baseRef(binding.local, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, binding.line, binding.moduleSpec), NewLexicalCallPatternEvidence(rc.filePath, line, "use of imported binding "+binding.local)}
	if dep == nil {
		if javascript.IsBuiltin(binding.moduleSpec) || !javascript.IsRelative(binding.moduleSpec) {
			ref.Resolution = domain.ResolutionExternal
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status == ModuleExternal {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: binding.importedName, Kind: "symbol", Language: domain.LangJavaScript, Container: binding.moduleSpec}
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status != ModuleResolved || dep.ResolvedPath == "" {
		ref.Resolution = mapModuleStatus(dep.Status)
		rc.refs = append(rc.refs, ref)
		return
	}
	if binding.importedName == "*" {
		ref.Resolution = domain.ResolutionResolved
		ref.Target = &domain.SymbolRef{Name: filepath.Base(dep.ResolvedPath), Kind: "module", File: dep.ResolvedPath, Language: dep.ResolvedLanguage}
		rc.refs = append(rc.refs, ref)
		return
	}
	syms := rc.resolveExportedSymbol(dep.ResolvedPath, binding.importedName, dep.ResolvedLanguage)
	switch len(syms) {
	case 0:
		ref.Resolution = domain.ResolutionUnresolved
	case 1:
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(syms[0])
		ref.Target = &t
	default:
		ref.Resolution = domain.ResolutionAmbiguous
		for _, s := range syms {
			ref.Candidates = append(ref.Candidates, symbolToRef(s))
		}
	}
	rc.refs = append(rc.refs, ref)
}

func mapModuleStatus(s ModuleResolutionStatus) domain.ResolutionStatus {
	switch s {
	case ModuleResolved:
		return domain.ResolutionResolved
	case ModuleAmbiguous:
		return domain.ResolutionAmbiguous
	case ModuleExternal:
		return domain.ResolutionExternal
	case ModuleDynamic:
		return domain.ResolutionDynamic
	default:
		return domain.ResolutionUnresolved
	}
}

func (rc *jsResolveContext) resolveName(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewLexicalCallPatternEvidence(rc.filePath, line, detail)}
	if jsBuiltinValue(name) {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "builtin", Language: domain.LangJavaScript}
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.localSymbolsByName(name, kind)
	// If a local (shadowing) is present in current scope, localSymbolsByName won't see it via symbols table except same-file; but hasLocal was checked by callers.
	if len(cands) == 0 {
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, ref)
		return
	}
	if len(cands) == 1 {
		ref.Resolution = domain.ResolutionResolved
		t := symbolToRef(cands[0])
		ref.Target = &t
		rc.refs = append(rc.refs, ref)
		return
	}
	ref.Resolution = domain.ResolutionAmbiguous
	for _, c := range cands {
		ref.Candidates = append(ref.Candidates, symbolToRef(c))
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *jsResolveContext) localSymbolsByName(name string, kind domain.ReferenceKind) []Symbol {
	var sameFile []Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Name != name || s.Kind == "file" || s.Kind == "module" {
			continue
		}
		// Filter by kind preference.
		if kind == domain.ReferenceCall && s.Kind != "func" && s.Kind != "method" && s.Kind != "type" {
			continue
		}
		if kind == domain.ReferenceType && s.Kind != "type" {
			continue
		}
		sameFile = append(sameFile, s)
	}
	if len(sameFile) == 0 && kind == domain.ReferenceCall {
		// Fallback: allow any same-file symbol with matching name.
		for _, s := range rc.byFile[rc.filePath] {
			if s.Name == name && s.Kind != "file" && s.Kind != "module" {
				sameFile = append(sameFile, s)
			}
		}
	}
	sort.Slice(sameFile, func(i, j int) bool { return sameFile[i].Line < sameFile[j].Line })
	return sameFile
}

func (rc *jsResolveContext) baseRef(name string, kind domain.ReferenceKind, line, col int) domain.Reference {
	return domain.Reference{
		Kind:     kind,
		Language: domain.LangJavaScript,
		Location: domain.Location{Path: rc.filePath, StartLine: line, StartColumn: col},
		Source: domain.SymbolRef{
			Name:     name,
			Kind:     string(kind),
			File:     rc.filePath,
			Line:     line,
			Column:   col,
			Language: domain.LangJavaScript,
		},
	}
}

func (rc *jsResolveContext) emitDynamic(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Resolution = domain.ResolutionDynamic
	ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, detail)}
	rc.refs = append(rc.refs, ref)
}

func jsControlWord(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "function", "class", "return", "throw", "new", "typeof", "instanceof", "delete", "void", "await", "yield", "else", "do", "try", "finally", "break", "continue", "case", "default", "import", "export", "from", "as", "const", "let", "var", "this", "super", "extends", "implements", "interface", "type", "enum", "async", "static", "get", "set", "of", "in", "with", "debugger":
		return true
	}
	return false
}

func jsBuiltinValue(name string) bool {
	switch name {
	case "console", "Math", "JSON", "Object", "Array", "String", "Number", "Boolean", "Symbol", "BigInt", "Date", "RegExp", "Error", "TypeError", "RangeError", "Promise", "Map", "Set", "WeakMap", "WeakSet", "Proxy", "Reflect", "parseInt", "parseFloat", "isNaN", "isFinite", "setTimeout", "setInterval", "clearTimeout", "clearInterval", "require", "module", "exports", "globalThis", "window", "document", "undefined", "null", "NaN", "Infinity", "fetch", "URL", "URLSearchParams", "Buffer", "process", "global":
		return true
	}
	return false
}
