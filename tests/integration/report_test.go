package integration

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/cli"
)

// suppressionOnlyChange adds one analyzer suppression and nothing else, so
// the run holds a single finding.
const suppressionOnlyChange = "def greet():\n    return \"hello\"  # noqa\n"

// sarifExport is the part of the SARIF document that the end to end tests
// read.
type sarifExport struct {
	Schema  string           `json:"$schema"`
	Version string           `json:"version"`
	Runs    []sarifExportRun `json:"runs"`
}

// sarifExportRun is one run of the document.
type sarifExportRun struct {
	Tool struct {
		Driver struct {
			Name           string `json:"name"`
			Version        string `json:"version"`
			InformationURI string `json:"informationUri"`
			Rules          []struct {
				ID                   string `json:"id"`
				DefaultConfiguration struct {
					Level string `json:"level"`
				} `json:"defaultConfiguration"`
			} `json:"rules"`
		} `json:"driver"`
	} `json:"tool"`
	Results    []sarifExportResult `json:"results"`
	Properties struct {
		SuppressedByBaseline int `json:"suppressedByBaseline"`
		MissingRules         []struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Reason  string `json:"reason"`
		} `json:"missingRules,omitempty"`
	} `json:"properties"`
}

// sarifExportResult is one finding of the document.
type sarifExportResult struct {
	RuleID    string `json:"ruleId"`
	RuleIndex int    `json:"ruleIndex"`
	Level     string `json:"level"`
	Message   struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
				EndLine   int `json:"endLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          struct {
		EpistemicStatus string `json:"epistemic_status"`
		SupportLevel    string `json:"support_level"`
		RuleVersion     string `json:"rule_version"`
	} `json:"properties"`
}

// parseSarif parses the SARIF document of the standard output.
//
// The document is not validated against the official JSON schema of SARIF:
// the test environment has no network access to fetch it. The test validates
// the structure that the decision 0009 fixes.
func parseSarif(t *testing.T, stdout, stderr string) sarifExport {
	t.Helper()
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("the standard output is not a JSON document:\n%s\n%s", stdout, stderr)
	}
	var document sarifExport
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("cannot parse the SARIF document: %v", err)
	}
	return document
}

func TestReportWritesTheSarifDocumentOfTheFindings(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	before := treeState(t, root)
	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif", "--worktree")
	after := treeState(t, root)

	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if ansiSequence(stdout) {
		t.Errorf("the SARIF document carries an escape sequence:\n%q", stdout)
	}
	if before != after {
		t.Errorf("the report changed the repository:\nbefore %s\nafter  %s", before, after)
	}

	document := parseSarif(t, stdout, stderr)
	if document.Version != "2.1.0" {
		t.Errorf("the version is %q, want 2.1.0", document.Version)
	}
	if !strings.Contains(document.Schema, "sarif-2.1.0") {
		t.Errorf("the schema is %q, want the schema of SARIF 2.1.0", document.Schema)
	}
	if len(document.Runs) != 1 {
		t.Fatalf("the document holds %d runs, want 1", len(document.Runs))
	}
	run := document.Runs[0]

	if run.Tool.Driver.Name != "codixai" {
		t.Errorf("the driver is %q, want codixai", run.Tool.Driver.Name)
	}
	if run.Tool.Driver.Version == "" || run.Tool.Driver.InformationURI == "" {
		t.Errorf("the driver does not declare its version and its origin: %+v", run.Tool.Driver)
	}

	rules := map[string]string{}
	for _, rule := range run.Tool.Driver.Rules {
		rules[rule.ID] = rule.DefaultConfiguration.Level
	}
	for rule, level := range map[string]string{"CDX-SUP-001": "warning", "CDX-PLH-001": "warning"} {
		if rules[rule] != level {
			t.Errorf("the catalog publishes %q with the level %q, want %q", rule, rules[rule], level)
		}
	}

	reported := map[string]sarifExportResult{}
	for _, result := range run.Results {
		reported[result.RuleID] = result
	}
	for _, rule := range []string{"CDX-SUP-001", "CDX-PLH-001"} {
		result, ok := reported[rule]
		if !ok {
			t.Fatalf("the document does not report %s: %+v", rule, run.Results)
		}
		if result.Level != "warning" {
			t.Errorf("the level of %s is %q, want warning", rule, result.Level)
		}
		if result.Message.Text == "" {
			t.Errorf("the result of %s has no message", rule)
		}
		if result.RuleIndex < 0 || result.RuleIndex >= len(run.Tool.Driver.Rules) {
			t.Errorf("the rule index of %s is %d, want a position of the catalog", rule, result.RuleIndex)
		}
		if got := run.Tool.Driver.Rules[result.RuleIndex].ID; got != rule {
			t.Errorf("the rule index of %s points at %q", rule, got)
		}
		if len(result.Locations) != 1 {
			t.Fatalf("the result of %s holds %d locations, want 1", rule, len(result.Locations))
		}
		location := result.Locations[0].PhysicalLocation
		if location.ArtifactLocation.URI != "app.py" {
			t.Errorf("the location of %s is %q, want the repository relative path app.py",
				rule, location.ArtifactLocation.URI)
		}
		if location.Region.StartLine < 1 || location.Region.EndLine < location.Region.StartLine {
			t.Errorf("the region of %s is %d-%d, want a range of the file",
				rule, location.Region.StartLine, location.Region.EndLine)
		}
		fingerprint := result.PartialFingerprints["codixaiFingerprint/v1"]
		if len(fingerprint) != 16 {
			t.Errorf("the fingerprint of %s is %q, want 16 hexadecimal characters", rule, fingerprint)
		}
		if result.Properties.EpistemicStatus == "" || result.Properties.SupportLevel == "" ||
			result.Properties.RuleVersion == "" {
			t.Errorf("the result of %s does not declare its properties: %+v", rule, result.Properties)
		}
	}

	if run.Properties.SuppressedByBaseline != 0 {
		t.Errorf("the run suppresses %d findings, want 0: the repository has no baseline",
			run.Properties.SuppressedByBaseline)
	}
	if !strings.Contains(stderr, "fail_on") {
		t.Errorf("the standard error does not explain the failure:\n%s", stderr)
	}
}

func TestReportAppliesTheBaselineToTheDocument(t *testing.T) {
	root := newRepository(t)
	base := headRevision(t, root)
	writePolicy(t, root)
	writeFile(t, root, "app.py", suppressionOnlyChange)
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "add the suppression")

	// Without a baseline the finding is new and the command fails.
	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif", "--base", base)
	if code != cli.ExitFindings {
		t.Fatalf("the report of the new finding exits with %d, want %d: %s",
			code, cli.ExitFindings, stderr)
	}
	if len(parseSarif(t, stdout, stderr).Runs[0].Results) != 1 {
		t.Fatalf("the document does not report the new finding:\n%s", stdout)
	}

	if code, _, stderr := execute(t, "--root", root, "baseline", "create", "--base", base); code != cli.ExitOK {
		t.Fatalf("baseline create exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}

	// With the baseline the finding is known: it is not a result and it
	// does not reach the threshold.
	code, stdout, stderr = execute(t, "--root", root, "report", "--format", "sarif", "--base", base)
	if code != cli.ExitOK {
		t.Fatalf("the report of the known finding exits with %d, want %d: %s", code, cli.ExitOK, stderr)
	}
	run := parseSarif(t, stdout, stderr).Runs[0]
	if len(run.Results) != 0 {
		t.Errorf("the document reports a finding that the baseline knows: %+v", run.Results)
	}
	if run.Properties.SuppressedByBaseline != 1 {
		t.Errorf("the run suppresses %d findings, want 1", run.Properties.SuppressedByBaseline)
	}
	// The catalog publishes the rules of the tool, so the identifier of the
	// known rule still appears in the document: it is the results list that
	// has to stay empty.
	if !strings.Contains(stderr, "1 finding(s) known by the baseline") {
		t.Errorf("the standard error does not count the known finding:\n%s", stderr)
	}
}

func TestReportJSONIsTheAuditDocument(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	auditCode, auditOut, auditErr := execute(t, "--root", root, "--format", "json", "audit", "--diff", "--worktree")
	reportCode, reportOut, reportErr := execute(t, "--root", root, "report", "--format", "json", "--worktree")

	if auditCode != cli.ExitFindings || reportCode != cli.ExitFindings {
		t.Fatalf("the exit codes are %d and %d, want %d: %s%s",
			auditCode, reportCode, cli.ExitFindings, auditErr, reportErr)
	}
	if !json.Valid([]byte(auditOut)) || !json.Valid([]byte(reportOut)) {
		t.Fatalf("the standard output is not a JSON document:\naudit:\n%s\nreport:\n%s", auditOut, reportOut)
	}

	// The two commands run at different instants, so the timestamp of the
	// run is the only field that can differ: the document describes the
	// same run.
	audit := documentWithoutTimestamp(t, auditOut)
	report := documentWithoutTimestamp(t, reportOut)
	if !reflect.DeepEqual(audit, report) {
		t.Errorf("the two documents differ:\naudit:  %v\nreport: %v", audit, report)
	}
}

// documentWithoutTimestamp parses the audit document and drops the instant of
// the run.
func documentWithoutTimestamp(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("cannot parse the document: %v", err)
	}
	if _, ok := document["generated_at"]; !ok {
		t.Fatalf("the document does not declare the instant of the run:\n%s", stdout)
	}
	delete(document, "generated_at")
	return document
}

func TestReportWithoutOriginFailsWithTheInvalidArgumentsCode(t *testing.T) {
	root := newRepository(t)

	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
	for _, expected := range []string{"--base", "--staged", "--worktree"} {
		if !strings.Contains(stderr, expected) {
			t.Errorf("the error does not say what to pass, it does not mention %q:\n%s", expected, stderr)
		}
	}
}

// TestTheFormatsBelongToTheCommandsTheyServe checks that a command never
// writes a document that nobody asked for: the report command has no text
// document and the audit command does not write SARIF.
func TestTheFormatsBelongToTheCommandsTheyServe(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "text", "--worktree")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("report --format text exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "audit") {
		t.Errorf("the error does not point at the command that writes the text report:\n%s", stderr)
	}

	code, stdout, stderr = execute(t, "--root", root, "audit", "--diff", "--format", "sarif", "--worktree")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("audit --format sarif exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "report") {
		t.Errorf("the error does not point at the command that writes SARIF:\n%s", stderr)
	}
}

// TestReportDirectorySARIF checks that report --directory writes a valid
// SARIF document for directory analysis: empty fingerprints, missing rules
// in properties, source_range evidence kind, no diff regions.
func TestReportDirectorySARIF(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport \"github.com/missing/package\"\n\nfunc main() {}\n")

	code, stdout, stderr := execute(t, "--format", "sarif", "report", "--directory", root)

	if code != cli.ExitFindings {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitFindings, stderr)
	}
	if ansiSequence(stdout) {
		t.Errorf("the SARIF document carries an escape sequence:\n%q", stdout)
	}

	document := parseSarif(t, stdout, stderr)
	if document.Version != "2.1.0" {
		t.Errorf("the version is %q, want 2.1.0", document.Version)
	}
	if !strings.Contains(document.Schema, "sarif-2.1.0") {
		t.Errorf("the schema is %q, want the schema of SARIF 2.1.0", document.Schema)
	}
	if len(document.Runs) != 1 {
		t.Fatalf("the document holds %d runs, want 1", len(document.Runs))
	}
	run := document.Runs[0]

	if run.Tool.Driver.Name != "codixai" {
		t.Errorf("the driver is %q, want codixai", run.Tool.Driver.Name)
	}

	// Results should have one finding with empty fingerprint.
	if len(run.Results) != 1 {
		t.Fatalf("the document holds %d results, want 1: %+v", len(run.Results), run.Results)
	}
	result := run.Results[0]
	if result.RuleID != "CDX-IMP-001" {
		t.Errorf("the rule is %q, want CDX-IMP-001", result.RuleID)
	}
	if result.Message.Text == "" {
		t.Errorf("the result has no message")
	}
	if len(result.Locations) != 1 {
		t.Fatalf("the result holds %d locations, want 1", len(result.Locations))
	}
	location := result.Locations[0].PhysicalLocation
	if location.ArtifactLocation.URI != "main.go" {
		t.Errorf("the location is %q, want main.go", location.ArtifactLocation.URI)
	}
	if location.Region.StartLine != 3 || location.Region.EndLine != 3 {
		t.Errorf("the region is %d-%d, want 3-3", location.Region.StartLine, location.Region.EndLine)
	}
	fingerprint := result.PartialFingerprints["codixaiFingerprint/v1"]
	if fingerprint != "" {
		t.Errorf("the fingerprint of a directory finding must be empty, got %q", fingerprint)
	}
	// The epistemic status for CDX-IMP-001 in directory mode is observed (direct manifest check).
	if result.Properties.EpistemicStatus != "observed" {
		t.Errorf("the epistemic status is %q, want observed", result.Properties.EpistemicStatus)
	}
	if result.Properties.SupportLevel != "syntactic" {
		t.Errorf("the support level is %q, want syntactic", result.Properties.SupportLevel)
	}

	// Missing rules must be in properties, not as results.
	if len(run.Properties.MissingRules) == 0 {
		t.Fatalf("the run properties do not declare missing rules")
	}
	foundMissing := false
	for _, mr := range run.Properties.MissingRules {
		if mr.ID == "CDX-TST-001" {
			foundMissing = true
			if mr.Reason == "" {
				t.Errorf("the missing rule CDX-TST-001 has no reason")
			}
		}
	}
	if !foundMissing {
		t.Errorf("the missing rules do not include CDX-TST-001: %+v", run.Properties.MissingRules)
	}
	// The known findings count should be 0 (no baseline in directory mode).
	if run.Properties.SuppressedByBaseline != 0 {
		t.Errorf("the run suppresses %d findings, want 0 in directory mode", run.Properties.SuppressedByBaseline)
	}
	if !strings.Contains(stderr, "fail_on") {
		t.Errorf("the standard error does not explain the failure:\n%s", stderr)
	}
}

// TestReportDirectoryJSON checks that report --format json --directory
// produces the same AnalysisRun document as audit --format json <path>.
func TestReportDirectoryJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.21\n")
	writeFile(t, root, "main.go", "package main\n\nimport \"github.com/missing/package\"\n\nfunc main() {}\n")

	auditCode, auditOut, auditErr := execute(t, "--format", "json", "audit", root)
	reportCode, reportOut, reportErr := execute(t, "--format", "json", "report", "--directory", root)

	if auditCode != cli.ExitFindings || reportCode != cli.ExitFindings {
		t.Fatalf("the exit codes are %d and %d, want %d: audit: %s, report: %s",
			auditCode, reportCode, cli.ExitFindings, auditErr, reportErr)
	}
	if !json.Valid([]byte(auditOut)) || !json.Valid([]byte(reportOut)) {
		t.Fatalf("the standard output is not a JSON document:\naudit:\n%s\nreport:\n%s", auditOut, reportOut)
	}

	// The two commands run at different instants, so the timestamp of the
	// run is the only field that can differ: the document describes the
	// same run.
	audit := documentWithoutTimestamp(t, auditOut)
	report := documentWithoutTimestamp(t, reportOut)
	if !reflect.DeepEqual(audit, report) {
		t.Errorf("the two documents differ:\naudit:  %v\nreport: %v", audit, report)
	}
}

// TestReportInvalidOriginCombination checks that mutually exclusive origin
// flags are rejected with a clear error message.
func TestReportInvalidOriginCombination(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	// base + directory
	code, stdout, stderr := execute(t, "--root", root, "report", "--format", "sarif", "--base", "main", "--directory", "/tmp")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("report --base --directory exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "several origins") && !strings.Contains(stderr, "mutually") {
		t.Errorf("the error does not explain the conflict:\n%s", stderr)
	}

	// staged + directory
	code, stdout, stderr = execute(t, "--root", root, "report", "--format", "sarif", "--staged", "--directory", "/tmp")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("report --staged --directory exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}

	// worktree + directory
	code, stdout, stderr = execute(t, "--root", root, "report", "--format", "sarif", "--worktree", "--directory", "/tmp")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("report --worktree --directory exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}

	// base + staged
	code, stdout, stderr = execute(t, "--root", root, "report", "--format", "sarif", "--base", "main", "--staged")
	if code != cli.ExitInvalidArgs {
		t.Fatalf("report --base --staged exits with %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output must stay empty, got %q", stdout)
	}
}

// TestReportDirectoryNoFindings checks that a clean directory succeeds with
// exit code 0 and empty results.
func TestReportDirectoryNoFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	code, stdout, stderr := execute(t, "--format", "sarif", "report", "--directory", root)
	if code != cli.ExitOK {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitOK, stderr)
	}

	document := parseSarif(t, stdout, stderr)
	if len(document.Runs[0].Results) != 0 {
		t.Errorf("the document reports %d findings, want 0", len(document.Runs[0].Results))
	}
	if document.Runs[0].Properties.SuppressedByBaseline != 0 {
		t.Errorf("the run suppresses %d findings, want 0", document.Runs[0].Properties.SuppressedByBaseline)
	}
	// For a clean directory with exit code 0, stderr may be empty (no failing findings).
	// The failure threshold is still applied, just no findings reach it.
}
