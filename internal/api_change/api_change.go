// Package api_change implements public API change detection.
// It identifies breaking changes (removed functions, signature changes) and affected consumers.
package api_change

import (
	"context"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
)

// BreakingChangeType represents the type of breaking change detected.
type BreakingChangeType string

const (
	// BreakingChangeFunctionRemoved indicates a public function was removed.
	BreakingChangeFunctionRemoved BreakingChangeType = "function_removed"
	// BreakingChangeFunctionSignatureChanged indicates a public function signature changed.
	BreakingChangeFunctionSignatureChanged BreakingChangeType = "function_signature_changed"
	// BreakingChangeTypeRemoved indicates a public type was removed.
	BreakingChangeTypeRemoved BreakingChangeType = "type_removed"
	// BreakingChangeTypeSignatureChanged indicates a public type signature changed.
	BreakingChangeTypeSignatureChanged BreakingChangeType = "type_signature_changed"
	// BreakingChangeMethodRemoved indicates a public method was removed.
	BreakingChangeMethodRemoved BreakingChangeType = "method_removed"
	// BreakingChangeMethodSignatureChanged indicates a public method signature changed.
	BreakingChangeMethodSignatureChanged BreakingChangeType = "method_signature_changed"
)

// BreakingChange represents a detected breaking change.
type BreakingChange struct {
	// Type is the type of breaking change.
	Type BreakingChangeType `json:"type"`
	// Language is the language of the changed symbol.
	Language string `json:"language"`
	// File is the file where the change was detected.
	File string `json:"file"`
	// Symbol is the name of the affected symbol.
	Symbol string `json:"symbol"`
	// OldSignature is the old signature (if applicable).
	OldSignature string `json:"old_signature,omitempty"`
	// NewSignature is the new signature (if applicable).
	NewSignature string `json:"new_signature,omitempty"`
	// Consumers lists the files that may be affected.
	Consumers []string `json:"consumers,omitempty"`
	// Severity is the severity of the breaking change.
	Severity domain.Severity `json:"severity"`
	// EpistemicStatus indicates how the finding was established.
	EpistemicStatus domain.EpistemicStatus `json:"epistemic_status"`
	// Evidence provides verifiable references.
	Evidence []domain.Evidence `json:"evidence,omitempty"`
}

// APIChangeResult holds the results of API change analysis.
type APIChangeResult struct {
	// BreakingChanges lists all detected breaking changes.
	BreakingChanges []BreakingChange `json:"breaking_changes"`
	// Summary provides a summary of changes by type.
	Summary map[BreakingChangeType]int `json:"summary"`
}

// Detector detects public API changes between two revisions.
type Detector struct {
	index *index.Index
}

// NewDetector creates a new API change detector.
func NewDetector(idx *index.Index) *Detector {
	return &Detector{index: idx}
}

// Detect analyzes the diff between base and head to find breaking changes.
func (d *Detector) Detect(ctx context.Context, changes *domain.ChangeSet) (*APIChangeResult, error) {
	if changes == nil || len(changes.Files) == 0 {
		return &APIChangeResult{
			BreakingChanges: []BreakingChange{},
			Summary:         map[BreakingChangeType]int{},
		}, nil
	}

	var breakingChanges []BreakingChange

	for _, file := range changes.Files {
		// Only analyze files in supported languages
		if !isSupportedLanguage(file.Language) {
			continue
		}

		// For each hunk, analyze the removed/added lines
		for _, hunk := range file.Hunks {
			removedLines := collectLines(hunk, domain.LineRemoved)
			addedLines := collectLines(hunk, domain.LineAdded)

			// Detect breaking changes based on language
			switch file.Language {
			case domain.LangGo:
				changes := d.detectGoBreakingChanges(file.Path, removedLines, addedLines)
				breakingChanges = append(breakingChanges, changes...)
			case domain.LangPython, domain.LangJavaScript, domain.LangTypeScript, domain.LangPHP:
				changes := d.detectLexicalBreakingChanges(file.Path, file.Language, removedLines, addedLines)
				breakingChanges = append(breakingChanges, changes...)
			}
		}
	}

	// Find consumers for each breaking change
	for i := range breakingChanges {
		breakingChanges[i].Consumers = d.findConsumers(changes, breakingChanges[i])
	}

	// Build summary
	summary := map[BreakingChangeType]int{}
	for _, bc := range breakingChanges {
		summary[bc.Type]++
	}

	return &APIChangeResult{
		BreakingChanges: breakingChanges,
		Summary:         summary,
	}, nil
}

