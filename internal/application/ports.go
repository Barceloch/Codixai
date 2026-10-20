// Package application holds the use cases of Codixai.
//
// It depends only on the domain models and on the ports declared here.
// Concrete providers (git, storage, language adapters) implement those ports.
package application

import (
	"context"

	"github.com/Barceloch/Codixai/internal/domain"
)

// DiffAnalyzer is the port implemented by the analysis engine.
type DiffAnalyzer interface {
	// Analyze inspects a change set and returns the findings it produced.
	Analyze(ctx context.Context, changes *domain.ChangeSet) (*domain.AnalysisRun, error)
	// Rules returns the rules exposed by the engine with their support matrix.
	Rules() []domain.Rule
}

// ChangeSource is the port implemented by the git integration.
type ChangeSource interface {
	// Changes collects the changes of the given origin.
	Changes(ctx context.Context, origin domain.ChangeOrigin, base string) (*domain.ChangeSet, error)
	// Head returns the current HEAD revision of the repository.
	Head(ctx context.Context) (string, error)
}
