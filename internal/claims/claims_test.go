package claims

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// parseError parses a claim file and returns the message of the error.
func parseError(t *testing.T, content string) string {
	t.Helper()
	_, err := Parse([]byte(content), "claims.json")
	if err == nil {
		t.Fatalf("expected an error for the document:\n%s", content)
	}
	return err.Error()
}

// requireLine checks that an error names the file and the line.
func requireLine(t *testing.T, message string, line int) {
	t.Helper()
	if !strings.Contains(message, "claims.json") {
		t.Errorf("the error does not name the file: %v", message)
	}
	if !strings.Contains(message, "line "+strconv.Itoa(line)) {
		t.Errorf("the error does not name the line %d: %v", line, message)
	}
}

func TestParseReadsAValidFile(t *testing.T) {
	content := `{
  "schema_version": "0.1",
  "base": "main",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["internal/billing/**"] },
    { "id": "c2", "type": "tests_passed", "report": "reports/go-test.json", "report_revision": "9f2c1ab" },
    { "id": "c3", "type": "dependency_added", "dependencies": ["github.com/acme/lib"] }
  ]
}
`
	file, err := Parse([]byte(content), "claims.json")
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if file.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version: got %q, want %q", file.SchemaVersion, SchemaVersion)
	}
	if file.Base != "main" {
		t.Errorf("base: got %q, want %q", file.Base, "main")
	}
	if len(file.Claims) != 3 {
		t.Fatalf("claims: got %d, want 3", len(file.Claims))
	}
	if file.Claims[1].Report != "reports/go-test.json" {
		t.Errorf("report: got %q", file.Claims[1].Report)
	}
	if file.Claims[1].ReportRevision != "9f2c1ab" {
		t.Errorf("report_revision: got %q", file.Claims[1].ReportRevision)
	}
	if len(file.Claims[2].Dependencies) != 1 || file.Claims[2].Dependencies[0] != "github.com/acme/lib" {
		t.Errorf("dependencies: got %v", file.Claims[2].Dependencies)
	}
}

func TestParseRejectsAMissingSchemaVersion(t *testing.T) {
	message := parseError(t, `{
  "claims": []
}
`)
	requireLine(t, message, 1)
	if !strings.Contains(message, "schema_version") {
		t.Errorf("the error does not name the field: %v", message)
	}
}

func TestParseRejectsAnUnsupportedSchemaVersion(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.2",
  "claims": []
}
`)
	requireLine(t, message, 2)
	if !strings.Contains(message, "unsupported schema_version") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsMissingClaims(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1"
}
`)
	requireLine(t, message, 1)
	if !strings.Contains(message, "\"claims\" is required") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsEmptyClaims(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": []
}
`)
	requireLine(t, message, 3)
	if !strings.Contains(message, "must not be empty") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsClaimsThatAreNotAList(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": "c1"
}
`)
	requireLine(t, message, 3)
	if !strings.Contains(message, "must be a list of claims") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAClaimWithoutIdentifier(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "type": "files_unchanged", "paths": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must declare an \"id\"") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsADuplicateIdentifier(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["internal/**"] },
    { "id": "c1", "type": "scope", "paths": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 5)
	if !strings.Contains(message, "already used at line 4") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAClaimWithoutType(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "paths": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must declare a \"type\"") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAnUnknownType(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "tests_written", "paths": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "unknown claim type \"tests_written\"") {
		t.Errorf("unexpected message: %v", message)
	}
	for _, name := range Types() {
		if !strings.Contains(message, name) {
			t.Errorf("the message does not list the supported type %q: %v", name, message)
		}
	}
}

func TestParseRejectsAnUnknownFieldOfTheFile(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [],
  "head": "abc"
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "unknown field \"head\"") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAnUnknownFieldOfAClaim(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "path": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "unknown field \"path\"") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAPathThatIsNotRelative(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["/etc/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must be relative to the root") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAPathThatEscapesTheRoot(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged", "paths": ["../outside/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must be relative to the root") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAReportThatIsNotRelative(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "tests_passed", "report": "/tmp/report.json" }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must be relative to the root") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAClaimOfPathsWithoutPaths(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": "c1", "type": "files_unchanged" }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "must declare at least one path") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsAMalformedDocument(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
`)
	if !strings.Contains(message, "claims.json") {
		t.Errorf("the error does not name the file: %v", message)
	}
	if !strings.Contains(message, "not valid JSON") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsADocumentThatIsNotAnObject(t *testing.T) {
	message := parseError(t, `[]`)
	requireLine(t, message, 1)
	if !strings.Contains(message, "must be a JSON object") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsATrailingDocument(t *testing.T) {
	message := parseError(t, `{"schema_version": "0.1", "claims": []} {}`)
	if !strings.Contains(message, "single JSON object") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestParseRejectsANonTextIdentifier(t *testing.T) {
	message := parseError(t, `{
  "schema_version": "0.1",
  "claims": [
    { "id": 7, "type": "files_unchanged", "paths": ["internal/**"] }
  ]
}
`)
	requireLine(t, message, 4)
	if !strings.Contains(message, "\"id\" must be a text value") {
		t.Errorf("unexpected message: %v", message)
	}
}

func TestLoadReadsAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claims.json")
	content := `{"schema_version": "0.1", "claims": [{"id": "c1", "type": "files_unchanged", "paths": ["internal/**"]}]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the claim file: %v", err)
	}
	file, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(file.Claims) != 1 {
		t.Errorf("claims: got %d, want 1", len(file.Claims))
	}
}

func TestLoadReportsAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claims.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !strings.Contains(err.Error(), "cannot read the claim file") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestValidateBaseRejectsAnotherBase(t *testing.T) {
	file := &File{SchemaVersion: SchemaVersion, Base: "main"}
	err := ValidateBase(file, "develop")
	if err == nil {
		t.Fatal("expected an error for another base")
	}
	if !strings.Contains(err.Error(), "another revision") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestValidateBaseAcceptsTheSameBase(t *testing.T) {
	file := &File{SchemaVersion: SchemaVersion, Base: "main"}
	if err := ValidateBase(file, "main"); err != nil {
		t.Fatalf("ValidateBase returned an error: %v", err)
	}
}

func TestValidateBaseAcceptsAFileWithoutBase(t *testing.T) {
	file := &File{SchemaVersion: SchemaVersion}
	if err := ValidateBase(file, "main"); err != nil {
		t.Fatalf("ValidateBase returned an error: %v", err)
	}
}

func TestCatalogHoldsEveryType(t *testing.T) {
	expected := []string{
		"api_unchanged",
		"behavior_preserved",
		"dependency_added",
		"dependency_unchanged",
		"files_unchanged",
		"scope",
		"tests_added",
		"tests_modified",
		"tests_passed",
	}
	types := Types()
	if len(types) != len(expected) {
		t.Fatalf("types: got %v, want %v", types, expected)
	}
	for index, name := range expected {
		if types[index] != name {
			t.Errorf("types[%d]: got %q, want %q", index, types[index], name)
		}
	}
	for _, entry := range Catalog() {
		if entry.Description == "" {
			t.Errorf("the type %q has no description", entry.ID)
		}
		if len(entry.Limitations) == 0 {
			t.Errorf("the type %q has no limitation", entry.ID)
		}
	}
}
