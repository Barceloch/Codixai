// Package measurement provides a minimal, deterministic, local-first harness for
// recording and comparing the token and context consumption of AI coding agents
// with and without Codixai.
//
// The harness never calls a remote service and never executes analyzed code. It
// consumes run records produced by an operator and computes a comparison report.
// Token counts are carried with an explicit provenance so that an estimate is
// never presented as a provider-reported measurement.
package measurement

import (
	"errors"
	"fmt"
	"time"
)

// SchemaVersion is the schema version of the measurement documents.
const SchemaVersion = "0.1"

// TokenSource describes the provenance of a token count.
type TokenSource string

const (
	// TokenSourceProviderReported is an actual count reported by a model provider.
	TokenSourceProviderReported TokenSource = "provider_reported"
	// TokenSourceTokenizerEstimate is a count produced by a local tokenizer.
	TokenSourceTokenizerEstimate TokenSource = "tokenizer_estimate"
	// TokenSourceByteProxy is the bytes/4 proxy, used for sizing only.
	TokenSourceByteProxy TokenSource = "byte_proxy"
	// TokenSourceUnknown means the count is not available.
	TokenSourceUnknown TokenSource = "unknown"
)

// TokenMetric is a token count together with its provenance.
type TokenMetric struct {
	Value  *int64      `json:"value"` // nil when the count is unknown
	Source TokenSource `json:"source"`
}

// Metrics holds all measured quantities for one run.
type Metrics struct {
	InputTokens    TokenMetric `json:"input_tokens"`
	OutputTokens   TokenMetric `json:"output_tokens"`
	TotalTokens    TokenMetric `json:"total_tokens"`
	FilesInspected *int        `json:"files_inspected"` // nil when unknown
	FilesReadFull  *int        `json:"files_read_full"`
	ToolCalls      *int        `json:"tool_calls"`
	TaskCompleted  *bool       `json:"task_completed"`
	TaskCorrect    *bool       `json:"task_correct"`
	DurationMs     *int64      `json:"duration_ms"`
}

// Repository identifies the repository state of a run.
type Repository struct {
	Root     string `json:"root"`
	Revision string `json:"revision"`
	State    string `json:"state"` // clean, dirty or unknown
}

// Model identifies the model used for a run.
type Model struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Version  string `json:"version,omitempty"`
}

// Environment identifies the tooling environment of a run.
type Environment struct {
	CodixaiVersion string `json:"codixai_version,omitempty"`
	OS             string `json:"os,omitempty"`
	Arch           string `json:"arch,omitempty"`
}

// Completeness records how many known references the run found.
type Completeness struct {
	ReferencesFound *int `json:"references_found,omitempty"`
	ReferencesKnown *int `json:"references_known,omitempty"`
}

// RunRecord is one recorded measurement run.
type RunRecord struct {
	SchemaVersion   string       `json:"schema_version"`
	RunID           string       `json:"run_id"`
	Scenario        string       `json:"scenario"` // baseline or codixai
	TaskID          string       `json:"task_id"`
	TaskDescription string       `json:"task_description,omitempty"`
	Repository      Repository   `json:"repository"`
	Model           Model        `json:"model"`
	Environment     Environment  `json:"environment,omitempty"`
	Metrics         Metrics      `json:"metrics"`
	Completeness    Completeness `json:"completeness,omitempty"`
	Notes           string       `json:"notes,omitempty"`
	RecordedAt      string       `json:"recorded_at"`
}

// validTokenSources is the set of accepted token sources.
var validTokenSources = map[TokenSource]bool{
	TokenSourceProviderReported:  true,
	TokenSourceTokenizerEstimate: true,
	TokenSourceByteProxy:         true,
	TokenSourceUnknown:           true,
}

