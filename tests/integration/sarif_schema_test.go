// Package integration holds the end to end tests of the command line
// interface.
//
// This file validates the SARIF documents of the real reporting paths against
// the authoritative OASIS 2.1.0 JSON schema. The schema is a local test
// fixture: neither the tool nor the tests fetch it over the network.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/mcpserver"
	"github.com/barceloch/codixai/internal/report"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// sarifSchemaID is the canonical identifier of the OASIS schema. The fixture
// is registered under this identifier so local references resolve offline.
const sarifSchemaID = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"

const sarifSchemaFile = "testdata/sarif-schema-2.1.0.json"

var (
	sarifSchemaOnce sync.Once
	sarifSchema     *jsonschema.Schema
	sarifSchemaErr  error
)

// sarifValidator compiles the authoritative schema from the local fixture.
// The compiler never loads a remote resource: the fixture is registered under
// its own $id and every reference of the schema is a local fragment.
func sarifValidator(t *testing.T) *jsonschema.Schema {
	t.Helper()
	sarifSchemaOnce.Do(func() {
		data, err := os.ReadFile(sarifSchemaFile)
		if err != nil {
			sarifSchemaErr = err
			return
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			sarifSchemaErr = err
			return
		}
		compiler := jsonschema.NewCompiler()
		// The schema uses uri formats; assert them instead of treating them
		// as annotations only.
		compiler.AssertFormat()
		if err := compiler.AddResource(sarifSchemaID, document); err != nil {
			sarifSchemaErr = err
			return
		}
		sarifSchema, sarifSchemaErr = compiler.Compile(sarifSchemaID)
	})
	if sarifSchemaErr != nil {
		t.Fatalf("cannot compile the official SARIF 2.1.0 schema: %v", sarifSchemaErr)
	}
	return sarifSchema
}

// assertOfficialSARIF validates a document produced by a real Codixai
// reporting path against the authoritative schema.
func assertOfficialSARIF(t *testing.T, path string, document []byte) {
	t.Helper()
	if !json.Valid(document) {
		t.Fatalf("%s did not produce a JSON document:\n%s", path, document)
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		t.Fatalf("%s produced an unreadable JSON document: %v", path, err)
	}
	if err := sarifValidator(t).Validate(value); err != nil {
		t.Errorf("%s violates the official SARIF 2.1.0 schema:\n%v\nDocument:\n%s", path, err, document)
	}
}

// assertSARIFHasResults fails a case that was supposed to exercise findings,
// so an empty document cannot make a validation case meaningless.
func assertSARIFHasResults(t *testing.T, document []byte, minimum int) {
	t.Helper()
	var parsed struct {
		Runs []struct {
			Results []json.RawMessage `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(document, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Runs) != 1 || len(parsed.Runs[0].Results) < minimum {
		t.Fatalf("the document holds %d runs and %d results, want 1 run and at least %d results",
			len(parsed.Runs), len(parsed.Runs[0].Results), minimum)
	}
}

// cliSARIF runs the report command and returns the document of the standard
// output. Any exit code is accepted: a failing audit still writes its document.
func cliSARIF(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	all := append([]string{"--root", root, "report", "--format", "sarif"}, args...)
	code, stdout, stderr := execute(t, all...)
	if stdout == "" {
		t.Fatalf("codixai %v produced no document (exit %d): %s", all, code, stderr)
	}
	return []byte(stdout)
}

// TestOfficialSchemaValidatorRejectsInvalidDocuments is a negative control:
// it proves that the local schema fixture really constrains documents and that
// the passing cases below are not an artifact of a permissive validator.
func TestOfficialSchemaValidatorRejectsInvalidDocuments(t *testing.T) {
	cases := map[string]string{
		"unsupported version": `{"$schema":"https://json.schemastore.org/sarif-2.1.0.json","version":"2.2.0","runs":[]}`,
		"unknown root field":  `{"$schema":"https://json.schemastore.org/sarif-2.1.0.json","version":"2.1.0","runs":[],"notAField":true}`,
		"missing runs":        `{"$schema":"https://json.schemastore.org/sarif-2.1.0.json","version":"2.1.0"}`,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			value, err := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(document)))
			if err != nil {
				t.Fatal(err)
			}
			if err := sarifValidator(t).Validate(value); err == nil {
				t.Fatalf("the validator accepted an invalid document: %s", document)
			}
		})
	}
}

func TestOfficialSchemaCLIDirectoryWithFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".codixai.yml", "version: 1\naudit:\n  fail_on: warning\n")
	writeFile(t, root, "app.py", failingChange)

	document := cliSARIF(t, root, "--directory", root)
	assertSARIFHasResults(t, document, 1)
	assertOfficialSARIF(t, "cli report --directory (findings)", document)
}

func TestOfficialSchemaCLIDirectoryWithoutFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".codixai.yml", "version: 1\naudit:\n  fail_on: warning\n")

	document := cliSARIF(t, root, "--directory", root)
	assertOfficialSARIF(t, "cli report --directory (no findings)", document)
}

func TestOfficialSchemaCLIWorktreeWithFindings(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	document := cliSARIF(t, root, "--worktree")
	assertSARIFHasResults(t, document, 2)
	assertOfficialSARIF(t, "cli report --worktree (findings)", document)
}

func TestOfficialSchemaCLIWorktreeWithoutFindings(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)

	document := cliSARIF(t, root, "--worktree")
	assertOfficialSARIF(t, "cli report --worktree (no findings)", document)
}

func TestOfficialSchemaCLIBaseWithFindings(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)
	runGit(t, root, "add", "app.py")
	runGit(t, root, "commit", "-q", "-m", "change")

	document := cliSARIF(t, root, "--base", "HEAD~1")
	assertSARIFHasResults(t, document, 2)
	assertOfficialSARIF(t, "cli report --base (findings)", document)
}

func TestOfficialSchemaCLIStagedWithFindings(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)
	runGit(t, root, "add", "app.py")

	document := cliSARIF(t, root, "--staged")
	assertSARIFHasResults(t, document, 2)
	assertOfficialSARIF(t, "cli report --staged (findings)", document)
}

// sarifMCPCall invokes the real MCP report tool through the JSON-RPC entry
// point and returns the text of its result. No transport or client is started.
func sarifMCPCall(t *testing.T, args map[string]any) string {
	t.Helper()
	s := mcpserver.New("schema-test")
	initialize := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"schema-test","version":"1.0.0"}}}`)
	if reply := s.HandleMessage(context.Background(), initialize); reply == nil {
		t.Fatal("the MCP server did not answer the initialize request")
	}

	message, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "report",
			"arguments": args,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reply := s.HandleMessage(context.Background(), message)
	response, ok := reply.(mcp.JSONRPCResponse)
	if !ok {
		if failure, failed := reply.(mcp.JSONRPCError); failed {
			t.Fatalf("the MCP report tool failed: %s", failure.Error.Message)
		}
		t.Fatalf("the MCP report tool returned %T", reply)
	}
	result, ok := response.Result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("the MCP report tool returned %T", response.Result)
	}
	// A failed audit still carries its SARIF document and marks the result
	// as failed; the document is what this test validates.
	if len(result.Content) != 1 {
		t.Fatalf("the MCP result holds %d content items, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("the MCP content is %T", result.Content[0])
	}
	return text.Text
}

func TestOfficialSchemaMCPDirectoryReport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".codixai.yml", "version: 1\naudit:\n  fail_on: warning\n")
	writeFile(t, root, "app.py", failingChange)

	document := []byte(sarifMCPCall(t, map[string]any{"root": root, "directory": root, "format": "sarif"}))
	assertSARIFHasResults(t, document, 1)
	assertOfficialSARIF(t, "mcp report (directory, findings)", document)
}

