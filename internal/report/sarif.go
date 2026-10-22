// Package report builds the SARIF export of an audit.
//
// The package turns the result of the audit use case into the SARIF 2.1.0
// document that the report command writes, so that a compatible platform can
// import the findings of Codixai. It never runs the analysis: the command
// runs the audit and passes the result here.
//
// The document describes the same run that the audit reports. The policy of
// the repository, the baseline and the failure threshold are applied before
// this package is called, so the results are the new findings of the run and
// the findings that the baseline knows only travel as the count of
// runs[].properties.suppressedByBaseline.
//
// The shape and the mapping are fixed by the decision 0009, in
// docs/decisions/0009-sarif-export.md, and the fingerprint of a finding by
// the decision 0008, in docs/decisions/0008-baseline-and-fingerprint.md.
//
// The document states its limits instead of hiding them:
//
//   - It is not validated against the official JSON schema of SARIF: the
//     environment has no network access to fetch it. The tests of the
//     project validate the structure that the decision fixes.
//   - The rule catalog publishes one description per rule and no human
//     readable name, so shortDescription and fullDescription carry the same
//     text and name carries the stable identifier of the rule.
//   - SARIF has no field for the epistemic status nor for the support
//     level, so both travel as properties of the result.
package report

import (
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/baseline"
	"github.com/barceloch/codixai/internal/domain"
)

const (
	// Version is the version of the SARIF format of the document.
	Version = "2.1.0"

	// SchemaURL is the location of the JSON schema of the version.
	//
	// The URL is written in the document and it is never fetched: the tool
	// has no network access.
	SchemaURL = "https://json.schemastore.org/sarif-2.1.0.json"

	// DriverName is the name of the driver of the run.
	DriverName = "codixai"

	// InformationURI points at the project that produces the document.
	InformationURI = "https://github.com/Barceloch/Codixai"

	// FingerprintKey is the key of the fingerprint inside
	// partialFingerprints.
	//
	// The key carries the version of the identity, so a consumer can tell
	// the fingerprints of two definitions apart.
	FingerprintKey = "codixaiFingerprint/v1"

	// Format is the name of the format on the command line.
	Format = "sarif"

	// unknownToolVersion is the version that the driver reports when
	// neither the caller nor the run declares one. The field is required
	// by the format, so it is never left empty.
	unknownToolVersion = "unknown"

	// missingRuleDescription explains a rule that a result references and
	// that the catalog of the run does not describe. The export never
	// leaves a required description empty.
	missingRuleDescription = "The rule is not described by the rule catalog of this version of the tool."
)

// SARIFOptions configures the document.
type SARIFOptions struct {
	// ToolVersion is the version of the tool that produced the run.
	//
	// When it is empty the version of the run is used, and "unknown" when
	// the run does not declare one either.
	ToolVersion string
	// Rules is the catalog that the document publishes.
	//
	// When it is empty the rules of the run are used. Every rule that a
	// result references is published, even when the catalog given here does
	// not describe it, because ruleIndex has to point into the array.
	Rules []domain.Rule
}

// SarifDocument is the SARIF 2.1.0 document of one audit run.
//
// The type marshals with the field order of the decision 0009: structs
// instead of maps, so the bytes of the document are stable.
type SarifDocument struct {
	// Schema is the location of the schema of the format.
	Schema string `json:"$schema"`
	// Version is the version of the format.
	Version string `json:"version"`
	// Runs holds the runs of the document. The export always writes one.
	Runs []sarifRun `json:"runs"`
}

// sarifRun is one run of the document.
type sarifRun struct {
	// Tool describes the tool that produced the run.
	Tool sarifTool `json:"tool"`
	// Results holds the new findings of the run.
	Results []sarifResult `json:"results"`
	// Properties holds the facts of the run that the format has no field
	// for.
	Properties sarifRunProperties `json:"properties"`
}

// sarifTool is the tool of a run.
type sarifTool struct {
	// Driver is the component that produced the results.
	Driver sarifDriver `json:"driver"`
}

// sarifDriver describes the tool and its rule catalog.
type sarifDriver struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Version is the version of the tool.
	Version string `json:"version"`
	// InformationURI points at the project.
	InformationURI string `json:"informationUri"`
	// Rules is the catalog, sorted by the identifier of the rule.
	Rules []sarifRule `json:"rules"`
}

