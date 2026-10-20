// Package lexical provides a shared, string and comment aware LINE-LEVEL
// scanner plus the profile type that describes the lexical conventions of a
// language.
//
// Scope and limitations:
//
//   - The scanner is line based. It processes one line at a time and keeps
//     only the minimal state required to continue a block comment or a multi
//     line string across line boundaries.
//   - The scanner does not build a syntax tree, does not resolve names and
//     does not evaluate expressions. Everything derived from it is syntactic
//     evidence, which corresponds to the "syntactic" support level of the
//     Codixai support matrix. It never reaches the "resolved" level.
//   - The scanner is conservative: when it cannot decide what a fragment is,
//     it prefers to treat the fragment as code, and it never reports a marker
//     that lives inside a string literal or inside a comment when the rule
//     asks for code only.
//
// The profile also carries the syntactic markers used by the rule engine
// (suppressions, skips, assertions, placeholders and error swallows). They
// live here so that every language package describes itself in a single
// place and the rule engine stays language agnostic.
package lexical

import (
	"regexp"
	"strings"
)

// Scope restricts the part of a line in which a Marker may match.
type Scope int

const (
	// ScopeAny matches against the whole text of the line.
	ScopeAny Scope = iota
	// ScopeCode matches against code segments only.
	ScopeCode
	// ScopeComment matches against comment segments only.
	ScopeComment
)

// Marker is a syntactic pattern recognized by the rule engine.
type Marker struct {
	// Pattern is the regular expression matched against the line text.
	Pattern *regexp.Regexp
	// Description explains, in English, what the marker means.
	Description string
	// Scope restricts the segments of the line in which the marker may
	// match. ScopeAny matches the whole line.
	Scope Scope
	// Require is an optional second pattern that must match the same line.
	// It keeps broad markers precise.
	Require *regexp.Regexp
	// Explicit states whether the marker is an explicit statement of an
	// unfinished implementation. The rules use it to tell an observed fact
	// apart from an inference.
	Explicit bool
}

// BlockComment describes a block comment delimiter pair.
type BlockComment struct {
	// Start opens the comment.
	Start string
	// End closes the comment.
	End string
}

// StringSpec describes one string literal form of a language.
type StringSpec struct {
	// Delimiter is the literal that opens and closes the string.
	Delimiter string
	// Escape states whether a backslash escapes the next character.
	Escape bool
	// Multiline states whether the string may span several lines.
	Multiline bool
}

// Profile describes the lexical conventions and the syntactic markers of a
// language.
type Profile struct {
	// ID is the language identifier used by the domain model.
	ID string
	// Name is the human readable name of the language.
	Name string
	// Extensions are the file extensions, in lower case and with the dot.
	Extensions []string
	// LineComments are the prefixes that open a comment until end of line.
	LineComments []string
	// BlockComments are the block comment delimiter pairs.
	BlockComments []BlockComment
	// Strings are the string literal forms of the language.
	Strings []StringSpec
	// RawPrefixes are the literal prefixes that disable escape processing,
	// such as the "r" of a Python raw string.
	RawPrefixes []string
	// RegexLiterals states whether the language has regular expression
	// literals that must not be confused with division.
	RegexLiterals bool

	// Suppressions lists the analyzer suppression markers.
	Suppressions []Marker
	// Skips lists the markers that disable or skip a test.
	Skips []Marker
	// Assertions lists the markers that express a test assertion.
	Assertions []Marker
	// Placeholders lists the markers that express an unfinished
	// implementation.
	Placeholders []Marker
	// Swallows lists the markers that discard an error value.
	Swallows []Marker

	// TestPaths are the glob patterns of the files that hold tests.
	TestPaths []string
}

// IsTestPath reports whether the given repository path is a test file.
func (p Profile) IsTestPath(path string) bool {
	for _, pattern := range p.TestPaths {
		if MatchPath(pattern, path) {
			return true
		}
	}
	return false
}

// Line is a source line with its number.
type Line struct {
	// Number is the line number in its side of the diff.
	Number int
	// Text is the content of the line, without the diff marker.
	Text string
}

// ImportSpec is a module reference extracted from source text.
type ImportSpec struct {
	// Module is the imported module or package.
	Module string
	// Relative states whether the import points to the local project.
	Relative bool
	// Line is the line number where the import was found.
	Line int
}

// MatchPath reports whether path matches pattern.
//
// The pattern supports "*" (any sequence of characters except "/"), "**"
// (any sequence of characters, including "/") and "?" (a single character).
// A leading "**/" also matches when the path has no directory part.
func MatchPath(pattern, path string) bool {
	re, err := regexp.Compile(globToRegexp(pattern))
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

// globToRegexp converts a glob pattern into an anchored regular expression.
func globToRegexp(pattern string) string {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
					continue
				}
				b.WriteString(".*")
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return b.String()
}
