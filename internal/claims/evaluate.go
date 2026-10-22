package claims

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/glob"
	"github.com/barceloch/codixai/internal/languages/registry"
)

// Evaluator computes the results of the claims over a change set.
//
// The evaluator never executes the code of the analyzed repository: it reads
// the manifests and the external reports of the repository as text.
type Evaluator struct {
	// root is the root of the analyzed repository. When it is empty, the
	// root of the change set is used.
	root string
	// base is the base revision of the comparison. When it is empty, the
	// base of the change set is used.
	base string
}

// Option configures an evaluator.
type Option func(*Evaluator)

// WithRoot sets the root of the analyzed repository.
//
// The root is the directory the manifests and the reports are read from. It
// defaults to the root of the change set.
func WithRoot(root string) Option {
	return func(e *Evaluator) {
		e.root = root
	}
}

// WithBase sets the base revision of the comparison.
//
// The base is the revision a claim file is checked against. It defaults to
// the base of the change set.
func WithBase(base string) Option {
	return func(e *Evaluator) {
		e.base = base
	}
}

// NewEvaluator returns an evaluator configured with the given options.
func NewEvaluator(options ...Option) *Evaluator {
	evaluator := &Evaluator{}
	for _, option := range options {
		option(evaluator)
	}
	return evaluator
}

// Evaluate computes the result of every claim of the file over the changes.
//
// The root of the repository, the one the manifests and the reports are read
// from, is taken from the change set unless the evaluator overrides it. The
// head is the revision a report is linked to; it defaults to the head of
// the change set.
//
// The results are sorted by the identifier of the claim.
func (e *Evaluator) Evaluate(ctx context.Context, changes *domain.ChangeSet, file *File, head string) ([]domain.ClaimResult, error) {
	if changes == nil {
		return nil, errors.New("the evaluation needs the changes of the repository")
	}
	if file == nil {
		return nil, errors.New("the evaluation needs a claim file")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := e.root
	if root == "" {
		root = changes.Root
	}
	base := e.base
	if base == "" {
		base = changes.Base
	}
	if err := ValidateBase(file, base); err != nil {
		return nil, err
	}
	if head == "" {
		head = changes.Head
	}
	results := make([]domain.ClaimResult, 0, len(file.Claims))
	for _, claim := range file.Claims {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		results = append(results, e.evaluate(root, changes, claim, head))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Claim.ID < results[j].Claim.ID
	})
	return results, nil
}

// evaluate computes the result of one claim.
func (e *Evaluator) evaluate(root string, changes *domain.ChangeSet, claim domain.Claim, head string) domain.ClaimResult {
	switch claim.Type {
	case domain.ClaimTypeFilesUnchanged:
		return evaluateFilesUnchanged(changes, claim)
	case domain.ClaimTypeScope:
		return evaluateScope(changes, claim)
	case domain.ClaimTypeTestsAdded:
		return evaluateTests(changes, claim, false)
	case domain.ClaimTypeTestsModified:
		return evaluateTests(changes, claim, true)
	case domain.ClaimTypeDependencyUnchanged:
		return evaluateDependencyUnchanged(root, changes, claim)
	case domain.ClaimTypeDependencyAdded:
		return evaluateDependencyAdded(root, changes, claim)
	case domain.ClaimTypeAPIUnchanged:
		return newResult(claim, domain.ClaimUnresolved,
			"the public API of the repository is not extracted yet, so the claim cannot be decided by the analysis", nil)
	case domain.ClaimTypeTestsPassed:
		return evaluateTestsPassed(root, claim, head)
	case domain.ClaimTypeBehaviorPreserved:
		return newResult(claim, domain.ClaimUnresolved,
			"static analysis does not decide the behavior of a change", nil)
	default:
		return newResult(claim, domain.ClaimUnresolved,
			fmt.Sprintf("the claim type %q is unknown, so the claim cannot be decided", claim.Type), nil)
	}
}

// evaluateFilesUnchanged checks that no path of the claim changed.
func evaluateFilesUnchanged(changes *domain.ChangeSet, claim domain.Claim) domain.ClaimResult {
	if len(changes.Files) == 0 {
		return newResult(claim, domain.ClaimUnresolved, "the diff is empty, so there is nothing to compare", nil)
	}
	matched := matchingFiles(claim, changes)
	if len(matched) == 0 {
		return newResult(claim, domain.ClaimVerified, "no changed file matches the paths of the claim", nil)
	}
	message := fmt.Sprintf("%d changed file(s) match the paths of the claim: %s", len(matched), strings.Join(filePaths(matched), ", "))
	return newResult(claim, domain.ClaimContradicted, message, fileEvidence(matched))
}

// evaluateScope checks that every change stays inside the paths of the
// claim.
func evaluateScope(changes *domain.ChangeSet, claim domain.Claim) domain.ClaimResult {
	if len(changes.Files) == 0 {
		return newResult(claim, domain.ClaimUnresolved, "the diff is empty, so there is nothing to compare", nil)
	}
	outside := outsideFiles(claim, changes)
	if len(outside) == 0 {
		return newResult(claim, domain.ClaimVerified, "every changed file matches the paths of the claim", nil)
	}
	message := fmt.Sprintf("%d changed file(s) are outside the paths of the claim: %s", len(outside), strings.Join(filePaths(outside), ", "))
	return newResult(claim, domain.ClaimContradicted, message, fileEvidence(outside))
}

