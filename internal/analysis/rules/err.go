package rules

import (
	"fmt"

	"github.com/barceloch/codixai/internal/domain"
	golang "github.com/barceloch/codixai/internal/languages/go"
	"github.com/barceloch/codixai/internal/languages/javascript"
	"github.com/barceloch/codixai/internal/languages/lexical"
	"github.com/barceloch/codixai/internal/languages/php"
	"github.com/barceloch/codixai/internal/languages/python"
	"github.com/barceloch/codixai/internal/languages/typescript"
)

// checkErrors implements the rule CDX-ERR-001.
func checkErrors(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	reported := map[int]bool{}
	for _, line := range emptyHandlerLines(ctx) {
		message := "The change adds an empty handler that discards the error."
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, line,
			"Handle the error or document why it can be ignored."))
		reported[line] = true
	}
	for _, marker := range ctx.Profile.Swallows {
		for _, line := range lexical.MatchMarker(marker, ctx.View.Added, ctx.View.NewSegments) {
			message := "The change assigns an error to the blank identifier and discards it."
			findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicInferred, message, line,
				"Handle the error or document why it can be ignored."))
			reported[line] = true
		}
	}
	// The lexical marker only recognizes the names that look like an error.
	// The declaration of the called function states the result type, so a
	// discarded result of a function declared in the package directory is
	// observed directly instead of inferred from the name.
	for _, assignment := range blankErrorAssignments(ctx) {
		if reported[assignment.Line] {
			continue
		}
		message := fmt.Sprintf("The change discards the error returned by %q.", assignment.Callee)
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, assignment.Line,
			"Handle the error or document why it can be ignored."))
		reported[assignment.Line] = true
	}
	return findings, nil
}

// blankErrorAssignments returns the added lines that discard in the blank
// identifier the result of a function that the package directory declares
// with an error result.
//
// The analysis abstains when it cannot read the source of the file or of its
// siblings: without a declaration there is no evidence of the result type.
func blankErrorAssignments(ctx *Context) []golang.BlankErrorAssignment {
	if ctx.Language != domain.LangGo {
		return nil
	}
	if ctx.View.Status == domain.FileDeleted {
		return nil
	}
	source, ok := goSource(ctx)
	if !ok {
		return nil
	}
	siblings, ok := goSiblings(ctx)
	if !ok {
		return nil
	}
	assignments, err := golang.FindBlankErrorAssignments(ctx.View.Path, source, siblings)
	if err != nil {
		return nil
	}
	added := make([]golang.BlankErrorAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		if ctx.View.IsAdded(assignment.Line) {
			added = append(added, assignment)
		}
	}
	return added
}

// emptyHandlerLines returns the lines that open an empty handler.
func emptyHandlerLines(ctx *Context) []int {
	switch ctx.Language {
	case domain.LangGo:
		return golang.EmptyErrorBlocks(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangPython:
		return python.EmptyHandlers(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangPHP:
		return php.EmptyHandlers(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangJavaScript:
		return javascript.EmptyHandlers(ctx.View.NewLines, ctx.View.NewSegments)
	case domain.LangTypeScript:
		return typescript.EmptyHandlers(ctx.View.NewLines, ctx.View.NewSegments)
	}
	return nil
}
