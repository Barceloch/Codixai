package rules

import (
	"fmt"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/lexical"
)

// checkSuppressions implements the rule CDX-SUP-001.
func checkSuppressions(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	reported := map[int]bool{}
	for _, marker := range ctx.Profile.Suppressions {
		for _, line := range lexical.MatchMarker(marker, ctx.View.Added, ctx.View.NewSegments) {
			if reported[line] {
				continue
			}
			reported[line] = true
			message := fmt.Sprintf("The change adds an analyzer suppression: %s.", marker.Description)
			findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, line,
				"Remove the suppression or justify why the analyzer must not run here."))
		}
	}
	if ctx.Language == domain.LangGo {
		findings = append(findings, goSuppressionFindings(ctx, ctx.Definition, reported)...)
	}
	return findings, nil
}

// goSuppressionFindings reports the suppression directives that the Go
// parser found in the comments of the changed file.
//
// The directives are confirmed with go/parser when the content of the file
// is available. The lexical scan of the changed lines is the primary source,
// so a finding is still reported when the file cannot be read.
func goSuppressionFindings(ctx *Context, definition Definition, reported map[int]bool) []domain.Finding {
	file, ok := goSourceFile(ctx)
	if !ok {
		return nil
	}
	var findings []domain.Finding
	for _, comment := range file.Comments {
		if reported[comment.Line] || !ctx.View.IsAdded(comment.Line) {
			continue
		}
		marker, ok := suppressionMarker(ctx.Profile, comment.Text)
		if !ok {
			continue
		}
		reported[comment.Line] = true
		message := fmt.Sprintf("The change adds an analyzer suppression: %s.", marker.Description)
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, comment.Line,
			"Remove the suppression or justify why the analyzer must not run here."))
	}
	return findings
}

// suppressionMarker returns the suppression marker that matches a comment.
func suppressionMarker(profile lexical.Profile, text string) (lexical.Marker, bool) {
	for _, marker := range profile.Suppressions {
		if marker.Pattern.MatchString(text) {
			return marker, true
		}
	}
	return lexical.Marker{}, false
}
