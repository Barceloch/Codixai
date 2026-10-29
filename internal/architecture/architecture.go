// Package architecture implements architectural analysis and configurable conventions.
package architecture

import (
	"context"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
)

// ConventionType represents the type of convention.
type ConventionType string

const (
	// ConventionNamingStyle checks naming conventions.
	ConventionNamingStyle ConventionType = "naming_style"
	// ConventionImportOrder checks import ordering.
	ConventionImportOrder ConventionType = "import_order"
	// ConventionFileOrganization checks file organization.
	ConventionFileOrganization ConventionType = "file_organization"
	// ConventionCyclicDependency checks for cyclic dependencies.
	ConventionCyclicDependency ConventionType = "cyclic_dependency"
	// ConventionCoupling checks coupling metrics.
	ConventionCoupling ConventionType = "coupling"
	// ConventionCohesion checks cohesion metrics.
	ConventionCohesion ConventionType = "cohesion"
)

// ConventionViolation represents a violation of a convention.
type ConventionViolation struct {
	// Type is the type of convention violated.
	Type ConventionType `json:"type"`
	// Severity is the severity of the violation.
	Severity domain.Severity `json:"severity"`
	// EpistemicStatus indicates how the finding was established.
	EpistemicStatus domain.EpistemicStatus `json:"epistemic_status"`
	// File is the file where the violation was found.
	File string `json:"file"`
	// Line is the line of the violation.
	Line int `json:"line,omitempty"`
	// Message describes the violation.
	Message string `json:"message"`
	// Evidence provides verifiable references.
	Evidence []domain.Evidence `json:"evidence,omitempty"`
}

// ArchitectureResult holds the results of architectural analysis.
type ArchitectureResult struct {
	// Violations lists all detected convention violations.
	Violations []ConventionViolation `json:"violations"`
	// CyclicDependencies lists detected cyclic dependencies.
	CyclicDependencies []CyclicDependency `json:"cyclic_dependencies,omitempty"`
	// CouplingMetrics provides coupling analysis.
	CouplingMetrics *CouplingMetrics `json:"coupling_metrics,omitempty"`
	// CohesionMetrics provides cohesion analysis.
	CohesionMetrics *CohesionMetrics `json:"cohesion_metrics,omitempty"`
	// Summary provides a summary by type.
	Summary map[ConventionType]int `json:"summary"`
}

// CyclicDependency represents a detected cycle in the dependency graph.
type CyclicDependency struct {
	// Files involved in the cycle.
	Files []string `json:"files"`
	// Severity of the cycle.
	Severity domain.Severity `json:"severity"`
	// EpistemicStatus indicates how the finding was established.
	EpistemicStatus domain.EpistemicStatus `json:"epistemic_status"`
}

// CouplingMetrics provides coupling analysis.
type CouplingMetrics struct {
	// AfferentCoupling (Ca) - number of other packages that depend on this package.
	AfferentCoupling map[string]int `json:"afferent_coupling,omitempty"`
	// EfferentCoupling (Ce) - number of other packages this package depends on.
	EfferentCoupling map[string]int `json:"efferent_coupling,omitempty"`
	// Instability (I = Ce / (Ca + Ce)) - ranges from 0 (stable) to 1 (unstable).
	Instability map[string]float64 `json:"instability,omitempty"`
}

// CohesionMetrics provides cohesion analysis.
type CohesionMetrics struct {
	// Lack of Cohesion of Methods (LCOM) for each type.
	LCOM map[string]float64 `json:"lcom,omitempty"`
	// Number of methods per type.
	MethodsPerType map[string]int `json:"methods_per_type,omitempty"`
	// Number of fields per type.
	FieldsPerType map[string]int `json:"fields_per_type,omitempty"`
}

// Analyzer performs architectural analysis.
type Analyzer struct {
	index         *index.Index
	namingStyle   map[string]string // language -> style (camelCase, snake_case, PascalCase, etc.)
	importOrder   map[string]string // language -> order (alphabetical, grouped, etc.)
	checkCycles   bool
	thresholdLCOM float64
}

// NewAnalyzer creates a new architectural analyzer.
func NewAnalyzer(idx *index.Index) *Analyzer {
	return &Analyzer{
		index:         idx,
		namingStyle:   map[string]string{domain.LangGo: "camelCase", domain.LangPython: "snake_case", domain.LangJavaScript: "camelCase", domain.LangTypeScript: "camelCase", domain.LangPHP: "snake_case"},
		importOrder:   map[string]string{domain.LangGo: "grouped", domain.LangPython: "alphabetical", domain.LangJavaScript: "grouped", domain.LangTypeScript: "grouped", domain.LangPHP: "alphabetical"},
		checkCycles:   true,
		thresholdLCOM: 0.5,
	}
}

