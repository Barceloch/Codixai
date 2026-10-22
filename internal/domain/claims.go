package domain

// The claim types of the catalog of Phase 2.
//
// The catalog is fixed by the decision 0007. A type that is not listed here
// is not part of the contract: an unknown type is an error, never an
// unresolved claim.
const (
	// ClaimTypeFilesUnchanged asserts that no path of the claim changed.
	ClaimTypeFilesUnchanged = "files_unchanged"
	// ClaimTypeScope asserts that every change stays inside the paths of
	// the claim.
	ClaimTypeScope = "scope"
	// ClaimTypeTestsAdded asserts that a test file of the paths was added
	// or modified.
	ClaimTypeTestsAdded = "tests_added"
	// ClaimTypeTestsModified asserts that a test file of the paths was
	// modified.
	ClaimTypeTestsModified = "tests_modified"
	// ClaimTypeDependencyUnchanged asserts that no manifest or lockfile
	// changed.
	ClaimTypeDependencyUnchanged = "dependency_unchanged"
	// ClaimTypeDependencyAdded asserts that the declared dependencies
	// appear in the manifests.
	ClaimTypeDependencyAdded = "dependency_added"
	// ClaimTypeAPIUnchanged asserts that the public API did not change. It
	// is always unresolved: the public API is not extracted yet.
	ClaimTypeAPIUnchanged = "api_unchanged"
	// ClaimTypeTestsPassed asserts that the tests passed, according to an
	// external report linked to the revision.
	ClaimTypeTestsPassed = "tests_passed"
	// ClaimTypeBehaviorPreserved asserts that the behavior of the change
	// is preserved. It is always unresolved: static analysis does not
	// decide the behavior of a change.
	ClaimTypeBehaviorPreserved = "behavior_preserved"
)

// Claim is an explicit assertion made about a change.
type Claim struct {
	// ID identifies the claim inside the claim file. It is unique.
	ID string `json:"id"`
	// Type is one of the types of the catalog.
	Type string `json:"type"`
	// Paths are the glob patterns of the repository relative paths the
	// claim is about.
	Paths []string `json:"paths,omitempty"`
	// Symbols are the names of the symbols the claim is about.
	Symbols []string `json:"symbols,omitempty"`
	// Report is the path of an external test report, relative to the root
	// of the repository.
	Report string `json:"report,omitempty"`
	// ReportRevision is the revision the external report belongs to.
	ReportRevision string `json:"report_revision,omitempty"`
	// Dependencies are the dependencies the claim declares, as they are
	// written in the manifests of the repository.
	Dependencies []string `json:"dependencies,omitempty"`
}

// Claim status values.
const (
	// ClaimVerified means the repository confirms the assertion.
	ClaimVerified = "verified"
	// ClaimContradicted means the repository shows that the assertion is
	// false.
	ClaimContradicted = "contradicted"
	// ClaimUnresolved means the evidence is not enough to decide.
	ClaimUnresolved = "unresolved"
)

// ClaimResult is the outcome of checking one claim.
type ClaimResult struct {
	// Claim is the assertion that was checked.
	Claim Claim `json:"claim"`
	// Status is one of verified, contradicted or unresolved.
	Status string `json:"status"`
	// Message says what was observed.
	Message string `json:"message"`
	// Evidence holds the references that support the status.
	Evidence []Evidence `json:"evidence,omitempty"`
	// Limitations holds the known limits of the check that produced the
	// status.
	Limitations []string `json:"limitations,omitempty"`
}

// ClaimFile is the versioned document that holds the claims of an agent.
type ClaimFile struct {
	// SchemaVersion is the version of the format of the claim file.
	SchemaVersion string `json:"schema_version"`
	// Base is the base revision of the comparison the claims are about. It
	// is optional.
	Base string `json:"base,omitempty"`
	// Claims holds the assertions of the file.
	Claims []Claim `json:"claims"`
}
