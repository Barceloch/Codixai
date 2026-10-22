// Package glob matches repository relative paths against glob patterns.
//
// The package holds the matcher used by the configuration of the analyzed
// repository and by the claims of a claim file, so that both share the same
// syntax without depending on each other.
//
// The supported syntax is a focused subset:
//
//   - "*" matches any sequence of characters inside a single path segment;
//   - "**" matches any number of path segments, including none;
//   - "?" matches a single character inside a path segment;
//   - a pattern without a "/" matches the file name at any depth;
//   - a pattern ending with "/" matches everything under that directory.
//
// The matcher is deterministic and never touches the filesystem.
package glob

import (
	"errors"
	"strings"
)

// Matcher matches repository relative paths against a compiled pattern.
type Matcher struct {
	pattern  string
	segments []string
	baseOnly bool
}

// Compile compiles a glob pattern.
//
// An empty pattern is rejected: a pattern that matches nothing is almost
// always a mistake in the caller.
func Compile(pattern string) (*Matcher, error) {
	raw := strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
	directory := strings.HasSuffix(raw, "/")
	normalized := strings.TrimSuffix(raw, "/")
	if normalized == "" {
		return nil, errors.New("the pattern is empty")
	}
	matcher := &Matcher{
		pattern:  pattern,
		segments: strings.Split(normalized, "/"),
	}
	if directory {
		// A pattern ending with "/" matches everything under the
		// directory it names.
		matcher.segments = append(matcher.segments, "**")
	}
	matcher.baseOnly = !strings.Contains(raw, "/")
	if normalized == "**" {
		// "**" alone matches every path, not only the file name.
		matcher.baseOnly = false
	}
	return matcher, nil
}

// Matches reports whether a repository relative path matches the pattern.
func (m *Matcher) Matches(path string) bool {
	clean := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	clean = strings.Trim(clean, "/")
	if clean == "" {
		return false
	}
	if m.baseOnly {
		name := clean
		if index := strings.LastIndex(clean, "/"); index >= 0 {
			name = clean[index+1:]
		}
		return matchSegment(m.segments[0], name)
	}
	return matchSegments(m.segments, strings.Split(clean, "/"))
}

// Pattern returns the pattern the matcher was compiled from.
func (m *Matcher) Pattern() string {
	return m.pattern
}

// Match reports whether a repository relative path matches a glob pattern.
//
// It is the helper used to decide whether a file is excluded from the
// analysis or whether it matches the paths of a claim.
func Match(pattern, path string) (bool, error) {
	matcher, err := Compile(pattern)
	if err != nil {
		return false, err
	}
	return matcher.Matches(path), nil
}

// matchSegments matches the segments of a pattern against the segments of a
// path. A "**" segment matches any number of path segments.
func matchSegments(pattern, parts []string) bool {
	if len(pattern) == 0 {
		return len(parts) == 0
	}
	if pattern[0] == "**" {
		rest := pattern[1:]
		if len(rest) == 0 {
			return true
		}
		for i := 0; i <= len(parts); i++ {
			if matchSegments(rest, parts[i:]) {
				return true
			}
		}
		return false
	}
	if len(parts) == 0 {
		return false
	}
	if !matchSegment(pattern[0], parts[0]) {
		return false
	}
	return matchSegments(pattern[1:], parts[1:])
}

// matchSegment matches a single path segment against a single wildcard
// pattern.
func matchSegment(pattern, name string) bool {
	p, n := 0, 0
	star, resume := -1, 0
	for n < len(name) {
		switch {
		case p < len(pattern) && (pattern[p] == '?' || pattern[p] == name[n]):
			p++
			n++
		case p < len(pattern) && pattern[p] == '*':
			star = p
			resume = n
			p++
		case star >= 0:
			p = star + 1
			resume++
			n = resume
		default:
			return false
		}
	}
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}
	return p == len(pattern)
}
