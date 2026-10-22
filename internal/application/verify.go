package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/policies"
)

// ClaimLoader loads a claim file and checks that the claims are about the
// base of the comparison.
//
// The port is implemented by the claims package. The application layer does
// not import it: the claims package is a provider, and the direction of the
// dependencies is the one of the architecture of the project.
type ClaimLoader interface {
	// Load reads and validates the claim file of the given path.
	Load(path string) (*domain.ClaimFile, error)
	// ValidateBase checks that the claims of the file are about the given
	// base revision.
	ValidateBase(file *domain.ClaimFile, base string) error
}

// ClaimEvaluator computes the results of the claims over a change set.
//
// The port is implemented by the claims package.
type ClaimEvaluator interface {
	// Evaluate computes the result of every claim of the file over the
	// changes. The head is the revision of the head of the compared
	// range.
	Evaluate(ctx context.Context, changes *domain.ChangeSet, file *domain.ClaimFile, head string) ([]domain.ClaimResult, error)
}

// VerificationSummary counts the results of a verification by status.
//
// The summary of a verification has its own type because the summary of the
// domain counts findings by severity, rule, language and epistemic status,
// and a claim has none of them: a claim is verified, contradicted or
// unresolved. The summary of the audit keeps its shape.
type VerificationSummary struct {
	// Total is the number of claims that were checked.
	Total int `json:"total"`
	// ByStatus counts the claims per status. Every status of the catalog
	// is always present, even when its count is zero.
	ByStatus map[string]int `json:"by_status,omitempty"`
}

// VerificationDocument is the versioned document of a verification.
//
// It holds the fields of the contract of the phase 2: the schema version,
// the version of the tool, the instant of the generation, the base and the
// head of the compared range, the result of every claim, the summary of the
// results and the limitations of the verification.
type VerificationDocument struct {
	// SchemaVersion is the version of the output contract.
	SchemaVersion string `json:"schema_version"`
	// ToolVersion is the version of the tool that produced the document.
	ToolVersion string `json:"tool_version"`
	// GeneratedAt is the instant of the generation of the document.
	GeneratedAt string `json:"generated_at"`
	// Base is the base revision of the comparison, as the user wrote it.
	Base string `json:"base"`
	// Head is the revision of the head of the compared range.
	Head string `json:"head"`
	// Results holds the result of every claim, sorted by the identifier
	// of the claim.
	Results []domain.ClaimResult `json:"results"`
	// Summary counts the results by status.
	Summary VerificationSummary `json:"summary"`
	// Limitations holds the known limits of the verification.
	Limitations []string `json:"limitations,omitempty"`
}

// VerifyRequest describes the verification to run.
//
// The use case never touches the command line: it receives the already
// resolved options and the providers that collect the changes and that
// check the claims.
type VerifyRequest struct {
	// Root is the absolute path of the analyzed repository.
	Root string
	// Base is the base revision of the comparison, as the user wrote it.
	// The claim file is checked against this ref and not against the
	// commit it resolves to, because the claim file names the base the
	// author of the claims wrote.
	Base string
	// ClaimsPath is the path of the claim file to verify.
	ClaimsPath string
	// Source collects the changes of the repository.
	Source ChangeSource
	// Loader loads the claim file and checks its base.
	Loader ClaimLoader
	// Evaluator computes the result of every claim.
	Evaluator ClaimEvaluator
	// Config holds the policy of the repository. A nil value loads the
	// configuration of the repository, and the defaults when the
	// repository has none.
	Config *policies.Config
	// ToolVersion is the version of the tool reported in the document.
	ToolVersion string
	// Now returns the instant of the generation of the document. It
	// defaults to the clock of the process.
	Now func() time.Time
}

// VerifyResult is the outcome of a verification.
type VerifyResult struct {
	// Document is the document produced by the verification.
	Document *VerificationDocument
	// Failed states whether a claim reaches the failure threshold.
	Failed bool
	// Contradicted counts the contradicted claims.
	Contradicted int
	// Unresolved counts the unresolved claims.
	Unresolved int
	// FailOnUnresolved tells whether an unresolved claim makes the
	// verification fail.
	FailOnUnresolved bool
}

