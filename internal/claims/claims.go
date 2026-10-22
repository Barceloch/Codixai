// Package claims loads a claim file and evaluates the claims of its catalog
// against a change set.
//
// The package implements the decision 0007: the format of the claim file, the
// catalog of the claim types and the linking of an external test report to
// the revision under analysis.
//
// The scope and the limits of the package:
//
//   - It never executes the code of the analyzed repository. It reads the
//     manifests and the reports of the repository as text.
//   - A report is always external evidence: Codixai did not produce it and
//     the evidence of a result says so.
//   - A claim that cannot be evaluated is never verified. The absence of
//     evidence is always unresolved.
//   - The catalog is closed. An unknown type is an error, not an unresolved
//     claim, because a typo in a claim file must not read as a pass.
package claims

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// SchemaVersion is the version of the format of the claim file.
//
// The format has its own lifecycle: it does not follow the version of the
// tool.
const SchemaVersion = "0.1"

// File is the versioned document that holds the claims of an agent.
//
// It is an alias of the model of the domain package, so that the CLI, the
// document of a verification and the evaluator share the same type.
type File = domain.ClaimFile

// Type describes a claim type of the catalog.
type Type struct {
	// ID is the identifier of the type, as it is written in a claim file.
	ID string
	// Description says what a claim of the type asserts.
	Description string
	// Limitations holds the known limits of the evaluation of the type.
	Limitations []string
}

// catalog holds the types of the catalog, sorted by identifier.
var catalog = []Type{
	{
		ID:          domain.ClaimTypeAPIUnchanged,
		Description: "the public API of the repository did not change",
		Limitations: []string{
			"The public API is not extracted yet, so the claim cannot be decided by the analysis.",
		},
	},
	{
		ID:          domain.ClaimTypeBehaviorPreserved,
		Description: "the behavior of the repository is preserved by the change",
		Limitations: []string{
			"Static analysis does not decide the behavior of a change.",
		},
	},
	{
		ID:          domain.ClaimTypeDependencyAdded,
		Description: "the declared dependencies appear in the manifests of the repository",
		Limitations: []string{
			"Only the manifests of the root of the repository are read, in their current state.",
			"A dependency that is declared in a manifest of a subdirectory is not found.",
		},
	},
	{
		ID:          domain.ClaimTypeDependencyUnchanged,
		Description: "no manifest or lockfile of the repository changed",
		Limitations: []string{
			"Only the manifests and the lockfiles of the root of the repository are considered.",
		},
	},
	{
		ID:          domain.ClaimTypeFilesUnchanged,
		Description: "no path of the claim changed",
		Limitations: []string{
			"The claim only holds for the compared range of revisions.",
		},
	},
	{
		ID:          domain.ClaimTypeScope,
		Description: "every change of the range stays inside the paths of the claim",
		Limitations: []string{
			"The claim only holds for the compared range of revisions.",
		},
	},
	{
		ID:          domain.ClaimTypeTestsAdded,
		Description: "a test file of the paths of the claim was added or modified",
		Limitations: []string{
			"The test files are recognized by the patterns of the language profile, so a test file with an unusual name is not recognized.",
		},
	},
	{
		ID:          domain.ClaimTypeTestsModified,
		Description: "a test file of the paths of the claim was modified",
		Limitations: []string{
			"The test files are recognized by the patterns of the language profile, so a test file with an unusual name is not recognized.",
			"Only a modified test file satisfies the claim: a newly added test file does not.",
		},
	},
	{
		ID:          domain.ClaimTypeTestsPassed,
		Description: "the tests of the revision passed, according to an external report",
		Limitations: []string{
			"The report was produced outside Codixai: Codixai did not run the tests and cannot vouch for the report.",
			"A report that cannot be linked to the revision under analysis leaves the claim unresolved.",
		},
	},
}

// Catalog returns the types of the catalog, sorted by identifier.
func Catalog() []Type {
	types := make([]Type, 0, len(catalog))
	for _, entry := range catalog {
		types = append(types, Type{
			ID:          entry.ID,
			Description: entry.Description,
			Limitations: append([]string(nil), entry.Limitations...),
		})
	}
	return types
}

// Types returns the identifiers of the types of the catalog, sorted.
func Types() []string {
	types := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		types = append(types, entry.ID)
	}
	sort.Strings(types)
	return types
}

// isCatalogType reports whether a type belongs to the catalog.
func isCatalogType(name string) bool {
	for _, entry := range catalog {
		if entry.ID == name {
			return true
		}
	}
	return false
}

// limitationsOf returns the known limits of the evaluation of a type.
func limitationsOf(name string) []string {
	for _, entry := range catalog {
		if entry.ID == name {
			return append([]string(nil), entry.Limitations...)
		}
	}
	return nil
}

