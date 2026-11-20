// Package domain contains the stable, transport-agnostic models of Codixai.
package domain

// ReferenceKind classifies the type of reference between symbols.
type ReferenceKind string

const (
	// ReferenceCall represents a function or method call.
	ReferenceCall ReferenceKind = "call"
	// ReferenceMethod represents a method call on a receiver.
	ReferenceMethod ReferenceKind = "method"
	// ReferenceType represents a type reference.
	ReferenceType ReferenceKind = "type"
	// ReferenceImport represents an import statement.
	ReferenceImport ReferenceKind = "import"
	// ReferenceInheritance represents inheritance or implementation.
	ReferenceInheritance ReferenceKind = "inheritance"
	// ReferenceVariable represents a variable reference.
	ReferenceVariable ReferenceKind = "variable"
	// ReferenceField represents a field or property access.
	ReferenceField ReferenceKind = "field"
)

// ResolutionStatus classifies the resolution state of a reference.
type ResolutionStatus string

const (
	// ResolutionResolved means the reference was resolved to a specific target.
	ResolutionResolved ResolutionStatus = "resolved"
	// ResolutionAmbiguous means the reference matches multiple candidates.
	ResolutionAmbiguous ResolutionStatus = "ambiguous"
	// ResolutionUnresolved means the reference could not be resolved.
	ResolutionUnresolved ResolutionStatus = "unresolved"
	// ResolutionExternal means the reference points to an external dependency.
	ResolutionExternal ResolutionStatus = "external"
	// ResolutionDynamic means the reference involves dynamic dispatch.
	ResolutionDynamic ResolutionStatus = "dynamic"
)

// ReferenceEvidenceKind classifies the type of evidence supporting a reference.
type ReferenceEvidenceKind string

const (
	// EvidenceASTCall represents an AST call expression node.
	EvidenceASTCall ReferenceEvidenceKind = "ast_call"
	// EvidenceASTMethodCall represents an AST method call expression.
	EvidenceASTMethodCall ReferenceEvidenceKind = "ast_method_call"
	// EvidenceASTTypeReference represents an AST type reference.
	EvidenceASTTypeReference ReferenceEvidenceKind = "ast_type_reference"
	// EvidenceASTImport represents an AST import declaration.
	EvidenceASTImport ReferenceEvidenceKind = "ast_import"
	// EvidenceLexicalCallPattern represents a lexical call pattern match.
	EvidenceLexicalCallPattern ReferenceEvidenceKind = "lexical_call_pattern"
	// EvidenceLexicalMethodCall represents a lexical method call pattern.
	EvidenceLexicalMethodCall ReferenceEvidenceKind = "lexical_method_call"
	// EvidenceLexicalTypeReference represents a lexical type reference pattern.
	EvidenceLexicalTypeReference ReferenceEvidenceKind = "lexical_type_reference"
	// EvidenceImportDeclaration represents an import declaration.
	EvidenceImportDeclaration ReferenceEvidenceKind = "import_declaration"
	// EvidenceModuleResolution represents a module resolution step.
	EvidenceModuleResolution ReferenceEvidenceKind = "module_resolution"
	// EvidenceHeuristicCandidate represents a heuristic candidate match.
	EvidenceHeuristicCandidate ReferenceEvidenceKind = "heuristic_candidate"
	// EvidenceDynamicDispatch represents a dynamic dispatch reference.
	EvidenceDynamicDispatch ReferenceEvidenceKind = "dynamic_dispatch"
)

// ReferenceEvidence supports a reference with verifiable evidence.
type ReferenceEvidence struct {
	// Kind is the evidence category.
	Kind ReferenceEvidenceKind `json:"kind"`
	// Ref is the stable reference of the evidence, e.g. "file.go:42:10" or "import:github.com/foo/bar".
	Ref string `json:"ref"`
	// Detail is an optional human readable clarification.
	Detail string `json:"detail,omitempty"`
	// Method is the method used to obtain this evidence.
	Method string `json:"method,omitempty"`
}

// Reference represents a reference between symbols.
type Reference struct {
	// ID is a stable identifier for this reference.
	ID string `json:"id,omitempty"`
	// Source is the symbol making the reference.
	Source SymbolRef `json:"source"`
	// Target is the symbol being referenced, nil when unresolved.
	Target *SymbolRef `json:"target,omitempty"`
	// Kind is the type of reference.
	Kind ReferenceKind `json:"kind"`
	// Resolution is the resolution status of this reference.
	Resolution ResolutionStatus `json:"resolution"`
	// Candidates are alternative targets when resolution is ambiguous.
	Candidates []SymbolRef `json:"candidates,omitempty"`
	// Evidence supports the reference.
	Evidence []ReferenceEvidence `json:"evidence,omitempty"`
	// Language is the language of the source code.
	Language string `json:"language"`
	// Location is the location of the reference in source code.
	Location Location `json:"location"`
}

// SymbolRef identifies a symbol for reference resolution.
type SymbolRef struct {
	// Name is the name of the symbol.
	Name string `json:"name"`
	// Kind is the kind of symbol (function, method, type, variable, package, etc.)
	Kind string `json:"kind"`
	// File is the repository-relative path of the file containing the symbol.
	File string `json:"file"`
	// Line is the line number of the symbol.
	Line int `json:"line"`
	// Column is the column of the symbol.
	Column int `json:"column,omitempty"`
	// Language is the programming language.
	Language string `json:"language"`
	// Container is the containing package, class, or namespace.
	Container string `json:"container,omitempty"`
	// Signature is the signature of the symbol when available.
	Signature string `json:"signature,omitempty"`
	// Exported indicates whether the symbol is exported/public.
	// When unknown, this field is omitted (zero value = false, but we use a pointer).
	Exported *bool `json:"exported,omitempty"`
	// SignatureHash is a hash of the signature for change detection.
	SignatureHash string `json:"signature_hash,omitempty"`
}
