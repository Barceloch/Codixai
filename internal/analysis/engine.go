// Package analysis implements the analysis engine of Codixai.
//
// The engine reads a change set and applies the deterministic rules of the
// catalog to it. It never executes code of the analyzed repository, it
// never invokes a package manager and it never accesses the network. The
// only files that it reads are the manifests of the repository and, for the
// rules that need it, the content of the changed files.
//
// The diagnostics that the analysis produces are appended to the Diagnostics
// field of the given change set, because the AnalysisRun model of the domain
// has no diagnostics field. The analysis never stops because of a single
// file that cannot be parsed.
package analysis

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/barceloch/codixai/internal/analysis/rules"
	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/languages/registry"
)

// ToolVersion is the version of the analyzer reported by the engine.
const ToolVersion = "0.1.0"

// Engine applies the rule catalog to a change set.
//
// It implements the application.DiffAnalyzer port.
type Engine struct {
	toolVersion string
	now         func() time.Time
}

// The engine implements the port of the application layer.
var _ application.DiffAnalyzer = (*Engine)(nil)

// Option configures an engine.
type Option func(*Engine)

// WithToolVersion sets the version that the engine reports.
func WithToolVersion(version string) Option {
	return func(engine *Engine) {
		engine.toolVersion = version
	}
}

// WithClock sets the clock that the engine uses to stamp the run.
//
// It exists so that the tests can produce a deterministic document.
func WithClock(now func() time.Time) Option {
	return func(engine *Engine) {
		engine.now = now
	}
}

// New returns an engine with the default configuration.
func New(options ...Option) *Engine {
	engine := &Engine{toolVersion: ToolVersion, now: time.Now}
	for _, option := range options {
		option(engine)
	}
	return engine
}

// Analyze applies the rule catalog to the given change set.
//
// The findings are sorted by path, line and rule identifier, so that two
// runs over the same change set produce the same document.
func (e *Engine) Analyze(ctx context.Context, changes *domain.ChangeSet) (*domain.AnalysisRun, error) {
	if changes == nil {
		return nil, fmt.Errorf("the change set is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	diagnostics := make([]domain.Diagnostic, 0, len(changes.Diagnostics))
	diagnostics = append(diagnostics, changes.Diagnostics...)

	var findings []domain.Finding
	for _, file := range changes.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file.IsBinary {
			continue
		}
		language, ok := registry.Detect(file.Path)
		if !ok {
			// The file does not belong to a supported language: no rule
			// applies to it.
			continue
		}
		profile, ok := registry.ProfileFor(language)
		if !ok {
			continue
		}
		change := file
		change.Language = language
		view := rules.BuildView(profile, &change)
		for _, definition := range rules.Catalog() {
			if !definition.Supports(language) {
				continue
			}
			ruleContext := rules.NewContext(changes, profile, &change, view, definition, &diagnostics)
			result, err := definition.Check(ruleContext)
			if err != nil {
				diagnostics = append(diagnostics, domain.Diagnostic{
					Path:     change.Path,
					Message:  fmt.Sprintf("the rule %s could not be applied: %v", definition.ID, err),
					Severity: domain.SeverityWarning,
				})
				continue
			}
			findings = append(findings, result...)
		}
	}

	sortFindings(findings)

	run := &domain.AnalysisRun{
		SchemaVersion: domain.SchemaVersion,
		ToolVersion:   e.toolVersion,
		GeneratedAt:   e.now().UTC().Format(time.RFC3339),
		Origin:        string(changes.Origin),
		Revision:      domain.Revision{Base: changes.Base, Head: changes.Head},
		Findings:      findings,
		Summary:       summarize(findings),
		Rules:         rules.Rules(),
		Limitations:   limitations(),
	}
	if run.Findings == nil {
		run.Findings = []domain.Finding{}
	}
	changes.Diagnostics = diagnostics
	return run, nil
}

// Rules returns the rules of the catalog with their support matrix.
func (e *Engine) Rules() []domain.Rule {
	return rules.Rules()
}

// sortFindings orders the findings by path, line and rule identifier.
func sortFindings(findings []domain.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Location.Path != right.Location.Path {
			return left.Location.Path < right.Location.Path
		}
		if left.Location.StartLine != right.Location.StartLine {
			return left.Location.StartLine < right.Location.StartLine
		}
		if left.RuleID != right.RuleID {
			return left.RuleID < right.RuleID
		}
		return left.Message < right.Message
	})
}

// summarize counts the findings by severity, rule, language and epistemic
// status.
func summarize(findings []domain.Finding) domain.Summary {
	summary := domain.Summary{Total: len(findings)}
	for _, finding := range findings {
		severity := string(finding.Severity)
		if summary.BySeverity == nil {
			summary.BySeverity = map[string]int{}
			summary.ByRule = map[string]int{}
			summary.ByLanguage = map[string]int{}
			summary.ByEpistemicStatus = map[string]int{}
		}
		summary.BySeverity[severity]++
		summary.ByRule[finding.RuleID]++
		summary.ByLanguage[finding.Language]++
		summary.ByEpistemicStatus[string(finding.EpistemicStatus)]++
	}
	return summary
}

// limitations returns the limitations that apply to every run.
func limitations() []string {
	return []string{
		"The analysis is static and non semantic: the findings come from the syntax and from the text of the diff, not from the behavior of the code.",
		"Codixai does not execute any code of the analyzed repository, does not invoke a package manager and does not access the network.",
		"The support level of every rule is at most syntactic in this milestone: no relation is resolved semantically.",
		"The absence of findings does not prove that the change is correct, safe or complete.",
	}
}