// evaluateTests checks the test files of the paths of the claim.
//
// When modifiedOnly is true, only a modified test file satisfies the claim:
// a newly added test file does not.
func evaluateTests(changes *domain.ChangeSet, claim domain.Claim, modifiedOnly bool) domain.ClaimResult {
	candidates := matchingFiles(claim, changes)
	if len(candidates) == 0 {
		return newResult(claim, domain.ClaimUnresolved,
			"no changed file matches the paths of the claim, so no test file can be identified", nil)
	}
	testFiles := testFilesOf(candidates)
	if len(testFiles) == 0 {
		return newResult(claim, domain.ClaimUnresolved,
			fmt.Sprintf("the paths of the claim hold no test file of the language of the changed files (%s)", strings.Join(languagesOf(candidates), ", ")),
			fileEvidence(candidates))
	}
	satisfied := make([]domain.FileChange, 0, len(testFiles))
	for _, file := range testFiles {
		if modifiedOnly && file.Status != domain.FileModified {
			continue
		}
		if !modifiedOnly && file.Status != domain.FileAdded && file.Status != domain.FileModified {
			continue
		}
		satisfied = append(satisfied, file)
	}
	if len(satisfied) > 0 {
		message := fmt.Sprintf("the test file(s) %s were added or modified", strings.Join(filePaths(satisfied), ", "))
		if modifiedOnly {
			message = fmt.Sprintf("the test file(s) %s were modified", strings.Join(filePaths(satisfied), ", "))
		}
		return newResult(claim, domain.ClaimVerified, message, fileEvidence(satisfied))
	}
	message := fmt.Sprintf("no test file that matches the paths of the claim was added or modified: %s", strings.Join(filePaths(testFiles), ", "))
	if modifiedOnly {
		message = fmt.Sprintf("no test file that matches the paths of the claim was modified: %s", strings.Join(filePaths(testFiles), ", "))
	}
	return newResult(claim, domain.ClaimContradicted, message, fileEvidence(testFiles))
}

// evaluateDependencyUnchanged checks that no manifest or lockfile changed.
func evaluateDependencyUnchanged(root string, changes *domain.ChangeSet, claim domain.Claim) domain.ClaimResult {
	if !rootHasManifest(root) {
		return newResult(claim, domain.ClaimUnresolved,
			"the repository has no manifest of any language, so the dependencies cannot be compared", nil)
	}
	changed := manifestFiles(changes)
	if len(changed) == 0 {
		return newResult(claim, domain.ClaimVerified, "no manifest or lockfile changed", nil)
	}
	message := fmt.Sprintf("%d manifest(s) or lockfile(s) changed: %s", len(changed), strings.Join(filePaths(changed), ", "))
	return newResult(claim, domain.ClaimContradicted, message, fileEvidence(changed))
}

// evaluateDependencyAdded checks that the declared dependencies appear in
// the manifests of the repository.
func evaluateDependencyAdded(root string, changes *domain.ChangeSet, claim domain.Claim) domain.ClaimResult {
	if len(claim.Dependencies) == 0 {
		return newResult(claim, domain.ClaimUnresolved, "the claim declares no dependency", nil)
	}
	if !rootHasManifest(root) {
		return newResult(claim, domain.ClaimUnresolved,
			"the repository has no manifest of any language, so the declared dependencies cannot be found", nil)
	}
	declarations := readManifests(root)
	missing := make([]string, 0, len(claim.Dependencies))
	for _, dependency := range claim.Dependencies {
		if !declarations.declares(dependency) {
			missing = append(missing, dependency)
		}
	}
	changed := manifestFiles(changes)
	if len(changed) == 0 {
		return newResult(claim, domain.ClaimContradicted,
			"no manifest or lockfile changed, so no dependency was added", nil)
	}
	if len(missing) > 0 {
		message := fmt.Sprintf("the declared dependencies are absent from the manifests of the repository: %s", strings.Join(missing, ", "))
		return newResult(claim, domain.ClaimContradicted, message, fileEvidence(changed))
	}
	message := fmt.Sprintf("the declared dependencies appear in the manifests of the repository: %s", strings.Join(claim.Dependencies, ", "))
	return newResult(claim, domain.ClaimVerified, message, fileEvidence(changed))
}