// ValidateRunRecord checks that a run record is well-formed.
func ValidateRunRecord(r RunRecord) error {
	if r.SchemaVersion == "" {
		return errors.New("schema_version is required")
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version %q does not match %q", r.SchemaVersion, SchemaVersion)
	}
	if r.RunID == "" {
		return errors.New("run_id is required")
	}
	if r.Scenario != "baseline" && r.Scenario != "codixai" {
		return fmt.Errorf("scenario must be \"baseline\" or \"codixai\", got %q", r.Scenario)
	}
	if r.TaskID == "" {
		return errors.New("task_id is required")
	}
	if r.Model.Provider == "" {
		return errors.New("model.provider is required")
	}
	if r.Model.Model == "" {
		return errors.New("model.model is required")
	}
	if r.Repository.State != "" && r.Repository.State != "clean" && r.Repository.State != "dirty" && r.Repository.State != "unknown" {
		return fmt.Errorf("repository.state must be clean, dirty or unknown, got %q", r.Repository.State)
	}
	if r.RecordedAt == "" {
		return errors.New("recorded_at is required")
	}
	if _, err := time.Parse(time.RFC3339, r.RecordedAt); err != nil {
		return fmt.Errorf("recorded_at must be RFC3339: %w", err)
	}
	if err := validateTokenMetric("input_tokens", r.Metrics.InputTokens); err != nil {
		return err
	}
	if err := validateTokenMetric("output_tokens", r.Metrics.OutputTokens); err != nil {
		return err
	}
	if err := validateTokenMetric("total_tokens", r.Metrics.TotalTokens); err != nil {
		return err
	}
	for name, v := range map[string]*int{
		"files_inspected": r.Metrics.FilesInspected,
		"files_read_full": r.Metrics.FilesReadFull,
		"tool_calls":      r.Metrics.ToolCalls,
	} {
		if v != nil && *v < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	if r.Metrics.DurationMs != nil && *r.Metrics.DurationMs < 0 {
		return errors.New("duration_ms must not be negative")
	}
	if r.Completeness.ReferencesFound != nil && *r.Completeness.ReferencesFound < 0 {
		return errors.New("completeness.references_found must not be negative")
	}
	if r.Completeness.ReferencesKnown != nil && *r.Completeness.ReferencesKnown < 0 {
		return errors.New("completeness.references_known must not be negative")
	}
	return nil
}

func validateTokenMetric(name string, m TokenMetric) error {
	if !validTokenSources[m.Source] {
		return fmt.Errorf("%s.source %q is not valid", name, m.Source)
	}
	if m.Value == nil {
		if m.Source != TokenSourceUnknown {
			return fmt.Errorf("%s has no value but source is %q", name, m.Source)
		}
		return nil
	}
	if m.Source == TokenSourceUnknown {
		return fmt.Errorf("%s has a value but source is unknown", name)
	}
	if *m.Value < 0 {
		return fmt.Errorf("%s must not be negative", name)
	}
	return nil
}

// Comparability describes whether two runs can be compared.
type Comparability struct {
	Comparable bool     `json:"comparable"`
	Reasons    []string `json:"reasons"`
}

// RunSummary is the compact metric view of a run.
type RunSummary struct {
	InputTokens    TokenMetric `json:"input_tokens"`
	OutputTokens   TokenMetric `json:"output_tokens"`
	TotalTokens    TokenMetric `json:"total_tokens"`
	FilesInspected *int        `json:"files_inspected"`
	FilesReadFull  *int        `json:"files_read_full"`
	ToolCalls      *int        `json:"tool_calls"`
	TaskCompleted  *bool       `json:"task_completed"`
	TaskCorrect    *bool       `json:"task_correct"`
	DurationMs     *int64      `json:"duration_ms"`
}

func summaryOf(r RunRecord) RunSummary {
	return RunSummary{
		InputTokens:    r.Metrics.InputTokens,
		OutputTokens:   r.Metrics.OutputTokens,
		TotalTokens:    r.Metrics.TotalTokens,
		FilesInspected: r.Metrics.FilesInspected,
		FilesReadFull:  r.Metrics.FilesReadFull,
		ToolCalls:      r.Metrics.ToolCalls,
		TaskCompleted:  r.Metrics.TaskCompleted,
		TaskCorrect:    r.Metrics.TaskCorrect,
		DurationMs:     r.Metrics.DurationMs,
	}
}

// Savings holds percentage savings when computable.
type Savings struct {
	InputTokens  *float64 `json:"input_tokens_percent,omitempty"`
	OutputTokens *float64 `json:"output_tokens_percent,omitempty"`
	TotalTokens  *float64 `json:"total_tokens_percent,omitempty"`
	Formula      string   `json:"formula"`
}

// CompletenessDelta records the change in reference completeness.
type CompletenessDelta struct {
	BaselineFound *int `json:"baseline_references_found,omitempty"`
	CodixaFound   *int `json:"codixai_references_found,omitempty"`
	BaselineKnown *int `json:"baseline_references_known,omitempty"`
	CodixaKnown   *int `json:"codixai_references_known,omitempty"`
}

// ComparisonReport is the result of comparing two runs.
type ComparisonReport struct {
	SchemaVersion  string             `json:"schema_version"`
	ToolVersion    string             `json:"tool_version"`
	TaskID         string             `json:"task_id"`
	Comparability  Comparability      `json:"comparability"`
	Baseline       RunSummary         `json:"baseline"`
	Codixai        RunSummary         `json:"codixai"`
	Savings        *Savings           `json:"savings"`
	Completeness   *CompletenessDelta `json:"completeness,omitempty"`
	Interpretation string             `json:"interpretation"`
	Limitations    []string           `json:"limitations"`
}

// Compare validates and compares two run records.
func Compare(baseline, withCodixai RunRecord, toolVersion string) (ComparisonReport, error) {
	if err := ValidateRunRecord(baseline); err != nil {
		return ComparisonReport{}, fmt.Errorf("baseline: %w", err)
	}
	if err := ValidateRunRecord(withCodixai); err != nil {
		return ComparisonReport{}, fmt.Errorf("codixai run: %w", err)
	}

	report := ComparisonReport{
		SchemaVersion: SchemaVersion,
		ToolVersion:   toolVersion,
		TaskID:        baseline.TaskID,
		Comparability: Comparability{Comparable: true, Reasons: []string{}},
		Baseline:      summaryOf(baseline),
		Codixai:       summaryOf(withCodixai),
		Limitations: []string{
			"Token counts are recorded by the operator; this tool does not call any model provider.",
			"Estimated or proxy values are never reported as provider-reported savings.",
			"Completeness is reported alongside any savings; a saving that drops a reference is a regression.",
		},
	}

	var reasons []string
	if baseline.Scenario != "baseline" {
		reasons = append(reasons, "baseline scenario is not \"baseline\"")
	}
	if withCodixai.Scenario != "codixai" {
		reasons = append(reasons, "codixai scenario is not \"codixai\"")
	}
	if baseline.TaskID != withCodixai.TaskID {
		reasons = append(reasons, "task_id differs between runs")
	}
	if baseline.Repository.Revision != withCodixai.Repository.Revision {
		reasons = append(reasons, "repository revision differs between runs")
	}
	if baseline.Model.Provider != withCodixai.Model.Provider || baseline.Model.Model != withCodixai.Model.Model {
		reasons = append(reasons, "model differs between runs")
	}
	if len(reasons) > 0 {
		report.Comparability = Comparability{Comparable: false, Reasons: reasons}
		report.Interpretation = "runs are not comparable: " + joinReasons(reasons)
		return report, nil
	}

	// Savings are computed only from provider-reported total tokens.
	bTotal := baseline.Metrics.TotalTokens
	cTotal := withCodixai.Metrics.TotalTokens
	if bTotal.Source == TokenSourceProviderReported && cTotal.Source == TokenSourceProviderReported &&
		bTotal.Value != nil && cTotal.Value != nil && *bTotal.Value > 0 {
		bv := float64(*bTotal.Value)
		cv := float64(*cTotal.Value)
		totalPct := (1.0 - cv/bv) * 100.0
		savings := &Savings{Formula: "Savings = (1 - Tc / Tb) * 100"}
		savings.TotalTokens = &totalPct

		if bIn := baseline.Metrics.InputTokens; bIn.Source == TokenSourceProviderReported && bIn.Value != nil && *bIn.Value > 0 {
			if cIn := withCodixai.Metrics.InputTokens; cIn.Source == TokenSourceProviderReported && cIn.Value != nil {
				v := (1.0 - float64(*cIn.Value)/float64(*bIn.Value)) * 100.0
				savings.InputTokens = &v
			}
		}
		if bOut := baseline.Metrics.OutputTokens; bOut.Source == TokenSourceProviderReported && bOut.Value != nil && *bOut.Value > 0 {
			if cOut := withCodixai.Metrics.OutputTokens; cOut.Source == TokenSourceProviderReported && cOut.Value != nil {
				v := (1.0 - float64(*cOut.Value)/float64(*bOut.Value)) * 100.0
				savings.OutputTokens = &v
			}
		}
		report.Savings = savings
		report.Interpretation = fmt.Sprintf("comparable runs; total-token savings = %.2f%%", totalPct)
	} else {
		report.Interpretation = "runs are comparable, but total-token savings cannot be computed: " +
			"both totals must be provider_reported with a positive baseline"
	}

	// Completeness delta.
	cd := &CompletenessDelta{}
	if baseline.Completeness.ReferencesFound != nil {
		v := *baseline.Completeness.ReferencesFound
		cd.BaselineFound = &v
	}
	if withCodixai.Completeness.ReferencesFound != nil {
		v := *withCodixai.Completeness.ReferencesFound
		cd.CodixaFound = &v
	}
	if baseline.Completeness.ReferencesKnown != nil {
		v := *baseline.Completeness.ReferencesKnown
		cd.BaselineKnown = &v
	}
	if withCodixai.Completeness.ReferencesKnown != nil {
		v := *withCodixai.Completeness.ReferencesKnown
		cd.CodixaKnown = &v
	}
	report.Completeness = cd

	return report, nil
}

// EstimateTokensFromBytes returns a deterministic byte/4 proxy. The result is an
// estimate and must never be reported as a provider-reported token count.
func EstimateTokensFromBytes(n int) int {
	if n < 0 {
		return 0
	}
	return n / 4
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	out := reasons[0]
	for i := 1; i < len(reasons); i++ {
		out += "; " + reasons[i]
	}
	return out
}