// sarifRule describes one rule of the catalog.
type sarifRule struct {
	// ID is the stable identifier of the rule.
	ID string `json:"id"`
	// Name is the short name of the rule. The catalog publishes the
	// identifier only, so the identifier travels here.
	Name string `json:"name"`
	// ShortDescription is the one line description of the rule.
	ShortDescription sarifMessage `json:"shortDescription"`
	// FullDescription is the description of the rule.
	FullDescription sarifMessage `json:"fullDescription"`
	// DefaultConfiguration holds the default severity of the rule.
	DefaultConfiguration sarifConfiguration `json:"defaultConfiguration"`
	// Properties holds the facts of the rule that the format has no field
	// for.
	Properties sarifRuleProperties `json:"properties"`
}

// sarifMessage is a message of the document.
type sarifMessage struct {
	// Text is the text of the message.
	Text string `json:"text"`
}

// sarifConfiguration is the default configuration of a rule.
type sarifConfiguration struct {
	// Level is the default level of the findings of the rule.
	Level string `json:"level"`
}

// sarifRuleProperties holds the facts of a rule that the format has no field
// for.
type sarifRuleProperties struct {
	// Support is the honest depth of the analysis of the rule.
	Support sarifSupport `json:"support"`
}

// sarifSupport declares how deep the analysis of a rule is.
type sarifSupport struct {
	// Level is the deepest support level of the rule across the languages.
	Level string `json:"level"`
	// Method explains how the rule checks the code.
	Method string `json:"method,omitempty"`
	// Limitations are the known limitations, prefixed with the language
	// that they affect.
	Limitations []string `json:"limitations,omitempty"`
}

// sarifResult is one new finding of the run.
type sarifResult struct {
	// RuleID is the identifier of the rule that produced the finding.
	RuleID string `json:"ruleId"`
	// RuleIndex is the position of the rule in the catalog of the driver.
	RuleIndex int `json:"ruleIndex"`
	// Level is the severity of the finding.
	Level string `json:"level"`
	// Message is the message of the finding.
	Message sarifMessage `json:"message"`
	// Locations points at the range of the finding. The export always
	// writes one location.
	Locations []sarifLocation `json:"locations"`
	// PartialFingerprints carries the stable identity of the finding.
	PartialFingerprints sarifPartialFingerprints `json:"partialFingerprints"`
	// Properties holds the facts of the finding that the format has no
	// field for.
	Properties sarifResultProperties `json:"properties"`
}

// sarifLocation is the location of a result.
type sarifLocation struct {
	// PhysicalLocation is the range inside a file.
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

// sarifPhysicalLocation is the file and the range of a result.
type sarifPhysicalLocation struct {
	// ArtifactLocation is the file, relative to the root of the
	// repository.
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	// Region is the range inside the file. It is absent when the finding
	// does not declare a line: SARIF has no line zero, and the export
	// never invents one.
	Region *sarifRegion `json:"region,omitempty"`
}

// sarifArtifactLocation is the file of a result.
type sarifArtifactLocation struct {
	// URI is the repository relative path, with forward slashes.
	URI string `json:"uri"`
}

// sarifRegion is the range of a result inside a file.
type sarifRegion struct {
	// StartLine is the first line of the range.
	StartLine int `json:"startLine"`
	// EndLine is the last line of the range.
	EndLine int `json:"endLine"`
}

// sarifPartialFingerprints is the stable identity of a result.
//
// The format defines the property as a map of string to string. A struct
// with the single key of the decision is used here, so the field order of
// the document is fixed by the declaration and not by the encoding of a map.
type sarifPartialFingerprints struct {
	// CodixaiV1 is the fingerprint of the decision 0008.
	CodixaiV1 string `json:"codixaiFingerprint/v1"`
}

// sarifResultProperties holds the facts of a finding that the format has no
// field for.
type sarifResultProperties struct {
	// EpistemicStatus is the state of the finding: observed, inferred or
	// unknown.
	EpistemicStatus string `json:"epistemic_status"`
	// SupportLevel is the depth of the analysis of the finding.
	SupportLevel string `json:"support_level"`
	// RuleVersion is the version of the rule that produced the finding.
	RuleVersion string `json:"rule_version"`
}

// sarifRunProperties holds the facts of the run that the format has no field
// for.
type sarifRunProperties struct {
	// SuppressedByBaseline counts the findings that the baseline already
	// knows. They are not results: they are not new findings.
	SuppressedByBaseline int `json:"suppressedByBaseline"`
}

// SARIF builds the SARIF 2.1.0 document of an audit result.
//
// The document is deterministic: the rules are sorted by their identifier and
// the results by path, line and rule, so two builds of the same result
// marshal to the same bytes.
//
// A nil result, or a result without a run, produces a document of one run
// without results: the document that the format requires is always
// structurally complete.
func SARIF(result *application.AuditResult, options SARIFOptions) SarifDocument {
	var (
		findings []domain.Finding
		rules    = options.Rules
		changes  *domain.ChangeSet
		version  = strings.TrimSpace(options.ToolVersion)
		known    int
	)
	if result != nil {
		changes = result.Changes
		known = result.KnownFindings
		if result.Run != nil {
			findings = result.Run.Findings
			if len(rules) == 0 {
				rules = result.Run.Rules
			}
			if version == "" {
				version = strings.TrimSpace(result.Run.ToolVersion)
			}
		}
	}
	if version == "" {
		version = unknownToolVersion
	}

	catalog, index := buildCatalog(rules, findings)
	return SarifDocument{
		Schema:  SchemaURL,
		Version: Version,
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           DriverName,
				Version:        version,
				InformationURI: InformationURI,
				Rules:          catalog,
			}},
			Results: buildResults(findings, changes, index),
			Properties: sarifRunProperties{
				SuppressedByBaseline: known,
			},
		}},
	}
}

