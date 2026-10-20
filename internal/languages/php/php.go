// Package php implements the PHP specific extraction of Codixai.
//
// The extraction is syntactic: use statements, markers and empty catch
// blocks are recognized from the text of the changed lines.
package php

import (
	"regexp"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
)

// Profile returns the lexical profile and the syntactic markers of PHP.
func Profile() lexical.Profile {
	return lexical.Profile{
		ID:            domain.LangPHP,
		Name:          "PHP",
		Extensions:    []string{".php"},
		LineComments:  []string{"//", "#"},
		BlockComments: []lexical.BlockComment{{Start: "/*", End: "*/"}},
		Strings: []lexical.StringSpec{
			{Delimiter: "\"", Escape: true},
			{Delimiter: "'", Escape: true},
		},
		RawPrefixes:   []string{},
		RegexLiterals: false,
		Suppressions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bphpcs:\s*(disable|ignore)\b`),
				Description: "phpcs suppression directive",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`\bphpstan-ignore\b`),
				Description: "phpstan suppression directive",
				Scope:       lexical.ScopeComment,
			},
			{
				Pattern:     regexp.MustCompile(`@noinspection\b`),
				Description: "IDE inspection suppression directive",
				Scope:       lexical.ScopeComment,
			},
		},
		Skips: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\bmarkTestSkipped\s*\(`),
				Description: "test skipped at run time",
				Scope:       lexical.ScopeCode,
			},
		},
		Assertions: []lexical.Marker{
			{
				Pattern:     regexp.MustCompile(`\$this->assert[A-Za-z]*\s*\(`),
				Description: "PHPUnit assertion of the test case",
				Scope:       lexical.ScopeCode,
			},
			{
				Pattern:     regexp.MustCompile(`\bassert[A-Za-z]*\s*\(`),
				Description: "PHPUnit assertion function",
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
				Require:     regexp.MustCompile(`(?i)\b(return\s+\$?null|return\s+null|pass)\b`),
			},
		},
		Swallows:  []lexical.Marker{},
		TestPaths: []string{"**/*Test.php", "**/tests/**"},
	}
}

// globalNames holds the classes and the functions of the PHP standard
// library that can be referenced without a namespace.
var globalNames = map[string]bool{}

func init() {
	names := []string{
		"ArrayAccess", "ArrayIterator", "ArrayObject", "Attribute", "BackedEnum",
		"BadFunctionCallException", "BadMethodCallException", "Closure", "Countable",
		"DateInterval", "DateTime", "DateTimeImmutable", "DateTimeInterface",
		"DateTimeZone", "DivisionByZeroError", "DomainException", "Error",
		"ErrorException", "Exception", "Generator", "InvalidArgumentException",
		"JsonException", "JsonSerializable", "LengthException", "LogicException",
		"OutOfBoundsException", "OutOfRangeException", "OverflowException",
		"RangeException", "RuntimeException", "SplStack", "SplObjectStorage",
		"Stringable", "Throwable", "Traversable", "TypeError", "UnexpectedValueException",
		"UnderflowException", "ValueError", "WeakMap",
		"array_change_key_case", "array_filter", "array_key_exists", "array_keys",
		"array_map", "array_merge", "array_reduce", "array_slice", "array_sum",
		"array_unique", "array_values", "call_user_func", "class_exists",
		"count", "date", "define", "dirname", "error_log", "file_exists",
		"file_get_contents", "filter_var", "func_get_args", "function_exists",
		"get_class", "get_object_vars", "gettype", "implode", "in_array",
		"interface_exists", "is_array", "is_bool", "is_callable", "is_float",
		"is_int", "is_null", "is_numeric", "is_object", "is_resource",
		"is_string", "isset", "json_decode", "json_encode", "method_exists",
		"property_exists", "serialize", "spl_autoload_register", "sprintf",
		"str_contains", "str_replace", "strlen", "strpos", "strtolower",
		"strtoupper", "substr", "trait_exists", "trigger_error", "unset",
		"var_dump", "var_export",
	}
	for _, name := range names {
		globalNames[name] = true
	}
}

// IsGlobal reports whether the given name belongs to the global namespace of
// the PHP standard library.
func IsGlobal(name string) bool {
	return globalNames[name]
}

// useStatementRe matches a use statement of a PHP file.
var useStatementRe = regexp.MustCompile(`^\s*use\s+(function\s+|const\s+)?([A-Za-z_\\][A-Za-z0-9_\\]*(\{[^}]*\}?)?)`)

// ExtractImports returns the namespaces referenced by the use statements of
// the given source lines.
//
// Only the code of the line is considered, so a use statement written inside
// a string or a comment is not reported.
func ExtractImports(lines []lexical.Line, segments map[int][]lexical.Segment) []lexical.ImportSpec {
	var imports []lexical.ImportSpec
	for _, line := range lines {
		match := useStatementRe.FindStringSubmatch(lexical.CodeOrText(line, segments))
		if match == nil {
			continue
		}
		if match[1] != "" {
			// Imports of functions and constants do not name a package.
			continue
		}
		namespace := match[2]
		for _, part := range expandGroup(namespace, line.Text) {
			imports = append(imports, lexical.ImportSpec{Module: part, Line: line.Number})
		}
	}
	return imports
}

// expandGroup expands the grouped use syntax, such as "App\{A, B}".
func expandGroup(namespace, line string) []string {
	open := strings.Index(namespace, "{")
	if open < 0 {
		return []string{namespace}
	}
	prefix := namespace[:open]
	close := strings.Index(namespace, "}")
	if close < 0 {
		return []string{namespace}
	}
	body := namespace[open+1 : close]
	var parts []string
	for _, item := range strings.Split(body, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts = append(parts, prefix+item)
	}
	return parts
}

// catchRe matches the header of a catch block.
var catchRe = regexp.MustCompile(`^\s*\}?\s*catch\s*\(.*\)\s*\{\s*$`)

// catchInlineRe matches a catch block with an inline body.
var catchInlineRe = regexp.MustCompile(`^\s*\}?\s*catch\s*\(.*\)\s*\{\s*\}\s*$`)

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
