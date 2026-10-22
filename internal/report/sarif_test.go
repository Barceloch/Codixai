package report

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/baseline"
	"github.com/barceloch/codixai/internal/domain"
)

// suppressionLine is the added line that the suppression finding points at.
//
// It belongs to the change set, so the fingerprint of the finding carries the
// normalized context of the line.
const suppressionLine = "    //nolint:errcheck"

// testRules returns a catalog in the reverse order of the identifiers.
//
// The order of the catalog is deliberately not the order of the document:
// the builder has to sort the rules by their identifier.
func testRules() []domain.Rule {
	return []domain.Rule{
		{
			ID:          "CDX-SUP-001",
			Version:     "1",
			Area:        "SUP",
			Description: "The change adds a suppression directive that silences an analyzer.",
			Severity:    domain.SeverityWarning,
			Languages: map[string]domain.SupportLevel{
				domain.LangGo:     domain.SupportSyntactic,
				domain.LangPython: domain.SupportSyntactic,
			},
			Method:      "lexical scan of the changed lines",
			Limitations: []string{"go: Only the nolint directives of the Go comments are detected."},
		},
		{
			ID:          "CDX-ERR-001",
			Version:     "1",
			Area:        "ERR",
			Description: "The change adds a pattern that discards an error or an empty handler.",
			Severity:    domain.SeverityError,
			Languages: map[string]domain.SupportLevel{
				domain.LangGo: domain.SupportSyntactic,
			},
			Method:      "lexical scan of the changed lines",
			Limitations: []string{"go: A swallowed error is not necessarily a defect."},
		},
	}
}

// testChanges returns the change set of the run.
func testChanges() *domain.ChangeSet {
	return &domain.ChangeSet{
		Origin: domain.OriginWorktree,
		Files: []domain.FileChange{{
			Path:   "internal/payment/service.go",
			Status: domain.FileModified,
			Hunks: []domain.Hunk{{Index: 0, Lines: []domain.DiffLine{
				{Kind: domain.LineAdded, NewLine: 12, Content: "    if err := save(); err != nil {}"},
				{Kind: domain.LineAdded, NewLine: 41, Content: suppressionLine},
			}}},
		}},
	}
}

// suppressionFinding is the finding of the change set with a context.
func suppressionFinding() domain.Finding {
	return domain.Finding{
		SchemaVersion:   domain.SchemaVersion,
		RuleID:          "CDX-SUP-001",
		RuleVersion:     "1",
		Severity:        domain.SeverityWarning,
		EpistemicStatus: domain.EpistemicObserved,
		Language:        domain.LangGo,
		Message:         "The change adds an analyzer suppression: nolint.",
		Location:        domain.Location{Path: "internal/payment/service.go", StartLine: 41, EndLine: 41},
		Support: domain.Support{
			Level:  domain.SupportSyntactic,
			Method: "lexical scan of the changed lines",
		},
	}
}

// errorFinding is an informational finding of another file.
func errorFinding() domain.Finding {
	return domain.Finding{
		SchemaVersion:   domain.SchemaVersion,
		RuleID:          "CDX-ERR-001",
		RuleVersion:     "1",
		Severity:        domain.SeverityInfo,
		EpistemicStatus: domain.EpistemicInferred,
		Language:        domain.LangGo,
		Message:         "The change adds a pattern that discards an error.",
		Location:        domain.Location{Path: "app.go", StartLine: 1, EndLine: 3},
		Support:         domain.Support{Level: domain.SupportSyntactic},
	}
}

// testResult returns an audit result with two findings, given out of order.
//
// It is built directly instead of running the audit use case: the builder
// only maps a result, and the shape of the result is the contract that the
// application layer publishes.
func testResult() *application.AuditResult {
	return &application.AuditResult{
		Run: &domain.AnalysisRun{
			SchemaVersion: domain.SchemaVersion,
			ToolVersion:   "0.1.0",
			GeneratedAt:   "2026-01-01T00:00:00Z",
			Origin:        string(domain.OriginWorktree),
			Rules:         testRules(),
			Findings:      []domain.Finding{suppressionFinding(), errorFinding()},
		},
		Changes:       testChanges(),
		KnownFindings: 3,
	}
}

