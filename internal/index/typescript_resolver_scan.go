package index

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/javascript"
	"github.com/barceloch/codixai/internal/languages/lexical"
	"github.com/barceloch/codixai/internal/languages/typescript"
)

func (rc *tsResolveContext) baseRef(name string, kind domain.ReferenceKind, line, col int) domain.Reference {
	return domain.Reference{
		Kind:     kind,
		Language: domain.LangTypeScript,
		Location: domain.Location{Path: rc.filePath, StartLine: line, StartColumn: col},
		Source:   domain.SymbolRef{Name: name, Kind: string(kind), File: rc.filePath, Line: line, Column: col, Language: domain.LangTypeScript},
	}
}

func (rc *tsResolveContext) findModuleDep(spec string, line int) *ModuleDependency {
	if rc.modGraph == nil {
		return nil
	}
	for i := range rc.modGraph.Dependencies {
		d := &rc.modGraph.Dependencies[i]
		if d.SourceFile == rc.filePath && d.Specifier == spec && (d.Line == line || line == 0 || d.Line == 0) {
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

func (rc *tsResolveContext) findModuleDepFromFile(srcFile, spec string) *ModuleDependency {
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

var tsQualifiedTypeRe = regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\.([A-Za-z_$][A-Za-z0-9_$]*)\b`)

func (rc *tsResolveContext) emitTypeExpr(expr string, line, col int, kind domain.ReferenceKind, detail string) {
	// Handle qualified names like Models.User first (namespace/type member access).
	consumed := map[string]bool{}
	for _, loc := range tsQualifiedTypeRe.FindAllStringSubmatchIndex(expr, -1) {
		left := expr[loc[2]:loc[3]]
		right := expr[loc[4]:loc[5]]
		if tsTypeKeyword(left) || tsTypeKeyword(right) {
			continue
		}
		consumed[expr[loc[0]:loc[1]]] = true
		if rc.shadowedType(left) {
			continue
		}
		if b, ok := rc.bindings[left]; ok && b.kind == "namespace" {
			rc.resolveNamespaceTypeMember(b, right, kind, line, col, detail+left+"."+right)
			continue
		}
		// Not a known namespace binding: abstain conservatively.
	}
	for _, name := range tsTypeIdents(expr) {
		if rc.shadowedType(name) {
			continue
		}
		skip := false
		for q := range consumed {
			if strings.HasSuffix(q, "."+name) || strings.HasPrefix(q, name+".") {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		rc.resolveTypeName(name, kind, line, col, detail+name)
	}
}

var tsTypeIdentRe = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

func tsTypeIdents(expr string) []string {
	// Remove string literals crudely and scan identifiers.
	expr = regexp.MustCompile(`"[^"]*"|'[^']*'|`+"`[^`]*`").ReplaceAllString(expr, " ")
	var out []string
	seen := map[string]bool{}
	for _, name := range tsTypeIdentRe.FindAllString(expr, -1) {
		if seen[name] || tsTypeKeyword(name) {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func tsTypeKeyword(name string) bool {
	switch name {
	case "string", "number", "boolean", "any", "unknown", "never", "void", "null", "undefined", "object", "symbol", "bigint", "this", "true", "false", "readonly", "keyof", "typeof", "infer", "extends", "implements", "in", "is", "as", "satisfies", "new", "abstract", "public", "private", "protected", "static", "override", "declare", "type", "interface", "enum", "namespace", "module", "class", "function", "const", "let", "var", "import", "export", "from", "default", "return", "if", "else", "for", "while", "do", "switch", "case", "break", "continue", "try", "catch", "finally", "throw", "async", "await", "yield", "get", "set", "of", "instanceof", "delete":
		return true
	}
	return false
}

func tsValueBuiltin(name string) bool {
	switch name {
	case "console", "Math", "JSON", "Object", "Array", "String", "Number", "Boolean", "Symbol", "BigInt", "Date", "RegExp", "Error", "Promise", "Map", "Set", "WeakMap", "WeakSet", "Proxy", "Reflect", "parseInt", "parseFloat", "setTimeout", "setInterval", "clearTimeout", "clearInterval", "undefined", "NaN", "Infinity", "globalThis", "process", "require", "module", "exports", "fetch", "URL":
		return true
	}
	return false
}

func (rc *tsResolveContext) resolveTypeName(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewLexicalTypeReferenceEvidence(rc.filePath, line, detail)}
	if binding, ok := rc.bindings[name]; ok {
		rc.resolveBindingRef(&ref, binding, name, kind, tsSpaceType)
		return
	}
	// Local type/value declarations in type space: class, interface, type alias, enum, namespace.
	if sp, ok := rc.localDecls[name]; ok && (sp == tsSpaceType || sp == tsSpaceBoth) {
		cands := rc.symbolCandidates(name, tsSpaceType)
		rc.finishCandidates(&ref, cands)
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.symbolCandidates(name, tsSpaceType)
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) resolveValueName(name string, kind domain.ReferenceKind, line, col int, detail string) {
	ref := rc.baseRef(name, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewLexicalCallPatternEvidence(rc.filePath, line, detail)}
	if tsValueBuiltin(name) {
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: name, Kind: "builtin", Language: domain.LangTypeScript}
		rc.refs = append(rc.refs, ref)
		return
	}
	if binding, ok := rc.bindings[name]; ok {
		rc.resolveBindingRef(&ref, binding, name, kind, tsSpaceValue)
		return
	}
	if sp, ok := rc.localDecls[name]; ok && (sp == tsSpaceValue || sp == tsSpaceBoth) {
		cands := rc.symbolCandidates(name, tsSpaceValue)
		rc.finishCandidates(&ref, cands)
		rc.refs = append(rc.refs, ref)
		return
	}
	// If it's a type-only local (interface/type alias), value usage is unresolved.
	if sp, ok := rc.localDecls[name]; ok && sp == tsSpaceType {
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(name, "type-only declaration used in value space"))
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.symbolCandidates(name, tsSpaceValue)
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) finishCandidates(ref *domain.Reference, cands []Symbol) {
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].File < cands[j].File || (cands[i].File == cands[j].File && cands[i].Line < cands[j].Line)
	})
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
}

func (rc *tsResolveContext) symbolCandidates(name string, space tsSpace) []Symbol {
	var out []Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Name != name || s.Kind == "file" || s.Kind == "module" {
			continue
		}
		if !tsSymbolInSpace(s, space) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func tsSymbolInSpace(s Symbol, space tsSpace) bool {
	sig := strings.ToLower(s.Signature)
	switch space {
	case tsSpaceType:
		return s.Kind == "type" || s.Kind == "interface" || strings.HasPrefix(sig, "interface") || strings.HasPrefix(sig, "type ") || strings.HasPrefix(sig, "enum") || strings.HasPrefix(sig, "class") || strings.Contains(sig, "namespace")
	case tsSpaceValue:
		if strings.HasPrefix(sig, "interface") || strings.HasPrefix(sig, "type ") {
			return false
		}
		return s.Kind == "func" || s.Kind == "method" || s.Kind == "var" || s.Kind == "field" || strings.HasPrefix(sig, "class") || strings.HasPrefix(sig, "enum") || strings.HasPrefix(sig, "function") || strings.Contains(sig, "namespace") || s.Kind == "module"
	default:
		return true
	}
}

func (rc *tsResolveContext) resolveBindingRef(ref *domain.Reference, binding tsBinding, name string, kind domain.ReferenceKind, space tsSpace) {
	ref.Evidence = append(ref.Evidence, NewImportDeclarationEvidence(rc.filePath, binding.line, binding.moduleSpec))
	if binding.space == tsSpaceType && space == tsSpaceValue {
		ref.Resolution = domain.ResolutionUnresolved
		ref.Evidence = append(ref.Evidence, NewHeuristicCandidateEvidence(name, "type-only import used in value space"))
		rc.refs = append(rc.refs, *ref)
		return
	}
	dep := rc.findModuleDep(binding.moduleSpec, binding.line)
	if dep == nil {
		if javascript.IsBuiltin(binding.moduleSpec) || !javascript.IsRelative(binding.moduleSpec) {
			ref.Resolution = domain.ResolutionExternal
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, *ref)
		return
	}
	switch dep.Status {
	case ModuleExternal:
		ref.Resolution = domain.ResolutionExternal
		ref.Target = &domain.SymbolRef{Name: binding.importedName, Kind: "symbol", Language: domain.LangTypeScript, Container: binding.moduleSpec}
		rc.refs = append(rc.refs, *ref)
		return
	case ModuleAmbiguous, ModuleDynamic, ModuleUnresolved:
		ref.Resolution = mapModuleStatus(dep.Status)
		rc.refs = append(rc.refs, *ref)
		return
	case ModuleResolved:
	default:
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, *ref)
		return
	}
	if dep.ResolvedPath == "" {
		ref.Resolution = domain.ResolutionUnresolved
		rc.refs = append(rc.refs, *ref)
		return
	}
	if binding.importedName == "*" {
		ref.Resolution = domain.ResolutionResolved
		ref.Target = &domain.SymbolRef{Name: filepath.Base(dep.ResolvedPath), Kind: "module", File: dep.ResolvedPath, Language: dep.ResolvedLanguage}
		rc.refs = append(rc.refs, *ref)
		return
	}
	cands := rc.resolveExportedSymbol(dep.ResolvedPath, binding.importedName, space, dep.ResolvedLanguage)
	rc.finishCandidates(ref, cands)
	rc.refs = append(rc.refs, *ref)
}

func (rc *tsResolveContext) resolveExportedSymbol(file, exportedName string, space tsSpace, lang string) []Symbol {
	file = filepath.ToSlash(file)
	exports := rc.getExports(file)
	localName := exportedName
	mapped := false
	if exportedName == "default" {
		if exports.defaultSymbol != "" {
			localName = exports.defaultSymbol
			mapped = true
		} else {
			return nil
		}
	} else if en, ok := exports.named[exportedName]; ok && en.local != "" {
		if space == tsSpaceValue && en.space == tsSpaceType {
			return nil
		}
		localName = en.local
		mapped = true
	}
	if exportedName != "default" {
		for _, re := range exports.reexports {
			if re.exportedName == exportedName {
				dep := rc.findModuleDepFromFile(file, re.moduleSpec)
				if dep != nil && dep.Status == ModuleResolved && dep.ResolvedPath != "" {
					name := re.localName
					if name == "" || name == "*" {
						name = exportedName
					}
					sp := space
					if re.typeOnly {
						sp = tsSpaceType
					}
					res := rc.resolveExportedSymbolWithStack(dep.ResolvedPath, name, sp, dep.ResolvedLanguage)
					if len(res) > 0 {
						return res
					}
				}
			}
		}
		if !mapped {
			for _, re := range exports.reexports {
				if re.exportedName == "*" {
					dep := rc.findModuleDepFromFile(file, re.moduleSpec)
					if dep != nil && dep.Status == ModuleResolved && dep.ResolvedPath != "" {
						res := rc.resolveExportedSymbolWithStack(dep.ResolvedPath, exportedName, space, dep.ResolvedLanguage)
						if len(res) > 0 {
							return res
						}
					}
				}
			}
		}
	}
	var out []Symbol
	for _, s := range rc.byFile[file] {
		if s.Name != localName || s.Kind == "file" || s.Kind == "module" {
			continue
		}
		if !tsSymbolInSpace(s, space) {
			continue
		}
		if lang != "" && s.Language != "" && s.Language != lang && !isJSTSLangPair(s.Language, lang) {
			continue
		}
		out = append(out, s)
	}
	if len(out) > 1 {
		var exported []Symbol
		for _, s := range out {
			if s.Exported != nil && *s.Exported {
				exported = append(exported, s)
			}
		}
		if len(exported) > 0 {
			out = exported
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

func (rc *tsResolveContext) resolveExportedSymbolWithStack(file, exportedName string, space tsSpace, lang string) []Symbol {
	key := filepath.ToSlash(file) + "\x00" + exportedName + "\x00" + string(rune('0'+space))
	if rc.reStack[key] {
		return nil
	}
	rc.reStack[key] = true
	defer delete(rc.reStack, key)
	return rc.resolveExportedSymbol(file, exportedName, space, lang)
}

func (rc *tsResolveContext) getExports(file string) *tsExportInfo {
	file = filepath.ToSlash(file)
	if info, ok := rc.exportCache[file]; ok {
		return info
	}
	abs := filepath.Join(rc.root, filepath.FromSlash(file))
	src, err := os.ReadFile(abs)
	if err != nil {
		info := &tsExportInfo{named: map[string]tsExportName{}}
		rc.exportCache[file] = info
		return info
	}
	profile := typescript.Profile()
	if strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".jsx") || strings.HasSuffix(file, ".mjs") || strings.HasSuffix(file, ".cjs") {
		profile = lexical.Profile{ID: "javascript", Name: "JavaScript", Extensions: []string{".js"}}
		// minimal; JS exports parsing is limited
	}
	scanner := lexical.NewScanner(profile)
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	codeLines := make([]string, len(raw))
	segLines := make([][]lexical.Segment, len(raw))
	for i, text := range raw {
		segs := scanner.Scan(text)
		segLines[i] = segs
		codeLines[i] = lexical.CodeText(segs)
	}
	tmp := &tsResolveContext{bindings: map[string]tsBinding{}, localDecls: map[string]tsSpace{}, exportCache: map[string]*tsExportInfo{}, reStack: map[string]bool{}}
	tmp.collectLocalDecls(codeLines)
	info := tmp.parseBindingsAndExports(codeLines, segLines)
	if info.named == nil {
		info.named = map[string]tsExportName{}
	}
	rc.exportCache[file] = info
	return info
}

func (rc *tsResolveContext) scan(codeLines []string, segLines [][]lexical.Segment) {
	for i, code := range codeLines {
		line := i + 1
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		indent := len(code) - len(strings.TrimLeft(code, " \t"))
		rc.popScopes(indent, trimmed)
		if strings.HasPrefix(trimmed, "import ") {
			spec := rc.moduleFromSegments(segLines[i])
			if spec != "" {
				rc.emitImportRefs(trimmed, spec, line)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "export") && strings.Contains(trimmed, " from ") {
			spec := rc.moduleFromSegments(segLines[i])
			if spec != "" {
				rc.emitExportFromRefs(trimmed, spec, line)
			}
			// fall through for export declarations? export from usually stands alone.
			continue
		}
		// Generic params shadowing (basic): only for declarations that introduce type params.
		if tsDeclaresGenericParams(trimmed) {
			if gen := tsGenericParams(trimmed); len(gen) > 0 {
				for _, g := range gen {
					if len(rc.scopes) > 0 {
						rc.scopes[len(rc.scopes)-1].typeVars[g] = true
					}
				}
			}
		}
		// Declarations + scopes
		if m := tsIfaceRe.FindStringSubmatch(trimmed); m != nil {
			rc.pushScope(tsScope{kind: "block", name: m[1], indent: indent})
			if idx := strings.Index(trimmed, " extends "); idx >= 0 {
				rc.emitTypeExpr(trimmed[idx+9:], line, 1, domain.ReferenceInheritance, "interface extends ")
			}
		}
		if m := tsTypeAliasRe.FindStringSubmatch(trimmed); m != nil {
			if idx := strings.Index(trimmed, "="); idx >= 0 {
				rc.emitTypeExpr(trimmed[idx+1:], line, 1, domain.ReferenceType, "type alias ")
			}
		}
		if m := tsClassRe.FindStringSubmatch(trimmed); m != nil {
			rc.pushScope(tsScope{kind: "class", name: m[1], indent: indent})
			rc.emitClassHeritage(trimmed, line)
		}
		if m := tsFuncRe.FindStringSubmatch(trimmed); m != nil {
			rc.pushScope(tsScope{kind: "func", name: m[1], indent: indent})
			rc.emitSignatureTypes(trimmed, line)
		}
		if m := tsVarRe.FindStringSubmatch(trimmed); m != nil {
			if len(rc.scopes) > 0 {
				rc.scopes[len(rc.scopes)-1].valueNames[m[1]] = true
			}
			rc.emitVarAnnotation(trimmed, code, line)
		}
		if rc.currentClass() != "" && indent > 0 && !strings.HasPrefix(trimmed, "class ") && !strings.HasPrefix(trimmed, "interface ") && !strings.HasPrefix(trimmed, "type ") {
			if mm := regexp.MustCompile(`^(?:public|private|protected|static|readonly|abstract|async|get|set|override|\s)*([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:<[^>]*>)?\s*\(([^)]*)\)\s*(?::([^{;=]+))?`).FindStringSubmatch(trimmed); mm != nil && !tsControlWord(mm[1]) {
				rc.pushScope(tsScope{kind: "func", name: mm[1], indent: indent})
				rc.emitParamTypes(mm[2], line)
				if len(mm) > 3 && strings.TrimSpace(mm[3]) != "" {
					rc.emitTypeExpr(strings.TrimSpace(mm[3]), line, 1, domain.ReferenceType, "return type ")
				}
			} else if fm := regexp.MustCompile(`^(?:public|private|protected|static|readonly|override|declare|\s)*([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:\?)?\s*:\s*([^=;]+)`).FindStringSubmatch(trimmed); fm != nil && !tsControlWord(fm[1]) {
				rc.emitTypeExpr(strings.TrimSpace(fm[2]), line, 1, domain.ReferenceType, "property type ")
			}
		}
		rc.scanValueRefs(code, trimmed, line)
	}
}

func tsDeclaresGenericParams(trimmed string) bool {
	// Only treat <...> as introducing generic type params on declaration forms.
	for _, prefix := range []string{"function ", "export function ", "export default function ", "async function ", "export async function ", "class ", "export class ", "export default class ", "abstract class ", "export abstract class ", "interface ", "export interface ", "type ", "export type "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func tsGenericParams(trimmed string) []string {
	idx := strings.Index(trimmed, "<")
	if idx < 0 {
		return nil
	}
	end := strings.Index(trimmed[idx:], ">")
	if end < 0 {
		return nil
	}
	inner := trimmed[idx+1 : idx+end]
	var out []string
	for _, part := range strings.Split(inner, ",") {
		p := strings.TrimSpace(part)
		p = strings.TrimPrefix(p, "const ")
		p = strings.TrimPrefix(p, "in ")
		p = strings.TrimPrefix(p, "out ")
		if ext := strings.Index(p, " extends "); ext >= 0 {
			p = strings.TrimSpace(p[:ext])
		}
		if eq := strings.IndexAny(p, " =\t"); eq >= 0 {
			p = p[:eq]
		}
		p = strings.TrimSpace(p)
		if tsIdentRe2.MatchString(p) && !tsTypeKeyword(p) {
			out = append(out, p)
		}
	}
	return out
}

func (rc *tsResolveContext) emitClassHeritage(trimmed string, line int) {
	if idx := strings.Index(trimmed, " extends "); idx >= 0 {
		rest := trimmed[idx+9:]
		if c := strings.Index(rest, " implements "); c >= 0 {
			rest = rest[:c]
		}
		if c := strings.Index(rest, "{"); c >= 0 {
			rest = rest[:c]
		}
		rc.emitTypeExpr(rest, line, 1, domain.ReferenceInheritance, "class extends ")
	}
	if idx := strings.Index(trimmed, " implements "); idx >= 0 {
		rest := trimmed[idx+12:]
		if c := strings.Index(rest, "{"); c >= 0 {
			rest = rest[:c]
		}
		rc.emitTypeExpr(rest, line, 1, domain.ReferenceInheritance, "implements ")
	}
}

func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depthAngle, depthParen, depthBracket, depthBrace := 0, 0, 0, 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depthAngle++
		case '>':
			if depthAngle > 0 {
				depthAngle--
			}
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case sep:
			if depthAngle == 0 && depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func (rc *tsResolveContext) emitParamTypes(params string, line int) {
	for _, part := range splitTopLevel(params, ',') {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		// Strip default value
		if eq := indexTopLevel(p, '='); eq >= 0 {
			p = strings.TrimSpace(p[:eq])
		}
		// Register simple parameter names as value locals to preserve shadowing semantics.
		namePart := p
		if colon := indexTopLevel(namePart, ':'); colon >= 0 {
			namePart = strings.TrimSpace(namePart[:colon])
		} else {
			namePart = strings.TrimSpace(namePart)
		}
		namePart = strings.TrimPrefix(namePart, "...")
		namePart = strings.TrimSpace(namePart)
		namePart = strings.TrimSuffix(namePart, "?")
		namePart = strings.TrimSpace(namePart)
		if tsIdentRe2.MatchString(namePart) && len(rc.scopes) > 0 {
			rc.scopes[len(rc.scopes)-1].valueNames[namePart] = true
		}
		// Find type annotation colon at top level
		colon := indexTopLevel(p, ':')
		if colon < 0 {
			continue
		}
		typeExpr := strings.TrimSpace(p[colon+1:])
		if typeExpr != "" {
			rc.emitTypeExpr(typeExpr, line, 1, domain.ReferenceType, "parameter type ")
		}
	}
}

func indexTopLevel(s string, ch byte) int {
	depthAngle, depthParen, depthBracket, depthBrace := 0, 0, 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Check for target char BEFORE modifying depth counters
		if c == ch && depthAngle == 0 && depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
			return i
		}
		switch c {
		case '<':
			depthAngle++
		case '>':
			if depthAngle > 0 {
				depthAngle--
			}
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		}
	}
	return -1
}

func (rc *tsResolveContext) emitSignatureTypes(trimmed string, line int) {
	l := strings.Index(trimmed, "(")
	r := -1
	if l >= 0 {
		r = matchingCloseParen(trimmed, l)
	}
	if l >= 0 && r > l {
		rc.emitParamTypes(trimmed[l+1:r], line)
	}
	if r >= 0 {
		rest := trimmed[r+1:]
		colon := indexTopLevel(rest, ':')
		if colon >= 0 {
			ret := strings.TrimSpace(rest[colon+1:])
			if b := indexTopLevel(ret, '{'); b >= 0 {
				ret = ret[:b]
			}
			ret = strings.TrimSpace(ret)
			if ret != "" {
				rc.emitTypeExpr(ret, line, 1, domain.ReferenceType, "return type ")
			}
		}
	}
}

// matchingCloseParen returns the index of the ')' that closes the '(' at open,
// honoring nested parentheses, or -1 when unbalanced. Using the matching close
// parenthesis keeps a one-line function body (for example
// "function f() { g(); }") from being parsed as part of the parameter list.
func matchingCloseParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func (rc *tsResolveContext) emitVarAnnotation(trimmed, code string, line int) {
	// const/let/var name: Type = ...
	if m := regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:const|let|var)\s+[A-Za-z_$][A-Za-z0-9_$]*\s*:\s*([^=;]+)`).FindStringSubmatch(trimmed); m != nil {
		rc.emitTypeExpr(m[1], line, 1, domain.ReferenceType, "variable type ")
	}
	// value as Type / satisfies Type
	for _, loc := range regexp.MustCompile(`\b(?:as|satisfies)\s+([A-Za-z_$][A-Za-z0-9_$<>\[\]|&,\s.?]+)`).FindAllStringSubmatch(code, -1) {
		rc.emitTypeExpr(loc[1], line, 1, domain.ReferenceType, "type assertion ")
	}
}

func (rc *tsResolveContext) emitImportRefs(trimmed, spec string, line int) {
	dep := rc.findModuleDep(spec, line)
	typeOnlyImport := strings.Contains(trimmed, "import type ")
	clause := rc.importClause(trimmed)
	if strings.HasPrefix(clause, "{") {
		for _, nb := range parseTSNamedBindings(clause) {
			local := nb.exported
			if local == "" {
				local = nb.local
			}
			kind := domain.ReferenceImport
			_ = kind
			rc.emitBindingRef(local, nb.local, spec, dep, line, typeOnlyImport || nb.typeOnly)
		}
		return
	}
	if m := regexp.MustCompile(`^\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)$`).FindStringSubmatch(clause); m != nil {
		rc.emitModuleBindingRef(m[1], spec, dep, line, typeOnlyImport)
		return
	}
	if idx := strings.Index(clause, ","); idx >= 0 {
		def := strings.TrimSpace(clause[:idx])
		rest := strings.TrimSpace(clause[idx+1:])
		if def != "" && tsIdentRe2.MatchString(def) {
			rc.emitBindingRef(def, "default", spec, dep, line, typeOnlyImport)
		}
		if strings.HasPrefix(rest, "{") {
			for _, nb := range parseTSNamedBindings(rest) {
				local := nb.exported
				if local == "" {
					local = nb.local
				}
				rc.emitBindingRef(local, nb.local, spec, dep, line, typeOnlyImport || nb.typeOnly)
			}
		} else if m := regexp.MustCompile(`^\*\s+as\s+([A-Za-z_$][A-Za-z0-9_$]*)$`).FindStringSubmatch(rest); m != nil {
			rc.emitModuleBindingRef(m[1], spec, dep, line, typeOnlyImport)
		}
		return
	}
	if tsIdentRe2.MatchString(clause) {
		rc.emitBindingRef(clause, "default", spec, dep, line, typeOnlyImport)
	}
}

func (rc *tsResolveContext) emitModuleBindingRef(name, spec string, dep *ModuleDependency, line int, typeOnly bool) {
	kind := domain.ReferenceImport
	if typeOnly {
		kind = domain.ReferenceType
	}
	ref := rc.baseRef(name, kind, line, 1)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, line, spec), NewModuleResolutionEvidence("namespace/type import " + spec)}
	if dep == nil {
		if !javascript.IsRelative(spec) {
			ref.Resolution = domain.ResolutionExternal
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	ref.Resolution = mapModuleStatus(dep.Status)
	if dep.Status == ModuleResolved && dep.ResolvedPath != "" {
		t := domain.SymbolRef{Name: filepath.Base(dep.ResolvedPath), Kind: "module", File: dep.ResolvedPath, Language: dep.ResolvedLanguage}
		ref.Target = &t
	}
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) emitBindingRef(local, imported, spec string, dep *ModuleDependency, line int, typeOnly bool) {
	kind := domain.ReferenceImport
	if typeOnly {
		kind = domain.ReferenceType
	}
	ref := rc.baseRef(local, kind, line, 1)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, line, spec), NewModuleResolutionEvidence("binding " + local + " <- " + imported)}
	if dep == nil {
		if !javascript.IsRelative(spec) {
			ref.Resolution = domain.ResolutionExternal
			ref.Target = &domain.SymbolRef{Name: imported, Kind: "symbol", Language: domain.LangTypeScript, Container: spec}
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status != ModuleResolved || dep.ResolvedPath == "" {
		ref.Resolution = mapModuleStatus(dep.Status)
		rc.refs = append(rc.refs, ref)
		return
	}
	space := tsSpaceValue
	if typeOnly {
		space = tsSpaceType
	}
	cands := rc.resolveExportedSymbol(dep.ResolvedPath, imported, space, dep.ResolvedLanguage)
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) emitExportFromRefs(trimmed, spec string, line int) {
	dep := rc.findModuleDep(spec, line)
	typeOnly := strings.Contains(trimmed, "export type")
	if strings.Contains(trimmed, "export *") {
		return
	}
	if l := strings.Index(trimmed, "{"); l >= 0 {
		if r := strings.LastIndex(trimmed, "}"); r > l {
			for _, nb := range parseTSNamedBindings(trimmed[l : r+1]) {
				rc.emitBindingRef(nb.local, nb.local, spec, dep, line, typeOnly || nb.typeOnly)
			}
		}
	}
}

func (rc *tsResolveContext) scanValueRefs(code, trimmed string, line int) {
	// Skip pure type declarations.
	if strings.HasPrefix(trimmed, "interface ") || strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "=>") || strings.HasPrefix(trimmed, "export interface ") || (strings.HasPrefix(trimmed, "export type ") && !strings.Contains(trimmed, "{")) {
		// but still allow scanning within? type alias RHS was handled above.
	}
	// new X(...)
	for _, loc := range regexp.MustCompile(`\bnew\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:<[^>]*>)?\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		name := code[loc[2]:loc[3]]
		col := loc[2] + 1
		rc.resolveValueName(name, domain.ReferenceCall, line, col, "constructor "+name)
	}
	// this.method()
	for _, loc := range regexp.MustCompile(`\bthis\.([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		name := code[loc[2]:loc[3]]
		col := loc[2] + 1
		rc.resolveThisMethod(name, line, col)
	}
	// calls ident(...) and obj.method(...)
	for _, loc := range regexp.MustCompile(`([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*)\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		chain := code[loc[2]:loc[3]]
		col := loc[2] + 1
		if tsControlWord(chain) || strings.HasPrefix(strings.TrimSpace(code[max(0, loc[0]-4):loc[0]]), "new") {
			continue
		}
		if chain == "constructor" && rc.currentClass() != "" && strings.Contains(trimmed, "constructor(") {
			continue
		}
		if strings.Contains(chain, ".") {
			parts := strings.Split(chain, ".")
			if parts[0] == "this" {
				continue
			}
			rc.resolveMemberCall(parts, line, col, code)
		} else {
			if rc.isDeclName(trimmed, chain) {
				continue
			}
			if rc.innerScopeShadowsValue(chain) {
				continue
			}
			rc.resolveValueName(chain, domain.ReferenceCall, line, col, "call "+chain)
		}
	}
	// computed member calls obj["m"]() / obj[m]()
	for _, loc := range regexp.MustCompile(`([A-Za-z_$][A-Za-z0-9_$]*)\s*\[[^\]]*\]\s*\(`).FindAllStringSubmatchIndex(code, -1) {
		name := code[loc[2]:loc[3]]
		if tsControlWord(name) {
			continue
		}
		ref := rc.baseRef(name, domain.ReferenceMethod, line, loc[2]+1)
		ref.Resolution = domain.ResolutionDynamic
		ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "computed call receiver "+name)}
		rc.refs = append(rc.refs, ref)
	}
	// Enum/namespace member access without call: X.Y
	for _, loc := range regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\.([A-Za-z_$][A-Za-z0-9_$]*)\b`).FindAllStringSubmatchIndex(code, -1) {
		left := code[loc[2]:loc[3]]
		end := loc[1]
		if end < len(code) && strings.HasPrefix(strings.TrimSpace(code[end:]), "(") {
			continue
		}
		if tsControlWord(left) || left == "this" {
			continue
		}
		if sp, ok := rc.localDecls[left]; ok && (sp == tsSpaceValue || sp == tsSpaceBoth) {
			rc.resolveValueName(left, domain.ReferenceVariable, line, loc[2]+1, "value member access "+left)
		}
	}
}

func (rc *tsResolveContext) innerScopeShadowsValue(name string) bool {
	for i := len(rc.scopes) - 1; i >= 1; i-- {
		if rc.scopes[i].valueNames[name] {
			return true
		}
	}
	return false
}

func (rc *tsResolveContext) isDeclName(trimmed, name string) bool {
	for _, p := range []string{"function " + name, "class " + name, "interface " + name, "type " + name, "enum " + name, "namespace " + name, "const " + name, "let " + name, "var " + name, "export function " + name, "export class " + name, "export interface " + name, "export type " + name, "export enum " + name, "export default function " + name, "export default class " + name, "export const " + name, "async function " + name, "export async function " + name} {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func (rc *tsResolveContext) resolveThisMethod(name string, line, col int) {
	class := rc.currentClass()
	ref := rc.baseRef(name, domain.ReferenceMethod, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewLexicalMethodCallEvidence(rc.filePath, line, "this."+name)}
	if class == "" {
		ref.Resolution = domain.ResolutionDynamic
		rc.refs = append(rc.refs, ref)
		return
	}
	var cands []Symbol
	for _, s := range rc.byFile[rc.filePath] {
		if s.Container == class && s.Name == name && (s.Kind == "method" || s.Kind == "func" || s.Kind == "field") {
			cands = append(cands, s)
		}
	}
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) resolveMemberCall(parts []string, line, col int, code string) {
	if len(parts) < 2 {
		return
	}
	receiver, member := parts[0], parts[len(parts)-1]
	if !rc.innerScopeShadowsValue(receiver) {
		if binding, ok := rc.bindings[receiver]; ok && (binding.kind == "namespace" || (binding.kind == "named" && binding.importedName == "*")) {
			rc.resolveNamespaceMember(binding, member, line, col)
			return
		}
	}
	if strings.Contains(code, "[") && strings.Contains(code, "]") {
		ref := rc.baseRef(member, domain.ReferenceMethod, line, col)
		ref.Resolution = domain.ResolutionDynamic
		ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, "computed member")}
		rc.refs = append(rc.refs, ref)
		return
	}
	ref := rc.baseRef(member, domain.ReferenceMethod, line, col)
	ref.Resolution = domain.ResolutionDynamic
	evidence := "member call on unresolved receiver " + receiver
	if rc.innerScopeShadowsValue(receiver) {
		evidence = "member call on local/shadowed receiver " + receiver
	}
	ref.Evidence = []domain.ReferenceEvidence{NewDynamicDispatchEvidence(rc.filePath, line, evidence)}
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) resolveNamespaceMember(binding tsBinding, member string, line, col int) {
	dep := rc.findModuleDep(binding.moduleSpec, binding.line)
	ref := rc.baseRef(member, domain.ReferenceCall, line, col)
	ref.Evidence = []domain.ReferenceEvidence{NewImportDeclarationEvidence(rc.filePath, binding.line, binding.moduleSpec), NewLexicalCallPatternEvidence(rc.filePath, line, "namespace member "+member)}
	if dep == nil || dep.Status == ModuleExternal {
		ref.Resolution = domain.ResolutionExternal
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status != ModuleResolved || dep.ResolvedPath == "" {
		ref.Resolution = mapModuleStatus(dep.Status)
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.resolveExportedSymbol(dep.ResolvedPath, member, tsSpaceValue, dep.ResolvedLanguage)
	if len(cands) == 0 {
		cands = rc.resolveExportedSymbol(dep.ResolvedPath, member, tsSpaceType, dep.ResolvedLanguage)
	}
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func (rc *tsResolveContext) resolveNamespaceTypeMember(binding tsBinding, member string, kind domain.ReferenceKind, line, col int, detail string) {
	dep := rc.findModuleDep(binding.moduleSpec, binding.line)
	ref := rc.baseRef(member, kind, line, col)
	ref.Evidence = []domain.ReferenceEvidence{
		NewImportDeclarationEvidence(rc.filePath, binding.line, binding.moduleSpec),
		NewLexicalTypeReferenceEvidence(rc.filePath, line, detail),
	}
	if dep == nil {
		if !javascript.IsRelative(binding.moduleSpec) {
			ref.Resolution = domain.ResolutionExternal
		} else {
			ref.Resolution = domain.ResolutionUnresolved
		}
		rc.refs = append(rc.refs, ref)
		return
	}
	if dep.Status != ModuleResolved || dep.ResolvedPath == "" {
		ref.Resolution = mapModuleStatus(dep.Status)
		rc.refs = append(rc.refs, ref)
		return
	}
	cands := rc.resolveExportedSymbol(dep.ResolvedPath, member, tsSpaceType, dep.ResolvedLanguage)
	rc.finishCandidates(&ref, cands)
	rc.refs = append(rc.refs, ref)
}

func tsControlWord(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "function", "class", "return", "throw", "new", "typeof", "instanceof", "delete", "void", "await", "yield", "else", "do", "try", "finally", "break", "continue", "case", "default", "import", "export", "from", "as", "const", "let", "var", "this", "super", "extends", "implements", "interface", "type", "enum", "async", "static", "get", "set", "of", "in", "satisfies":
		return true
	}
	return false
}
