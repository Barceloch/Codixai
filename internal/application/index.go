// Package application holds the use cases of Codixai.
package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
	"github.com/barceloch/codixai/internal/storage"
)

// IndexRequest describes an index operation.
type IndexRequest struct {
	// Root is the absolute path of the repository to index.
	Root string
	// IndexPath is the path to the SQLite index file.
	// If empty, a default location (.codixai/index.db) is used.
	IndexPath string
	// ForceRebuild forces a full rebuild of the index.
	ForceRebuild bool
	// ToolVersion is the version of the tool reported in the index.
	ToolVersion string
	// AnalyzerVersion is the version of the analyzer.
	AnalyzerVersion string
}

// IndexResult is the outcome of an index operation.
type IndexResult struct {
	// Root is the repository root that was indexed.
	Root string
	// IndexPath is the path to the index file.
	IndexPath string
	// FilesIndexed is the number of files indexed.
	FilesIndexed int
	// SymbolsCount is the number of symbols indexed.
	SymbolsCount int
	// DependenciesCount is the number of dependencies indexed.
	DependenciesCount int
	// Duration is the time taken to build the index.
	Duration time.Duration
}

// RunIndex builds or updates the structural index of a repository.
func RunIndex(ctx context.Context, request IndexRequest) (*IndexResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Resolve index path
	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	// Ensure index directory exists
	indexDir := filepath.Dir(indexPath)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	// Open or create index store
	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()

	var filesIndexed, symbolsCount, depsCount int
	start := time.Now()

	// Check if we should force rebuild or if index doesn't exist
	forceRebuild := request.ForceRebuild
	if !forceRebuild {
		status, err := store.Status(ctx, request.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to check index status: %w", err)
		}
		if status.FileCount == 0 {
			forceRebuild = true
		}
	}

	if forceRebuild {
		// Build index from source
		idx := index.New()
		if err := idx.Build(ctx, request.Root); err != nil {
			return nil, fmt.Errorf("failed to build index: %w", err)
		}

		// Save to storage
		if err := store.Save(ctx, idx, request.Root, request.ToolVersion, request.AnalyzerVersion); err != nil {
			return nil, fmt.Errorf("failed to save index: %w", err)
		}

		// Get counts
		symbols := idx.Symbols()
		deps := idx.Dependencies()

		// Count unique files
		fileSet := make(map[string]bool)
		for _, sym := range idx.Symbols() {
			fileSet[sym.File] = true
		}
		for _, dep := range idx.Dependencies() {
			fileSet[dep.File] = true
		}

		filesIndexed = len(fileSet)
		symbolsCount = len(symbols)
		depsCount = len(deps)
	} else {
		// Load existing index
		_, err := store.Load(ctx, request.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to load index: %w", err)
		}

		// Get counts from status
		status, err := store.Status(ctx, request.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to get index status: %w", err)
		}
		filesIndexed = status.FileCount
		symbolsCount = status.SymbolCount
		depsCount = status.DepCount
	}

	elapsed := time.Since(start)

	return &IndexResult{
		Root:              request.Root,
		IndexPath:         indexPath,
		FilesIndexed:      filesIndexed,
		SymbolsCount:      symbolsCount,
		DependenciesCount: depsCount,
		Duration:          elapsed,
	}, nil
}

// IndexStatusRequest describes a request to get index status.
type IndexStatusRequest struct {
	Root      string
	IndexPath string
}

// RunIndexStatus returns the status of an index.
func RunIndexStatus(ctx context.Context, request IndexStatusRequest) (*storage.IndexStatus, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}

	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()

	return store.Status(ctx, request.Root)
}

// IndexFindRequest describes a request to find symbols in the index.
type IndexFindRequest struct {
	Root      string
	IndexPath string
	Query     string
}

// RunIndexFind searches for symbols in the index.
func RunIndexFind(ctx context.Context, request IndexFindRequest) ([]storage.SymbolResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}
	// Empty query is allowed - it returns all symbols (used for overview)

	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()

	return store.ListSymbols(ctx, request.Root, request.Query)
}

// IndexRefsRequest describes a request to find references of a symbol.
type IndexRefsRequest struct {
	Root      string
	IndexPath string
	Symbol    string
}