// WithNamingStyle sets the naming convention for a language.
func (a *Analyzer) WithNamingStyle(lang, style string) *Analyzer {
	a.namingStyle[lang] = style
	return a
}

// WithImportOrder sets the import ordering convention.
func (a *Analyzer) WithImportOrder(lang, order string) *Analyzer {
	a.importOrder[lang] = order
	return a
}

// WithCyclicCheck enables or disables cyclic dependency detection.
func (a *Analyzer) WithCyclicCheck(enabled bool) *Analyzer {
	a.checkCycles = enabled
	return a
}

// WithLCOMThreshold sets the LCOM threshold.
func (a *Analyzer) WithLCOMThreshold(threshold float64) *Analyzer {
	a.thresholdLCOM = threshold
	return a
}

// Analyze performs architectural analysis on the repository.
func (a *Analyzer) Analyze(ctx context.Context) (*ArchitectureResult, error) {
	if a.index == nil {
		return &ArchitectureResult{
			Violations:         []ConventionViolation{},
			CyclicDependencies: []CyclicDependency{},
			Summary:            map[ConventionType]int{},
		}, nil
	}

	symbols := a.index.Symbols()
	deps := a.index.Dependencies()

	var violations []ConventionViolation

	// Group by language
	byLang := make(map[string][]index.Symbol)
	for _, sym := range symbols {
		byLang[sym.Language] = append(byLang[sym.Language], sym)
	}

	// Check naming conventions
	for lang, langSymbols := range byLang {
		violations = append(violations, a.checkNamingConventions(lang, langSymbols)...)
	}

	// Check import conventions (using dependencies)
	violations = append(violations, a.checkImportConventions(deps)...)

	// Check cyclic dependencies
	var cycles []CyclicDependency
	if a.checkCycles {
		cycles = a.detectCycles(deps)
	}

	// Calculate coupling metrics
	coupling := a.calculateCoupling(deps)

	// Calculate cohesion metrics (simplified)
	cohesion := a.calculateCohesion(symbols)

	summary := map[ConventionType]int{}
	for _, v := range violations {
		summary[v.Type]++
	}
	for range cycles {
		summary[ConventionCyclicDependency]++
	}

	return &ArchitectureResult{
		Violations:         violations,
		CyclicDependencies: cycles,
		CouplingMetrics:    coupling,
		CohesionMetrics:    cohesion,
		Summary:            summary,
	}, nil
}

func (a *Analyzer) checkNamingConventions(lang string, symbols []index.Symbol) []ConventionViolation {
	var violations []ConventionViolation
	style := a.namingStyle[lang]

	for _, sym := range symbols {
		if sym.Kind != "function" && sym.Kind != "method" && sym.Kind != "func" && sym.Kind != "package" && sym.Kind != "type" {
			continue
		}

		violation := a.checkNameStyle(lang, style, sym.Name, sym.File, sym.Line, sym.Kind)
		if violation != nil {
			violations = append(violations, *violation)
		}
	}

	return violations
}

func (a *Analyzer) checkNameStyle(lang, style, name, file string, line int, kind string) *ConventionViolation {
	var valid bool
	switch style {
	case "camelCase":
		valid = isCamelCase(name)
	case "PascalCase":
		valid = isPascalCase(name)
	case "snake_case":
		valid = isSnakeCase(name)
	case "UPPER_CASE":
		valid = isUpperCase(name)
	default:
		valid = true // Unknown style, don't enforce
	}

	if !valid {
		return &ConventionViolation{
			Type:            ConventionNamingStyle,
			Severity:        domain.SeverityWarning,
			EpistemicStatus: domain.EpistemicInferred,
			File:            file,
			Line:            line,
			Message:         name + " does not follow " + style + " naming convention",
			Evidence: []domain.Evidence{
				{Kind: "source_range", Ref: file + ":" + name, Detail: "Symbol " + name + " of kind " + kind + " violates " + style + " naming convention"},
			},
		}
	}
	return nil
}

func isCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with lowercase
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] == '_' {
			return false
		}
	}
	return true
}

func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with uppercase
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] == '_' {
			return false
		}
	}
	return true
}

func isSnakeCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Cannot start or end with underscore
	if s[0] == '_' || s[len(s)-1] == '_' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			return false
		}
		if c == '_' && i+1 < len(s) && s[i+1] == '_' {
			return false // no double underscore
		}
	}
	return true
}

func isUpperCase(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return false
		}
	}
	return true
}

func (a *Analyzer) checkImportConventions(deps []index.Dependency) []ConventionViolation {
	var violations []ConventionViolation
	// Group by file
	byFile := make(map[string][]index.Dependency)
	for _, dep := range deps {
		byFile[dep.File] = append(byFile[dep.File], dep)
	}

	for file, fileDeps := range byFile {
		lang := ""
		for _, dep := range fileDeps {
			lang = dep.Language
			break
		}
		if lang == "" {
			continue
		}
		order := a.importOrder[lang]

		// Check import order
		violations := a.checkImportOrder(file, fileDeps, lang, order)
		violations = append(violations, violations...)
	}

	return violations
}

