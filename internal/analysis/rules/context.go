package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/languages/lexical"
)

// View is the diff oriented view of one changed file.
//
// It holds the lines that the change adds and removes, with the segments
// that the lexical scanner found in each one of them.
type View struct {
	// Path is the path of the changed file.
	Path string
	// Language is the language identifier of the file.
	Language string
	// Status is the change status of the file.
	Status string
	// Added are the lines that the change introduces, with their number in
	// the new side of the diff.
	Added []lexical.Line
	// Removed are the lines that the change removes, with their number in
	// the old side of the diff.
	Removed []lexical.Line
	// NewLines are the new side lines of the hunks, in order.
	NewLines []lexical.Line
	// OldLines are the old side lines of the hunks, in order.
	OldLines []lexical.Line
	// NewSegments holds the segments of the new side lines, by line number.
	NewSegments map[int][]lexical.Segment
	// OldSegments holds the segments of the old side lines, by line number.
	OldSegments map[int][]lexical.Segment
	// NewHunk holds the index of the hunk that holds a new side line.
	NewHunk map[int]int
	// OldHunk holds the index of the hunk that holds an old side line.
	OldHunk map[int]int
}

// BuildView builds the diff oriented view of a changed file.
func BuildView(profile lexical.Profile, file *domain.FileChange) *View {
	view := &View{
		Path:        file.Path,
		Language:    file.Language,
		Status:      file.Status,
		NewSegments: map[int][]lexical.Segment{},
		OldSegments: map[int][]lexical.Segment{},
		NewHunk:     map[int]int{},
		OldHunk:     map[int]int{},
	}
	newScanner := lexical.NewScanner(profile)
	oldScanner := lexical.NewScanner(profile)
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			switch line.Kind {
			case domain.LineContext:
				new := lexical.Line{Number: line.NewLine, Text: line.Content}
				old := lexical.Line{Number: line.OldLine, Text: line.Content}
				view.NewSegments[new.Number] = newScanner.Scan(new.Text)
				view.OldSegments[old.Number] = oldScanner.Scan(old.Text)
				view.NewHunk[new.Number] = hunk.Index
				view.OldHunk[old.Number] = hunk.Index
				view.NewLines = append(view.NewLines, new)
				view.OldLines = append(view.OldLines, old)
			case domain.LineAdded:
				added := lexical.Line{Number: line.NewLine, Text: line.Content}
				view.NewSegments[added.Number] = newScanner.Scan(added.Text)
				view.NewHunk[added.Number] = hunk.Index
				view.NewLines = append(view.NewLines, added)
				view.Added = append(view.Added, added)
			case domain.LineRemoved:
				removed := lexical.Line{Number: line.OldLine, Text: line.Content}
				view.OldSegments[removed.Number] = oldScanner.Scan(removed.Text)
				view.OldHunk[removed.Number] = hunk.Index
				view.OldLines = append(view.OldLines, removed)
				view.Removed = append(view.Removed, removed)
			}
		}
	}
	return view
}

// IsAdded reports whether a line of the new side is introduced by the change.
func (v *View) IsAdded(line int) bool {
	_, ok := v.NewHunk[line]
	if !ok {
		return false
	}
	for _, added := range v.Added {
		if added.Number == line {
			return true
		}
	}
	return false
}

// Context carries everything that a rule needs to analyze one changed file.
type Context struct {
	// Changes is the change set under analysis.
	Changes *domain.ChangeSet
	// Language is the language identifier of the file.
	Language string
	// Profile is the lexical profile of the language.
	Profile lexical.Profile
	// Definition is the rule that is being applied.
	Definition Definition
	// File is the change of the file.
	File *domain.FileChange
	// View is the diff oriented view of the file.
	View *View
	// Root is the absolute path of the analyzed repository.
	Root string
	// Revision identifies the compared endpoints of the diff.
	Revision domain.Revision
	// Diagnostics collects the problems that the analysis found.
	Diagnostics *[]domain.Diagnostic
}

// NewContext builds the context of one changed file.
func NewContext(changes *domain.ChangeSet, profile lexical.Profile, file *domain.FileChange, view *View, definition Definition, diagnostics *[]domain.Diagnostic) *Context {
	return &Context{
		Changes:     changes,
		Language:    file.Language,
		Profile:     profile,
		Definition:  definition,
		File:        file,
		View:        view,
		Root:        changes.Root,
		Revision:    domain.Revision{Base: changes.Base, Head: changes.Head},
		Diagnostics: diagnostics,
	}
}

// Diagnostic records a problem that the analysis found without stopping it.
func (c *Context) Diagnostic(message string) {
	if c.Diagnostics == nil {
		return
	}
	*c.Diagnostics = append(*c.Diagnostics, domain.Diagnostic{
		Path:     c.View.Path,
		Message:  message,
		Severity: domain.SeverityWarning,
	})
}

// supportLevel returns the support level of the rule for the language of the
// file.
func (c *Context) supportLevel(definition Definition) domain.SupportLevel {
	return definition.Languages[c.Language]
}

// evidence returns the evidence of a finding placed in a line of the new
// side of the diff.
func (c *Context) evidence(line int) domain.Evidence {
	hunk, ok := c.View.NewHunk[line]
	if !ok {
		hunk = 0
	}
	return domain.Evidence{
		Kind: "diff_hunk",
		Ref:  fmt.Sprintf("hunk:%s:%d", c.View.Path, hunk),
	}
}

// finding builds a finding of the given rule.
func (c *Context) finding(definition Definition, status domain.EpistemicStatus, message string, line int, recommendation string) domain.Finding {
	location := domain.Location{Path: c.View.Path, StartLine: line, EndLine: line}
	finding := domain.Finding{
		SchemaVersion:   domain.SchemaVersion,
		RuleID:          definition.ID,
		RuleVersion:     definition.Version,
		Severity:        definition.Severity,
		EpistemicStatus: status,
		Language:        c.Language,
		Message:         message,
		Location:        location,
		Revision:        c.Revision,
		Evidence:        []domain.Evidence{c.evidence(line)},
		Support: domain.Support{
			Level:       c.supportLevel(definition),
			Method:      definition.Method,
			Limitations: definition.Limitations[c.Language],
		},
		Recommendation: recommendation,
	}
	return finding
}

// directory returns the directory of the changed file inside the repository.
func (c *Context) directory() string {
	return filepath.Dir(filepath.ToSlash(c.View.Path))
}

// siblingPath returns the path of a file of the same directory, relative to
// the repository root.
func (c *Context) siblingPath(name string) string {
	directory := c.directory()
	if directory == "." || directory == "" {
		return name
	}
	return strings.TrimSuffix(directory, "/") + "/" + name
}