// The fields of the claim file and of a claim. An unknown field is rejected:
// the format is a contract and a typo must not read as a pass.
var (
	// fileFields are the fields of the top level object of a claim file.
	fileFields = []string{"base", "claims", "schema_version"}
	// claimFields are the fields of a claim of a claim file.
	claimFields = []string{"dependencies", "id", "paths", "report", "report_revision", "symbols", "type"}
)

// Load reads the claim file of the given path and validates it.
//
// The path is only used to build the error messages: every error names the
// file and the line of the problem.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the claim file %s: %w", path, err)
	}
	return Parse(data, path)
}

// Loader reads the claim files.
//
// It is the provider of the port that the application layer declares, so
// the application layer does not import this package.
type Loader struct{}

// Load reads the claim file of the given path and validates it.
func (Loader) Load(path string) (*File, error) {
	return Load(path)
}

// ValidateBase checks that the claims of the file are about the given base
// revision.
func (Loader) ValidateBase(file *File, base string) error {
	return ValidateBase(file, base)
}

// Parse validates the claim file held in data.
//
// The path is only used to build the error messages: every error names the
// file and the line of the problem.
func Parse(data []byte, path string) (*File, error) {
	positions, err := scanJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	file, err := buildFile(data, positions)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return file, nil
}

// buildFile validates the document of a claim file.
func buildFile(data []byte, positions *jsonPositions) (*File, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, &jsonError{line: lineOf(data, offsetOf(err)), message: "the claim file is not valid JSON"}
	}
	for _, key := range sortedFields(positions.fields) {
		if !contains(fileFields, key) {
			return nil, &jsonError{
				line:    positions.fields[key],
				message: fmt.Sprintf("unknown field %q, the supported fields are %s", key, strings.Join(fileFields, ", ")),
			}
		}
	}

	file := &File{}

	raw, ok := document["schema_version"]
	if !ok {
		return nil, &jsonError{
			line:    1,
			message: fmt.Sprintf("the field \"schema_version\" is required: the format of the claim file is versioned, the supported version is %q", SchemaVersion),
		}
	}
	version, err := textValue(raw)
	if err != nil {
		return nil, &jsonError{line: positions.fields["schema_version"], message: "the field \"schema_version\" must be a text value"}
	}
	if version != SchemaVersion {
		return nil, &jsonError{
			line:    positions.fields["schema_version"],
			message: fmt.Sprintf("unsupported schema_version %q, the supported version is %q", version, SchemaVersion),
		}
	}
	file.SchemaVersion = version

	if raw, ok := document["base"]; ok {
		base, err := textValue(raw)
		if err != nil {
			return nil, &jsonError{line: positions.fields["base"], message: "the field \"base\" must be a text value"}
		}
		if strings.TrimSpace(base) == "" {
			return nil, &jsonError{line: positions.fields["base"], message: "the field \"base\" must not be empty"}
		}
		file.Base = base
	}

	raw, ok = document["claims"]
	if !ok {
		return nil, &jsonError{line: 1, message: "the field \"claims\" is required"}
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		return nil, &jsonError{line: positions.fields["claims"], message: "the field \"claims\" must be a list of claims"}
	}
	if len(elements) == 0 {
		return nil, &jsonError{line: positions.fields["claims"], message: "the field \"claims\" must not be empty"}
	}

	seen := map[string]int{}
	for index, element := range elements {
		position := jsonClaimPosition{line: 1, keys: map[string]int{}}
		if index < len(positions.claims) {
			position = positions.claims[index]
		}
		claim, err := buildClaim(element, position)
		if err != nil {
			return nil, err
		}
		if line, ok := seen[claim.ID]; ok {
			return nil, &jsonError{
				line:    position.line,
				message: fmt.Sprintf("the claim id %q is already used at line %d", claim.ID, line),
			}
		}
		seen[claim.ID] = position.line
		file.Claims = append(file.Claims, claim)
	}
	return file, nil
}