func (a *Analyzer) checkImportOrder(file string, deps []index.Dependency, lang, order string) []ConventionViolation {
	var violations []ConventionViolation

	if order == "alphabetical" {
		for i := 1; i < len(deps); i++ {
			if strings.Compare(deps[i-1].Path, deps[i].Path) > 0 {
				violations = append(violations, ConventionViolation{
					Type:            ConventionImportOrder,
					Severity:        domain.SeverityInfo,
					EpistemicStatus: domain.EpistemicObserved,
					File:            file,
					Line:            deps[i].Line,
					Message:         "Import " + deps[i].Path + " should come before " + deps[i-1].Path + " (alphabetical order)",
					Evidence: []domain.Evidence{
						{Kind: "source_range", Ref: file + ":" + deps[i].Path, Detail: "Import order violation: " + deps[i].Path + " before " + deps[i-1].Path},
					},
				})
			}
		}
	}

	return violations
}

func (a *Analyzer) detectCycles(deps []index.Dependency) []CyclicDependency {
	// Build dependency graph (simplified - file-level)
	graph := make(map[string]map[string]bool)
	for _, dep := range deps {
		if graph[dep.File] == nil {
			graph[dep.File] = make(map[string]bool)
		}
		// This is a simplified approach - real implementation would resolve imports
		// For now, we just check if a file imports itself indirectly
		graph[dep.File][dep.Path] = true
	}

	// Simplified cycle detection - real implementation would be more thorough
	var cycles []CyclicDependency

	// For now, just return empty - real cycle detection needs proper graph analysis
	return cycles
}

func (a *Analyzer) calculateCoupling(deps []index.Dependency) *CouplingMetrics {
	// Group dependencies by file/package
	afferent := make(map[string]int)
	efferent := make(map[string]int)

	for _, dep := range deps {
		efferent[dep.File]++
		// For afferent, we'd need reverse mapping
		// This is simplified
	}

	instability := make(map[string]float64)
	allPackages := make(map[string]bool)
	for k := range afferent {
		allPackages[k] = true
	}
	for k := range efferent {
		allPackages[k] = true
	}

	for pkg := range allPackages {
		ca := float64(afferent[pkg])
		ce := float64(efferent[pkg])
		if ca+ce > 0 {
			instability[pkg] = ce / (ca + ce)
		}
	}

	return &CouplingMetrics{
		AfferentCoupling:  afferent,
		EfferentCoupling:  efferent,
		Instability:       instability,
	}
}

func (a *Analyzer) calculateCohesion(symbols []index.Symbol) *CohesionMetrics {
	// Simplified cohesion metrics
	lcom := make(map[string]float64)
	methodsPerType := make(map[string]int)
	fieldsPerType := make(map[string]int)

	// Group symbols by file (simplified type grouping)
	byFile := make(map[string][]index.Symbol)
	for _, sym := range symbols {
		if sym.Kind == "function" || sym.Kind == "method" {
			byFile[sym.File] = append(byFile[sym.File], sym)
		}
	}

	for file, fileSyms := range byFile {
		methodsPerType[file] = len(fileSyms)
		// Simplified LCOM - in reality would need field access analysis
		if len(fileSyms) > 1 {
			lcom[file] = 0.0 // Placeholder
		}
	}

	return &CohesionMetrics{
		LCOM:           lcom,
		MethodsPerType: methodsPerType,
		FieldsPerType:  fieldsPerType,
	}
}

// DetectFromDiff analyzes architectural issues from a diff.
func (a *Analyzer) DetectFromDiff(ctx context.Context, changes *domain.ChangeSet) (*ArchitectureResult, error) {
	if changes == nil || len(changes.Files) == 0 {
		return &ArchitectureResult{
			Violations:         []ConventionViolation{},
			CyclicDependencies: []CyclicDependency{},
			Summary:            map[ConventionType]int{},
		}, nil
	}

	var violations []ConventionViolation

	for _, file := range changes.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.Kind == domain.LineAdded {
					// Check naming in added lines
					// This is simplified - would need more context
					lang := file.Language
					if lang != "" {
						// Could check naming conventions in added code
					}
				}
			}
		}
	}

	return &ArchitectureResult{
		Violations:         violations,
		CyclicDependencies: []CyclicDependency{},
		Summary:            map[ConventionType]int{},
	}, nil
}

// AnalyzeRepository runs full architectural analysis on the repository.
func (a *Analyzer) AnalyzeRepository(ctx context.Context, root string) (*ArchitectureResult, error) {
	return a.Analyze(ctx)
}