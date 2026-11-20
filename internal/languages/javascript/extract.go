package javascript

import (
	"regexp"
	"strings"

	"github.com/barceloch/codixai/internal/languages/lexical"
)

// Import is an import or dependency declaration extracted from JavaScript/TypeScript source.
type Import struct {
	// Module is the imported module specifier.
	Module string
	// Name is the imported binding name when known (named import/export), empty otherwise.
	Name string
	// Alias is the local alias when present.
	Alias string
	// Default reports whether the binding is a default import/export.
	Default bool
	// Namespace reports whether the binding is a namespace import (* as).
	Namespace bool
	// TypeOnly reports whether the import/export is type-only (TypeScript).
	TypeOnly bool
	// Relative reports whether the specifier points to a local file.
	Relative bool
	// Reexport reports whether this is a re-export from another module.
	Reexport bool
	// Kind is a conservative classification: "esm_import", "esm_export", "require", "cjs_export".
	Kind string
	// Line is the line number.
	Line int
}

// Symbol is a named declaration extracted from JavaScript/TypeScript source.
type Symbol struct {
	Name      string
	Kind      string
	File      string
	Line      int
	Column    int
	Language  string
	Container string
	Signature string
	Exported  bool
}

// File is the extracted structure of a JavaScript/TypeScript source file.
type File struct {
	Imports []Import
	Symbols []Symbol
}

// Extract parses JavaScript source conservatively and returns imports/exports and symbols.
// It uses lexical segmentation to avoid matching inside strings and comments.
// It does not execute analyzed code.
func Extract(filename string, src []byte) (*File, error) {
	return ExtractWithProfile(filename, src, Profile())
}

// ExtractWithProfile parses source with the given lexical profile and language id.
func ExtractWithProfile(filename string, src []byte, profile lexical.Profile) (*File, error) {
	scanner := lexical.NewScanner(profile)
	rawLines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	lines := make([]lexical.Line, 0, len(rawLines))
	segments := make(map[int][]lexical.Segment, len(rawLines))
	for i, text := range rawLines {
		n := i + 1
		lines = append(lines, lexical.Line{Number: n, Text: text})
		segments[n] = scanner.Scan(text)
	}
	ex := &extractor{
		filename: filename,
		lang:     profile.ID,
		profile:  profile,
		lines:    lines,
		segments: segments,
		out:      &File{},
	}
	ex.run()
	return ex.out, nil
}

type classCtx struct {
	name   string
	indent int
}

type extractor struct {
	filename string
	lang     string
	profile  lexical.Profile
	lines    []lexical.Line
	segments map[int][]lexical.Segment
	out      *File

	classStack []classCtx
	funcIndent []int
	fieldSeen  map[string]bool
}

