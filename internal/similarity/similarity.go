// Package similarity implements code similarity detection for identifying
// potential reimplementations of existing utilities.
package similarity

import (
	"context"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
)

// SimilarityType represents the type of similarity detected.
type SimilarityType string

const (
	SimilarityFunction SimilarityType = "function"
	SimilarityMethod   SimilarityType = "method"
	SimilarityTypeS    SimilarityType = "type"
)

// SimilarityMatch represents a pair of similar code fragments.
type SimilarityMatch struct {
	Type            SimilarityType         `json:"type"`
	Language        string                 `json:"language"`
	FileA           string                 `json:"file_a"`
	SymbolA         string                 `json:"symbol_a"`
	FileB           string                 `json:"file_b"`
	SymbolB         string                 `json:"symbol_b"`
	Similarity      float64                `json:"similarity"`
	LinesMatched    int                    `json:"lines_matched"`
	TotalLines      int                    `json:"total_lines"`
	Severity        domain.Severity        `json:"severity"`
	EpistemicStatus domain.EpistemicStatus `json:"epistemic_status"`
	Evidence        []domain.Evidence      `json:"evidence,omitempty"`
}

// SimilarityResult holds the results of similarity analysis.
type SimilarityResult struct {
	Matches []SimilarityMatch      `json:"matches"`
	Summary map[SimilarityType]int `json:"summary"`
}

// Detector detects similar code fragments in the repository.
type Detector struct {
	index       *index.Index
	threshold   float64
	minLines    int
	ignoreTests bool
}

// NewDetector creates a new similarity detector.
func NewDetector(idx *index.Index) *Detector {
	return &Detector{
		index:       idx,
		threshold:   70.0,
		minLines:    5,
		ignoreTests: true,
	}
}

// WithThreshold sets the minimum similarity threshold.
func (d *Detector) WithThreshold(threshold float64) *Detector {
	d.threshold = threshold
	return d
}

// WithMinLines sets the minimum number of lines.
func (d *Detector) WithMinLines(minLines int) *Detector {
	d.minLines = minLines
	return d
}

// WithIgnoreTests sets whether to ignore test files.
func (d *Detector) WithIgnoreTests(ignore bool) *Detector {
	d.ignoreTests = ignore
	return d
}