// RunIndexRefs finds references of a symbol in the index.
func RunIndexRefs(ctx context.Context, request IndexRefsRequest) ([]storage.SymbolResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}
	if strings.TrimSpace(request.Symbol) == "" {
		return nil, fmt.Errorf("symbol name is required")
	}

	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()

	return store.ListSymbols(ctx, request.Root, request.Symbol)
}

// IndexDepsRequest describes a request to list dependencies.
type IndexDepsRequest struct {
	Root      string
	IndexPath string
}

// RunIndexDeps lists declared dependencies from the index.
func RunIndexDeps(ctx context.Context, request IndexDepsRequest) ([]storage.DependencyResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}

	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()

	return store.ListDependencies(ctx, request.Root)
}

// IndexDiffRequest describes a request to compare blobs between base and HEAD.
type IndexDiffRequest struct {
	Root string
	Base string
}

// IndexDiffResult holds the result of a blob comparison.
type IndexDiffResult struct {
	Root  string
	Base  string
	Head  string
	Files []DiffFileInfo
}

// DiffFileInfo holds information about a changed file.
type DiffFileInfo struct {
	Path     string
	OldPath  string
	Status   string
	Language string
	Added    int
	Removed  int
}

// RunIndexDiff compares blobs between base and HEAD without checkout.
func RunIndexDiff(ctx context.Context, request IndexDiffRequest) (*IndexDiffResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}
	if strings.TrimSpace(request.Base) == "" {
		return nil, fmt.Errorf("base revision is required")
	}

	// This uses the existing git integration, not the structural index
	// It's kept here for CLI compatibility
	return nil, fmt.Errorf("not implemented: use 'codixai audit --diff' for diff analysis")
}

// ResolveReferencesRequest describes a request to resolve and persist references.
type ResolveReferencesRequest struct {
	Root      string
	IndexPath string
	Changes   *domain.ChangeSet
}

// ResolveReferencesResult holds resolved references.
type ResolveReferencesResult struct {
	Root       string
	References []domain.Reference
}

// RunResolveReferences resolves references for a changeset using the structural index
// and persists them. It supports Go (Phase 5.2.2), Python (Phase 5.2.3),
// JavaScript (Phase 5.2.4.3), and TypeScript (Phase 5.2.4.4). The JS/TS module
// graph is resolved from the index when JS/TS references are resolved.
func RunResolveReferences(ctx context.Context, request ResolveReferencesRequest) (*ResolveReferencesResult, error) {
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the index needs the root of the repository")
	}
	if request.Changes == nil {
		return nil, fmt.Errorf("changeset is required")
	}
	indexPath := request.IndexPath
	if indexPath == "" {
		indexPath = filepath.Join(request.Root, ".codixai", "index.db")
	}

	idx := index.New()
	if err := idx.Build(ctx, request.Root); err != nil {
		return nil, fmt.Errorf("failed to build index for resolution: %w", err)
	}

	var refs []domain.Reference
	goRefs, err := index.NewGoReferenceResolver().Resolve(ctx, idx, request.Changes)
	if err != nil {
		return nil, err
	}
	refs = append(refs, goRefs...)
	pyRefs, err := index.NewPythonReferenceResolver().Resolve(ctx, idx, request.Changes)
	if err != nil {
		return nil, err
	}
	refs = append(refs, pyRefs...)
	jsRefs, err := index.NewJavaScriptReferenceResolver().Resolve(ctx, idx, request.Changes)
	if err != nil {
		return nil, err
	}
	refs = append(refs, jsRefs...)
	tsRefs, err := index.NewTypeScriptReferenceResolver().Resolve(ctx, idx, request.Changes)
	if err != nil {
		return nil, err
	}
	refs = append(refs, tsRefs...)
	refs = index.NormalizeReferences(refs)

	store, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index store: %w", err)
	}
	defer store.Close()
	if err := store.Save(ctx, idx, request.Root, "codixai", "0.1"); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}
	if err := store.SaveReferences(ctx, refs, request.Root); err != nil {
		return nil, fmt.Errorf("failed to save references: %w", err)
	}

	return &ResolveReferencesResult{Root: request.Root, References: refs}, nil
}
