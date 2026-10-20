// Package javascript implements the JavaScript specific extraction of
// Codixai.
//
// The extraction is syntactic: module specifiers, markers and empty catch
// blocks are recognized from the text of the changed lines with a scanner
// that knows the comment and string syntax of JavaScript.
package javascript

import (
	"regexp"
	"strings"

	"codixai/internal/domain"
	"codixai/internal/languages/lexical"
)

// Profile returns the lexical profile and the syntactic markers of
// JavaScript.
func Profile() lexical.Profile {
	return lexical.Profile{
		ID:            domain.LangJavaScript,
		Name:          "JavaScript",
		Extensions:    []string{".js", ".mjs", ".cjs", ".jsx"},
		LineComments:  []string{"//"},
		BlockComments: []lexical.BlockComment{{Start: "/*", End: "*/"}},
		Strings: []lexical.StringSpec{
			{Delimiter: "`", Escape: true, Multiline: true},
			{Delimiter: "\"", Escape: true},
			{Delimiter: "'", Escape: true},
		},
		RawPrefixes:   []string{},
		RegexLiterals: true,
		Suppressions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\beslint-disable(?:-next-line|-line)?\b`),
				Description: "eslint suppression directive",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bprettier-ignore\b`),
				Description: "prettier suppression directive",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bnolint\b`),
				Description: "nolint analyzer suppression directive",
				Scope:       lexical.ScopeComment,
			},
		},
		Skips: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\b(describe|it|test)\.skip\s*\(`),
				Description: "test suite or test case skipped",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bx(describe|it|test)\s*\(`),
				Description: "test suite or test case disabled by prefix",
				Scope:       lexical.ScopeCode,
			},
		},
		Assertions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bexpect\s*\(`),
				Description: "expect assertion",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bassert\s*(\.\s*[A-Za-z]+)?\s*\(`),
				Description: "assertion call",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bt\.(equal|deepEqual|strictEqual|ok|same)\s*\(`),
				Description: "test runner assertion",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bshould\b`),
				Description: "should style assertion",
				Scope:       lexical.ScopeCode,
			},
		},
		Placeholders: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bTODO\b`),
				Description: "TODO marker",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bFIXME\b`),
				Description: "FIXME marker",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`not\s+implemented`),
				Description: "explicit not implemented marker in a comment",
				Scope:       lexical.ScopeComment,
				Explicit:    true,
			},
			{
				Pattern:     regexp.MustCompile(`not\s+implemented`),
				Description: "explicit not implemented marker in a thrown error",
				Scope:       lexical.ScopeAny,
				Require:     regexp.MustCompile(`(?i)\b(panic|throw|raise)\b`),
				Explicit:    true,
			},
			{
				Pattern:     regexp.MustCompile(`\bplaceholder\b`),
				Description: "placeholder marker",
				Scope:       lexical.ScopeAny,
				Require:     regexp.MustCompile(`(?i)\b(return\s+null|return\s+undefined)\b`),
			},
		},
		Swallows:  []lexical.Marker{},
		TestPaths: []string{"**/*.test.js", "**/*.spec.js", "**/*.test.jsx", "**/*.spec.jsx", "**/__tests__/**"},
	}
}

// builtinModules holds the modules provided by the Node.js runtime.
var builtinModules = map[string]bool{}

func init() {
	names := []string{
		"assert", "async_hooks", "buffer", "child_process", "cluster", "console",
		"constants", "crypto", "dgram", "diagnostics_channel", "dns", "domain",
		"events", "fs", "http", "http2", "https", "inspector", "module", "net",
		"os", "path", "perf_hooks", "process", "punycode", "querystring",
		"readline", "repl", "stream", "string_decoder", "sys", "timers", "tls",
		"trace_events", "tty", "url", "util", "v8", "vm", "wasi",
		"worker_threads", "zlib",
	}
	for _, name := range names {
		builtinModules[name] = true
	}
}

// IsBuiltin reports whether the given module specifier refers to a module
// of the Node.js runtime.
func IsBuiltin(specifier string) bool {
	name := strings.TrimPrefix(specifier, "node:")
	if idx := strings.Index(name, "/"); idx > 0 {
		name = name[:idx]
	}
	return builtinModules[name]
}

// IsRelative reports whether the module specifier points to a local file.
func IsRelative(specifier string) bool {
	return strings.HasPrefix(specifier, "./") || strings.HasPrefix(specifier, "../") ||
		strings.HasPrefix(specifier, "/") || specifier == "." || specifier == ".."
}

// PackageOf returns the package name of a module specifier.
//
// The specifier "lodash/fp" belongs to the package "lodash" and the
// specifier "@scope/pkg/sub" belongs to the package "@scope/pkg".
func PackageOf(specifier string) string {
	if strings.HasPrefix(specifier, "@") {
		parts := strings.Split(specifier, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return specifier
	}
	if idx := strings.Index(specifier, "/"); idx > 0 {
		return specifier[:idx]
	}
	return specifier
}

// importFromRe matches the tail of an import or export declaration.
var importFromRe = regexp.MustCompile(`(?:^|[^\w$.])(?:import|export)\b[^;]*?\bfrom\s*$`)

// ExtractImports returns the module specifiers imported by the given source
// lines.
//
// The specifier is read from the string literals of the line, so a module
// name written inside a comment is not reported.
func ExtractImports(lines []lexical.Line, segments map[int][]lexical.Segment) []lexical.ImportSpec {
	var imports []lexical.ImportSpec
	for _, line := range lines {
		for _, segment := range segments[line.Number] {
			if segment.Kind != lexical.KindString {
				continue
			}
			specifier := strings.Trim(segment.Text, "\"'`")
			if specifier == "" {
				continue
			}
			before := codeBefore(line.Text, segments[line.Number], segment.Start)
			if !isModuleSpecifier(before) {
				continue
			}
			imports = append(imports, lexical.ImportSpec{
				Module:   specifier,
				Relative: IsRelative(specifier),
				Line:     line.Number,
			})
		}
	}
	return imports
}

