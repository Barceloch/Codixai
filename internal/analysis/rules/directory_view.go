// Package rules contains the deterministic rule catalog of Codixai.
//
// This file provides the file view construction for full directory analysis,
// where every line of a file is treated as part of the analysis, not just
// lines added by a change.
package rules

import (
	"strings"

	"github.com/barceloch/codixai/internal/languages/lexical"
)

// BuildFileView builds a View for a file's full content for directory analysis.
//
// Unlike BuildView which focuses on diff lines (added/removed), this builds
// a view where every line of the file is present in NewLines and marked as
// added, so rules that scan lines can operate on full files.
func BuildFileView(profile lexical.Profile, path string, content string) *View {
	lines := strings.Split(content, "\n")
	// Remove trailing empty string from split if content ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" && len(content) > 0 && content[len(content)-1] == '\n' {
		lines = lines[:len(lines)-1]
	}

	view := &View{
		Path:        path,
		NewSegments: map[int][]lexical.Segment{},
		OldSegments: map[int][]lexical.Segment{},
		NewHunk:     map[int]int{},
		OldHunk:     map[int]int{},
	}

	scanner := lexical.NewScanner(profile)

	for i, lineText := range lines {
		lineNumber := i + 1
		line := lexical.Line{Number: lineNumber, Text: lineText}
		segments := scanner.Scan(lineText)

		view.NewSegments[lineNumber] = segments
		view.OldSegments[lineNumber] = segments
		view.NewHunk[lineNumber] = 0
		view.OldHunk[lineNumber] = 0
		view.NewLines = append(view.NewLines, line)
		view.Added = append(view.Added, line)
		view.OldLines = append(view.OldLines, line)
	}

	return view
}
