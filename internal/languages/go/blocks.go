package golang

import (
	"regexp"
	"strings"

	"codixai/internal/languages/lexical"
)

// emptyErrorCheckRe matches the header of an error check.
var emptyErrorCheckRe = regexp.MustCompile(`^\s*if\s+err\s*(!=|==)\s*nil\s*\{\s*$`)

// EmptyErrorBlocks returns the numbers of the lines that open an error check
// with an empty body.
//
// Go has no exception handler. The equivalent pattern is an error check
// whose body is empty. Only the checks whose closing brace is visible in the
// given lines are reported.
func EmptyErrorBlocks(lines []lexical.Line, segments map[int][]lexical.Segment) []int {
	var found []int
	for i, line := range lines {
		trimmed := strings.TrimSpace(lexical.CodeOrText(line, segments))
		if !emptyErrorCheckRe.MatchString(trimmed) {
			continue
		}
		if i+1 < len(lines) && strings.TrimSpace(lexical.CodeOrText(lines[i+1], segments)) == "}" {
			found = append(found, line.Number)
		}
	}
	return found
}