// build returns the document of the test result.
func build(t *testing.T) SarifDocument {
	t.Helper()
	return SARIF(testResult(), SARIFOptions{ToolVersion: "0.1.0"})
}

// runOf returns the only run of a document.
func runOf(t *testing.T, document SarifDocument) sarifRun {
	t.Helper()
	if len(document.Runs) != 1 {
		t.Fatalf("the document holds %d runs, want 1", len(document.Runs))
	}
	return document.Runs[0]
}

// resultOf returns the result with a rule identifier.
func resultOf(t *testing.T, run sarifRun, ruleID string) sarifResult {
	t.Helper()
	for _, result := range run.Results {
		if result.RuleID == ruleID {
			return result
		}
	}
	t.Fatalf("the document holds no result for %s: %+v", ruleID, run.Results)
	return sarifResult{}
}

// ruleOf returns the published rule with an identifier.
func ruleOf(t *testing.T, run sarifRun, ruleID string) sarifRule {
	t.Helper()
	for _, rule := range run.Tool.Driver.Rules {
		if rule.ID == ruleID {
			return rule
		}
	}
	t.Fatalf("the catalog holds no rule %s: %+v", ruleID, run.Tool.Driver.Rules)
	return sarifRule{}
}

func TestDocumentDeclaresTheVersionAndTheSchema(t *testing.T) {
	document := build(t)

	if document.Version != Version {
		t.Errorf("the version is %q, want %q", document.Version, Version)
	}
	if document.Version != "2.1.0" {
		t.Errorf("the version is %q, want the version of SARIF that the decision 0009 fixes: 2.1.0", document.Version)
	}
	if document.Schema != SchemaURL {
		t.Errorf("the schema is %q, want %q", document.Schema, SchemaURL)
	}
	if !strings.Contains(document.Schema, "sarif-2.1.0") {
		t.Errorf("the schema %q does not identify the version 2.1.0", document.Schema)
	}
}

func TestDriverDescribesTheTool(t *testing.T) {
	driver := runOf(t, build(t)).Tool.Driver

	if driver.Name != "codixai" {
		t.Errorf("the driver name is %q, want codixai", driver.Name)
	}
	if driver.Version != "0.1.0" {
		t.Errorf("the driver version is %q, want 0.1.0", driver.Version)
	}
	if driver.InformationURI != InformationURI {
		t.Errorf("the information URI is %q, want %q", driver.InformationURI, InformationURI)
	}
}

func TestDriverFallsBackToTheVersionOfTheRun(t *testing.T) {
	document := SARIF(testResult(), SARIFOptions{})
	if version := runOf(t, document).Tool.Driver.Version; version != "0.1.0" {
		t.Errorf("the driver version is %q, want the version of the run: 0.1.0", version)
	}

	// The version is a required field of the format: it is never empty,
	// even when neither the caller nor the run declares one.
	result := &application.AuditResult{Run: &domain.AnalysisRun{}}
	document = SARIF(result, SARIFOptions{ToolVersion: " "})
	if version := runOf(t, document).Tool.Driver.Version; version == "" {
		t.Error("the driver version is empty: the format requires a version")
	}
}

