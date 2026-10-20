package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/policies"
)

// AuditRequest describes the audit to run.
//
// The use case never touches the command line: it receives the already
// resolved options and the providers that collect the changes and that
// apply the rules.
type AuditRequest struct {
	// Root is the absolute path of the analyzed repository.
	Root string
	// Origin selects the changes to audit.
	Origin domain.ChangeOrigin
	// Base is the base revision, when the origin is base.
	Base string
	// Source collects the changes of the repository.
	Source ChangeSource
	// Analyzer applies the rule catalog to the changes.
	Analyzer DiffAnalyzer
	// Config holds the policy of the repository. A nil value loads the
	// configuration of the repository, and the defaults when the
	// repository has none.
	Config *policies.Config
}

// AuditResult is the outcome of an audit.
type AuditResult struct {
	// Run is the document produced by the analysis.
	Run *domain.AnalysisRun
	// FailOn is the severity that makes the audit fail.
	FailOn string
	// Failed states whether a finding reaches the threshold.
	Failed bool
	// FailingFindings counts the findings that reach the threshold.
	FailingFindings int
	// Suppressed holds the findings that the configuration suppresses.
	Suppressed []domain.Finding
	// ExcludedFiles holds the paths that the configuration excludes from
	// the analysis.
	ExcludedFiles []string
	// ChangedFiles counts the files that the analysis considered.
	ChangedFiles int
}

// RunAudit audits the changes of a repository.
//
// The use case collects the changes with the given source, applies the
// policy of the repository to the findings of the analyzer and decides
// whether the findings reach the configured failure threshold.
//
// The policy is applied after the analysis, because the analyzer knows the
// rules and the policy knows the decisions of the repository: a rule can
// be disabled, its severity can be overridden and a finding can be
// suppressed with an explicit and justified exception.
func RunAudit(ctx context.Context, request AuditRequest) (*AuditResult, error) {
	if request.Source == nil {
		return nil, fmt.Errorf("the audit needs a change source")
	}
	if request.Analyzer == nil {
		return nil, fmt.Errorf("the audit needs a diff analyzer")
	}
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the audit needs the root of the repository")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	config := request.Config
	if config == nil {
		loaded, err := policies.Load(request.Root)
		if err != nil {
			return nil, err
		}
		config = loaded
	}

	changes, err := request.Source.Changes(ctx, request.Origin, request.Base)
	if err != nil {
		return nil, err
	}
	// The change source does not know the root of the repository: the
	// rules need it to read the manifests of the analyzed project.
	changes.Root = request.Root
	excluded := excludedPaths(changes, config)
	changes = excludeFiles(changes, config)

	run, err := request.Analyzer.Analyze(ctx, changes)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("the analyzer did not produce any result")
	}

	findings, suppressed := applyPolicy(run.Findings, config)
	run.Findings = findings
	run.Summary = summarize(findings)

	failOn := config.Audit.FailOn
	if failOn == "" {
		failOn = policies.FailOnError
	}
	result := &AuditResult{
		Run:           run,
		FailOn:        failOn,
		Suppressed:    suppressed,
		ExcludedFiles: excluded,
		ChangedFiles:  len(changes.Files),
	}
	threshold := thresholdRank(failOn)
	for _, finding := range findings {
		if severityRank(finding.Severity) >= threshold && threshold > 0 {
			result.FailingFindings++
		}
	}
	result.Failed = result.FailingFindings > 0
	return result, nil
}

// excludeFiles removes the files that the configuration excludes from the
// analysis.
func excludeFiles(changes *domain.ChangeSet, config *policies.Config) *domain.ChangeSet {
	if changes == nil {
		return changes
	}
	files := make([]domain.FileChange, 0, len(changes.Files))
	for _, file := range changes.Files {
		if config.Excluded(file.Path) {
			continue
		}
		files = append(files, file)
	}
	changes.Files = files
	return changes
}

// excludedPaths returns the paths of the change set that the configuration
// excludes.
//
// The change set no longer holds them, so they are collected before the
// exclusion.
func excludedPaths(changes *domain.ChangeSet, config *policies.Config) []string {
	if changes == nil {
		return nil
	}
	var paths []string
	for _, file := range changes.Files {
		if config.Excluded(file.Path) {
			paths = append(paths, file.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

// applyPolicy applies the rule overrides and the suppressions of the
// configuration to the findings.
//
// It returns the findings that survive the policy and the findings that
// the suppressions remove.
func applyPolicy(findings []domain.Finding, config *policies.Config) ([]domain.Finding, []domain.Finding) {
	kept := make([]domain.Finding, 0, len(findings))
	var suppressed []domain.Finding
	for _, finding := range findings {
		if override, ok := config.Rule(finding.RuleID); ok {
			if !override.IsEnabled() {
				continue
			}
			if override.Severity != "" {
				finding.Severity = domain.Severity(override.Severity)
			}
		}
		if config.IsSuppressed(finding.RuleID, finding.Location.Path) {
			suppressed = append(suppressed, finding)
			continue
		}
		kept = append(kept, finding)
	}
	return kept, suppressed
}

// summarize counts the findings by severity, rule, language and epistemic
// status.
func summarize(findings []domain.Finding) domain.Summary {
	summary := domain.Summary{Total: len(findings)}
	for _, finding := range findings {
		if summary.BySeverity == nil {
			summary.BySeverity = map[string]int{}
			summary.ByRule = map[string]int{}
			summary.ByLanguage = map[string]int{}
			summary.ByEpistemicStatus = map[string]int{}
		}
		summary.BySeverity[string(finding.Severity)]++
		summary.ByRule[finding.RuleID]++
		summary.ByLanguage[finding.Language]++
		summary.ByEpistemicStatus[string(finding.EpistemicStatus)]++
	}
	return summary
}

// severityRank orders the severities.
//
// The ranks are categorical: they express an order, not a measurement.
func severityRank(severity domain.Severity) int {
	switch severity {
	case domain.SeverityInfo:
		return 1
	case domain.SeverityWarning:
		return 2
	case domain.SeverityError:
		return 3
	default:
		return 0
	}
}

// thresholdRank returns the rank of a failure threshold.
//
// The threshold none never reaches a finding.
func thresholdRank(failOn string) int {
	switch strings.ToLower(strings.TrimSpace(failOn)) {
	case policies.FailOnError:
		return severityRank(domain.SeverityError)
	case policies.FailOnWarning:
		return severityRank(domain.SeverityWarning)
	case policies.FailOnNone:
		return 0
	default:
		return severityRank(domain.SeverityError)
	}
}
