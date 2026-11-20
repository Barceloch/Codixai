//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/integrations/git"
	"github.com/barceloch/codixai/internal/integrations/mcpserver"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMCPVerifyClaimsParityWithCLI verifies the contract of P4-04: the MCP
// verify_claims tool and the application use case used by the CLI compose the
// same verification document for the same inputs. The tool is invoked through
// the real JSON-RPC entry point; the two documents are compared after removing
// the fields that describe the run rather than its result. The transport itself
// belongs to mcp-go and is not under test here.
func TestMCPVerifyClaimsParityWithCLI(t *testing.T) {
	requireGit(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	writeFile(t, repo, "file.txt", "hello\n")
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-q", "-m", "initial")

	writeFile(t, repo, "file.txt", "hello world\n")
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-q", "-m", "change")

	claimsPath := filepath.Join(repo, "claims.json")
	claimsDoc := `{
  "schema_version": "0.1",
  "base": "HEAD~1",
  "claims": [
    {"id": "c1", "type": "files_unchanged", "paths": ["file.txt"]},
    {"id": "c2", "type": "files_unchanged", "paths": ["missing.txt"]}
  ]
}`
	if err := os.WriteFile(claimsPath, []byte(claimsDoc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run via application (same use case as the CLI).
	source, err := git.New(repo)
	if err != nil {
		t.Fatal(err)
	}
	result, err := application.RunVerify(context.Background(), application.VerifyRequest{
		Root:        repo,
		Base:        "HEAD~1",
		ClaimsPath:  claimsPath,
		Source:      source,
		Loader:      claims.Loader{},
		Evaluator:   claims.NewEvaluator(claims.WithRoot(repo), claims.WithBase("HEAD~1")),
		ToolVersion: "parity-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	applicationDoc, err := json.Marshal(result.Document)
	if err != nil {
		t.Fatal(err)
	}

	// Run the real tool through the server's JSON-RPC handling path.
	server := mcpserver.New("parity-test")
	ctx := context.Background()
	initialize := json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"parity-test","version":"1.0.0"}}}`)
	if reply := server.HandleMessage(ctx, initialize); reply == nil {
		t.Fatal("the server did not answer the initialize request")
	}
	call, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "verify_claims",
			"arguments": map[string]any{
				"root":        repo,
				"base":        "HEAD~1",
				"claims_path": claimsPath,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reply := server.HandleMessage(ctx, call)
	response, ok := reply.(mcp.JSONRPCResponse)
	if !ok {
		if failure, failed := reply.(mcp.JSONRPCError); failed {
			t.Fatalf("verify_claims returned a JSON-RPC error (%d): %s", failure.Error.Code, failure.Error.Message)
		}
		t.Fatalf("verify_claims returned %T", reply)
	}
	var toolResult *mcp.CallToolResult
	switch value := response.Result.(type) {
	case *mcp.CallToolResult:
		toolResult = value
	case mcp.CallToolResult:
		toolResult = &value
	default:
		t.Fatalf("verify_claims returned an unexpected result %T", response.Result)
	}
	if len(toolResult.Content) != 1 {
		t.Fatalf("verify_claims returned %d content items, want 1", len(toolResult.Content))
	}
	text, ok := toolResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("verify_claims content is %T", toolResult.Content[0])
	}

	// A contradicted claim marks the tool result as failed; the document is
	// still the payload to compare. Error semantics are covered by the WU2
	// contract tests and are not repeated here.
	applicationParsed := documentWithoutTimestamp(t, string(applicationDoc))
	mcpParsed := documentWithoutTimestamp(t, text.Text)
	// The tool version names the caller, not the verification result; the CLI
	// and the MCP server stamp different callers on purpose.
	delete(applicationParsed, "tool_version")
	delete(mcpParsed, "tool_version")
	// The fixture has two claims: a document with no results must not pass
	// only because both sides are trivial.
	for name, parsed := range map[string]map[string]any{"application": applicationParsed, "MCP": mcpParsed} {
		results, ok := parsed["results"].([]any)
		if !ok || len(results) != 2 {
			t.Fatalf("the %s document holds %v results, want 2", name, parsed["results"])
		}
	}
	if !reflect.DeepEqual(applicationParsed, mcpParsed) {
		t.Errorf("the application and MCP documents differ:\napplication: %v\nMCP:         %v", applicationParsed, mcpParsed)
	}
}