func TestRulesAreSortedAndDescribeTheCatalog(t *testing.T) {
	rules := runOf(t, build(t)).Tool.Driver.Rules

	if len(rules) != len(testRules()) {
		t.Fatalf("the catalog holds %d rules, want %d", len(rules), len(testRules()))
	}
	want := []string{"CDX-ERR-001", "CDX-SUP-001"}
	for index, identifier := range want {
		if rules[index].ID != identifier {
			t.Errorf("the rule %d is %q, want %q: the catalog has to be sorted by identifier",
				index, rules[index].ID, identifier)
		}
	}
	for _, rule := range rules {
		// The catalog does not publish a human readable name, so the
		// identifier is the name of the descriptor: it is never empty.
		if rule.Name != rule.ID {
			t.Errorf("the name of %s is %q, want its identifier", rule.ID, rule.Name)
		}
		if rule.ShortDescription.Text == "" || rule.FullDescription.Text == "" {
			t.Errorf("the rule %s does not describe itself: %+v", rule.ID, rule)
		}
		if rule.ShortDescription.Text != rule.FullDescription.Text {
			t.Errorf("the rule %s publishes two different descriptions: %q and %q",
				rule.ID, rule.ShortDescription.Text, rule.FullDescription.Text)
		}
		if rule.Properties.Support.Method == "" {
			t.Errorf("the rule %s does not declare the method of its analysis", rule.ID)
		}
		if len(rule.Properties.Support.Limitations) == 0 {
			t.Errorf("the rule %s hides its limitations", rule.ID)
		}
		if rule.Properties.Support.Level != string(domain.SupportSyntactic) {
			t.Errorf("the support level of %s is %q, want syntactic",
				rule.ID, rule.Properties.Support.Level)
		}
	}

	if level := ruleOf(t, runOf(t, build(t)), "CDX-ERR-001").DefaultConfiguration.Level; level != "error" {
		t.Errorf("the default level of CDX-ERR-001 is %q, want error", level)
	}
	if level := ruleOf(t, runOf(t, build(t)), "CDX-SUP-001").DefaultConfiguration.Level; level != "warning" {
		t.Errorf("the default level of CDX-SUP-001 is %q, want warning", level)
	}
}

func TestResultMapsTheFinding(t *testing.T) {
	run := runOf(t, build(t))
	result := resultOf(t, run, "CDX-SUP-001")
	finding := suppressionFinding()

	if result.RuleID != finding.RuleID {
		t.Errorf("the rule identifier is %q, want %q", result.RuleID, finding.RuleID)
	}
	if result.RuleIndex != 1 {
		t.Errorf("the rule index is %d, want the position of CDX-SUP-001 in the sorted catalog: 1", result.RuleIndex)
	}
	if run.Tool.Driver.Rules[result.RuleIndex].ID != finding.RuleID {
		t.Errorf("the rule index %d points at %q, want %q",
			result.RuleIndex, run.Tool.Driver.Rules[result.RuleIndex].ID, finding.RuleID)
	}
	if result.Level != "warning" {
		t.Errorf("the level is %q, want warning", result.Level)
	}
	if result.Message.Text != finding.Message {
		t.Errorf("the message is %q, want %q", result.Message.Text, finding.Message)
	}

	if len(result.Locations) != 1 {
		t.Fatalf("the result holds %d locations, want 1", len(result.Locations))
	}
	location := result.Locations[0].PhysicalLocation
	if location.ArtifactLocation.URI != "internal/payment/service.go" {
		t.Errorf("the URI is %q, want the repository relative path", location.ArtifactLocation.URI)
	}
	if location.Region == nil {
		t.Fatal("the region of the finding is missing")
	}
	if location.Region.StartLine != 41 || location.Region.EndLine != 41 {
		t.Errorf("the region is %d-%d, want 41-41", location.Region.StartLine, location.Region.EndLine)
	}

	// The fingerprint is the identity of the decision 0008: the rule, the
	// path and the normalized context of the line, never the line number.
	want := baseline.Fingerprint(finding.RuleID, finding.Location.Path, "//nolint:errcheck")
	if result.PartialFingerprints.CodixaiV1 != want {
		t.Errorf("the fingerprint is %q, want %q", result.PartialFingerprints.CodixaiV1, want)
	}
	if len(result.PartialFingerprints.CodixaiV1) != 16 {
		t.Errorf("the fingerprint %q is not 16 hexadecimal characters",
			result.PartialFingerprints.CodixaiV1)
	}

	if result.Properties.EpistemicStatus != "observed" {
		t.Errorf("the epistemic status is %q, want observed", result.Properties.EpistemicStatus)
	}
	if result.Properties.SupportLevel != "syntactic" {
		t.Errorf("the support level is %q, want syntactic", result.Properties.SupportLevel)
	}
	if result.Properties.RuleVersion != "1" {
		t.Errorf("the rule version is %q, want 1", result.Properties.RuleVersion)
	}
}

