package lexical

import (
	"strings"
)

// Kind classifies a segment of a source line.
type Kind int

const (
	// KindCode is executable source text.
	KindCode Kind = iota
	// KindString is the interior and delimiters of a string literal.
	KindString
	// KindComment is the interior and delimiters of a comment.
	KindComment
)

// Segment is a homogeneous piece of a source line.
type Segment struct {
	// Kind classifies the segment.
	Kind Kind
	// Text is the content of the segment.
	Text string
	// Start is the byte offset of the segment inside the line.
	Start int
	// End is the byte offset just after the segment.
	End int
}

// Scanner splits source lines into code, string and comment segments.
//
// The scanner keeps the minimal state needed to continue a block comment or
// a multi line string across lines. It never fails: a fragment it cannot
// classify is reported as code.
type Scanner struct {
	profile Profile
	// blockEnd is the terminator of the block comment that is still open.
	blockEnd string
	// stringEnd is the delimiter of the multi line string that is still open.
	stringEnd string
	// stringEscape states whether the open string processes escapes.
	stringEscape bool
}

// NewScanner returns a scanner for the given language profile.
func NewScanner(profile Profile) *Scanner {
	return &Scanner{profile: profile}
}

// Scan splits one line into segments.
//
// The returned segments are ordered and cover the whole line.
func (s *Scanner) Scan(line string) []Segment {
	segments := make([]Segment, 0, 4)
	codeStart := -1

	flushCode := func(end int) {
		if codeStart >= 0 {
			segments = append(segments, Segment{Kind: KindCode, Text: line[codeStart:end], Start: codeStart, End: end})
			codeStart = -1
		}
	}
	add := func(kind Kind, start, end int) {
		flushCode(start)
		if end > start {
			segments = append(segments, Segment{Kind: kind, Text: line[start:end], Start: start, End: end})
		}
	}

	pos := 0
	for pos < len(line) {
		// Continuation of a block comment opened in a previous line.
		if s.blockEnd != "" {
			idx := strings.Index(line[pos:], s.blockEnd)
			if idx < 0 {
				add(KindComment, pos, len(line))
				pos = len(line)
				break
			}
			end := pos + idx + len(s.blockEnd)
			add(KindComment, pos, end)
			pos = end
			s.blockEnd = ""
			continue
		}

		// Continuation of a multi line string opened in a previous line.
		if s.stringEnd != "" {
			end, ok := findStringEnd(line, pos, s.stringEnd, s.stringEscape)
			if !ok {
				add(KindString, pos, len(line))
				pos = len(line)
				break
			}
			add(KindString, pos, end)
			pos = end
			s.stringEnd = ""
			s.stringEscape = false
			continue
		}

		rest := line[pos:]

		// Line comment.
		if prefix := s.matchLineComment(rest); prefix != "" {
			add(KindComment, pos, len(line))
			pos = len(line)
			break
		}

		// Block comment.
		if start, end, ok := s.matchBlockComment(rest); ok {
			if idx := strings.Index(rest[len(start):], end); idx < 0 {
				add(KindComment, pos, len(line))
				pos = len(line)
				s.blockEnd = end
				break
			} else {
				stop := len(start) + idx + len(end)
				add(KindComment, pos, pos+stop)
				pos += stop
				continue
			}
		}

		// String literal.
		if spec, size, ok := s.matchString(rest, codeText(line, codeStart, pos)); ok {
			end, closed := findStringEnd(line, pos+size, spec.Delimiter, spec.Escape)
			if !closed {
				add(KindString, pos, len(line))
				pos = len(line)
				if spec.Multiline {
					s.stringEnd = spec.Delimiter
					s.stringEscape = spec.Escape
				}
				break
			}
			add(KindString, pos, end)
			pos = end
			continue
		}

		// Regular expression literal.
		if s.profile.RegexLiterals && rest[0] == '/' {
			if end, ok := s.matchRegex(line, pos, codeText(line, codeStart, pos)); ok {
				add(KindString, pos, end)
				pos = end
				continue
			}
		}

		// Plain code character.
		if codeStart < 0 {
			codeStart = pos
		}
		pos++
	}
	flushCode(len(line))
	return segments
}

// codeText returns the code text accumulated so far, used to decide whether
// a slash starts a regular expression literal.
func codeText(line string, codeStart, pos int) string {
	if codeStart < 0 {
		return ""
	}
	return line[codeStart:pos]
}

// matchLineComment returns the line comment prefix that matches the text.
func (s *Scanner) matchLineComment(text string) string {
	for _, prefix := range s.profile.LineComments {
		if prefix != "" && strings.HasPrefix(text, prefix) {
			return prefix
		}
	}
	return ""
}

// matchBlockComment returns the block comment delimiters that match.
func (s *Scanner) matchBlockComment(text string) (start string, end string, ok bool) {
	best := ""
	for _, block := range s.profile.BlockComments {
		if block.Start == "" || !strings.HasPrefix(text, block.Start) {
			continue
		}
		if len(block.Start) > len(best) {
			best = block.Start
		}
	}
	if best == "" {
		return "", "", false
	}
	for _, block := range s.profile.BlockComments {
		if block.Start == best {
			return block.Start, block.End, true
		}
	}
	return "", "", false
}

