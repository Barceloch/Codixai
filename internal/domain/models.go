// Package domain contains the stable, transport-agnostic models of Codixai.
//
// The models in this package are the contract shared by the CLI, the future
// MCP server and any other consumer. They must not import CLI, storage,
// framework or AI provider packages.
package domain

// SchemaVersion is the version of the JSON output contract. It changes when a
// consumer-visible field is added, removed or changes meaning in a way that is
// not backward compatible.
const SchemaVersion = "0.1"

// EpistemicStatus classifies how a Finding was established.
type EpistemicStatus string

const (
	// EpistemicObserved means the finding was checked directly against code or diff.
	EpistemicObserved EpistemicStatus = "observed"
	// EpistemicInferred means the finding was derived by a heuristic.
	EpistemicInferred EpistemicStatus = "inferred"
	// EpistemicUnknown means the tool could not determine the outcome.
	EpistemicUnknown EpistemicStatus = "unknown"
)

// Resolution classifies the state of a Relation between symbols or files.
type Resolution string

const (
	ResolutionConfirmed  Resolution = "confirmed"
	ResolutionInferred   Resolution = "inferred"
	ResolutionUnresolved Resolution = "unresolved"
)

// Severity is the impact class of a Finding.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// SupportLevel declares how deep the analysis is for a rule and a language.
type SupportLevel string

const (
	// SupportNone means the rule is not available for that language.
	SupportNone SupportLevel = "none"
	// SupportSyntactic means the rule works on syntax only (Level 1 of the spec).
	SupportSyntactic SupportLevel = "syntactic"
	// SupportResolved means the rule relies on resolved relations (Level 2 or above).
	SupportResolved SupportLevel = "resolved"
)

// Language identifiers supported by the product from the first milestone.
const (
	LangGo         = "go"
	LangPython     = "python"
	LangPHP        = "php"
	LangJavaScript = "javascript"
	LangTypeScript = "typescript"
)

// Location points at a range inside a file of the analyzed repository.
type Location struct {
	Path        string `json:"path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line,omitempty"`
	StartColumn int    `json:"start_column,omitempty"`
	EndColumn   int    `json:"end_column,omitempty"`
}

// Evidence is a verifiable reference that supports a Finding or a Claim result.
type Evidence struct {
	// Kind is the evidence category, e.g. "diff_hunk", "source_range", "manifest_entry".
	Kind string `json:"kind"`
	// Ref is the stable reference of the evidence, e.g. "hunk:3" or "path:line".
	Ref string `json:"ref"`
	// Detail is an optional human readable clarification.
	Detail string `json:"detail,omitempty"`
}

// Support declares the analysis depth and its known limitations.
type Support struct {
	Level       SupportLevel `json:"level"`
	Method      string       `json:"method,omitempty"`
	Limitations []string     `json:"limitations,omitempty"`
}

// Revision identifies the compared endpoints of a diff.
type Revision struct {
	Base string `json:"base,omitempty"`
	Head string `json:"head,omitempty"`
}

// Finding is the result of a rule applied to a change or a repository.
type Finding struct {
	SchemaVersion   string          `json:"schema_version"`
	RuleID          string          `json:"rule_id"`
	RuleVersion     string          `json:"rule_version"`
	Severity        Severity        `json:"severity"`
	EpistemicStatus EpistemicStatus `json:"epistemic_status"`
	Language        string          `json:"language,omitempty"`
	Message         string          `json:"message"`
	Location        Location        `json:"location"`
	Revision        Revision        `json:"revision,omitempty"`
	Evidence        []Evidence      `json:"evidence,omitempty"`
	Support         Support         `json:"support,omitempty"`
	Recommendation  string          `json:"recommendation,omitempty"`
}

// Rule is a deterministic check with a stable identifier.
type Rule struct {
	ID           string                  `json:"id"`
	Version      string                  `json:"version"`
	Area         string                  `json:"area"`
	Description  string                  `json:"description"`
	Severity     Severity                `json:"severity"`
	Languages    map[string]SupportLevel `json:"languages"`
	Method       string                  `json:"method,omitempty"`
	Limitations  []string                `json:"limitations,omitempty"`
	EvidenceKind string                  `json:"evidence_kind,omitempty"`
}

// Summary aggregates findings or claim results.
type Summary struct {
	Total             int            `json:"total"`
	BySeverity        map[string]int `json:"by_severity,omitempty"`
	ByRule            map[string]int `json:"by_rule,omitempty"`
	ByLanguage        map[string]int `json:"by_language,omitempty"`
	ByEpistemicStatus map[string]int `json:"by_epistemic_status,omitempty"`
}

// AnalysisRun is the document produced by an analysis over a diff.
type AnalysisRun struct {
	SchemaVersion string    `json:"schema_version"`
	ToolVersion   string    `json:"tool_version"`
	GeneratedAt   string    `json:"generated_at"`
	Origin        string    `json:"origin"`
	Revision      Revision  `json:"revision"`
	Findings      []Finding `json:"findings"`
	Summary       Summary   `json:"summary"`
	Rules         []Rule    `json:"rules,omitempty"`
	Limitations   []string  `json:"limitations,omitempty"`
}

// Claim is an explicit assertion made about a change.
type Claim struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Paths   []string `json:"paths,omitempty"`
	Symbols []string `json:"symbols,omitempty"`
	Report  string   `json:"report,omitempty"`
}

// Claim status values.
const (
	ClaimVerified     = "verified"
	ClaimContradicted = "contradicted"
	ClaimUnresolved   = "unresolved"
)

// ClaimResult is the outcome of checking one claim.
type ClaimResult struct {
	Claim    Claim      `json:"claim"`
	Status   string     `json:"status"`
	Message  string     `json:"message"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// VerificationRun is the document produced by verifying a set of claims.
type VerificationRun struct {
	SchemaVersion string        `json:"schema_version"`
	ToolVersion   string        `json:"tool_version"`
	GeneratedAt   string        `json:"generated_at"`
	Base          string        `json:"base"`
	Head          string        `json:"head"`
	Results       []ClaimResult `json:"results"`
	Summary       Summary       `json:"summary"`
	Limitations   []string      `json:"limitations,omitempty"`
}

// Repository describes the analyzed project.
type Repository struct {
	Root      string   `json:"root"`
	Revision  Revision `json:"revision"`
	Languages []string `json:"languages,omitempty"`
	Tracked   bool     `json:"tracked"`
}

// File is a file considered by the analysis.
type File struct {
	Path     string `json:"path"`
	Language string `json:"language,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Tracked  bool   `json:"tracked"`
}

// Symbol is a named declaration with its position.
type Symbol struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Location  Location `json:"location"`
	Container string   `json:"container,omitempty"`
}

// Relation links two symbols or files.
type Relation struct {
	Type       string     `json:"type"`
	From       Location   `json:"from"`
	To         Location   `json:"to,omitempty"`
	Target     string     `json:"target,omitempty"`
	Method     string     `json:"method"`
	Resolution Resolution `json:"resolution"`
}

// Diagnostic reports a parsing or analysis problem.
type Diagnostic struct {
	Path     string   `json:"path"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line,omitempty"`
}