func TestOfficialSchemaMCPWorktreeReport(t *testing.T) {
	root := newRepository(t)
	writePolicy(t, root)
	writeFile(t, root, "app.py", failingChange)

	document := []byte(sarifMCPCall(t, map[string]any{"root": root, "worktree": true, "format": "sarif"}))
	assertSARIFHasResults(t, document, 2)
	assertOfficialSARIF(t, "mcp report (worktree, findings)", document)
}

// TestOfficialSchemaBuilderEdgeCases validates shapes that the end to end
// paths cannot easily produce, using the production builders themselves.
func TestOfficialSchemaBuilderEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		build func() any
	}{
		{
			name: "nil audit result",
			build: func() any {
				return report.SARIF(nil, report.SARIFOptions{ToolVersion: "schema-test"})
			},
		},
		{
			name: "finding without a line",
			build: func() any {
				finding := domain.Finding{
					SchemaVersion:   domain.SchemaVersion,
					RuleID:          "CDX-SUP-001",
					RuleVersion:     "1",
					Severity:        domain.SeverityWarning,
					EpistemicStatus: domain.EpistemicInferred,
					Message:         "The repository declares a suppression.",
					Location:        domain.Location{Path: "docs/guide.md"},
					Support:         domain.Support{Level: domain.SupportSyntactic},
				}
				return report.SARIF(applicationResultWith(finding), report.SARIFOptions{ToolVersion: "schema-test"})
			},
		},
		{
			name: "unknown rule and missing description",
			build: func() any {
				finding := domain.Finding{
					SchemaVersion:   domain.SchemaVersion,
					RuleID:          "CDX-XYZ-999",
					Severity:        domain.SeverityError,
					EpistemicStatus: domain.EpistemicObserved,
					Message:         "A finding of a rule outside the catalog.",
					Location:        domain.Location{Path: "pkg\\file.go", StartLine: 7, EndLine: 9},
				}
				return report.SARIF(applicationResultWith(finding), report.SARIFOptions{ToolVersion: "schema-test"})
			},
		},
		{
			name: "directory with missing rules",
			build: func() any {
				run := &domain.AnalysisRun{
					SchemaVersion: domain.SchemaVersion,
					ToolVersion:   "schema-test",
					MissingRules: []domain.MissingRule{{
						ID:      "CDX-REF-001",
						Version: "1",
						Reason:  "requires a diff",
					}},
				}
				return report.SARIFDirectory(run, report.SARIFOptions{})
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := json.Marshal(testCase.build())
			if err != nil {
				t.Fatal(err)
			}
			assertOfficialSARIF(t, "builder/"+testCase.name, data)
		})
	}
}

// applicationResultWith wraps a finding in the application result that the
// production builder consumes.
func applicationResultWith(finding domain.Finding) *application.AuditResult {
	return &application.AuditResult{
		Run: &domain.AnalysisRun{
			SchemaVersion: domain.SchemaVersion,
			ToolVersion:   "schema-test",
			Findings:      []domain.Finding{finding},
		},
	}
}