// matchString returns the string specification that opens a literal at the
// beginning of the text.
func (s *Scanner) matchString(text, precedingCode string) (StringSpec, int, bool) {
	best := StringSpec{}
	bestLen := 0
	for _, spec := range s.profile.Strings {
		if spec.Delimiter == "" || !strings.HasPrefix(text, spec.Delimiter) {
			continue
		}
		if len(spec.Delimiter) > bestLen {
			best = spec
			bestLen = len(spec.Delimiter)
		}
	}
	if bestLen == 0 {
		return StringSpec{}, 0, false
	}
	if !best.Escape && s.hasRawPrefix(precedingCode) {
		best.Escape = false
	}
	return best, bestLen, true
}

// hasRawPrefix reports whether the code preceding a string literal ends with
// a prefix that disables escape processing.
func (s *Scanner) hasRawPrefix(code string) bool {
	for _, prefix := range s.profile.RawPrefixes {
		if prefix == "" {
			continue
		}
		if strings.HasSuffix(code, prefix) {
			return true
		}
	}
	return false
}

// matchRegex matches a regular expression literal. It returns the offset
// just after the literal.
func (s *Scanner) matchRegex(line string, pos int, precedingCode string) (int, bool) {
	if !regexAllowed(precedingCode) {
		return 0, false
	}
	i := pos + 1
	inClass := false
	for i < len(line) {
		switch line[i] {
		case '\\':
			i += 2
			continue
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if inClass {
				i++
				continue
			}
			return i + 1, true
		case '\n':
			return 0, false
		}
		i++
	}
	return 0, false
}

// regexAllowed reports whether a slash may start a regular expression
// literal instead of a division operator.
func regexAllowed(precedingCode string) bool {
	trimmed := strings.TrimRight(precedingCode, " \t")
	if trimmed == "" {
		return true
	}
	switch trimmed[len(trimmed)-1] {
	case '(', ',', '=', ':', '[', '!', '&', '|', '?', '{', '}', ';', '+', '-', '*', '%', '^', '~', '<', '>':
		return true
	}
	word := lastWord(trimmed)
	switch word {
	case "return", "typeof", "case", "in", "of", "new", "delete", "void", "do", "else", "yield", "await":
		return true
	}
	return false
}

// lastWord returns the trailing identifier of a code fragment.
func lastWord(code string) string {
	end := len(code)
	start := end
	for start > 0 && isWordByte(code[start-1]) {
		start--
	}
	return code[start:end]
}

// isWordByte reports whether b can be part of an identifier.
func isWordByte(b byte) bool {
	return b == '_' || b == '$' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// findStringEnd returns the offset just after the closing delimiter of a
// string literal. It reports false when the literal is not closed in the
// given text.
func findStringEnd(text string, start int, delimiter string, escape bool) (int, bool) {
	i := start
	for i < len(text) {
		if escape && text[i] == '\\' {
			i += 2
			continue
		}
		if strings.HasPrefix(text[i:], delimiter) {
			return i + len(delimiter), true
		}
		i++
	}
	return len(text), false
}

// CodeText returns the concatenation of the code segments of a line.
func CodeText(segments []Segment) string {
	var b strings.Builder
	for _, segment := range segments {
		if segment.Kind == KindCode {
			b.WriteString(segment.Text)
		}
	}
	return b.String()
}

// CommentText returns the concatenation of the comment segments of a line.
func CommentText(segments []Segment) string {
	var b strings.Builder
	for _, segment := range segments {
		if segment.Kind == KindComment {
			b.WriteString(segment.Text)
		}
	}
	return b.String()
}

// CodeOrText returns the code of a line.
//
// It falls back to the raw text of the line when the scanner did not
// produce any segment for it.
func CodeOrText(line Line, segments map[int][]Segment) string {
	if found, ok := segments[line.Number]; ok && len(found) > 0 {
		return CodeText(found)
	}
	return line.Text
}

// MatchMarker reports the lines in which the marker matches.
//
// The marker scope decides the text that is matched: ScopeAny uses the whole
// line, ScopeCode only the code segments and ScopeComment only the comment
// segments. When the marker declares a Require pattern, the same line must
// match it too.
func MatchMarker(marker Marker, lines []Line, segments map[int][]Segment) []int {
	var matched []int
	for _, line := range lines {
		var text string
		switch marker.Scope {
		case ScopeCode:
			text = CodeText(segments[line.Number])
		case ScopeComment:
			text = CommentText(segments[line.Number])
		default:
			text = line.Text
		}
		if text == "" || !marker.Pattern.MatchString(text) {
			continue
		}
		if marker.Require != nil && !marker.Require.MatchString(line.Text) {
			continue
		}
		matched = append(matched, line.Number)
	}
	return matched
}