// verificationLimitations are the known limits of a verification.
//
// They are part of the document: a consumer that reads only the states of
// the claims has to know what a verification decides and what it does not.
var verificationLimitations = []string{
	"The verification is a static analysis of the compared range: Codixai did not run the code of the repository.",
	"The test reports are external evidence: they were produced outside Codixai and Codixai cannot vouch for them.",
	"An unresolved claim is not a failure: it means that the available evidence was not enough to decide.",
	"The claims are checked against the whole compared range: the paths that the configuration excludes from the analysis are not excluded from the claims.",
}

// RunVerify verifies the claims of a file against the state of a repository.
//
// The use case collects the changes between the base and the head of the
// repository, loads the claim file, checks that the claims are about that
// range and evaluates every claim against the changes.
//
// The verification never modifies the analyzed repository: the manifests and
// the reports of the repository are read as data.
func RunVerify(ctx context.Context, request VerifyRequest) (*VerifyResult, error) {
	if request.Source == nil {
		return nil, fmt.Errorf("the verification needs a change source")
	}
	if request.Loader == nil {
		return nil, fmt.Errorf("the verification needs a claim loader")
	}
	if request.Evaluator == nil {
		return nil, fmt.Errorf("the verification needs a claim evaluator")
	}
	if strings.TrimSpace(request.Root) == "" {
		return nil, fmt.Errorf("the verification needs the root of the repository")
	}
	base := strings.TrimSpace(request.Base)
	if base == "" {
		return nil, fmt.Errorf("the verification needs a base revision")
	}
	claimsPath := strings.TrimSpace(request.ClaimsPath)
	if claimsPath == "" {
		return nil, fmt.Errorf("the verification needs the path of the claim file")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	config := request.Config
	if config == nil {
		loaded, err := policies.Load(request.Root)
		if err != nil {
			return nil, err
		}
		config = loaded
	}

	file, err := request.Loader.Load(claimsPath)
	if err != nil {
		return nil, err
	}
	// The base of the claim file is compared with the ref as the user
	// wrote it: a claim file that declares another revision is about
	// another state of the repository.
	if err := request.Loader.ValidateBase(file, base); err != nil {
		return nil, err
	}

	changes, err := request.Source.Changes(ctx, domain.OriginBase, base)
	if err != nil {
		return nil, err
	}
	// The change source does not know the root of the repository: the
	// manifests and the reports of the analyzed project are read from it.
	changes.Root = request.Root

	head, err := request.Source.Head(ctx)
	if err != nil {
		return nil, err
	}

	results, err := request.Evaluator.Evaluate(ctx, changes, file, head)
	if err != nil {
		return nil, err
	}

	summary := summarizeClaims(results)
	now := request.Now
	if now == nil {
		now = time.Now
	}

	result := &VerifyResult{
		Document: &VerificationDocument{
			SchemaVersion: domain.SchemaVersion,
			ToolVersion:   request.ToolVersion,
			GeneratedAt:   now().UTC().Format(time.RFC3339),
			Base:          base,
			Head:          head,
			Results:       results,
			Summary:       summary,
			Limitations:   append([]string(nil), verificationLimitations...),
		},
		Failed:           claimsFailed(results, config.Verify.FailOnUnresolved),
		Contradicted:     summary.ByStatus[domain.ClaimContradicted],
		Unresolved:       summary.ByStatus[domain.ClaimUnresolved],
		FailOnUnresolved: config.Verify.FailOnUnresolved,
	}
	return result, nil
}

// claimsFailed reports whether the results of a verification reach the
// failure threshold.
//
// A contradicted claim always fails. An unresolved claim only fails when
// the policy of the repository says so, because an unresolved claim is
// usually a limitation of the analysis and not a failure of the change.
func claimsFailed(results []domain.ClaimResult, failOnUnresolved bool) bool {
	for _, result := range results {
		switch result.Status {
		case domain.ClaimContradicted:
			return true
		case domain.ClaimUnresolved:
			if failOnUnresolved {
				return true
			}
		}
	}
	return false
}

// summarizeClaims counts the results of a verification by status.
//
// Every status of the catalog is always counted, so that the summary of a
// verification holds the same keys whatever the claims were.
func summarizeClaims(results []domain.ClaimResult) VerificationSummary {
	summary := VerificationSummary{
		Total: len(results),
		ByStatus: map[string]int{
			domain.ClaimVerified:     0,
			domain.ClaimContradicted: 0,
			domain.ClaimUnresolved:   0,
		},
	}
	for _, result := range results {
		summary.ByStatus[result.Status]++
	}
	return summary
}