// evaluateTestsPassed checks the result of an external test report.
//
// The report is never presented as observed by Codixai: it was produced
// outside, and the evidence of the result says so.
func evaluateTestsPassed(root string, claim domain.Claim, head string) domain.ClaimResult {
	if strings.TrimSpace(claim.Report) == "" {
		return newResult(claim, domain.ClaimUnresolved,
			"the claim declares no report, so the result of the tests is unknown", nil)
	}
	if root == "" {
		return newResult(claim, domain.ClaimUnresolved,
			"the root of the repository is unknown, so the report cannot be read", nil)
	}
	path := filepath.Join(root, filepath.FromSlash(claim.Report))
	data, err := os.ReadFile(path)
	if err != nil {
		message := fmt.Sprintf("the report %q cannot be read: %v", claim.Report, err)
		return newResult(claim, domain.ClaimUnresolved, message, nil)
	}
	parsed, err := parseReport(claim.Report, data)
	if err != nil {
		return newResult(claim, domain.ClaimUnresolved, err.Error(), nil)
	}
	linked, reason := linkReport(claim, parsed, head)
	if !linked {
		return newResult(claim, domain.ClaimUnresolved, reason, reportEvidence(claim.Report))
	}
	evidence := reportEvidence(claim.Report)
	if parsed.success() {
		message := fmt.Sprintf("the linked report %q reports %d test(s), no failure and no error", claim.Report, parsed.Tests)
		return newResult(claim, domain.ClaimVerified, message, evidence)
	}
	message := fmt.Sprintf("the linked report %q reports %d test(s), %d failure(s) and %d error(s)", claim.Report, parsed.Tests, parsed.Failures, parsed.Errors)
	return newResult(claim, domain.ClaimContradicted, message, evidence)
}

// matchingFiles returns the changed files whose path matches the paths of
// the claim.
func matchingFiles(claim domain.Claim, changes *domain.ChangeSet) []domain.FileChange {
	matched := make([]domain.FileChange, 0, len(changes.Files))
	for _, file := range changes.Files {
		if matchesPaths(claim, file) {
			matched = append(matched, file)
		}
	}
	return matched
}

// outsideFiles returns the changed files whose path does not match the paths
// of the claim.
func outsideFiles(claim domain.Claim, changes *domain.ChangeSet) []domain.FileChange {
	outside := make([]domain.FileChange, 0, len(changes.Files))
	for _, file := range changes.Files {
		if !matchesPaths(claim, file) {
			outside = append(outside, file)
		}
	}
	return outside
}

// matchesPaths reports whether a changed file matches the paths of a claim.
// The old path of a renamed file is considered too.
func matchesPaths(claim domain.Claim, file domain.FileChange) bool {
	if len(claim.Paths) == 0 {
		return false
	}
	for _, pattern := range claim.Paths {
		if matched, err := glob.Match(pattern, file.Path); err == nil && matched {
			return true
		}
		if file.OldPath != "" {
			if matched, err := glob.Match(pattern, file.OldPath); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// testFilesOf returns the files that hold tests of their language.
func testFilesOf(files []domain.FileChange) []domain.FileChange {
	tests := make([]domain.FileChange, 0, len(files))
	for _, file := range files {
		profile, ok := registry.ProfileForPath(file.Path)
		if !ok {
			continue
		}
		if profile.IsTestPath(file.Path) {
			tests = append(tests, file)
		}
	}
	return tests
}

// languagesOf returns the languages of the changed files, sorted and without
// repetitions.
func languagesOf(files []domain.FileChange) []string {
	seen := map[string]bool{}
	languages := make([]string, 0, len(files))
	for _, file := range files {
		language, ok := registry.Detect(file.Path)
		if !ok || seen[language] {
			continue
		}
		seen[language] = true
		languages = append(languages, language)
	}
	sort.Strings(languages)
	if len(languages) == 0 {
		return []string{"unknown"}
	}
	return languages
}

// manifestFiles returns the changed files that are a manifest or a lockfile.
func manifestFiles(changes *domain.ChangeSet) []domain.FileChange {
	manifests := make([]domain.FileChange, 0, len(changes.Files))
	for _, file := range changes.Files {
		if isManifestPath(file.Path) {
			manifests = append(manifests, file)
		}
	}
	return manifests
}

// filePaths returns the paths of the changed files.
func filePaths(files []domain.FileChange) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

// fileEvidence returns the evidence of a set of changed files.
func fileEvidence(files []domain.FileChange) []domain.Evidence {
	evidence := make([]domain.Evidence, 0, len(files))
	for _, file := range files {
		evidence = append(evidence, domain.Evidence{
			Kind:   "diff_file",
			Ref:    file.Path,
			Detail: "the file appears in the compared change set",
		})
	}
	return evidence
}

// reportEvidence returns the evidence of an external report. The detail
// always states that the provenance is external.
func reportEvidence(path string) []domain.Evidence {
	return []domain.Evidence{{
		Kind:   "external_report",
		Ref:    path,
		Detail: "the report was produced outside Codixai: Codixai did not run the tests of the repository",
	}}
}

// newResult builds the result of a claim.
func newResult(claim domain.Claim, status, message string, evidence []domain.Evidence) domain.ClaimResult {
	return domain.ClaimResult{
		Claim:       claim,
		Status:      status,
		Message:     message,
		Evidence:    evidence,
		Limitations: limitationsOf(claim.Type),
	}
}

// Failed reports whether the results of a verification reach the failure
// threshold.
//
// A contradicted claim always fails. An unresolved claim only fails when the
// policy of the repository says so, because an unresolved claim is usually a
// limitation of the analysis and not a failure of the change.
func Failed(results []domain.ClaimResult, failOnUnresolved bool) bool {
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
