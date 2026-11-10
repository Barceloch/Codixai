// Package index implements the structural indexing of a repository.
package index

import (
	"fmt"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// NewReferenceEvidence creates a new ReferenceEvidence with the given parameters.
func NewReferenceEvidence(kind domain.ReferenceEvidenceKind, ref, detail, method string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   kind,
		Ref:    ref,
		Detail: detail,
		Method: method,
	}
}

// NewASTCallEvidence creates evidence for an AST call expression.
func NewASTCallEvidence(file string, line, column int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceASTCall,
		Ref:    fmt.Sprintf("%s:%d:%d", file, line, column),
		Detail: detail,
		Method: "ast",
	}
}

// NewASTMethodCallEvidence creates evidence for an AST method call expression.
func NewASTMethodCallEvidence(file string, line, column int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceASTMethodCall,
		Ref:    fmt.Sprintf("%s:%d:%d", file, line, column),
		Detail: detail,
		Method: "ast",
	}
}

// NewASTTypeReferenceEvidence creates evidence for an AST type reference.
func NewASTTypeReferenceEvidence(file string, line, column int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceASTTypeReference,
		Ref:    fmt.Sprintf("%s:%d:%d", file, line, column),
		Detail: detail,
		Method: "ast",
	}
}

// NewASTImportEvidence creates evidence for an AST import declaration.
func NewASTImportEvidence(file string, line int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceASTImport,
		Ref:    fmt.Sprintf("%s:%d", file, line),
		Detail: detail,
		Method: "ast",
	}
}

// NewLexicalCallPatternEvidence creates evidence for a lexical call pattern.
func NewLexicalCallPatternEvidence(file string, line int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceLexicalCallPattern,
		Ref:    fmt.Sprintf("%s:%d", file, line),
		Detail: detail,
		Method: "lexical_pattern",
	}
}

// NewLexicalMethodCallEvidence creates evidence for a lexical method call.
func NewLexicalMethodCallEvidence(file string, line int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceLexicalMethodCall,
		Ref:    fmt.Sprintf("%s:%d", file, line),
		Detail: detail,
		Method: "lexical_pattern",
	}
}

// NewLexicalTypeReferenceEvidence creates evidence for a lexical type reference.
func NewLexicalTypeReferenceEvidence(file string, line int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceLexicalTypeReference,
		Ref:    fmt.Sprintf("%s:%d", file, line),
		Detail: detail,
		Method: "lexical_pattern",
	}
}

// NewImportDeclarationEvidence creates evidence for an import declaration.
func NewImportDeclarationEvidence(file string, line int, importPath string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceImportDeclaration,
		Ref:    fmt.Sprintf("import:%s", importPath),
		Detail: fmt.Sprintf("Import declaration at %s:%d", file, line),
		Method: "import_graph",
	}
}

// NewModuleResolutionEvidence creates evidence for a module resolution step.
func NewModuleResolutionEvidence(detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceModuleResolution,
		Ref:    "module_resolution",
		Detail: detail,
		Method: "module_resolution",
	}
}

// NewHeuristicCandidateEvidence creates evidence for a heuristic candidate match.
func NewHeuristicCandidateEvidence(candidateName, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceHeuristicCandidate,
		Ref:    "heuristic:" + candidateName,
		Detail: detail,
		Method: "heuristic",
	}
}

// NewDynamicDispatchEvidence creates evidence for a dynamic dispatch reference.
func NewDynamicDispatchEvidence(file string, line int, detail string) domain.ReferenceEvidence {
	return domain.ReferenceEvidence{
		Kind:   domain.EvidenceDynamicDispatch,
		Ref:    fmt.Sprintf("%s:%d", file, line),
		Detail: detail,
		Method: "dynamic",
	}
}

// MergeEvidence merges multiple evidence slices, removing semantically identical
// duplicates while preserving distinct evidence records that differ in meaningful
// provenance details. The merged result is deterministically ordered.
func MergeEvidence(evidenceList ...[]domain.ReferenceEvidence) []domain.ReferenceEvidence {
	var all []domain.ReferenceEvidence
	for _, evList := range evidenceList {
		all = append(all, evList...)
	}
	return NormalizeEvidence(all)
}

// ReferenceKindString returns a human-readable string for a ReferenceKind.
func ReferenceKindString(kind domain.ReferenceKind) string {
	switch kind {
	case domain.ReferenceCall:
		return "call"
	case domain.ReferenceMethod:
		return "method"
	case domain.ReferenceType:
		return "type"
	case domain.ReferenceImport:
		return "import"
	case domain.ReferenceInheritance:
		return "inheritance"
	case domain.ReferenceVariable:
		return "variable"
	case domain.ReferenceField:
		return "field"
	default:
		return string(kind)
	}
}

// ResolutionStatusString returns a human-readable string for a ResolutionStatus.
func ResolutionStatusString(status domain.ResolutionStatus) string {
	switch status {
	case domain.ResolutionResolved:
		return "resolved"
	case domain.ResolutionAmbiguous:
		return "ambiguous"
	case domain.ResolutionUnresolved:
		return "unresolved"
	case domain.ResolutionExternal:
		return "external"
	case domain.ResolutionDynamic:
		return "dynamic"
	default:
		return string(status)
	}
}

// ResolutionStatusFromString parses a ResolutionStatus from string.
func ResolutionStatusFromString(s string) domain.ResolutionStatus {
	switch strings.ToLower(s) {
	case "resolved":
		return domain.ResolutionResolved
	case "ambiguous":
		return domain.ResolutionAmbiguous
	case "unresolved":
		return domain.ResolutionUnresolved
	case "external":
		return domain.ResolutionExternal
	case "dynamic":
		return domain.ResolutionDynamic
	default:
		return domain.ResolutionUnresolved
	}
}
