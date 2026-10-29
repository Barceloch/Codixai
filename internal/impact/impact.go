// Package impact implements impact analysis for changes.
// It identifies affected findings, likely failing tests, and recommends tests to run.
package impact

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
)

// ImpactResult holds the results of impact analysis.
type ImpactResult struct {
	// AffectedFindings lists findings affected by the change.
	AffectedFindings []ImpactedFinding `json:"affected_findings"`
	// LikelyFailingTests lists tests that likely fail due to the change.
	LikelyFailingTests []string `json:"likely_failing_tests"`
	// RecommendedTests lists tests recommended to run.
	RecommendedTests []string `json:"recommended_tests"`
	// Impacted marks findings that are directly impacted.
	Impacted []string `json:"impacted"`
}

// ImpactedFinding represents a finding with impact information.
type ImpactedFinding struct {
	Finding    domain.Finding `json:"finding"`
	ImpactType string         `json:"impact_type"` // "direct", "indirect", "unknown"
	Reason     string         `json:"reason"`
}

// Analyzer analyzes the impact of a change set.
type Analyzer struct {
	index *index.Index
}

// NewAnalyzer creates a new impact analyzer.
func NewAnalyzer(idx *index.Index) *Analyzer {
	return &Analyzer{index: idx}
}

// AnalyzeImpact analyzes the impact of a change set.
func (a *Analyzer) AnalyzeImpact(ctx context.Context, changes *domain.ChangeSet) (*ImpactResult, error) {
	if changes == nil || len(changes.Files) == 0 {
		return &ImpactResult{}, nil
	}

	// Build index if not already built
	// For now we just analyze the changes directly

	affectedFindings := []ImpactedFinding{}
	likelyFailing := []string{}
	recommended := []string{}
	impacted := []string{}

	// Analyze each changed file
	for _, file := range changes.Files {
		// Determine impact type based on file status
		impactType := a.determineImpactType(file)

		// Create an impacted finding entry for each changed file
		// In a real implementation, we'd correlate with actual findings from audit
		affectedFindings = append(affectedFindings, ImpactedFinding{
			Finding: domain.Finding{
				Message: fmt.Sprintf("File %s changed (%s)", file.Path, file.Status),
				Location: domain.Location{
					Path: file.Path,
				},
				Severity:        domain.SeverityInfo,
				EpistemicStatus: domain.EpistemicObserved,
			},
			ImpactType: impactType,
			Reason:     fmt.Sprintf("File %s was %s", file.Path, file.Status),
		})

		// Find tests related to this file
		tests := a.findRelatedTests(file.Path)
		likelyFailing = append(likelyFailing, tests...)
		recommended = append(recommended, tests...)

		// Mark as impacted
		impacted = append(impacted, file.Path)
	}

	// Deduplicate
	likelyFailing = dedup(likelyFailing)
	recommended = dedup(recommended)
	impacted = dedup(impacted)

	return &ImpactResult{
		AffectedFindings:   affectedFindings,
		LikelyFailingTests: likelyFailing,
		RecommendedTests:   recommended,
		Impacted:           impacted,
	}, nil
}

func (a *Analyzer) determineImpactType(file domain.FileChange) string {
	switch file.Status {
	case domain.FileAdded:
		return "direct"
	case domain.FileModified:
		return "direct"
	case domain.FileDeleted:
		return "direct"
	case domain.FileRenamed:
		return "indirect"
	default:
		return "unknown"
	}
}

func (a *Analyzer) findRelatedTests(filePath string) []string {
	var tests []string
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), ext)
	dir := filepath.Dir(filePath)

	// Common test file patterns
	patterns := []string{
		filePath + "_test" + ext,
		filePath + ".test" + ext,
		"test_" + filePath,
		filepath.Join(dir, base+"_test"+ext),
		filepath.Join(dir, "test_"+base+ext),
		filepath.Join(dir, base+"_test.go"),
	}

	for _, p := range patterns {
		if _, err := filepath.Abs(p); err == nil {
			tests = append(tests, p)
		}
	}

	// Also check for test files in the same directory
	// This is a heuristic; in reality we'd search the index
	return tests
}

func dedup(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// AnalyzeImpactFromDiff runs impact analysis on a diff directly.
// This is a convenience function for CLI usage.
func AnalyzeImpactFromDiff(ctx context.Context, root string, changes *domain.ChangeSet) (*ImpactResult, error) {
	idx := index.New()
	if err := idx.Build(ctx, root); err != nil {
		return nil, fmt.Errorf("failed to build index: %w", err)
	}

	analyzer := NewAnalyzer(idx)
	return analyzer.AnalyzeImpact(ctx, changes)
}