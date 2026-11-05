// Package analysis implements the analysis engine of Codixai.
//
// This file provides directory analysis capability: analyzing all files of
// a directory without requiring a Git repository. It reuses the rule catalog
// and the engine infrastructure, but does not simulate Git diffs.
package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/barceloch/codixai/internal/analysis/rules"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/registry"
	"github.com/barceloch/codixai/internal/source"
)

// DirectoryAnalyzer implements directory analysis by reusing the rule catalog.
//
// It does not require a Git repository and does not simulate changes.
// Only rules compatible with full-file analysis run in this mode.
// Rules that require diff comparison (CDX-TST-001) are reported as not
// applicable and never appear as passed or findings.
type DirectoryAnalyzer struct {
	toolVersion string
	now         func() time.Time
}

// NewDirectoryAnalyzer creates a directory analyzer reusing the same rule
// catalog as the diff engine.
//
// The options are applied through an engine, so the analyzer honors
// WithToolVersion and WithClock exactly like the diff engine does.
func NewDirectoryAnalyzer(options ...Option) *DirectoryAnalyzer {
	engine := New(options...)
	return &DirectoryAnalyzer{
		toolVersion: engine.toolVersion,
		now:         engine.now,
	}
}

// AnalyzeDirectory analyzes all files in a FileSet using rules compatible
// with full-file analysis.
//
// Rules that require diff comparison (CDX-TST-001) are listed in
// MissingRules and never executed.
func (a *DirectoryAnalyzer) AnalyzeDirectory(ctx context.Context, fileSet *source.FileSetInfo) (*domain.AnalysisRun, error) {
	if fileSet == nil {
		return nil, fmt.Errorf("the file set info is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var (
		findings    []domain.Finding
		diagnostics []domain.Diagnostic
		analyzed    int
		skipped     = fileSet.Skipped
	)

	// Rules that require diff comparison are not applicable in directory mode.
	missingRuleIDs := notApplicableRuleIDs()

	for _, file := range fileSet.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file.IsBinary {
			skipped++
			continue
		}
		language := file.Language
		if language == "" {
			language, _ = registry.Detect(file.Path)
		}
		if language == "" {
			skipped++
			continue
		}

		profile, ok := registry.ProfileFor(language)
		if !ok {
			skipped++
			continue
		}

		// Build a file change representation that rules can use.
		fileChange := &domain.FileChange{
			Path:     file.Path,
			Language: language,
			Status:   domain.FileAdded,
		}

		// Build a view where all lines are present and marked as added.
		view := rules.BuildFileView(profile, file.Path, file.Content)

		// Run compatible rules.
		for _, definition := range rules.Catalog() {
			if !definition.Supports(language) {
				continue
			}
			if missingRuleIDs[definition.ID] {
				// Rule not applicable in directory mode: record and skip.
				continue
			}

			// Build a ChangeSet with Origin set to directory mode so that
			// rules can detect the analysis mode (e.g. for evidence kind).
			changes := &domain.ChangeSet{
				Root:   fileSet.Root,
				Origin: domain.OriginDirectory,
			}
			ctxRule := rules.NewContext(changes, profile, fileChange, view, definition, &diagnostics)
			result, err := definition.Check(ctxRule)
			if err != nil {
				diagnostics = append(diagnostics, domain.Diagnostic{
					Path:     fileChange.Path,
					Message:  fmt.Sprintf("the rule %s could not be applied: %v", definition.ID, err),
					Severity: domain.SeverityWarning,
				})
				continue
			}
			findings = append(findings, result...)
		}

		// One file is counted once, no matter how many rules ran over it.
		analyzed++
	}

	sortFindings(findings)

	// Build rules catalog excluding not-applicable rules.
	var rulesOutput []domain.Rule
	for _, rule := range rules.Rules() {
		if missingRuleIDs[rule.ID] {
			continue
		}
		rulesOutput = append(rulesOutput, rule)
	}

	// Build missing rules list.
	var missingRules []domain.MissingRule
	for _, rule := range rules.Rules() {
		if missingRuleIDs[rule.ID] {
			missingRules = append(missingRules, domain.MissingRule{
				ID:      rule.ID,
				Version: rule.Version,
				Reason:  "requires diff analysis to compare test states between revisions",
			})
		}
	}

	run := &domain.AnalysisRun{
		SchemaVersion: domain.SchemaVersion,
		ToolVersion:   a.toolVersion,
		GeneratedAt:   a.now().UTC().Format(time.RFC3339),
		Origin:        string(domain.OriginDirectory),
		Revision:      domain.Revision{}, // empty for directory analysis
		Findings:      findings,
		Summary:       summarize(findings),
		Rules:         rulesOutput,
		MissingRules:  missingRules,
		Files: &domain.DirectoryFilesInfo{
			Discovered:    fileSet.Discovered,
			Analyzed:      analyzed,
			Excluded:      fileSet.Excluded,
			ExcludedPaths: fileSet.ExcludedPaths,
			Skipped:       skipped,
		},
		Limitations: directoryLimitations(),
	}
	if run.Findings == nil {
		run.Findings = []domain.Finding{}
	}

	return run, nil
}

// directoryLimitations returns the limitations specific to directory analysis.
func directoryLimitations() []string {
	return []string{
		"The analysis is static and non semantic: the findings come from the syntax and text of files, not from the behavior of the code.",
		"Codixai does not execute any code of the analyzed repository, does not invoke a package manager and does not access the network.",
		"The support level of every rule is at most syntactic in this milestone: no relation is resolved semantically.",
		"The absence of findings does not prove that the directory is correct, safe or complete.",
		"CDX-TST-001 (test deletion/weakening) requires diff analysis and is not applicable to directory analysis.",
		"This analysis reports the current state of files, not changes between revisions.",
		"The analysis.respect_gitignore setting has no effect in directory mode: only the analysis.exclude patterns of the configuration are applied, so a path that only the .gitignore of the directory ignores is analyzed.",
		"Symbolic links to directories are not traversed in this mode: a linked directory is never entered, so the files below it are not analyzed.",
		"The containment check of a path and the read of the file are two separate steps: a process that can modify the analyzed directory concurrently could change a path between the check and the read (a time-of-check to time-of-use race). No exploitation has been demonstrated.",
	}
}

// notApplicableRuleIDs returns the rules that cannot run without a diff.
func notApplicableRuleIDs() map[string]bool {
	return map[string]bool{
		rules.IDTests: true,
	}
}

// Rules returns the rules that directory analysis applies, with their support
// matrix. The rules that require a diff are not part of the result: they are
// reported as not applicable in MissingRules instead.
func (a *DirectoryAnalyzer) Rules() []domain.Rule {
	missing := notApplicableRuleIDs()
	catalog := rules.Rules()
	applicable := make([]domain.Rule, 0, len(catalog))
	for _, rule := range catalog {
		if missing[rule.ID] {
			continue
		}
		applicable = append(applicable, rule)
	}
	return applicable
}