func isSupportedLanguage(lang string) bool {
	switch lang {
	case domain.LangGo, domain.LangPython, domain.LangJavaScript, domain.LangTypeScript, domain.LangPHP:
		return true
	default:
		return false
	}
}

func collectLines(hunk domain.Hunk, kind string) []string {
	var lines []string
	for _, line := range hunk.Lines {
		if line.Kind == kind {
			lines = append(lines, line.Content)
		}
	}
	return lines
}

func (d *Detector) detectGoBreakingChanges(filePath string, removedLines, addedLines []string) []BreakingChange {
	var changes []BreakingChange

	removedContent := strings.Join(removedLines, "\n")
	addedContent := strings.Join(addedLines, "\n")

	// Extract public symbols from Go diff content using lexical patterns
	removedSymbols := extractGoLexicalPublicSymbols(removedContent)
	addedSymbols := extractGoLexicalPublicSymbols(addedContent)

	// Find removed functions (breaking change: function removed)
	for _, sym := range removedSymbols {
		if _, exists := addedSymbols[sym.Name]; !exists {
			changes = append(changes, BreakingChange{
				Type:            BreakingChangeFunctionRemoved,
				Language:        domain.LangGo,
				File:            filePath,
				Symbol:          sym.Name,
				OldSignature:    sym.Signature,
				Severity:        domain.SeverityError,
				EpistemicStatus: domain.EpistemicInferred,
				Evidence: []domain.Evidence{
					{Kind: "diff_hunk", Ref: filePath + ":function_removed:" + sym.Name, Detail: "Public function " + sym.Name + " was removed"},
				},
			})
		}
	}

	// Find signature changes
	for _, oldSym := range removedSymbols {
		if newSym, exists := addedSymbols[oldSym.Name]; exists {
			if oldSym.Signature != newSym.Signature {
				changes = append(changes, BreakingChange{
					Type:            BreakingChangeFunctionSignatureChanged,
					Language:        domain.LangGo,
					File:            filePath,
					Symbol:          oldSym.Name,
					OldSignature:    oldSym.Signature,
					NewSignature:    newSym.Signature,
					Severity:        domain.SeverityError,
					EpistemicStatus: domain.EpistemicInferred,
					Evidence: []domain.Evidence{
						{Kind: "diff_hunk", Ref: filePath + ":signature_changed:" + oldSym.Name, Detail: "Function signature changed from " + oldSym.Signature + " to " + newSym.Signature},
					},
				})
			}
		}
	}

	return changes
}

func (d *Detector) detectLexicalBreakingChanges(filePath, language string, removedLines, addedLines []string) []BreakingChange {
	var changes []BreakingChange

	removedContent := strings.Join(removedLines, "\n")
	addedContent := strings.Join(addedLines, "\n")

	// Extract function/type definitions using lexical patterns
	removedSymbols := extractLexicalPublicSymbols(language, removedContent)
	addedSymbols := extractLexicalPublicSymbols(language, addedContent)

	// Find removed functions
	for _, sym := range removedSymbols {
		if _, exists := addedSymbols[sym.Name]; !exists {
			changes = append(changes, BreakingChange{
				Type:            BreakingChangeFunctionRemoved,
				Language:        language,
				File:            filePath,
				Symbol:          sym.Name,
				OldSignature:    sym.Signature,
				Severity:        domain.SeverityError,
				EpistemicStatus: domain.EpistemicInferred,
				Evidence: []domain.Evidence{
					{Kind: "diff_hunk", Ref: filePath + ":function_removed:" + sym.Name, Detail: "Public function " + sym.Name + " was removed"},
				},
			})
		}
	}

	// Find signature changes
	for _, oldSym := range removedSymbols {
		if newSym, exists := addedSymbols[oldSym.Name]; exists {
			if oldSym.Signature != newSym.Signature {
				changes = append(changes, BreakingChange{
					Type:            BreakingChangeFunctionSignatureChanged,
					Language:        language,
					File:            filePath,
					Symbol:          oldSym.Name,
					OldSignature:    oldSym.Signature,
					NewSignature:    newSym.Signature,
					Severity:        domain.SeverityError,
					EpistemicStatus: domain.EpistemicInferred,
					Evidence: []domain.Evidence{
						{Kind: "diff_hunk", Ref: filePath + ":signature_changed:" + oldSym.Name, Detail: "Function signature changed from " + oldSym.Signature + " to " + newSym.Signature},
					},
				})
			}
		}
	}

	return changes
}

