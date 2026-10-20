package rules

import (
	"fmt"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/lexical"
)

// checkPlaceholders implements the rule CDX-PLH-001.
func checkPlaceholders(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	reported := map[int]bool{}
	// The explicit markers are checked first, so that a line that holds
	// both an explicit marker and a generic one is reported as explicit.
	for _, explicit := range []bool{true, false} {
		for _, marker := range ctx.Profile.Placeholders {
			if marker.Explicit != explicit {
				continue
			}
			for _, line := range lexical.MatchMarker(marker, ctx.View.Added, ctx.View.NewSegments) {
				if reported[line] {
					continue
				}
				reported[line] = true
				status := domain.EpistemicInferred
				if marker.Explicit {
					status = domain.EpistemicObserved
				}
				message := fmt.Sprintf("The change adds a placeholder: %s.", marker.Description)
				findings = append(findings, ctx.finding(ctx.Definition, status, message, line,
					"Implement the behavior or link the task that tracks it."))
			}
		}
	}
	return findings, nil
}
