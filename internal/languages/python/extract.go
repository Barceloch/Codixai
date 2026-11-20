package python

import (
	"regexp"
	"strings"

	"github.com/barceloch/codixai/internal/languages/lexical"
)

// Import is an import declaration extracted from Python source.
type Import struct {
	// Module is the imported module path.
	Module string
	// Name is the imported symbol name for from-imports, empty for module imports.
	Name string
	// Alias is the local alias, when present.
	Alias string
	// Relative reports whether the import is relative.
	Relative bool
	// Level is the relative import level (number of leading dots), 0 for absolute.
	Level int
	// Line is the line number.
	Line int
}

// Symbol is a named declaration extracted from Python source.
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

// File is the extracted structure of a Python source file.
type File struct {
	Imports []Import
	Symbols []Symbol
}

var (
	pyClassRe      = regexp.MustCompile(`^class\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	pyDefRe        = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	pyAssignRe     = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*(?::[^=]+)?=`)
	pyImportRe     = regexp.MustCompile(`^import\s+(.+)$`)
	pyFromRe       = regexp.MustCompile(`^from\s*([.]*(?:[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)?|[.]+)\s+import\s+(.+)$`)
	pyDecoratorRe  = regexp.MustCompile(`^@`)
	pyAnnOnlyRe    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*[^=]+$`)
	pySelfAssignRe = regexp.MustCompile(`^self\.([A-Za-z_][A-Za-z0-9_]*)\s*(?::[^=]+)?=`)
)

// Extract parses Python source conservatively and returns imports and symbols.
// It uses lexical segmentation to avoid matching inside strings and comments.
// It does not execute analyzed code.
func Extract(filename string, src []byte) (*File, error) {
	profile := Profile()
	scanner := lexical.NewScanner(profile)
	rawLines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	lines := make([]lexical.Line, 0, len(rawLines))
	segments := make(map[int][]lexical.Segment, len(rawLines))
	for i, text := range rawLines {
		n := i + 1
		lines = append(lines, lexical.Line{Number: n, Text: text})
		segments[n] = scanner.Scan(text)
	}

	f := &File{}
	var stack []classCtx
	var funcIndents []int
	fieldSeen := map[string]bool{}

	for _, line := range lines {
		code := strings.TrimRight(lexical.CodeOrText(line, segments), " \t")
		trimmed := strings.TrimLeft(code, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(code) - len(trimmed)
		// Pop function scopes that are no longer enclosing this line.
		for len(funcIndents) > 0 && indent <= funcIndents[len(funcIndents)-1] && !pyDecoratorRe.MatchString(trimmed) {
			funcIndents = funcIndents[:len(funcIndents)-1]
		}
		// Pop class contexts that are no longer enclosing this line.
		for len(stack) > 0 && indent <= stack[len(stack)-1].indent && !pyClassRe.MatchString(trimmed) && !pyDecoratorRe.MatchString(trimmed) {
			// Decorators can appear at class body indent before def; keep context unless clearly outdented.
			stack = stack[:len(stack)-1]
		}
		// If outdented past all classes, clear.
		for len(stack) > 0 && indent < stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}

		if pyDecoratorRe.MatchString(trimmed) {
			continue
		}

		if m := pyFromRe.FindStringSubmatch(trimmed); m != nil {
			modulePart := strings.TrimSpace(m[1])
			namesPart := strings.TrimSpace(m[2])
			level := 0
			for level < len(modulePart) && modulePart[level] == '.' {
				level++
			}
			mod := strings.TrimPrefix(modulePart, strings.Repeat(".", level))
			relative := level > 0
			f.Imports = append(f.Imports, parseFromNames(filename, mod, namesPart, relative, level, line.Number)...)
			continue
		}
		if m := pyImportRe.FindStringSubmatch(trimmed); m != nil {
			for _, part := range splitImportParts(m[1]) {
				mod, alias := parseImportAlias(part)
				if mod == "" {
					continue
				}
				f.Imports = append(f.Imports, Import{Module: mod, Alias: alias, Relative: strings.HasPrefix(mod, "."), Level: 0, Line: line.Number})
			}
			continue
		}

		if m := pyClassRe.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			container := containerFromStack(stack)
			col := strings.Index(code, "class") + 1
			if col < 1 {
				col = indent + 1
			}
			f.Symbols = append(f.Symbols, Symbol{
				Name:      name,
				Kind:      "type",
				File:      filename,
				Line:      line.Number,
				Column:    col,
				Language:  "python",
				Container: container,
				Signature: "class " + name,
				Exported:  !strings.HasPrefix(name, "_"),
			})
			stack = append(stack, classCtx{name: name, indent: indent})
			continue
		}

		if m := pyDefRe.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			params := strings.TrimSpace(m[2])
			kind := "func"
			container := containerFromStack(stack)
			// Method if directly inside a class body (indent greater than class indent).
			if len(stack) > 0 && indent > stack[len(stack)-1].indent {
				kind = "method"
				container = stack[len(stack)-1].name
			} else {
				// Top-level or outdented function resets class context relevance.
				for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
					stack = stack[:len(stack)-1]
				}
				container = containerFromStack(stack)
			}
			col := strings.Index(code, "def") + 1
			if col < 1 {
				col = indent + 1
			}
			sig := "def " + name + "(" + params + ")"
			if strings.HasPrefix(trimmed, "async def") {
				sig = "async " + sig
			}
			f.Symbols = append(f.Symbols, Symbol{
				Name:      name,
				Kind:      kind,
				File:      filename,
				Line:      line.Number,
				Column:    col,
				Language:  "python",
				Container: container,
				Signature: sig,
				Exported:  !strings.HasPrefix(name, "_"),
			})
			funcIndents = append(funcIndents, indent)
			continue
		}

		// Instance attributes assigned through self inside a method.
		if len(stack) > 0 && len(funcIndents) > 0 {
			if m := pySelfAssignRe.FindStringSubmatch(trimmed); m != nil {
				name := m[1]
				container := stack[len(stack)-1].name
				key := container + "." + name
				if !fieldSeen[key] {
					fieldSeen[key] = true
					f.Symbols = append(f.Symbols, Symbol{
						Name:      name,
						Kind:      "field",
						File:      filename,
						Line:      line.Number,
						Column:    indent + 1,
						Language:  "python",
						Container: container,
						Exported:  !strings.HasPrefix(name, "_"),
					})
				}
				continue
			}
		}

		// Class attributes declared directly in the class body.
		if len(stack) > 0 && len(funcIndents) == 0 && indent > stack[len(stack)-1].indent {
			if name, ok := pyAssignedName(trimmed); ok {
				container := stack[len(stack)-1].name
				key := container + "." + name
				if !fieldSeen[key] {
					fieldSeen[key] = true
					f.Symbols = append(f.Symbols, Symbol{
						Name:      name,
						Kind:      "field",
						File:      filename,
						Line:      line.Number,
						Column:    indent + 1,
						Language:  "python",
						Container: container,
						Exported:  !strings.HasPrefix(name, "_"),
					})
				}
				continue
			}
		}

		// Conservative top-level assignments as variables/constants only at module level.
		if len(stack) == 0 && indent == 0 {
			if name, ok := pyAssignedName(trimmed); ok {
				f.Symbols = append(f.Symbols, Symbol{
					Name:     name,
					Kind:     "var",
					File:     filename,
					Line:     line.Number,
					Column:   1,
					Language: "python",
					Exported: !strings.HasPrefix(name, "_"),
				})
			}
		}
	}
	return f, nil
}

func pyAssignedName(trimmed string) (string, bool) {
	if m := pyAssignRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		if name == "" || name == "_" || isKeyword(name) {
			return "", false
		}
		return name, true
	}
	if m := pyAnnOnlyRe.FindStringSubmatch(trimmed); m != nil {
		name := m[1]
		if name == "" || name == "_" || isKeyword(name) {
			return "", false
		}
		return name, true
	}
	return "", false
}

type classCtx struct {
	name   string
	indent int
}

func containerFromStack(stack []classCtx) string {
	if len(stack) == 0 {
		return ""
	}
	names := make([]string, 0, len(stack))
	for _, c := range stack {
		names = append(names, c.name)
	}
	return strings.Join(names, ".")
}

func splitImportParts(s string) []string {
	// Remove inline comments conservatively.
	if idx := strings.Index(s, "#"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseImportAlias(part string) (string, string) {
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

func parseFromNames(filename, module, namesPart string, relative bool, level, line int) []Import {
	if idx := strings.Index(namesPart, "#"); idx >= 0 {
		namesPart = namesPart[:idx]
	}
	namesPart = strings.TrimSpace(namesPart)
	if strings.HasPrefix(namesPart, "(") {
		namesPart = strings.TrimSuffix(strings.TrimPrefix(namesPart, "("), ")")
	}
	if namesPart == "*" {
		return []Import{{Module: module, Name: "*", Relative: relative, Level: level, Line: line}}
	}
	parts := strings.Split(namesPart, ",")
	out := make([]Import, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		name := fields[0]
		alias := ""
		if len(fields) >= 3 && fields[1] == "as" {
			alias = fields[2]
		}
		if name == "" {
			continue
		}
		out = append(out, Import{Module: module, Name: name, Alias: alias, Relative: relative, Level: level, Line: line})
	}
	return out
}

func isKeyword(s string) bool {
	switch s {
	case "False", "None", "True", "and", "as", "assert", "async", "await", "break", "class", "continue", "def", "del", "elif", "else", "except", "finally", "for", "from", "global", "if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass", "raise", "return", "try", "while", "with", "yield":
		return true
	}
	return false
}