// catalogIndex is the position of every published rule and the rule itself.
//
// The result needs both: the position is the ruleIndex of the format and the
// rule describes a finding that does not declare the depth of its analysis.
type catalogIndex struct {
	// rules holds the published rules by identifier.
	rules map[string]domain.Rule
	// positions holds the position of a rule in the catalog of the driver.
	positions map[string]int
}

// position returns the position of a rule in the catalog, and -1 when the
// rule is not published.
func (c catalogIndex) position(ruleID string) int {
	if position, ok := c.positions[ruleID]; ok {
		return position
	}
	return -1
}

// rule returns the published rule with an identifier.
func (c catalogIndex) rule(ruleID string) domain.Rule {
	return c.rules[ruleID]
}

// buildCatalog returns the catalog of the document and the index of its
// rules.
//
// The catalog is sorted by the identifier of the rule and it holds every rule
// that a result references: a result whose rule is missing from the catalog
// given by the caller gets a minimal entry, because ruleIndex has to point
// into the array.
func buildCatalog(source []domain.Rule, findings []domain.Finding) ([]sarifRule, catalogIndex) {
	byID := make(map[string]domain.Rule, len(source)+len(findings))
	for _, rule := range source {
		if rule.ID == "" {
			continue
		}
		if _, ok := byID[rule.ID]; ok {
			continue
		}
		byID[rule.ID] = rule
	}
	for _, finding := range findings {
		if finding.RuleID == "" {
			continue
		}
		if _, ok := byID[finding.RuleID]; ok {
			continue
		}
		byID[finding.RuleID] = domain.Rule{
			ID:          finding.RuleID,
			Version:     finding.RuleVersion,
			Description: missingRuleDescription,
			Severity:    finding.Severity,
		}
	}

	identifiers := make([]string, 0, len(byID))
	for identifier := range byID {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	catalog := make([]sarifRule, 0, len(identifiers))
	index := catalogIndex{
		rules:     byID,
		positions: make(map[string]int, len(identifiers)),
	}
	for _, identifier := range identifiers {
		index.positions[identifier] = len(catalog)
		catalog = append(catalog, ruleEntry(byID[identifier]))
	}
	return catalog, index
}

// ruleEntry maps a rule of the catalog onto its SARIF descriptor.
func ruleEntry(rule domain.Rule) sarifRule {
	description := rule.Description
	if description == "" {
		description = missingRuleDescription
	}
	support := supportOf(rule)
	return sarifRule{
		// The catalog publishes the identifier and no human readable
		// name, so the identifier is the name of the descriptor: it is
		// unique and it is stable.
		ID:                   rule.ID,
		Name:                 rule.ID,
		ShortDescription:     sarifMessage{Text: description},
		FullDescription:      sarifMessage{Text: description},
		DefaultConfiguration: sarifConfiguration{Level: sarifLevel(rule.Severity)},
		Properties:           sarifRuleProperties{Support: support},
	}
}

// supportOf aggregates the support matrix of a rule.
//
// The published level is the deepest level that the rule supports in any
// language: a rule that resolves relations in one language is not a syntactic
// rule, and the depth of the analysis is what the reader has to know. The
// limitations already carry the language that they affect.
func supportOf(rule domain.Rule) sarifSupport {
	level := domain.SupportNone
	for _, candidate := range rule.Languages {
		if supportRank(candidate) > supportRank(level) {
			level = candidate
		}
	}
	return sarifSupport{
		Level:       string(level),
		Method:      rule.Method,
		Limitations: rule.Limitations,
	}
}

// supportRank orders the support levels. The ranks are categorical: they
// express an order, not a measurement.
func supportRank(level domain.SupportLevel) int {
	switch level {
	case domain.SupportSyntactic:
		return 1
	case domain.SupportResolved:
		return 2
	default:
		return 0
	}
}

// buildResults maps the findings of the run onto the results of the document.
//
// The results are sorted by path, line and rule identifier, like the findings
// of the audit document, so that two runs over the same change set produce
// the same document.
func buildResults(findings []domain.Finding, changes *domain.ChangeSet, index catalogIndex) []sarifResult {
	sorted := make([]domain.Finding, len(findings))
	copy(sorted, findings)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, right := sorted[i], sorted[j]
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

	results := make([]sarifResult, 0, len(sorted))
	for _, finding := range sorted {
		results = append(results, resultEntry(finding, changes, index))
	}
	return results
}

// resultEntry maps one finding onto its SARIF result.
func resultEntry(finding domain.Finding, changes *domain.ChangeSet, index catalogIndex) sarifResult {
	return sarifResult{
		RuleID:    finding.RuleID,
		RuleIndex: index.position(finding.RuleID),
		Level:     sarifLevel(finding.Severity),
		Message:   sarifMessage{Text: finding.Message},
		Locations: []sarifLocation{{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: artifactURI(finding.Location.Path)},
				Region:           regionOf(finding.Location),
			},
		}},
		PartialFingerprints: sarifPartialFingerprints{CodixaiV1: fingerprintOf(finding, changes)},
		Properties: sarifResultProperties{
			EpistemicStatus: string(finding.EpistemicStatus),
			SupportLevel:    supportLevelOf(finding, index.rule(finding.RuleID)),
			RuleVersion:     finding.RuleVersion,
		},
	}
}

