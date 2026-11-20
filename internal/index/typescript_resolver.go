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
	"github.com/barceloch/codixai/internal/languages/typescript"
)

// TypeScriptReferenceResolver resolves TypeScript references conservatively.
// It distinguishes type-space and value-space references, supports type-only
// imports/exports, annotations, inheritance/implementation, and local/module
// resolution using the existing module graph. It never type-checks or executes
// analyzed code.
type TypeScriptReferenceResolver struct{}

// NewTypeScriptReferenceResolver creates a new TypeScript reference resolver.
func NewTypeScriptReferenceResolver() *TypeScriptReferenceResolver {
	return &TypeScriptReferenceResolver{}
}

// Resolve implements ReferenceResolver for TypeScript.
func (r *TypeScriptReferenceResolver) Resolve(ctx context.Context, idx *Index, changes *domain.ChangeSet) ([]domain.Reference, error) {
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
		if fc.Language != domain.LangTypeScript {
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

type tsSpace int

const (
	tsSpaceValue tsSpace = iota
	tsSpaceType
	tsSpaceBoth
)

type tsBinding struct {
	local        string
	moduleSpec   string
	importedName string
	kind         string // named,default,namespace
	space        tsSpace
	line         int
}

type tsExportInfo struct {
	defaultSymbol string
	defaultSpace  tsSpace
	named         map[string]tsExportName
	reexports     []tsReexport
}

type tsExportName struct {
	local string
	space tsSpace
}

type tsReexport struct {
	exportedName string
	localName    string
	moduleSpec   string
	typeOnly     bool
	line         int
}

type tsScope struct {
	kind       string // file,func,class,block
	name       string
	indent     int
	typeNames  map[string]bool
	valueNames map[string]bool
	typeVars   map[string]bool
}

type tsResolveContext struct {
	root     string
	filePath string
	idx      *Index
	modGraph *ModuleGraph

	byFile      map[string][]Symbol
	bindings    map[string]tsBinding
	exports     *tsExportInfo
	exportCache map[string]*tsExportInfo
	reStack     map[string]bool
	scopes      []tsScope
	refs        []domain.Reference
	localDecls  map[string]tsSpace
}

func (r *TypeScriptReferenceResolver) resolveFile(ctx context.Context, root, filePath string, src []byte, idx *Index, modGraph *ModuleGraph) ([]domain.Reference, error) {
	_ = ctx
	rc := &tsResolveContext{
		root:        root,
		filePath:    filePath,
		idx:         idx,
		modGraph:    modGraph,
		byFile:      map[string][]Symbol{},
		bindings:    map[string]tsBinding{},
		exportCache: map[string]*tsExportInfo{},
		reStack:     map[string]bool{},
		localDecls:  map[string]tsSpace{},
	}
	if idx != nil {
		for _, s := range idx.Symbols() {
			cp := s
			cp.File = relToRoot(root, s.File)
			rc.byFile[cp.File] = append(rc.byFile[cp.File], cp)
		}
	}
	rc.pushScope(tsScope{kind: "file", indent: -1})
	scanner := lexical.NewScanner(typescript.Profile())
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	codeLines := make([]string, len(raw))
	segLines := make([][]lexical.Segment, len(raw))
	for i, text := range raw {
		segs := scanner.Scan(text)
		segLines[i] = segs
		codeLines[i] = lexical.CodeText(segs)
	}
	rc.collectLocalDecls(codeLines)
	rc.exports = rc.parseBindingsAndExports(codeLines, segLines)
	rc.scan(codeLines, segLines)
	return rc.refs, nil
}

func (rc *tsResolveContext) pushScope(s tsScope) {
	if s.typeNames == nil {
		s.typeNames = map[string]bool{}
	}
	if s.valueNames == nil {
		s.valueNames = map[string]bool{}
	}
	if s.typeVars == nil {
		s.typeVars = map[string]bool{}
	}
	rc.scopes = append(rc.scopes, s)
}

func (rc *tsResolveContext) popScopes(indent int, trimmed string) {
	for len(rc.scopes) > 1 {
		top := rc.scopes[len(rc.scopes)-1]
		if indent > top.indent {
			return
		}
		if indent < top.indent || (indent == top.indent && top.kind != "file" && !strings.HasPrefix(trimmed, "else") && !strings.HasPrefix(trimmed, "catch") && !strings.HasPrefix(trimmed, "finally")) {
			rc.scopes = rc.scopes[:len(rc.scopes)-1]
			continue
		}
		return
	}
}

func (rc *tsResolveContext) shadowedType(name string) bool {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if rc.scopes[i].typeVars[name] || rc.scopes[i].typeNames[name] {
			return true
		}
	}
	return false
}

func (rc *tsResolveContext) currentClass() string {
	for i := len(rc.scopes) - 1; i >= 0; i-- {
		if rc.scopes[i].kind == "class" && rc.scopes[i].name != "" {
			return rc.scopes[i].name
		}
	}
	return ""
}

var (
	tsIfaceRe       = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsTypeAliasRe   = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=<]`)
	tsEnumRe        = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?(?:const\s+)?enum\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsNamespaceRe   = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?(?:namespace|module)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsClassRe       = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:abstract\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsFuncRe        = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsVarRe         = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsImportFromRe  = regexp.MustCompile(`^\s*import\s+(type\s+)?(.+?)\s+from\s+`)
	tsExportFromRe2 = regexp.MustCompile(`^\s*export\s+(type\s+)?(.+?)\s+from\s+`)
	tsIdentRe2      = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

func (rc *tsResolveContext) collectLocalDecls(codeLines []string) {
	for _, code := range codeLines {
		trimmed := strings.TrimSpace(code)
		if m := tsIfaceRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceType)
			continue
		}
		if m := tsTypeAliasRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceType)
			continue
		}
		if m := tsEnumRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceBoth)
			continue
		}
		if m := tsNamespaceRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceBoth)
			continue
		}
		if m := tsClassRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceBoth)
			continue
		}
		if m := tsFuncRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceValue)
			continue
		}
		if m := tsVarRe.FindStringSubmatch(trimmed); m != nil {
			rc.localDecls[m[1]] = mergeSpace(rc.localDecls[m[1]], tsSpaceValue)
			continue
		}
	}
}

func mergeSpace(a, b tsSpace) tsSpace {
	if a == tsSpaceBoth || b == tsSpaceBoth {
		return tsSpaceBoth
	}
	if a == 0 && b == 0 {
		return b
	}
	if a == b {
		return a
	}
	if (a == tsSpaceType && b == tsSpaceValue) || (a == tsSpaceValue && b == tsSpaceType) {
		return tsSpaceBoth
	}
	return b
}

func (rc *tsResolveContext) moduleFromSegments(segs []lexical.Segment) string {
	var mod string
	for _, seg := range segs {
		if seg.Kind != lexical.KindString {
			continue
		}
		text := strings.Trim(seg.Text, "\"'`")
		if text != "" {
			mod = text
		}
	}
	return mod
}

