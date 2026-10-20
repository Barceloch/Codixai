package application

import (
	"context"
	"testing"

	"codixai/internal/domain"
	"codixai/internal/policies"
)

// fakeSource is a change source that always returns the same change set.
type fakeSource struct {
	changes *domain.ChangeSet
	err     error
}

// Changes implements the ChangeSource port.
func (s *fakeSource) Changes(ctx context.Context, origin domain.ChangeOrigin, base string) (*domain.ChangeSet, error) {
	if s.err != nil {
		return nil, s.err
	}
	clone := *s.changes
	clone.Origin = origin
	clone.Base = base
	return &clone, nil
}

// Head implements the ChangeSource port.
func (s *fakeSource) Head(ctx context.Context) (string, error) {
	return "abc123", nil
}

// fakeAnalyzer is an analyzer that always returns the same findings.
type fakeAnalyzer struct {
	findings []domain.Finding
	rules    []domain.Rule
}

// Analyze implements the DiffAnalyzer port.
func (a *fakeAnalyzer) Analyze(ctx context.Context, changes *domain.ChangeSet) (*domain.AnalysisRun, error) {
	return &domain.AnalysisRun{
		SchemaVersion: domain.SchemaVersion,
		ToolVersion:   "test",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Origin:        string(changes.Origin),
		Revision:      domain.Revision{Base: changes.Base, Head: changes.Head},
		Findings:      a.findings,
		Rules:         a.rules,
	}, nil
}

// Rules implements the DiffAnalyzer port.
func (a *fakeAnalyzer) Rules() []domain.Rule { return a.rules }

// finding builds a finding of the given severity.
func finding(id string, severity domain.Severity, path string) domain.Finding {
	return domain.Finding{
		SchemaVersion:   domain.SchemaVersion,
		RuleID:          id,
		RuleVersion:     "1",
		Severity:        severity,
		EpistemicStatus: domain.EpistemicObserved,
		Language:        domain.LangPython,
		Message:         "a finding",
		Location:        domain.Location{Path: path, StartLine: 1, EndLine: 1},
	}
}

// newRequest builds a request over one changed file.
func newRequest(config *policies.Config, findings ...domain.Finding) AuditRequest {
	return AuditRequest{
		Root:   "/tmp/repository",
		Origin: domain.OriginWorktree,
		Source: &fakeSource{changes: &domain.ChangeSet{
			Files: []domain.FileChange{
				{Path: "app.py", Status: domain.FileModified},
				{Path: "vendor/lib.py", Status: domain.FileModified},
			},
		}},
		Analyzer: &fakeAnalyzer{findings: findings},
		Config:   config,
	}
}

func TestRunAuditAppliesTheFailureThreshold(t *testing.T) {
	cases := []struct {
		failOn string
		failed bool
		count  int
	}{
		{policies.FailOnError, false, 0},
		{policies.FailOnWarning, true, 1},
		{policies.FailOnNone, false, 0},
	}
	for _, testCase := range cases {
		config := policies.Defaults()
		config.Audit.FailOn = testCase.failOn
		result, err := RunAudit(context.Background(), newRequest(config,
			finding("CDX-SUP-001", domain.SeverityWarning, "app.py")))
		if err != nil {
			t.Fatalf("the audit failed: %v", err)
		}
		if result.Failed != testCase.failed {
			t.Errorf("with fail_on %s the audit failed = %v, want %v", testCase.failOn, result.Failed, testCase.failed)
		}
		if result.FailingFindings != testCase.count {
			t.Errorf("with fail_on %s the failing findings are %d, want %d", testCase.failOn, result.FailingFindings, testCase.count)
		}
	}
}

func TestRunAuditFailsOnAnErrorFinding(t *testing.T) {
	config := policies.Defaults()
	result, err := RunAudit(context.Background(), newRequest(config,
		finding("CDX-IMP-001", domain.SeverityError, "app.py")))
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if !result.Failed {
		t.Error("the audit must fail when a finding reaches the threshold")
	}
}

func TestRunAuditAppliesTheRuleOverrides(t *testing.T) {
	config := policies.Defaults()
	disabled := false
	config.Audit.Rules["CDX-SUP-001"] = policies.RuleConfig{Enabled: &disabled}
	config.Audit.Rules["CDX-PLH-001"] = policies.RuleConfig{Severity: policies.SeverityError}

	result, err := RunAudit(context.Background(), newRequest(config,
		finding("CDX-SUP-001", domain.SeverityWarning, "app.py"),
		finding("CDX-PLH-001", domain.SeverityWarning, "app.py")))
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if len(result.Run.Findings) != 1 {
		t.Fatalf("the audit reports %d findings, want 1", len(result.Run.Findings))
	}
	if result.Run.Findings[0].RuleID != "CDX-PLH-001" {
		t.Errorf("the audit reports %s, want CDX-PLH-001", result.Run.Findings[0].RuleID)
	}
	if result.Run.Findings[0].Severity != domain.SeverityError {
		t.Errorf("the severity override was not applied, got %s", result.Run.Findings[0].Severity)
	}
	if !result.Failed {
		t.Error("the audit must fail when the overridden severity reaches the threshold")
	}
}

func TestRunAuditAppliesTheSuppressions(t *testing.T) {
	config := policies.Defaults()
	config.Audit.Suppressions = []policies.Suppression{
		{Rule: "CDX-SUP-001", Path: "app.py", Reason: "the analyzer is wrong here"},
	}
	result, err := RunAudit(context.Background(), newRequest(config,
		finding("CDX-SUP-001", domain.SeverityWarning, "app.py"),
		finding("CDX-PLH-001", domain.SeverityWarning, "app.py")))
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if len(result.Run.Findings) != 1 {
		t.Fatalf("the audit reports %d findings, want 1", len(result.Run.Findings))
	}
	if len(result.Suppressed) != 1 {
		t.Fatalf("the audit suppresses %d findings, want 1", len(result.Suppressed))
	}
	if result.Run.Summary.Total != 1 {
		t.Errorf("the summary counts %d findings, want 1", result.Run.Summary.Total)
	}
}

func TestRunAuditExcludesTheConfiguredPaths(t *testing.T) {
	config := policies.Defaults()
	result, err := RunAudit(context.Background(), newRequest(config))
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if result.ChangedFiles != 1 {
		t.Errorf("the audit considers %d files, want 1", result.ChangedFiles)
	}
	if len(result.ExcludedFiles) != 1 || result.ExcludedFiles[0] != "vendor/lib.py" {
		t.Errorf("the excluded files are %v, want [vendor/lib.py]", result.ExcludedFiles)
	}
}

func TestRunAuditRejectsAnIncompleteRequest(t *testing.T) {
	if _, err := RunAudit(context.Background(), AuditRequest{Root: "/tmp"}); err == nil {
		t.Error("the audit must reject a request without a source")
	}
	if _, err := RunAudit(context.Background(), AuditRequest{Source: &fakeSource{}}); err == nil {
		t.Error("the audit must reject a request without an analyzer")
	}
	if _, err := RunAudit(context.Background(), AuditRequest{
		Source:   &fakeSource{},
		Analyzer: &fakeAnalyzer{},
	}); err == nil {
		t.Error("the audit must reject a request without a root")
	}
}