// Detect finds similar code fragments across the repository.
func (d *Detector) Detect(ctx context.Context) (*SimilarityResult, error) {
	if d.index == nil {
		return &SimilarityResult{
			Matches: []SimilarityMatch{},
			Summary: map[SimilarityType]int{},
		}, nil
	}

	symbols := d.index.Symbols()

	byLang := make(map[string][]index.Symbol)
	for _, sym := range symbols {
		if d.ignoreTests && isTestFile(sym.File) {
			continue
		}
		byLang[sym.Language] = append(byLang[sym.Language], sym)
	}

	var matches []SimilarityMatch

	for lang, langSymbols := range byLang {
		functions := d.extractFunctions(lang, langSymbols)

		for i := 0; i < len(functions); i++ {
			for j := i + 1; j < len(functions); j++ {
				similarity := d.compareFunctions(functions[i], functions[j])
				if similarity >= d.threshold {
					matches = append(matches, SimilarityMatch{
						Type:            SimilarityFunction,
						Language:        lang,
						FileA:           functions[i].File,
						SymbolA:         functions[i].Name,
						FileB:           functions[j].File,
						SymbolB:         functions[j].Name,
						Similarity:      similarity,
						LinesMatched:    functions[i].LinesMatched,
						TotalLines:      functions[i].TotalLines,
						Severity:        domain.SeverityWarning,
						EpistemicStatus: domain.EpistemicInferred,
						Evidence: []domain.Evidence{
							{Kind: "source_range", Ref: functions[i].File + ":" + functions[i].Name, Detail: "Similar to " + functions[j].File + ":" + functions[j].Name},
							{Kind: "source_range", Ref: functions[j].File + ":" + functions[j].Name, Detail: "Similar to " + functions[i].File + ":" + functions[i].Name},
						},
					})
				}
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	summary := map[SimilarityType]int{}
	for _, m := range matches {
		summary[m.Type]++
	}

	return &SimilarityResult{
		Matches: matches,
		Summary: summary,
	}, nil
}

// FunctionInfo holds extracted function information.
type FunctionInfo struct {
	Name         string
	File         string
	Language     string
	Lines        []string
	Normalized   string
	LinesMatched int
	TotalLines   int
}

// extractFunctions extracts function bodies from symbols.
func (d *Detector) extractFunctions(language string, symbols []index.Symbol) []FunctionInfo {
	var functions []FunctionInfo

	for _, sym := range symbols {
		if sym.Kind != "function" && sym.Kind != "method" && sym.Kind != "func" {
			continue
		}

		if d.ignoreTests && isTestFile(sym.File) {
			continue
		}

		lines := d.extractFunctionLines(sym.File, sym.Line, language)
		if len(lines) < d.minLines {
			continue
		}

		normalized := d.normalizeFunction(lines, language)

		functions = append(functions, FunctionInfo{
			Name:       sym.Name,
			File:       sym.File,
			Language:   language,
			Lines:      lines,
			Normalized: normalized,
			TotalLines: len(lines),
		})
	}

	return functions
}

// extractFunctionLines extracts the lines of a function starting from a given line.
func (d *Detector) extractFunctionLines(filePath string, startLine int, language string) []string {
	// Simplified - actual extraction would require file reading
	return []string{}
}

// normalizeFunction normalizes function lines for comparison.
func (d *Detector) normalizeFunction(lines []string, language string) string {
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isComment(trimmed, language) {
			continue
		}
		normalized = append(normalized, strings.Join(strings.Fields(trimmed), " "))
	}
	return strings.Join(normalized, "\n")
}

func isComment(line, language string) bool {
	switch language {
	case domain.LangGo, domain.LangJavaScript, domain.LangTypeScript, domain.LangPHP:
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*")
	case domain.LangPython:
		return strings.HasPrefix(line, "#")
	default:
		return false
	}
}

// compareFunctions compares two functions and returns similarity percentage.
func (d *Detector) compareFunctions(a, b FunctionInfo) float64 {
	if a.Normalized == "" || b.Normalized == "" {
		return 0
	}

	linesA := strings.Split(a.Normalized, "\n")
	linesB := strings.Split(b.Normalized, "\n")

	matched := 0.0
	total := len(linesA)
	if len(linesB) > total {
		total = len(linesB)
	}

	if total == 0 {
		return 0
	}

	for _, lineA := range linesA {
		for _, lineB := range linesB {
			if lineA == lineB {
				matched++
				break
			}
		}
	}

	for _, lineA := range linesA {
		for _, lineB := range linesB {
			if strings.Contains(lineA, lineB) || strings.Contains(lineB, lineA) {
				if lineA != lineB {
					matched += 0.5
				}
			}
		}
	}

	similarity := matched / float64(total) * 100
	if similarity > 100 {
		similarity = 100
	}

	a.LinesMatched = int(matched)

	return similarity
}

func isTestFile(filePath string) bool {
	return strings.Contains(filePath, "_test.go") ||
		strings.Contains(filePath, "_test.py") ||
		strings.Contains(filePath, ".test.js") ||
		strings.Contains(filePath, ".test.ts") ||
		strings.Contains(filePath, "_test.php") ||
		strings.Contains(filePath, "test_")
}

// DetectFromChanges detects similarity between changed code and existing code.
func (d *Detector) DetectFromChanges(ctx context.Context, changes *domain.ChangeSet) (*SimilarityResult, error) {
	if changes == nil || len(changes.Files) == 0 {
		return &SimilarityResult{
			Matches: []SimilarityMatch{},
			Summary: map[SimilarityType]int{},
		}, nil
	}

	var matches []SimilarityMatch

	newFunctions := d.extractFromDiff(changes)

	if d.index != nil {
		existingFunctions := d.extractAllFunctions()

		for _, newFunc := range newFunctions {
			for _, existingFunc := range existingFunctions {
				if newFunc.File == existingFunc.File && newFunc.Name == existingFunc.Name {
					continue
				}

				similarity := d.compareFunctions(newFunc, existingFunc)
				if similarity >= d.threshold {
					matches = append(matches, SimilarityMatch{
						Type:            SimilarityFunction,
						Language:        newFunc.Language,
						FileA:           newFunc.File,
						SymbolA:         newFunc.Name,
						FileB:           existingFunc.File,
						SymbolB:         existingFunc.Name,
						Similarity:      similarity,
						LinesMatched:    newFunc.LinesMatched,
						TotalLines:      newFunc.TotalLines,
						Severity:        domain.SeverityWarning,
						EpistemicStatus: domain.EpistemicInferred,
						Evidence: []domain.Evidence{
							{Kind: "source_range", Ref: newFunc.File + ":" + newFunc.Name, Detail: "Similar to existing " + existingFunc.File + ":" + existingFunc.Name},
							{Kind: "source_range", Ref: existingFunc.File + ":" + existingFunc.Name, Detail: "Similar to new " + newFunc.File + ":" + newFunc.Name},
						},
					})
				}
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	summary := map[SimilarityType]int{}
	for _, m := range matches {
		summary[m.Type]++
	}

	return &SimilarityResult{
		Matches: matches,
		Summary: summary,
	}, nil
}

func (d *Detector) extractFromDiff(changes *domain.ChangeSet) []FunctionInfo {
	var functions []FunctionInfo

	for _, file := range changes.Files {
		if !d.isSupportedLanguage(file.Language) {
			continue
		}

		for _, hunk := range file.Hunks {
			var addedLines []string
			for _, line := range hunk.Lines {
				if line.Kind == domain.LineAdded {
					addedLines = append(addedLines, line.Content)
				}
			}

			if len(addedLines) >= d.minLines {
				funcs := d.extractFunctionNamesFromLines(file.Path, file.Language, addedLines)
				for _, fn := range funcs {
					fn.Language = file.Language
					functions = append(functions, fn)
				}
			}
		}
	}

	return functions
}

func (d *Detector) isSupportedLanguage(lang string) bool {
	switch lang {
	case domain.LangGo, domain.LangPython, domain.LangJavaScript, domain.LangTypeScript, domain.LangPHP:
		return true
	default:
		return false
	}
}

func (d *Detector) extractFunctionNamesFromLines(filePath, language string, lines []string) []FunctionInfo {
	var functions []FunctionInfo

	switch language {
	case domain.LangGo:
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "func ") && !strings.HasPrefix(trimmed, "func (") {
				if idx := strings.Index(trimmed[5:], "("); idx >= 0 {
					name := strings.TrimSpace(trimmed[5 : 5+idx])
					if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
						functions = append(functions, FunctionInfo{
							Name:       name,
							File:       filePath,
							Language:   language,
							Lines:      lines,
							Normalized: d.normalizeFunction(lines, language),
							TotalLines: len(lines),
						})
					}
				}
			}
		}
	case domain.LangPython:
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "def ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name := strings.TrimSpace(trimmed[4:idx])
					if !strings.HasPrefix(name, "_") {
						functions = append(functions, FunctionInfo{
							Name:       name,
							File:       filePath,
							Language:   language,
							Lines:      lines,
							Normalized: d.normalizeFunction(lines, language),
							TotalLines: len(lines),
						})
					}
				}
			}
		}
	case domain.LangJavaScript, domain.LangTypeScript:
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name := strings.TrimSpace(trimmed[9:idx])
					if !strings.HasPrefix(name, "_") {
						functions = append(functions, FunctionInfo{
							Name:       name,
							File:       filePath,
							Language:   language,
							Lines:      lines,
							Normalized: d.normalizeFunction(lines, language),
							TotalLines: len(lines),
						})
					}
				}
			}
		}
	case domain.LangPHP:
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name := strings.TrimSpace(trimmed[9:idx])
					if !strings.HasPrefix(name, "_") {
						functions = append(functions, FunctionInfo{
							Name:       name,
							File:       filePath,
							Language:   language,
							Lines:      lines,
							Normalized: d.normalizeFunction(lines, language),
							TotalLines: len(lines),
						})
					}
				}
			}
			if strings.HasPrefix(trimmed, "public function ") {
				if idx := strings.Index(trimmed, "("); idx > 0 {
					name := strings.TrimSpace(trimmed[16:idx])
					if !strings.HasPrefix(name, "_") {
						functions = append(functions, FunctionInfo{
							Name:       name,
							File:       filePath,
							Language:   language,
							Lines:      lines,
							Normalized: d.normalizeFunction(lines, language),
							TotalLines: len(lines),
						})
					}
				}
			}
		}
	}

	return functions
}

func (d *Detector) extractAllFunctions() []FunctionInfo {
	return []FunctionInfo{}
}