// buildClaim validates one claim of a claim file.
func buildClaim(element json.RawMessage, position jsonClaimPosition) (domain.Claim, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(element, &fields); err != nil {
		return domain.Claim{}, &jsonError{line: position.line, message: "every claim must be a JSON object"}
	}
	for _, key := range sortedFields(position.keys) {
		if !contains(claimFields, key) {
			return domain.Claim{}, &jsonError{
				line:    position.keys[key],
				message: fmt.Sprintf("unknown field %q in the claim, the supported fields are %s", key, strings.Join(claimFields, ", ")),
			}
		}
	}

	claim := domain.Claim{}

	raw, ok := fields["id"]
	if !ok {
		return domain.Claim{}, &jsonError{line: position.line, message: "every claim must declare an \"id\""}
	}
	id, err := textValue(raw)
	if err != nil {
		return domain.Claim{}, &jsonError{line: position.keys["id"], message: "the field \"id\" must be a text value"}
	}
	if strings.TrimSpace(id) == "" {
		return domain.Claim{}, &jsonError{line: position.keys["id"], message: "the field \"id\" must not be empty"}
	}
	claim.ID = id

	raw, ok = fields["type"]
	if !ok {
		return domain.Claim{}, &jsonError{line: position.line, message: fmt.Sprintf("every claim must declare a \"type\", the supported types are %s", strings.Join(Types(), ", "))}
	}
	claimType, err := textValue(raw)
	if err != nil {
		return domain.Claim{}, &jsonError{line: position.keys["type"], message: "the field \"type\" must be a text value"}
	}
	if !isCatalogType(claimType) {
		return domain.Claim{}, &jsonError{
			line:    position.keys["type"],
			message: fmt.Sprintf("unknown claim type %q, the supported types are %s", claimType, strings.Join(Types(), ", ")),
		}
	}
	claim.Type = claimType

	if raw, ok := fields["paths"]; ok {
		values, err := textList(raw)
		if err != nil {
			return domain.Claim{}, &jsonError{line: position.keys["paths"], message: "the field \"paths\" must be a list of text values"}
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return domain.Claim{}, &jsonError{line: position.keys["paths"], message: "the field \"paths\" must not contain empty values"}
			}
			if !isRelativePath(value) {
				return domain.Claim{}, &jsonError{
					line:    position.keys["paths"],
					message: fmt.Sprintf("the path %q must be relative to the root of the repository", value),
				}
			}
		}
		claim.Paths = values
	}

	if raw, ok := fields["symbols"]; ok {
		values, err := textList(raw)
		if err != nil {
			return domain.Claim{}, &jsonError{line: position.keys["symbols"], message: "the field \"symbols\" must be a list of text values"}
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return domain.Claim{}, &jsonError{line: position.keys["symbols"], message: "the field \"symbols\" must not contain empty values"}
			}
		}
		claim.Symbols = values
	}

	if raw, ok := fields["dependencies"]; ok {
		values, err := textList(raw)
		if err != nil {
			return domain.Claim{}, &jsonError{line: position.keys["dependencies"], message: "the field \"dependencies\" must be a list of text values"}
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return domain.Claim{}, &jsonError{line: position.keys["dependencies"], message: "the field \"dependencies\" must not contain empty values"}
			}
		}
		claim.Dependencies = values
	}

	if raw, ok := fields["report"]; ok {
		report, err := textValue(raw)
		if err != nil {
			return domain.Claim{}, &jsonError{line: position.keys["report"], message: "the field \"report\" must be a text value"}
		}
		if strings.TrimSpace(report) == "" {
			return domain.Claim{}, &jsonError{line: position.keys["report"], message: "the field \"report\" must not be empty"}
		}
		if !isRelativePath(report) {
			return domain.Claim{}, &jsonError{
				line:    position.keys["report"],
				message: fmt.Sprintf("the report %q must be relative to the root of the repository", report),
			}
		}
		claim.Report = report
	}

	if raw, ok := fields["report_revision"]; ok {
		revision, err := textValue(raw)
		if err != nil {
			return domain.Claim{}, &jsonError{line: position.keys["report_revision"], message: "the field \"report_revision\" must be a text value"}
		}
		claim.ReportRevision = revision
	}

	if requiresPaths(claim.Type) && len(claim.Paths) == 0 {
		return domain.Claim{}, &jsonError{
			line:    position.line,
			message: fmt.Sprintf("a claim of type %q must declare at least one path", claim.Type),
		}
	}

	return claim, nil
}

// requiresPaths reports whether a type of the catalog is about paths. A
// claim of these types without paths would assert nothing, and an assertion
// that cannot be checked must not read as a pass.
func requiresPaths(claimType string) bool {
	switch claimType {
	case domain.ClaimTypeFilesUnchanged, domain.ClaimTypeScope, domain.ClaimTypeTestsAdded, domain.ClaimTypeTestsModified:
		return true
	}
	return false
}

// ValidateBase checks that the base of a claim file matches the base of the
// comparison.
//
// A claim file that declares another base is about another revision, so it
// is rejected instead of being evaluated.
func ValidateBase(file *File, base string) error {
	if file == nil {
		return fmt.Errorf("the validation needs a claim file")
	}
	if file.Base == "" {
		return nil
	}
	if base == "" {
		return fmt.Errorf("the claim file declares the base %q but the comparison has no base: the claims would be about another revision", file.Base)
	}
	if file.Base != base {
		return fmt.Errorf("the claim file declares the base %q but the comparison uses %q: the claims would be about another revision", file.Base, base)
	}
	return nil
}

// textValue reads a JSON string.
func textValue(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

// textList reads a JSON list of strings.
func textList(raw json.RawMessage) ([]string, error) {
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// isRelativePath reports whether a path is relative to the root of the
// repository and does not escape it.
func isRelativePath(value string) bool {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return false
	}
	if len(normalized) > 1 && normalized[1] == ':' {
		// A Windows drive letter, such as "C:/reports".
		return false
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

// contains reports whether a list holds a value.
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// sortedFields returns the keys of a map, sorted.
func sortedFields(fields map[string]int) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
