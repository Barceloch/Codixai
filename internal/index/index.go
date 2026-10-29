// Package index implements the structural indexing of a repository.
// It uses pure-Go AST parsers and lexical profiles to extract symbols.
//
// The index is built on demand, stored in memory, and satisfies the
// structural analysis requirements of Phase 5.
package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/barceloch/codixai/internal/domain"
	golang "github.com/barceloch/codixai/internal/languages/go"
	"github.com/barceloch/codixai/internal/languages/registry"
)

// Symbol represents a basic symbol extracted from source code.
type Symbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Language string `json:"language"`
}

// Dependency represents a declared dependency in the repository.
type Dependency struct {
	Path     string `json:"path"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Language string `json:"language"`
}

// Index holds the structural symbols and dependencies of the repository.
type Index struct {
	mu           sync.RWMutex
	symbols      []Symbol
	dependencies []Dependency
}

// New creates an empty Index.
func New() *Index {
	return &Index{
		symbols:      []Symbol{},
		dependencies: []Dependency{},
	}
}

// shouldSkipDir checks if a directory should be skipped during indexing.
// Uses exact directory name matching to avoid false positives like vendor_test.go.
func shouldSkipDir(name string) bool {
	return name == ".git" || name == "vendor" || name == "node_modules"
}

// isInsideSkippedDir checks if a path is inside a skipped directory.
func isInsideSkippedDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		if shouldSkipDir(part) {
			return true
		}
	}
	return false
}

// Build traverses the repository root and builds the index.
func (i *Index) Build(ctx context.Context, root string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.symbols = []Symbol{}
	i.dependencies = []Dependency{}

	// Ensure root is absolute for reliable path comparisons
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return nil // Continue walking, log error if needed
		}

		// Skip directories that should be excluded
		if info.IsDir() {
			if shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks to avoid following outside the root
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Ensure path doesn't escape the root
		if !strings.HasPrefix(path, absRoot) {
			return nil
		}

		detectedLang, ok := registry.Detect(path)
		if !ok {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		syms, deps := extract(path, src, detectedLang)
		i.symbols = append(i.symbols, syms...)
		i.dependencies = append(i.dependencies, deps...)
		return nil
	})
	return err
}

// Find searches for symbols matching a query.
func (i *Index) Find(query string) []Symbol {
	i.mu.RLock()
	defer i.mu.RUnlock()

	results := []Symbol{}
	queryLower := strings.ToLower(query)
	for _, sym := range i.symbols {
		if strings.Contains(strings.ToLower(sym.Name), queryLower) {
			results = append(results, sym)
		}
	}
	return results
}

// Symbols returns all symbols.
func (i *Index) Symbols() []Symbol {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.symbols
}

// Dependencies returns all declared dependencies.
func (i *Index) Dependencies() []Dependency {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dependencies
}

func extract(path string, src []byte, lang string) ([]Symbol, []Dependency) {
	if lang == domain.LangGo {
		return extractGo(path, src)
	}
	// Fallback for other languages: file-level symbols only
	return []Symbol{
		{
			Name:     filepath.Base(path),
			Kind:     "file",
			File:     path,
			Line:     0,
			Language: lang,
		},
	}, nil
}

func extractGo(path string, src []byte) ([]Symbol, []Dependency) {
	f, err := golang.Parse(filepath.Base(path), src)
	if err != nil {
		return nil, nil
	}

	var syms []Symbol
	var deps []Dependency
	if f.Package != "" {
		syms = append(syms, Symbol{
			Name:     f.Package,
			Kind:     "package",
			File:     path,
			Line:     0,
			Language: domain.LangGo,
		})
	}
	for _, imp := range f.Imports {
		deps = append(deps, Dependency{
			Path:     imp.Path,
			File:     path,
			Line:     imp.Line,
			Language: domain.LangGo,
		})
	}
	return syms, deps
}