// codeBefore returns the code of the line that precedes the given offset,
// with the comments replaced by spaces.
func codeBefore(line string, segments []lexical.Segment, offset int) string {
	var b strings.Builder
	for _, segment := range segments {
		if segment.Start >= offset {
			break
		}
		if segment.Kind == lexical.KindComment {
			b.WriteString(strings.Repeat(" ", segment.End-segment.Start))
			continue
		}
		b.WriteString(segment.Text)
	}
	return b.String()
}

// isModuleSpecifier reports whether the code that precedes a string literal
// declares it as an imported module.
func isModuleSpecifier(before string) bool {
	trimmed := strings.TrimRight(before, " \t")
	if trimmed == "" {
		return false
	}
	if importFromRe.MatchString(trimmed) {
		return true
	}
	if regexp.MustCompile(`(?:^|[^\w$.])import\s*$`).MatchString(trimmed) {
		return true
	}
	if regexp.MustCompile(`(?:^|[^\w$.])import\s*\($`).MatchString(trimmed) {
		return true
	}
	if regexp.MustCompile(`(?:^|[^\w$.])require\s*\($`).MatchString(trimmed) {
		return true
	}
	return false
}

// catchRe matches the header of a catch block.
var catchRe = regexp.MustCompile(`^\s*\}?\s*catch\s*(\([^)]*\))?\s*\{\s*$`)

// catchInlineRe matches a catch block with an inline body.
var catchInlineRe = regexp.MustCompile(`^\s*\}?\s*catch\s*(\([^)]*\))?\s*\{\s*\}\s*$`)

// EmptyHandlers returns the numbers of the lines that open an empty catch
// block.
//
// Only the blocks whose closing brace is visible in the given lines are
// reported.
func EmptyHandlers(lines []lexical.Line, segments map[int][]lexical.Segment) []int {
	var found []int
	for i, line := range lines {
		trimmed := strings.TrimSpace(lexical.CodeOrText(line, segments))
		if catchInlineRe.MatchString(trimmed) {
			found = append(found, line.Number)
			continue
		}
		if !catchRe.MatchString(trimmed) {
			continue
		}
		if i+1 < len(lines) && strings.TrimSpace(lexical.CodeOrText(lines[i+1], segments)) == "}" {
			found = append(found, line.Number)
		}
	}
	return found
}