func (rc *tsResolveContext) parseBindingsAndExports(codeLines []string, segLines [][]lexical.Segment) *tsExportInfo {
	info := &tsExportInfo{named: map[string]tsExportName{}}
	for i, code := range codeLines {
		line := i + 1
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		segs := segLines[i]
		if strings.HasPrefix(trimmed, "import ") {
			spec := rc.moduleFromSegments(segs)
			if spec == "" {
				continue
			}
			typeOnlyImport := strings.Contains(trimmed, "import type ")
			clause := rc.importClause(trimmed)
			rc.parseImportClause(clause, spec, line, typeOnlyImport)
			continue
		}
		if strings.HasPrefix(trimmed, "export") && strings.Contains(trimmed, " from ") {
			spec := rc.moduleFromSegments(segs)
			if spec == "" {
				continue
			}
			typeOnly := strings.Contains(trimmed, "export type")
			if strings.Contains(trimmed, "export *") {
				info.reexports = append(info.reexports, tsReexport{exportedName: "*", localName: "*", moduleSpec: spec, typeOnly: typeOnly, line: line})
				continue
			}
			if l := strings.Index(trimmed, "{"); l >= 0 {
				if r := strings.LastIndex(trimmed, "}"); r > l {
					for _, nb := range parseTSNamedBindings(trimmed[l : r+1]) {
						exp := nb.exported
						if exp == "" {
							exp = nb.local
						}
						space := tsSpaceValue
						if typeOnly || nb.typeOnly {
							space = tsSpaceType
						}
						info.named[exp] = tsExportName{local: nb.local, space: space}
						info.reexports = append(info.reexports, tsReexport{exportedName: exp, localName: nb.local, moduleSpec: spec, typeOnly: typeOnly || nb.typeOnly, line: line})
					}
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "export type ") || strings.HasPrefix(trimmed, "export interface ") {
			if m := tsTypeAliasRe.FindStringSubmatch(trimmed); m != nil {
				info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceType}
			}
			if m := tsIfaceRe.FindStringSubmatch(trimmed); m != nil {
				info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceType}
			}
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+(?:declare\s+)?(?:const\s+)?enum\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceBoth}
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+(?:declare\s+)?(?:namespace|module)\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceBoth}
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+default\s+(?:abstract\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			info.defaultSpace = tsSpaceBoth
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+default\s+(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			info.defaultSpace = tsSpaceValue
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+default\s+interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			info.defaultSpace = tsSpaceType
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+default\s+(?:const\s+)?enum\s+([A-Za-z_$][A-Za-z0-9_$]*)`).FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			info.defaultSpace = tsSpaceBoth
			continue
		}
		if m := regexp.MustCompile(`^\s*export\s+default\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*;?\s*$`).FindStringSubmatch(trimmed); m != nil {
			info.defaultSymbol = m[1]
			info.defaultSpace = rc.localDecls[m[1]]
			continue
		}
		if strings.HasPrefix(trimmed, "export default ") {
			info.defaultSpace = tsSpaceValue
			continue
		}
		if m := tsClassRe.FindStringSubmatch(trimmed); m != nil && strings.HasPrefix(trimmed, "export") {
			info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceBoth}
			continue
		}
		if m := tsFuncRe.FindStringSubmatch(trimmed); m != nil && strings.HasPrefix(trimmed, "export") {
			info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceValue}
			continue
		}
		if m := tsVarRe.FindStringSubmatch(trimmed); m != nil && strings.HasPrefix(trimmed, "export") {
			info.named[m[1]] = tsExportName{local: m[1], space: tsSpaceValue}
			continue
		}
		if strings.HasPrefix(trimmed, "export {") || strings.HasPrefix(trimmed, "export type {") {
			typeOnly := strings.HasPrefix(trimmed, "export type")
			if l := strings.Index(trimmed, "{"); l >= 0 {
				if r := strings.LastIndex(trimmed, "}"); r > l {
					for _, nb := range parseTSNamedBindings(trimmed[l : r+1]) {
						exp := nb.exported
						if exp == "" {
							exp = nb.local
						}
						sp := rc.localDecls[nb.local]
						if typeOnly || nb.typeOnly {
							sp = tsSpaceType
						}
						info.named[exp] = tsExportName{local: nb.local, space: sp}
					}
				}
			}
		}
	}
	return info
}

func (rc *tsResolveContext) importClause(trimmed string) string {
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

type tsNamedBinding struct {
	local, exported string
	typeOnly        bool
}

func parseTSNamedBindings(clause string) []tsNamedBinding {
	clause = strings.TrimSpace(clause)
	clause = strings.TrimPrefix(clause, "{")
	clause = strings.TrimSuffix(clause, "}")
	var out []tsNamedBinding
	for _, part := range strings.Split(clause, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		typeOnly := false
		if strings.HasPrefix(p, "type ") {
			typeOnly = true
			p = strings.TrimSpace(strings.TrimPrefix(p, "type "))
		}
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		local := fields[0]
		exported := local
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "as" {
				exported = fields[i+1]
				break
			}
		}
		if local != "" {
			out = append(out, tsNamedBinding{local: local, exported: exported, typeOnly: typeOnly})
		}
	}
	return out
}

func (rc *tsResolveContext) parseImportClause(clause, spec string, line int, typeOnlyImport bool) {
	clause = strings.TrimSpace(clause)
	if clause == "" {
		return
	}
	if m := regexp.MustCompile(`^\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)$`).FindStringSubmatch(clause); m != nil {
		space := tsSpaceValue
		if typeOnlyImport {
			space = tsSpaceType
		}
		rc.bindings[m[1]] = tsBinding{local: m[1], moduleSpec: spec, importedName: "*", kind: "namespace", space: space, line: line}
		return
	}
	if strings.HasPrefix(clause, "{") {
		for _, nb := range parseTSNamedBindings(clause) {
			local := nb.exported
			if local == "" {
				local = nb.local
			}
			space := tsSpaceValue
			if typeOnlyImport || nb.typeOnly {
				space = tsSpaceType
			}
			rc.bindings[local] = tsBinding{local: local, moduleSpec: spec, importedName: nb.local, kind: "named", space: space, line: line}
		}
		return
	}
	if idx := strings.Index(clause, ","); idx >= 0 {
		def := strings.TrimSpace(clause[:idx])
		rest := strings.TrimSpace(clause[idx+1:])
		if def != "" && tsIdentRe2.MatchString(def) {
			space := tsSpaceValue
			if typeOnlyImport {
				space = tsSpaceType
			}
			rc.bindings[def] = tsBinding{local: def, moduleSpec: spec, importedName: "default", kind: "default", space: space, line: line}
		}
		rc.parseImportClause(rest, spec, line, typeOnlyImport)
		return
	}
	if tsIdentRe2.MatchString(clause) {
		space := tsSpaceValue
		if typeOnlyImport {
			space = tsSpaceType
		}
		rc.bindings[clause] = tsBinding{local: clause, moduleSpec: spec, importedName: "default", kind: "default", space: space, line: line}
	}
}