// extractGoLexicalPublicSymbols extracts public Go symbols from diff content
// using lexical patterns (since diff content may not be a complete valid Go file).
func extractGoLexicalPublicSymbols(content string) map[string]LexicalSymbol {
	symbols := map[string]LexicalSymbol{}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Go: func FuncName(...) { or func FuncName(...) (
		if strings.HasPrefix(trimmed, "func ") {
			// Skip methods (func (receiver) MethodName)
			if strings.HasPrefix(trimmed, "func (") {
				continue
			}
			// func FuncName(...)
			if idx := strings.Index(trimmed[5:], "("); idx >= 0 {
				name := strings.TrimSpace(trimmed[5 : 5+idx])
				// Only public functions (capitalized)
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					symbols[name] = LexicalSymbol{Name: name, Signature: trimmed}
				}
			}
		}

		// type TypeName struct/interface
		if strings.HasPrefix(trimmed, "type ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := parts[1]
				// Only public types
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					symbols[name] = LexicalSymbol{Name: name, Signature: trimmed}
				}
			}
		}
	}

	return symbols
}

// LexicalSymbol represents a symbol extracted via lexical analysis.
type LexicalSymbol struct {
	Name      string
	Signature string
}

// extractLexicalPublicSymbols extracts public symbols using language-specific lexical patterns.
func extractLexicalPublicSymbols(language, content string) map[string]LexicalSymbol {
	symbols := map[string]LexicalSymbol{}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		var sig string
		var name string

		switch language {
		case domain.LangPython:
			// Python: def func_name(...):
			if strings.HasPrefix(trimmed, "def ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[4:idx])
					sig = trimmed
				}
			}
			// async def func_name(...):
			if strings.HasPrefix(trimmed, "async def ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[10:idx])
					sig = trimmed
				}
			}
			// class ClassName:
			if strings.HasPrefix(trimmed, "class ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[6:idx])
				} else if idx := strings.Index(trimmed, ":"); idx > 0 {
					name = strings.TrimSpace(trimmed[6:idx])
				} else {
					name = strings.TrimSpace(trimmed[6:])
				}
				sig = trimmed
			}
		case domain.LangJavaScript, domain.LangTypeScript:
			// function funcName(...)
			if strings.HasPrefix(trimmed, "function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[9:idx])
					sig = trimmed
				}
			}
			// const funcName = (...) => ...
			if strings.HasPrefix(trimmed, "const ") && strings.Contains(trimmed, "=>") {
				parts := strings.Split(trimmed, "=")
				if len(parts) == 2 {
					name = strings.TrimSpace(strings.TrimPrefix(parts[0], "const"))
					sig = trimmed
				}
			}
			// export function funcName(...)
			if strings.HasPrefix(trimmed, "export function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[16:idx])
					sig = trimmed
				}
			}
			// class ClassName ...
			if strings.HasPrefix(trimmed, "class ") {
				parts := strings.Fields(trimmed)
				if len(parts) > 1 {
					name = parts[1]
					sig = trimmed
				}
			}
		case domain.LangPHP:
			// function funcName(...)
			if strings.HasPrefix(trimmed, "function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[9:idx])
					sig = trimmed
				}
			}
			// public function funcName(...)
			if strings.HasPrefix(trimmed, "public function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name = strings.TrimSpace(trimmed[16:idx])
					sig = trimmed
				}
			}
			// class ClassName ...
			if strings.HasPrefix(trimmed, "class ") {
				parts := strings.Fields(trimmed)
				if len(parts) > 1 {
					name = parts[1]
					sig = trimmed
				}
			}
		}

		if name != "" && sig != "" {
			// Only consider public symbols (no leading underscore, not private)
			if !strings.HasPrefix(name, "_") {
				symbols[name] = LexicalSymbol{Name: name, Signature: sig}
			}
		}
	}

	return symbols
}

// findConsumers finds files that may consume the changed symbol.
func (d *Detector) findConsumers(changes *domain.ChangeSet, bc BreakingChange) []string {
	var consumers []string

	// Search in all files in the index for references to the symbol
	// This is a simplified approach - a real implementation would use proper reference resolution
	if d.index != nil {
		allSymbols := d.index.Symbols()
		for _, sym := range allSymbols {
			// Check if this symbol file might consume the broken symbol
			// This is a heuristic: if the file is in the same package/directory or imports it
			if sym.File != bc.File && strings.Contains(strings.ToLower(sym.File), strings.ToLower(bc.Symbol)) {
				consumers = append(consumers, sym.File)
			}
		}
	}

	// Also check changed files for direct references
	for _, file := range changes.Files {
		if file.Path == bc.File {
			continue
		}
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if strings.Contains(line.Content, bc.Symbol) {
					if !contains(consumers, file.Path) {
						consumers = append(consumers, file.Path)
					}
				}
			}
		}
	}

	return consumers
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}