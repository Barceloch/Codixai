// Package index implements the structural indexing of a repository.
package index

import (
	"context"

	"github.com/barceloch/codixai/internal/domain"
)

// ReferenceResolver resolves references from syntactic occurrences to declared symbols.
type ReferenceResolver interface {
	// Resolve resolves references in the given change set using the index.
	// The index provides the symbol table and import graph.
	// Changes provides the syntactic occurrences to resolve.
	Resolve(ctx context.Context, idx *Index, changes *domain.ChangeSet) ([]domain.Reference, error)
}

// ReferenceOccurrence represents a syntactic occurrence of a reference in source code.
type ReferenceOccurrence struct {
	// Name is the name being referenced.
	Name string
	// Kind is the type of reference.
	Kind domain.ReferenceKind
	// File is the file containing the reference.
	File string
	// Line is the line number.
	Line int
	// Column is the column number.
	Column int
	// Language is the programming language.
	Language string
	// Container is the containing symbol (function, class, etc.) when known.
	Container string
	// Evidence is the syntactic evidence for this occurrence.
	Evidence []domain.ReferenceEvidence
}

// SymbolRefWithID extends SymbolRef with an internal ID for resolution.
type SymbolRefWithID struct {
	domain.SymbolRef
	// ID is the internal database ID of the symbol.
	ID int64
}

// ResolverConfig configures the reference resolver.
type ResolverConfig struct {
	// IncludeUnresolved includes unresolved references in the output.
	IncludeUnresolved bool
	// MaxCandidates limits the number of candidates for ambiguous references.
	MaxCandidates int
}

// DefaultResolverConfig returns a sensible default configuration.
func DefaultResolverConfig() ResolverConfig {
	return ResolverConfig{
		IncludeUnresolved: true,
		MaxCandidates:     10,
	}
}