// fingerprintOf returns the identity of a finding.
//
// The identity is computed from the path as the baseline stores it, never
// from the URI of the document: the fingerprint of the export and the
// fingerprint of the baseline have to be the same value.
func fingerprintOf(finding domain.Finding, changes *domain.ChangeSet) string {
	return baseline.Fingerprint(finding.RuleID, finding.Location.Path, baseline.Context(changes, finding))
}

// supportLevelOf returns the support level of a finding.
//
// A finding carries the level of its own analysis. A finding that does not
// declare it is a finding of the rule, so the depth of the rule describes it,
// and "none" when the rule is not described either: the export never invents
// a depth of analysis.
func supportLevelOf(finding domain.Finding, rule domain.Rule) string {
	if finding.Support.Level != "" {
		return string(finding.Support.Level)
	}
	return supportOf(rule).Level
}

// sarifLevel maps a severity of Codixai onto an SARIF level.
//
//   - error stays error, warning stays warning, info becomes note.
//   - A severity outside the catalog becomes none, which is SARIF's way of
//     saying that the result does not fail anything: the export never
//     escalates a severity it does not know.
func sarifLevel(severity domain.Severity) string {
	switch severity {
	case domain.SeverityError:
		return "error"
	case domain.SeverityWarning:
		return "warning"
	case domain.SeverityInfo:
		return "note"
	default:
		return "none"
	}
}

// artifactURI returns the URI of a path of the repository.
//
// SARIF locations are URIs, so the separators of the analyzed platform are
// normalized to forward slashes. The path stays relative to the root of the
// repository: the document never exposes the absolute path of the machine.
func artifactURI(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// regionOf returns the range of a finding.
//
// A region needs a line, and SARIF has no line zero. A finding without a line
// therefore has no region; its location is the file, which is what the
// analysis knows.
func regionOf(location domain.Location) *sarifRegion {
	if location.StartLine <= 0 {
		return nil
	}
	end := location.EndLine
	if end < location.StartLine {
		end = location.StartLine
	}
	return &sarifRegion{StartLine: location.StartLine, EndLine: end}
}
