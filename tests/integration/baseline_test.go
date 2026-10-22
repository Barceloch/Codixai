package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/baseline"
	"github.com/barceloch/codixai/internal/cli"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/policies"
)

// baselineSource is a change source that always returns the same change set.
type baselineSource struct {
	changes *domain.ChangeSet
}

// Changes implements the application.ChangeSource port.
func (s *baselineSource) Changes(ctx context.Context, origin domain.ChangeOrigin, base string) (*domain.ChangeSet, error) {
	clone := *s.changes
	clone.Origin = origin
	clone.Base = base
	return &clone, nil
}

// Head implements the application.ChangeSource port.
func (s *baselineSource) Head(ctx context.Context) (string, error) { return "abc123", nil }

// baselineAnalyzer is an analyzer that always returns the same findings.
type baselineAnalyzer struct {
	findings []domain.Finding
}

// Analyze implements the application.DiffAnalyzer port.
func (a *baselineAnalyzer) Analyze(ctx context.Context, changes *domain.ChangeSet) (*domain.AnalysisRun, error) {
	return &domain.AnalysisRun{
		SchemaVersion: domain.SchemaVersion,
		ToolVersion:   "test",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Origin:        string(changes.Origin),
		Findings:      a.findings,
	}, nil
}

// Rules implements the application.DiffAnalyzer port.
func (a *baselineAnalyzer) Rules() []domain.Rule { return nil }

// baselineFinding builds a warning finding on a line of a file.
func baselineFinding(ruleID, path string, line int) domain.Finding {
	return domain.Finding{
		SchemaVersion:   domain.SchemaVersion,
		RuleID:          ruleID,
		RuleVersion:     "1",
		Severity:        domain.SeverityWarning,
		EpistemicStatus: domain.EpistemicObserved,
		Message:         "a finding",
		Location:        domain.Location{Path: path, StartLine: line, EndLine: line},
	}
}

// TestRunAuditKeepsTheKnownFindingsOutOfTheThreshold checks the application
// contract of the baseline: a known finding is not reported and does not
// fail the audit, and a new finding does.
func TestRunAuditKeepsTheKnownFindingsOutOfTheThreshold(t *testing.T) {
	root := t.TempDir()
	changes := &domain.ChangeSet{
		Files: []domain.FileChange{{
			Path:   "app.py",
			Status: domain.FileModified,
			Hunks: []domain.Hunk{{Index: 0, Lines: []domain.DiffLine{
				{Kind: domain.LineAdded, NewLine: 1, Content: "    return a  # noqa"},
				{Kind: domain.LineAdded, NewLine: 2, Content: "    # TODO: b"},
			}}},
		}},
	}
	known := baselineFinding("CDX-SUP-001", "app.py", 1)
	fresh := baselineFinding("CDX-PLH-001", "app.py", 2)

	if err := baseline.Save(filepath.Join(root, policies.DefaultBaseline), &baseline.File{
		SchemaVersion: baseline.SchemaVersion,
		CreatedAt:     "2026-01-01T00:00:00Z",
		ToolVersion:   "test",
		Findings:      baseline.Entries([]domain.Finding{known}, changes),
	}); err != nil {
		t.Fatalf("cannot write the baseline: %v", err)
	}

	config := policies.Defaults()
	config.Audit.FailOn = policies.FailOnWarning
	request := application.AuditRequest{
		Root:     root,
		Origin:   domain.OriginWorktree,
		Source:   &baselineSource{changes: changes},
		Analyzer: &baselineAnalyzer{findings: []domain.Finding{known, fresh}},
		Config:   config,
	}

	result, err := application.RunAudit(context.Background(), request)
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if result.KnownFindings != 1 || len(result.Known) != 1 {
		t.Fatalf("the audit knows %d findings, want 1", result.KnownFindings)
	}
	if result.Known[0].RuleID != "CDX-SUP-001" {
		t.Errorf("the known finding is %s, want CDX-SUP-001", result.Known[0].RuleID)
	}
	if len(result.Run.Findings) != 1 || result.Run.Findings[0].RuleID != "CDX-PLH-001" {
		t.Fatalf("the document holds %+v, want only the new finding", result.Run.Findings)
	}
	if result.FailingFindings != 1 || !result.Failed {
		t.Errorf("the audit fails with %d findings = %v, want 1 and true",
			result.FailingFindings, result.Failed)
	}

	// The baseline that is being created is never applied to its own run.
	request.SkipBaseline = true
	result, err = application.RunAudit(context.Background(), request)
	if err != nil {
		t.Fatalf("the audit without the baseline failed: %v", err)
	}
	if result.KnownFindings != 0 {
		t.Errorf("the audit without the baseline knows %d findings, want 0", result.KnownFindings)
	}
	if len(result.Run.Findings) != 2 {
		t.Errorf("the audit without the baseline reports %d findings, want 2", len(result.Run.Findings))
	}
}

