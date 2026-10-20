package rules

import (
	"fmt"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/lexical"
)

// checkTests implements the rule CDX-TST-001.
func checkTests(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	if ctx.View.Status == domain.FileDeleted {
		if ctx.Profile.IsTestPath(ctx.View.Path) {
			line := 0
			if len(ctx.View.Removed) > 0 {
				line = ctx.View.Removed[0].Number
			}
			message := fmt.Sprintf("The change deletes the test file %q.", ctx.View.Path)
			findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, line,
				"Restore the test or explain in the change why it is not needed any more."))
		}
		return findings, nil
	}
	if !ctx.Profile.IsTestPath(ctx.View.Path) {
		return nil, nil
	}
	for _, marker := range ctx.Profile.Skips {
		for _, line := range lexical.MatchMarker(marker, ctx.View.Added, ctx.View.NewSegments) {
			message := fmt.Sprintf("The change disables a test: %s.", marker.Description)
			findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, line,
				"Remove the marker that disables the test or justify why it is disabled."))
		}
	}
	removed, removedLine := assertionLines(ctx.View.Removed, ctx.View.OldSegments, ctx.Profile.Assertions)
	added, _ := assertionLines(ctx.View.Added, ctx.View.NewSegments, ctx.Profile.Assertions)
	if removed > added {
		message := fmt.Sprintf("The change removes %d assertion(s) of the test file and does not add an equivalent one.", removed)
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicInferred, message, removedLine,
			"Confirm that another assertion still covers the same behavior."))
	}
	return findings, nil
}

// assertionLines returns the number of distinct lines that hold an assertion
// and the number of the first one of them.
func assertionLines(lines []lexical.Line, segments map[int][]lexical.Segment, markers []lexical.Marker) (int, int) {
	seen := map[int]bool{}
	first := 0
	for _, marker := range markers {
		for _, line := range lexical.MatchMarker(marker, lines, segments) {
			if seen[line] {
				continue
			}
			seen[line] = true
			if first == 0 || line < first {
				first = line
			}
		}
	}
	return len(seen), first
}
