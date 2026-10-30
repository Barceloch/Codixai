package visualization

import (
	"fmt"
	"sort"

	"github.com/barceloch/codixai/internal/index"
)

// VisualizationResult holds the results of structural visualization.
type VisualizationResult struct {
	// Symbols lists all symbols in the index.
	Symbols []SymbolInfo `json:"symbols"`
	// Dependencies lists all dependencies.
	Dependencies []DependencyInfo `json:"dependencies"`
	// Summary provides a summary of the visualization.
	Summary map[string]int `json:"summary"`
}

// SymbolInfo holds information about a symbol.
type SymbolInfo struct {
	// Name is the symbol name.
	Name string `json:"name"`
	// Kind is the kind of symbol.
	Kind string `json:"kind"`
	// File is the file containing the symbol.
	File string `json:"file"`
	// Language is the language of the symbol.
	Language string `json:"language"`
}

// DependencyInfo holds information about a dependency.
type DependencyInfo struct {
	// Path is the dependency path.
	Path string `json:"path"`
	// File is the file containing the dependency.
	File string `json:"file"`
	// Language is the language of the dependency.
	Language string `json:"language"`
}

// Visualize performs structural visualization of the repository.
func Visualize(idx *index.Index) *VisualizationResult {
	result := &VisualizationResult{
		Summary: map[string]int{},
	}

	// Collect symbols
	symbols := idx.Symbols()
	symbolInfo := make([]SymbolInfo, len(symbols))
	for i, sym := range symbols {
		symbolInfo[i] = SymbolInfo{
			Name:     sym.Name,
			Kind:     sym.Kind,
			File:     sym.File,
			Language: sym.Language,
		}
		result.Summary["total_symbols"] = len(symbols)
	}

	// Sort symbols by name
	sort.Slice(symbolInfo, func(i, j int) bool {
		return symbolInfo[i].Name < symbolInfo[j].Name
	})
	result.Symbols = symbolInfo

	// Collect dependencies
	deps := idx.Dependencies()
	depInfo := make([]DependencyInfo, len(deps))
	for i, dep := range deps {
		depInfo[i] = DependencyInfo{
			Path:     dep.Path,
			File:     dep.File,
			Language: dep.Language,
		}
	}
	result.Dependencies = depInfo

	// Add summary entries
	result.Summary["total_dependencies"] = len(deps)

	return result
}

// PrintVisualization prints the visualization result to stdout.
func PrintVisualization(result *VisualizationResult) {
	fmt.Println("=== Structural Visualization ===")
	fmt.Println()

	fmt.Println("=== Symbols ===")
	for _, sym := range result.Symbols {
		fmt.Printf("  %s (%s) [%s]\n", sym.Name, sym.Kind, sym.File)
	}

	fmt.Println()
	fmt.Println("=== Dependencies ===")
	for _, dep := range result.Dependencies {
		fmt.Printf("  %s [%s]\n", dep.Path, dep.Language)
	}

	fmt.Println()
	fmt.Println("Summary:")
	for k, v := range result.Summary {
		fmt.Printf("  %s: %d\n", k, v)
	}
}
