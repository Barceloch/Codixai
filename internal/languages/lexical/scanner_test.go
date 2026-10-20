package lexical_test

import (
	"regexp"
	"testing"

	"github.com/Barceloch/Codixai/internal/languages/lexical"
	"github.com/Barceloch/Codixai/internal/languages/python"
)

// segmentsOf returns the segments of a line.
func segmentsOf(profile lexical.Profile, line string) []lexical.Segment {
	return lexical.NewScanner(profile).Scan(line)
}

// kindsOf returns the kind of every segment of a line.
func kindsOf(profile lexical.Profile, line string) []lexical.Kind {
	var kinds []lexical.Kind
	for _, segment := range segmentsOf(profile, line) {
		kinds = append(kinds, segment.Kind)
	}
	return kinds
}

// TestScannerSeparatesCodeStringsAndComments checks that the scanner tells
// the code, the strings and the comments apart.
func TestScannerSeparatesCodeStringsAndComments(t *testing.T) {
	profile := python.Profile()
	line := `value = "text"  # comment`
	kinds := kindsOf(profile, line)
	if len(kinds) != 4 {
		t.Fatalf("the line produced %d segments, want 4: %v", len(kinds), kinds)
	}
	if kinds[0] != lexical.KindCode {
		t.Errorf("the first segment is %v, want code", kinds[0])
	}
	if kinds[1] != lexical.KindString {
		t.Errorf("the second segment is %v, want string", kinds[1])
	}
	if kinds[3] != lexical.KindComment {
		t.Errorf("the last segment is %v, want comment", kinds[3])
	}
}

// TestScannerIgnoresMarkersInsideStrings checks that a marker that lives
// inside a string literal is not reported as a comment.
func TestScannerIgnoresMarkersInsideStrings(t *testing.T) {
	profile := python.Profile()
	scanner := lexical.NewScanner(profile)
	segments := scanner.Scan(`message = "# noqa: E501"`)
	if got := lexical.CommentText(segments); got != "" {
		t.Errorf("the comment text is %q, want an empty string", got)
	}
	marker := lexical.Marker{
		Pattern: regexp.MustCompile(`\bnoqa\b`),
		Scope:   lexical.ScopeComment,
	}
	lines := []lexical.Line{{Number: 1, Text: `message = "# noqa: E501"`}}
	segmentsByLine := map[int][]lexical.Segment{1: segments}
	if matched := lexical.MatchMarker(marker, lines, segmentsByLine); len(matched) != 0 {
		t.Errorf("the marker matched the lines %v, want none", matched)
	}
}

// TestScannerFindsMarkersInsideComments checks that a marker that lives in a
// comment is reported.
func TestScannerFindsMarkersInsideComments(t *testing.T) {
	profile := python.Profile()
	scanner := lexical.NewScanner(profile)
	segments := scanner.Scan(`value = 1  # noqa: E501`)
	marker := lexical.Marker{
		Pattern: regexp.MustCompile(`\bnoqa\b`),
		Scope:   lexical.ScopeComment,
	}
	lines := []lexical.Line{{Number: 1, Text: `value = 1  # noqa: E501`}}
	segmentsByLine := map[int][]lexical.Segment{1: segments}
	matched := lexical.MatchMarker(marker, lines, segmentsByLine)
	if len(matched) != 1 || matched[0] != 1 {
		t.Errorf("the marker matched the lines %v, want [1]", matched)
	}
}

// TestScannerContinuesBlockComments checks that a block comment that spans
// several lines stays a comment.
func TestScannerContinuesBlockComments(t *testing.T) {
	profile := lexical.Profile{
		ID:            "test",
		LineComments:  []string{"//"},
		BlockComments: []lexical.BlockComment{{Start: "/*", End: "*/"}},
		Strings:       []lexical.StringSpec{{Delimiter: "\"", Escape: true}},
	}
	scanner := lexical.NewScanner(profile)
	first := scanner.Scan("/* eslint-disable")
	if got := lexical.CommentText(first); got != "/* eslint-disable" {
		t.Errorf("the comment text is %q, want the whole line", got)
	}
	second := scanner.Scan("no-console */")
	if got := lexical.CommentText(second); got != "no-console */" {
		t.Errorf("the comment text is %q, want the whole line", got)
	}
	third := scanner.Scan("code")
	if got := lexical.CommentText(third); got != "" {
		t.Errorf("the comment text is %q, want an empty string", got)
	}
	if got := lexical.CodeText(third); got != "code" {
		t.Errorf("the code text is %q, want %q", got, "code")
	}
}

// TestScannerContinuesMultiLineStrings checks that a multi line string stays
// a string.
func TestScannerContinuesMultiLineStrings(t *testing.T) {
	profile := python.Profile()
	scanner := lexical.NewScanner(profile)
	first := scanner.Scan(`value = """start`)
	if got := lexical.CodeText(first); got != "value = " {
		t.Errorf("the code text is %q, want %q", got, "value = ")
	}
	second := scanner.Scan(`end"""`)
	if got := lexical.CodeText(second); got != "" {
		t.Errorf("the code text is %q, want an empty string", got)
	}
	third := scanner.Scan("value = 1")
	if got := lexical.CodeText(third); got != "value = 1" {
		t.Errorf("the code text is %q, want %q", got, "value = 1")
	}
}

// TestMatchMarkerRequiresBothPatterns checks that the second pattern of a
// marker has to match too.
func TestMatchMarkerRequiresBothPatterns(t *testing.T) {
	marker := lexical.Marker{
		Pattern: regexp.MustCompile(`\bplaceholder\b`),
		Require: regexp.MustCompile(`(?i)\bpass\b`),
		Scope:   lexical.ScopeAny,
	}
	lines := []lexical.Line{
		{Number: 1, Text: "pass  # placeholder"},
		{Number: 2, Text: "# placeholder without a body"},
	}
	matched := lexical.MatchMarker(marker, lines, nil)
	if len(matched) != 1 || matched[0] != 1 {
		t.Errorf("the marker matched the lines %v, want [1]", matched)
	}
}

// TestMatchPath checks the glob patterns of the test paths.
func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/*_test.go", "internal/payment/service_test.go", true},
		{"**/*_test.go", "service_test.go", true},
		{"**/*_test.go", "internal/payment/service.go", false},
		{"**/test_*.py", "tests/test_service.py", true},
		{"**/test_*.py", "tests/service_test.py", false},
		{"**/*Test.php", "tests/ServiceTest.php", true},
		{"**/*.test.js", "src/service.test.js", true},
		{"**/*.spec.ts", "src/service.spec.ts", true},
		{"**/*.spec.ts", "src/service.ts", false},
	}
	for _, testCase := range cases {
		got := lexical.MatchPath(testCase.pattern, testCase.path)
		if got != testCase.want {
			t.Errorf("MatchPath(%q, %q) is %v, want %v", testCase.pattern, testCase.path, got, testCase.want)
		}
	}
}