func TestLevelMapping(t *testing.T) {
	// The mapping of the decision 0009: error stays error, warning stays
	// warning and info becomes note. A severity outside the catalog never
	// escalates.
	cases := []struct {
		severity domain.Severity
		want     string
	}{
		{domain.SeverityError, "error"},
		{domain.SeverityWarning, "warning"},
		{domain.SeverityInfo, "note"},
		{domain.Severity("fatal"), "none"},
	}
	for _, testCase := range cases {
		if got := sarifLevel(testCase.severity); got != testCase.want {
			t.Errorf("the severity %q maps to %q, want %q", testCase.severity, got, testCase.want)
		}
	}

	// The informational finding of the fixture travels as a note.
	if level := resultOf(t, runOf(t, build(t)), "CDX-ERR-001").Level; level != "note" {
		t.Errorf("the level of the informational finding is %q, want note", level)
	}
}

func TestKnownFindingsTravelInTheRunProperties(t *testing.T) {
	if known := runOf(t, build(t)).Properties.SuppressedByBaseline; known != 3 {
		t.Errorf("the count of the known findings is %d, want 3", known)
	}

	// A run without known findings states the zero instead of omitting the
	// property: a consumer can tell "nothing was suppressed" from "the tool
	// did not say".
	result := testResult()
	result.KnownFindings = 0
	if known := runOf(t, SARIF(result, SARIFOptions{})).Properties.SuppressedByBaseline; known != 0 {
		t.Errorf("the count of the known findings is %d, want 0", known)
	}
}

func TestResultsAreSortedByPathLineAndRule(t *testing.T) {
	run := runOf(t, build(t))

	got := make([]string, 0, len(run.Results))
	for _, result := range run.Results {
		location := result.Locations[0].PhysicalLocation
		line := 0
		if location.Region != nil {
			line = location.Region.StartLine
		}
		got = append(got, location.ArtifactLocation.URI+"#"+result.RuleID+"@"+strconv.Itoa(line))
	}
	want := []string{"app.go#CDX-ERR-001@1", "internal/payment/service.go#CDX-SUP-001@41"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the results are %v, want %v: the order is path, line and rule", got, want)
	}
}

func TestDocumentIsDeterministic(t *testing.T) {
	first, err := json.Marshal(build(t))
	if err != nil {
		t.Fatalf("cannot marshal the document: %v", err)
	}
	second, err := json.Marshal(build(t))
	if err != nil {
		t.Fatalf("cannot marshal the document: %v", err)
	}
	if string(first) != string(second) {
		t.Errorf("two builds of the same result differ:\n%s\n%s", first, second)
	}

	// The order of the input does not change the document.
	reversed := testResult()
	reversed.Run.Findings = []domain.Finding{errorFinding(), suppressionFinding()}
	rules := testRules()
	reversed.Run.Rules = []domain.Rule{rules[1], rules[0]}
	third, err := json.Marshal(SARIF(reversed, SARIFOptions{ToolVersion: "0.1.0"}))
	if err != nil {
		t.Fatalf("cannot marshal the reversed document: %v", err)
	}
	if string(first) != string(third) {
		t.Errorf("the order of the input changes the document:\n%s\n%s", first, third)
	}
}

func TestFindingWithoutALineOmitsTheRegion(t *testing.T) {
	result := testResult()
	result.Run.Findings = []domain.Finding{{
		RuleID:          "CDX-SUP-001",
		Severity:        domain.SeverityWarning,
		EpistemicStatus: domain.EpistemicObserved,
		Message:         "a finding of a deleted file",
		Location:        domain.Location{Path: "internal/payment/service.go"},
	}}

	location := runOf(t, SARIF(result, SARIFOptions{})).Results[0].Locations[0].PhysicalLocation
	if location.Region != nil {
		t.Errorf("the region is %+v, want none: SARIF has no line zero", location.Region)
	}
	if location.ArtifactLocation.URI == "" {
		t.Error("the location of a finding without a line has to keep the file")
	}
}