var (
	jsFuncDeclRe   = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)`)
	jsClassRe      = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:abstract\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsVarRe        = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:declare\s+)?(const|let|var)\s+(.+)$`)
	jsMethodRe     = regexp.MustCompile(`^(?:export\s+)?(?:(?:public|private|protected|static|readonly|override|async|abstract|get|set)\s+)*\*?\s*([A-Za-z_$#][A-Za-z0-9_$]*)\s*(?:<[^>]*>)?\s*\(([^)]*)\)`)
	jsArrowAssign  = regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?::[^=]+)?=\s*(?:async\s*)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][A-Za-z0-9_$]*\s*=>)`)
	jsImportStmtRe = regexp.MustCompile(`^import\b`)
	jsExportStmtRe = regexp.MustCompile(`^export\b`)
	jsRequireRe    = regexp.MustCompile(`\brequire\s*\(\s*["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `]\s*\)`)
	jsCJSExportRe  = regexp.MustCompile(`^(?:module\.exports|exports)(?:\.([A-Za-z_$][A-Za-z0-9_$]*))?\s*=`)
	jsTSInterface  = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?interface\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsTSType       = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?type\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*[=<]`)
	jsTSEnum       = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?(?:const\s+)?enum\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	jsTSNamespace  = regexp.MustCompile(`^(?:export\s+(?:declare\s+)?)?(?:namespace|module)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
)

func (e *extractor) run() {
	e.fieldSeen = map[string]bool{}
	for i, line := range e.lines {
		_ = i
		code := strings.TrimRight(lexical.CodeOrText(line, e.segments), " \t")
		trimmed := strings.TrimSpace(code)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		indent := len(code) - len(strings.TrimLeft(code, " \t"))
		e.popScopes(indent, trimmed)

		e.scanImportsExports(line.Number, code, trimmed)
		e.scanDeclarations(line.Number, code, trimmed, indent)
	}
}

func (e *extractor) popScopes(indent int, trimmed string) {
	for len(e.funcIndent) > 0 && indent <= e.funcIndent[len(e.funcIndent)-1] && !strings.HasPrefix(trimmed, "@") && !strings.HasPrefix(trimmed, "}") {
		e.funcIndent = e.funcIndent[:len(e.funcIndent)-1]
	}
	for len(e.classStack) > 0 && indent <= e.classStack[len(e.classStack)-1].indent && !jsClassRe.MatchString(trimmed) && !strings.HasPrefix(trimmed, "@") {
		e.classStack = e.classStack[:len(e.classStack)-1]
	}
}

func (e *extractor) container() string {
	if len(e.classStack) == 0 {
		return ""
	}
	names := make([]string, 0, len(e.classStack))
	for _, c := range e.classStack {
		names = append(names, c.name)
	}
	return strings.Join(names, ".")
}

func (e *extractor) exportedPrefix(trimmed string) bool {
	return strings.HasPrefix(trimmed, "export ")
}

func (e *extractor) addSymbol(name, kind string, line, col int, container, signature string, exported bool) {
	if name == "" || name == "_" || jsKeyword(name) {
		return
	}
	if container == "" {
		// top-level non-exported in ESM files may still be module-local; keep Exported as declared.
	}
	e.out.Symbols = append(e.out.Symbols, Symbol{
		Name:      name,
		Kind:      kind,
		File:      e.filename,
		Line:      line,
		Column:    col,
		Language:  e.lang,
		Container: container,
		Signature: signature,
		Exported:  exported,
	})
}

func (e *extractor) addDep(module string, line int, kind string, typeOnly bool) {
	if module == "" {
		return
	}
	e.out.Imports = append(e.out.Imports, Import{
		Module:   module,
		Relative: IsRelative(module),
		TypeOnly: typeOnly,
		Kind:     kind,
		Line:     line,
	})
}

func (e *extractor) scanImportsExports(lineNo int, code, trimmed string) {
	if jsImportStmtRe.MatchString(trimmed) {
		e.parseImportStatement(lineNo, trimmed, e.segments[lineNo])
		return
	}
	if jsExportStmtRe.MatchString(trimmed) {
		e.parseExportStatement(lineNo, trimmed, e.segments[lineNo])
		return
	}
	for _, m := range jsRequireRe.FindAllStringSubmatch(trimmed, -1) {
		if len(m) > 1 {
			e.addDep(m[1], lineNo, "require", false)
		}
	}
	// require() module specifiers live in string literals, which are stripped from code text.
	for _, mod := range e.requireModulesFromSegments(lineNo, code) {
		e.addDep(mod, lineNo, "require", false)
	}
	if m := jsCJSExportRe.FindStringSubmatch(trimmed); m != nil {
		kind := "cjs_export"
		if m[1] != "" {
			e.out.Imports = append(e.out.Imports, Import{Module: "module.exports", Name: m[1], Relative: false, Kind: kind, Line: lineNo})
		} else {
			e.out.Imports = append(e.out.Imports, Import{Module: "module.exports", Default: true, Relative: false, Kind: kind, Line: lineNo})
		}
	}
}

func (e *extractor) moduleFromSegments(segs []lexical.Segment) string {
	// Use the last string literal on import/export/from statements; specifiers appear as strings.
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

func (e *extractor) hasFromKeyword(trimmed string) bool {
	// conservative: from appears in import/export-from forms
	return strings.Contains(trimmed, " from ")
}

func (e *extractor) requireModulesFromSegments(lineNo int, code string) []string {
	segs := e.segments[lineNo]
	var mods []string
	for _, seg := range segs {
		if seg.Kind != lexical.KindString {
			continue
		}
		mod := strings.Trim(seg.Text, "\"'`")
		if mod == "" {
			continue
		}
		before := codeBefore(code, segs, seg.Start)
		if regexp.MustCompile(`(?:^|[^\w$.])require\s*\($`).MatchString(strings.TrimRight(before, " \t")) {
			mods = append(mods, mod)
		}
	}
	return mods
}

// codeBefore is defined in javascript.go.
func (e *extractor) parseImportStatement(lineNo int, trimmed string, segs []lexical.Segment) {
	typeOnly := strings.Contains(trimmed, "import type ") || strings.HasPrefix(trimmed, "import type{") || strings.Contains(trimmed, "import type{")
	module := e.moduleFromSegments(segs)
	if module == "" {
		return
	}
	// Side-effect import has no bindings but still a dependency.
	e.out.Imports = append(e.out.Imports, Import{Module: module, Relative: IsRelative(module), TypeOnly: typeOnly, Kind: "esm_import", Line: lineNo})
}

func (e *extractor) parseExportStatement(lineNo int, trimmed string, segs []lexical.Segment) {
	if module := e.moduleFromSegments(segs); module != "" && (e.hasFromKeyword(trimmed) || strings.Contains(trimmed, "export *")) {
		typeOnly := strings.Contains(trimmed, "export type ") || strings.Contains(trimmed, " type {") || strings.Contains(trimmed, " type *")
		e.out.Imports = append(e.out.Imports, Import{Module: module, Relative: IsRelative(module), TypeOnly: typeOnly, Kind: "esm_export", Reexport: true, Line: lineNo})
		return
	}
	// export { a, b as c } (local re-export list, no from)
	if strings.HasPrefix(trimmed, "export {") || strings.HasPrefix(trimmed, "export type {") {
		typeOnly := strings.Contains(trimmed, "export type")
		names := extractNamedBindings(trimmed)
		for _, nb := range names {
			name := nb.exported
			if name == "" {
				name = nb.local
			}
			if name == "" {
				continue
			}
			e.addSymbol(name, "var", lineNo, 1, "", "export "+name, true)
			_ = typeOnly
		}
	}
}

type namedBinding struct{ local, exported string }

func (nb namedBinding) localOrExported() string {
	if nb.exported != "" {
		return nb.exported
	}
	return nb.local
}

func extractNamedBindings(stmt string) []namedBinding {
	l := strings.Index(stmt, "{")
	r := strings.LastIndex(stmt, "}")
	if l < 0 || r <= l {
		return nil
	}
	inner := stmt[l+1 : r]
	var out []namedBinding
	for _, part := range strings.Split(inner, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "type ")
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		nb := namedBinding{local: fields[0], exported: fields[0]}
		if len(fields) >= 3 && fields[1] == "as" {
			nb.exported = fields[2]
		}
		out = append(out, nb)
	}
	return out
}

// module specifiers are extracted from string segments in scanImportsExports/parseImportStatement.
func (e *extractor) scanInlineClassMembers(lineNo int, trimmed, className string) {
	// Conservative same-line member extraction for compact fixtures/declarations.
	idx := strings.Index(trimmed, "{")
	if idx < 0 {
		return
	}
	body := trimmed[idx+1:]
	// methods: name(...) {  or name(...): type {
	re := regexp.MustCompile(`\b([A-Za-z_$][A-Za-z0-9_$]*)\s*\([^)]*\)\s*(?::[^{;]+)?\{`)
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		name := m[1]
		if name == "function" || name == "if" || name == "for" || name == "while" || name == "catch" || name == "constructor" && false {
			continue
		}
		if name == "constructor" {
			e.addSymbol("constructor", "method", lineNo, 1, className, "constructor(...)", false)
			continue
		}
		if !jsMemberKeyword(name) && !jsKeyword(name) {
			e.addSymbol(name, "method", lineNo, 1, className, name+"(...)", false)
		}
	}
}