// headRevision returns the revision of HEAD of a repository.
func headRevision(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_SYSTEM="+os.DevNull,
	)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("cannot read the revision of HEAD: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// TestBaselineCreateMakesTheKnownFindingsQuiet is the end to end contract of
// the baseline: the command records the debt of the repository, the audit
// stops failing on it and a new defect still fails.
func TestBaselineCreateMakesTheKnownFindingsQuiet(t *testing.T) {
	root := newRepository(t)
	base := headRevision(t, root)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "add the findings")

	code, _, stderr := execute(t, "--root", root, "audit", "--diff", "--base", base)
	if code != cli.ExitFindings {
		t.Fatalf("the audit of the debt exits with %d, want %d: %s", code, cli.ExitFindings, stderr)
	}

	code, stdout, stderr := execute(t, "--root", root, "baseline", "create", "--base", base)
	if code != cli.ExitOK {
		t.Fatalf("baseline create exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stdout, "baseline.json") {
		t.Errorf("the command does not report the path it wrote:\n%s", stdout)
	}

	path := filepath.Join(root, filepath.FromSlash(policies.DefaultBaseline))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the baseline file does not exist: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("the baseline is not a JSON document:\n%s", data)
	}
	var document struct {
		SchemaVersion string           `json:"schema_version"`
		CreatedAt     string           `json:"created_at"`
		ToolVersion   string           `json:"tool_version"`
		Findings      []baseline.Entry `json:"findings"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("cannot parse the baseline: %v", err)
	}
	if document.SchemaVersion != baseline.SchemaVersion {
		t.Errorf("the baseline schema version is %q, want %q", document.SchemaVersion, baseline.SchemaVersion)
	}
	if document.CreatedAt == "" || document.ToolVersion == "" {
		t.Errorf("the baseline does not declare its metadata: %+v", document)
	}
	if len(document.Findings) != 2 {
		t.Fatalf("the baseline holds %d findings, want 2: %+v", len(document.Findings), document.Findings)
	}

	before := treeState(t, root)
	code, stdout, stderr = execute(t, "--root", root, "audit", "--diff", "--base", base)
	after := treeState(t, root)

	if code != cli.ExitOK {
		t.Fatalf("the audit of the known findings exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stderr, "2 finding(s) known by the baseline") {
		t.Errorf("the standard error does not count the known findings:\n%s", stderr)
	}
	if strings.Contains(stdout, "CDX-SUP-001") || strings.Contains(stdout, "CDX-PLH-001") {
		t.Errorf("the document reports a finding that the baseline knows:\n%s", stdout)
	}
	if before != after {
		t.Errorf("the audit changed the repository:\nbefore %s\nafter  %s", before, after)
	}

	// Without --force the command never replaces an existing baseline.
	code, _, stderr = execute(t, "--root", root, "baseline", "create", "--base", base)
	if code != cli.ExitError {
		t.Errorf("baseline create over an existing baseline exits with %d, want %d: %s", code, cli.ExitError, stderr)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("the error does not explain how to force the replacement:\n%s", stderr)
	}
	replaced, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the baseline after the refused replacement: %v", err)
	}
	if !bytes.Equal(data, replaced) {
		t.Error("baseline create replaced the existing baseline without --force")
	}

	// A defect that changes the code is a new finding.
	writeFile(t, root, "app.py", "def greet():\n    # TODO: return the greeting of the user\n    return \"hi\"  # noqa\n")
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "change the line of the finding")
	code, stdout, stderr = execute(t, "--root", root, "audit", "--diff", "--base", base)
	if code != cli.ExitFindings {
		t.Fatalf("the audit of the new finding exits with %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if !strings.Contains(stdout, "CDX-SUP-001") {
		t.Errorf("the document does not report the new finding:\n%s", stdout)
	}
	if !strings.Contains(stderr, "1 finding(s) known by the baseline") {
		t.Errorf("the standard error does not count the known finding:\n%s", stderr)
	}
}

func TestBaselineCreateWithoutOriginFailsWithTheInvalidArgumentsCode(t *testing.T) {
	root := newRepository(t)

	code, _, stderr := execute(t, "--root", root, "baseline", "create")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if !strings.Contains(stderr, "--base") || !strings.Contains(stderr, "--worktree") {
		t.Errorf("the error does not say what to pass:\n%s", stderr)
	}
}

// TestBaselineSurvivesTheMovementOfTheCode checks the promise of the
// fingerprint: the same findings, moved down the file, stay known.
func TestBaselineSurvivesTheMovementOfTheCode(t *testing.T) {
	root := newRepository(t)
	base := headRevision(t, root)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "add the findings")

	if code, _, stderr := execute(t, "--root", root, "baseline", "create", "--base", base); code != cli.ExitOK {
		t.Fatalf("baseline create exits with %d: %s", code, stderr)
	}

	// The same lines, one line further down.
	moved := "import os\n\n" + failingChange
	writeFile(t, root, "app.py", moved)
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "move the findings down")

	code, stdout, stderr := execute(t, "--root", root, "audit", "--diff", "--base", base)
	if code != cli.ExitOK {
		t.Fatalf("the audit of the moved findings exits with %d, want %d: %s",
			code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stderr, "2 finding(s) known by the baseline") {
		t.Errorf("the standard error does not count the known findings:\n%s", stderr)
	}
	if strings.Contains(stdout, "CDX-SUP-001") || strings.Contains(stdout, "CDX-PLH-001") {
		t.Errorf("the document reports a finding that only moved:\n%s", stdout)
	}
}

// TestBaselineFollowsTheConfiguredPath checks that both commands use the
// audit.baseline key of the configuration, resolved against the root.
func TestBaselineFollowsTheConfiguredPath(t *testing.T) {
	root := newRepository(t)
	writeFile(t, root, ".codixai.yml", "version: 1\naudit:\n  fail_on: warning\n  baseline: config/known.json\n")
	writeFile(t, root, "app.py", failingChange)

	if code, _, stderr := execute(t, "--root", root, "baseline", "create", "--worktree"); code != cli.ExitOK {
		t.Fatalf("baseline create exits with %d: %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "config", "known.json")); err != nil {
		t.Fatalf("the baseline was not written at the configured path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(policies.DefaultBaseline))); !os.IsNotExist(err) {
		t.Error("the baseline was written at the default path instead of the configured one")
	}

	code, _, stderr := execute(t, "--root", root, "audit", "--diff", "--worktree")
	if code != cli.ExitOK {
		t.Fatalf("the audit of the configured baseline exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stderr, "2 finding(s) known by the baseline") {
		t.Errorf("the audit does not read the configured baseline:\n%s", stderr)
	}
}

// TestBaselineCreateForceReplacesAnUnreadableBaseline checks that --force does
// not depend on the document it replaces.
func TestBaselineCreateForceReplacesAnUnreadableBaseline(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)
	path := filepath.Join(root, filepath.FromSlash(policies.DefaultBaseline))
	writeFile(t, root, policies.DefaultBaseline, "{ broken\n")

	code, _, stderr := execute(t, "--root", root, "baseline", "create", "--worktree")
	if code != cli.ExitError {
		t.Errorf("baseline create over an unreadable baseline exits with %d, want %d: %s", code, cli.ExitError, stderr)
	}
	if content, err := os.ReadFile(path); err != nil || string(content) != "{ broken\n" {
		t.Errorf("the unreadable baseline was modified without --force: %q, %v", content, err)
	}

	// The audit never ignores a baseline it cannot read: a malformed file
	// is an error, not a silent empty baseline.
	code, _, stderr = execute(t, "--root", root, "audit", "--diff", "--worktree")
	if code != cli.ExitError {
		t.Errorf("the audit over an unreadable baseline exits with %d, want %d: %s", code, cli.ExitError, stderr)
	}
	if !strings.Contains(stderr, "baseline.json") {
		t.Errorf("the error does not name the unreadable baseline:\n%s", stderr)
	}

	code, stdout, stderr := execute(t, "--root", root, "baseline", "create", "--worktree", "--force")
	if code != cli.ExitOK {
		t.Fatalf("baseline create --force exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	if !strings.Contains(stdout, "2 finding(s) recorded") {
		t.Errorf("the command does not report the findings it recorded:\n%s", stdout)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the replaced baseline: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("the replaced baseline is not a JSON document:\n%s", data)
	}
}