func TestResultWhoseRuleIsMissingGetsAPublishedEntry(t *testing.T) {
	result := testResult()
	result.Run.Rules = nil
	result.Run.Findings = []domain.Finding{{
		RuleID:          "CDX-UNK-001",
		RuleVersion:     "7",
		Severity:        domain.SeverityWarning,
		EpistemicStatus: domain.EpistemicObserved,
		Message:         "a finding of a rule that the catalog does not describe",
		Location:        domain.Location{Path: "app.go", StartLine: 1},
	}}

	run := runOf(t, SARIF(result, SARIFOptions{}))
	if len(run.Tool.Driver.Rules) != 1 {
		t.Fatalf("the catalog holds %d rules, want the rule of the result", len(run.Tool.Driver.Rules))
	}
	if run.Tool.Driver.Rules[0].ShortDescription.Text == "" {
		t.Error("the published rule has an empty description: the format requires one")
	}
	if index := run.Results[0].RuleIndex; index < 0 || index >= len(run.Tool.Driver.Rules) {
		t.Errorf("the rule index %d does not point into the catalog", index)
	}
}

// TestDocumentHasTheRequiredStructure validates the structure that the
// decision 0009 fixes.
//
// The document is NOT validated against the official JSON schema of SARIF
// 2.1.0: the environment has no network access to fetch the schema, so the
// claim of this test stops at the fields that the decision lists. Every
// required field is present and none of them is empty.
func TestDocumentHasTheRequiredStructure(t *testing.T) {
	data, err := json.Marshal(build(t))
	if err != nil {
		t.Fatalf("cannot marshal the document: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("the document is not a JSON object: %v", err)
	}

	nonEmptyString(t, document, "$schema", "the document")
	nonEmptyString(t, document, "version", "the document")

	runs := listField(t, document, "runs", "the document")
	if len(runs) != 1 {
		t.Fatalf("the document holds %d runs, want 1", len(runs))
	}
	run := objectOf(t, runs[0], "runs[0]")

	driver := objectField(t, objectField(t, run, "tool", "runs[0]"), "driver", "runs[0].tool")
	nonEmptyString(t, driver, "name", "runs[0].tool.driver")
	nonEmptyString(t, driver, "version", "runs[0].tool.driver")
	nonEmptyString(t, driver, "informationUri", "runs[0].tool.driver")

	rules := listField(t, driver, "rules", "runs[0].tool.driver")
	if len(rules) == 0 {
		t.Fatal("the driver publishes no rule")
	}
	for index, value := range rules {
		context := "runs[0].tool.driver.rules[" + strconv.Itoa(index) + "]"
		rule := objectOf(t, value, context)
		nonEmptyString(t, rule, "id", context)
		nonEmptyString(t, rule, "name", context)
		nonEmptyString(t, objectField(t, rule, "shortDescription", context), "text", context+".shortDescription")
		nonEmptyString(t, objectField(t, rule, "fullDescription", context), "text", context+".fullDescription")
		configuration := objectField(t, rule, "defaultConfiguration", context)
		switch level := nonEmptyString(t, configuration, "level", context+".defaultConfiguration"); level {
		case "error", "warning", "note":
		default:
			t.Errorf("%s.defaultConfiguration.level is %q, want error, warning or note", context, level)
		}
		support := objectField(t, objectField(t, rule, "properties", context), "support", context+".properties")
		nonEmptyString(t, support, "level", context+".properties.support")
	}

	results := listField(t, run, "results", "runs[0]")
	if len(results) == 0 {
		t.Fatal("the document of the fixture holds no result")
	}
	for index, value := range results {
		context := "runs[0].results[" + strconv.Itoa(index) + "]"
		result := objectOf(t, value, context)
		nonEmptyString(t, result, "ruleId", context)
		nonEmptyString(t, objectField(t, result, "message", context), "text", context+".message")
		switch level := nonEmptyString(t, result, "level", context); level {
		case "error", "warning", "note", "none":
		default:
			t.Errorf("%s.level is %q, want a level of the format", context, level)
		}

		locations := listField(t, result, "locations", context)
		if len(locations) != 1 {
			t.Fatalf("%s holds %d locations, want 1", context, len(locations))
		}
		physical := objectField(t, objectOf(t, locations[0], context+".locations[0]"),
			"physicalLocation", context+".locations[0]")
		artifact := objectField(t, physical, "artifactLocation", context+".locations[0].physicalLocation")
		if uri := nonEmptyString(t, artifact, "uri", context+".locations[0].physicalLocation.artifactLocation"); strings.Contains(uri, "\\") {
			t.Errorf("%s carries a backslash in the URI %q", context, uri)
		}
		region := objectField(t, physical, "region", context+".locations[0].physicalLocation")
		if start := numberField(t, region, "startLine", context+".region"); start < 1 {
			t.Errorf("%s.region.startLine is %d, want a line of the file", context, start)
		}
		if end := numberField(t, region, "endLine", context+".region"); end < 1 {
			t.Errorf("%s.region.endLine is %d, want a line of the file", context, end)
		}

		fingerprints := objectField(t, result, "partialFingerprints", context)
		fingerprint := nonEmptyString(t, fingerprints, FingerprintKey, context+".partialFingerprints")
		if len(fingerprint) != 16 {
			t.Errorf("%s.partialFingerprints[%q] is %q, want 16 hexadecimal characters",
				context, FingerprintKey, fingerprint)
		}

		properties := objectField(t, result, "properties", context)
		nonEmptyString(t, properties, "epistemic_status", context+".properties")
		nonEmptyString(t, properties, "support_level", context+".properties")
		nonEmptyString(t, properties, "rule_version", context+".properties")
	}

	if _, ok := run["properties"]; !ok {
		t.Error("runs[0] does not hold the properties of the run")
	}
	if _, err := jsonField(run, "properties", "suppressedByBaseline"); err != nil {
		t.Errorf("runs[0].properties does not hold the count of the known findings: %v", err)
	}
}

// objectOf returns a value of the document as an object.
func objectOf(t *testing.T, value any, context string) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object: %v", context, value)
	}
	return object
}

