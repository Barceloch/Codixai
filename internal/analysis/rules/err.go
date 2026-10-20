package rules

import (
	"codixai/internal/domain"
	golang "codixai/internal/languages/go"
	"codixai/internal/languages/javascript"
	"codixai/internal/languages/lexical"
	"codixai/internal/languages/php"
	"codixai/internal/languages/python"
	"codixai/internal/languages/typescript"
)

// checkErrors implements the rule CDX-ERR-001.
func checkErrors(ctx *Context) ([]domain.Finding, error) {
	var findings []domain.Finding
	for _, line := range emptyHandlerLines(ctx) {
		message := "The change adds an empty handler that discards the error."
		findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicObserved, message, line,
			"Handle the error or document why it can be ignored."))
	}
	for _, marker := range ctx.Profile.Swallows {
		for _, line := range lexical.MatchMarker(marker, ctx.View.Added, ctx.View.NewSegments) {
			message := "The change assigns an error to the blank identifier and discards it."
			findings = append(findings, ctx.finding(ctx.Definition, domain.EpistemicInferred, message, line,
				"Handle the error or document why it can be ignored."))
		}
	}
	return findings, nil
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