func (e *extractor) scanDeclarations(lineNo int, code, trimmed string, indent int) {
	exported := e.exportedPrefix(trimmed)
	col := indent + 1

	if m := jsFuncDeclRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		sig := "function " + name + "(" + strings.TrimSpace(m[2]) + ")"
		e.addSymbol(name, "func", lineNo, firstCol(code, "function", col), e.container(), sig, exported)
		e.funcIndent = append(e.funcIndent, indent)
		return
	}

	if m := jsArrowAssign.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		e.addSymbol(name, "func", lineNo, firstCol(code, name, col), e.container(), name+" = (...) =>", exported)
		e.funcIndent = append(e.funcIndent, indent)
		return
	}

	if m := jsClassRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		sig := "class " + name
		if strings.Contains(trimmed, " implements ") {
			if idx := strings.Index(trimmed, " implements "); idx >= 0 {
				rest := trimmed[idx+len(" implements "):]
				rest = strings.TrimSuffix(strings.TrimSpace(rest), "{")
				rest = strings.TrimSpace(rest)
				if rest != "" {
					sig += " implements " + rest
				}
			}
		}
		if strings.Contains(trimmed, " extends ") && !strings.Contains(sig, "extends") {
			if idx := strings.Index(trimmed, " extends "); idx >= 0 {
				rest := trimmed[idx+len(" extends "):]
				for _, cut := range []string{" implements ", "{"} {
					if c := strings.Index(rest, cut); c >= 0 {
						rest = rest[:c]
					}
				}
				rest = strings.TrimSpace(rest)
				if rest != "" {
					sig += " extends " + rest
				}
			}
		}
		e.addSymbol(name, "type", lineNo, firstCol(code, "class", col), e.container(), sig, exported)
		e.classStack = append(e.classStack, classCtx{name: name, indent: indent})
		e.scanInlineClassMembers(lineNo, trimmed, name)
		return
	}

	// TypeScript declarations
	if e.lang == "typescript" || e.lang == "ts" || strings.Contains(e.profile.Name, "TypeScript") {
		if m := jsTSInterface.FindStringSubmatch(trimmed); m != nil {
			e.addSymbol(m[1], "type", lineNo, firstCol(code, "interface", col), e.container(), "interface "+m[1], exported)
			return
		}
		if m := jsTSType.FindStringSubmatch(trimmed); m != nil {
			e.addSymbol(m[1], "type", lineNo, firstCol(code, "type", col), e.container(), "type "+m[1], exported)
			return
		}
		if m := jsTSEnum.FindStringSubmatch(trimmed); m != nil {
			e.addSymbol(m[1], "type", lineNo, firstCol(code, "enum", col), e.container(), "enum "+m[1], exported)
			return
		}
		if m := jsTSNamespace.FindStringSubmatch(trimmed); m != nil {
			e.addSymbol(m[1], "module", lineNo, col, e.container(), "namespace "+m[1], exported)
			return
		}
	}

	// Class members: methods/fields when inside a class and indented past class.
	if len(e.classStack) > 0 && indent > e.classStack[len(e.classStack)-1].indent {
		container := e.classStack[len(e.classStack)-1].name
		if strings.HasPrefix(trimmed, "constructor") && strings.Contains(trimmed, "(") {
			e.addSymbol("constructor", "method", lineNo, firstCol(code, "constructor", col), container, "constructor(...)", false)
			e.funcIndent = append(e.funcIndent, indent)
			return
		}
		if m := jsMethodRe.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			if !jsMemberKeyword(name) && !strings.HasPrefix(name, "#") {
				// Avoid matching control flow.
				if !isControlFlow(trimmed) {
					sig := name + "(" + strings.TrimSpace(m[2]) + ")"
					e.addSymbol(name, "method", lineNo, firstCol(code, name, col), container, sig, false)
					if strings.Contains(trimmed, "{") || !strings.HasSuffix(trimmed, ";") {
						e.funcIndent = append(e.funcIndent, indent)
					}
					return
				}
			}
		}
		// Class field: name = ... or name: type / name;
		if m := regexp.MustCompile(`^([A-Za-z_$#][A-Za-z0-9_$]*)\s*(?:[?!])?\s*(?::[^=;{]+)?(?:=|;)`).FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			key := container + "." + name
			if !e.fieldSeen[key] && !jsMemberKeyword(name) {
				e.fieldSeen[key] = true
				e.addSymbol(name, "field", lineNo, firstCol(code, name, col), container, "", false)
			}
			return
		}
	}

	if m := jsVarRe.FindStringSubmatch(trimmed); m != nil {
		kind := "var"
		rest := strings.TrimSpace(m[2])
		// simple first declarator: name = or name: or name,
		name := leadingIdent(rest)
		if name != "" {
			sig := m[1] + " " + name
			e.addSymbol(name, kind, lineNo, firstCol(code, name, col), e.container(), sig, exported)
		}
		return
	}
}

func firstCol(code, needle string, fallback int) int {
	idx := strings.Index(code, needle)
	if idx < 0 {
		return fallback
	}
	return idx + 1
}

func leadingIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// destructuring not supported conservatively
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return ""
	}
	re := regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}

func isControlFlow(trimmed string) bool {
	for _, p := range []string{"if (", "if(", "for (", "for(", "while (", "while(", "switch (", "switch(", "catch (", "catch("} {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func jsKeyword(name string) bool {
	switch name {
	case "break", "case", "catch", "class", "const", "continue", "debugger", "default", "delete", "do", "else", "export", "extends", "finally", "for", "function", "if", "import", "in", "instanceof", "new", "return", "super", "switch", "this", "throw", "try", "typeof", "var", "void", "while", "with", "yield", "let", "static", "enum", "implements", "interface", "package", "private", "protected", "public", "await", "async", "of", "as", "from", "type":
		return true
	}
	return false
}

func jsMemberKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "catch", "return", "throw", "else", "do", "try", "finally", "new", "delete", "typeof", "void", "await", "yield", "super", "this":
		return true
	}
	return false
}