// objectField returns an object field of the document.
func objectField(t *testing.T, object map[string]any, key, context string) map[string]any {
	t.Helper()
	value, err := jsonField(object, key, context)
	if err != nil {
		t.Fatal(err)
	}
	return objectOf(t, value, context+"."+key)
}

// listField returns a list field of the document.
func listField(t *testing.T, object map[string]any, key, context string) []any {
	t.Helper()
	value, err := jsonField(object, key, context)
	if err != nil {
		t.Fatal(err)
	}
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%s.%s is not a list: %v", context, key, value)
	}
	return list
}

// nonEmptyString returns a string field of the document and fails when it is
// missing or empty.
func nonEmptyString(t *testing.T, object map[string]any, key, context string) string {
	t.Helper()
	value, err := jsonField(object, key, context)
	if err != nil {
		t.Fatal(err)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s.%s is not a string: %v", context, key, value)
	}
	if strings.TrimSpace(text) == "" {
		t.Errorf("%s.%s is empty", context, key)
	}
	return text
}

// numberField returns a number field of the document.
func numberField(t *testing.T, object map[string]any, key, context string) int {
	t.Helper()
	value, err := jsonField(object, key, context)
	if err != nil {
		t.Fatal(err)
	}
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("%s.%s is not a number: %v", context, key, value)
	}
	return int(number)
}

// jsonField returns a field of the document, and the error that explains the
// missing one.
func jsonField(object map[string]any, key, context string) (any, error) {
	value, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("%s does not hold the required field %s", context, key)
	}
	return value, nil
}
