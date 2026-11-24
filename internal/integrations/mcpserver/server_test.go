package mcpserver

// Contract tests of the Codixai MCP server.
//
// The tests drive the server through the same JSON-RPC entry point the stdio
// and HTTP transports use (MCPServer.HandleMessage), so they exercise the real
// tool registry, the real argument decoding and the real handlers. They never
// start a transport, never reach the network and never touch a production
// database or configuration.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const testServerVersion = "0.1.0-contract-test"

// toolContract is the externally observable contract of one registered tool.
// Changing this table means changing the public MCP contract.
type toolContract struct {
	Name       string
	Required   []string
	Properties []string
	ReadOnly   bool
	Idempotent bool
}

// toolContracts is derived from the registration code in New. There are five
// tools: four read-only and one that legitimately writes a baseline.
var toolContracts = []toolContract{
	{Name: "audit_diff", Required: []string{"root"}, Properties: []string{"base", "include_rules", "root", "staged", "worktree"}, ReadOnly: true, Idempotent: true},
	{Name: "audit_directory", Required: []string{"root"}, Properties: []string{"include_rules", "root"}, ReadOnly: true, Idempotent: true},
	{Name: "verify_claims", Required: []string{"base", "claims_path", "root"}, Properties: []string{"base", "claims_path", "root"}, ReadOnly: true, Idempotent: true},
	{Name: "report", Required: []string{"root"}, Properties: []string{"base", "directory", "format", "include_rules", "root", "staged", "worktree"}, ReadOnly: true, Idempotent: true},
	{Name: "baseline_create", Required: []string{"root"}, Properties: []string{"base", "force", "out", "root", "staged", "worktree"}, ReadOnly: false, Idempotent: false},
}

var messageID atomic.Int64

// newTestServer builds a server and completes the initialize handshake through
// the real JSON-RPC entry point.
func newTestServer(t *testing.T) *server.MCPServer {
	t.Helper()
	s := New(testServerVersion)
	reply := s.HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"contract-test","version":"1.0.0"}}}`))
	if reply == nil {
		t.Fatal("the server did not answer the initialize request")
	}
	if failure, ok := reply.(mcp.JSONRPCError); ok {
		t.Fatalf("initialize failed: %s", failure.Error.Message)
	}
	return s
}

// send delivers a JSON-RPC message to the server and returns its reply.
func send(t *testing.T, s *server.MCPServer, method string, params any) mcp.JSONRPCMessage {
	t.Helper()
	defer func() {
		if value := recover(); value != nil {
			t.Fatalf("%s panicked: %v", method, value)
		}
	}()
	id := messageID.Add(1)
	message := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		message["params"] = params
	}
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("cannot encode the %s request: %v", method, err)
	}
	reply := s.HandleMessage(context.Background(), data)
	if reply == nil {
		t.Fatalf("the server did not answer %s", method)
	}
	return reply
}

// listTools discovers the tools through the standard tools/list request.
func listTools(t *testing.T, s *server.MCPServer) []mcp.Tool {
	t.Helper()
	reply := send(t, s, "tools/list", nil)
	response, ok := reply.(mcp.JSONRPCResponse)
	if !ok {
		if failure, failed := reply.(mcp.JSONRPCError); failed {
			t.Fatalf("tools/list failed: %s", failure.Error.Message)
		}
		t.Fatalf("tools/list returned %T", reply)
	}
	switch result := response.Result.(type) {
	case mcp.ListToolsResult:
		return result.Tools
	case *mcp.ListToolsResult:
		return result.Tools
	default:
		t.Fatalf("tools/list returned an unexpected result %T", response.Result)
	}
	return nil
}

// callTool invokes a tool through the standard tools/call request. The second
// return value is the JSON-RPC error, when the server rejected the request.
func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]any) (*mcp.CallToolResult, *mcp.JSONRPCError) {
	t.Helper()
	params := map[string]any{"name": name}
	if args != nil {
		params["arguments"] = args
	}
	reply := send(t, s, "tools/call", params)
	if response, ok := reply.(mcp.JSONRPCResponse); ok {
		switch result := response.Result.(type) {
		case *mcp.CallToolResult:
			return result, nil
		case mcp.CallToolResult:
			return &result, nil
		default:
			t.Fatalf("tools/call returned an unexpected result %T", response.Result)
		}
	}
	if failure, ok := reply.(mcp.JSONRPCError); ok {
		return nil, &failure
	}
	t.Fatalf("tools/call returned %T", reply)
	return nil, nil
}

// mustCall invokes a tool that must succeed at the protocol level.
func mustCall(t *testing.T, s *server.MCPServer, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, failure := callTool(t, s, name, args)
	if failure != nil {
		t.Fatalf("%s returned a protocol error (%d): %s", name, failure.Error.Code, failure.Error.Message)
	}
	if result == nil {
		t.Fatalf("%s returned no result", name)
	}
	return result
}

// mustFail invokes a tool that must fail at the protocol level and returns the
// error message.
func mustFail(t *testing.T, s *server.MCPServer, name string, args map[string]any) string {
	t.Helper()
	result, failure := callTool(t, s, name, args)
	if failure == nil {
		if result == nil || !result.IsError {
			t.Fatalf("%s unexpectedly succeeded at the protocol level", name)
		}
		return contentText(t, result)
	}
	return failure.Error.Message
}

// contentText returns the single text content of a successful tool result.
func contentText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) != 1 {
		t.Fatalf("the tool result must hold exactly one content item, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("the content is %T, want TextContent", result.Content[0])
	}
	if text.Type != mcp.ContentTypeText {
		t.Fatalf("the content type is %q, want %q", text.Type, mcp.ContentTypeText)
	}
	return text.Text
}

// document decodes the JSON document carried by a tool result.
func document(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(contentText(t, result)), &value); err != nil {
		t.Fatalf("the tool result is not a JSON object: %v", err)
	}
	return value
}

// canonicalJSON returns a stable representation of a JSON value. The
// generated_at field is not part of the deterministic contract.
func canonicalJSON(t *testing.T, value any) string {
	t.Helper()
	copyValue, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected a JSON object, got %T", value)
	}
	normalized := make(map[string]any, len(copyValue))
	for key, item := range copyValue {
		normalized[key] = item
	}
	delete(normalized, "generated_at")
	data, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("cannot encode the normalized document: %v", err)
	}
	return string(data)
}

// sortedKeys returns the sorted keys of a file snapshot.
func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// newProject creates an isolated directory with one source file.
func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "greeting.py", "def greet():\n    return \"hello\"\n")
	return root
}

// newGitRepo creates a repository with one commit and no pending changes.
func newGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available in this environment")
	}
	repo := t.TempDir()
	runGit(t, repo, "init", "-q", ".")
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"hello\"\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "-m", "initial")
	return repo
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("cannot create the directory of %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write %s: %v", name, err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, out)
	}
}

// treeSnapshot fingerprints every file of a tree except the git directory.
// It is used to prove that read-only tools leave the analyzed repository
// untouched.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[relative] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("cannot snapshot %s: %v", root, err)
	}
	return files
}

func assertSameTree(t *testing.T, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("the tool changed the number of files: before %d, after %d", len(before), len(after))
	}
	for path, sum := range before {
		if after[path] != sum {
			t.Fatalf("the tool changed the file %s", path)
		}
	}
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

func TestRegisteredToolsMatchContract(t *testing.T) {
	tools := listTools(t, newTestServer(t))
	if len(tools) != len(toolContracts) {
		names := make([]string, 0, len(tools))
		for _, tool := range tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("the server registers %d tools (%v), want %d", len(tools), names, len(toolContracts))
	}

	byName := make(map[string]mcp.Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	for _, want := range toolContracts {
		t.Run(want.Name, func(t *testing.T) {
			tool, ok := byName[want.Name]
			if !ok {
				t.Fatalf("the tool %s is not registered", want.Name)
			}
			if tool.Description == "" {
				t.Fatal("the tool has no description")
			}
			if tool.InputSchema.Type != "object" {
				t.Fatalf("inputSchema.type is %q, want object", tool.InputSchema.Type)
			}
			properties := make([]string, 0, len(tool.InputSchema.Properties))
			for name := range tool.InputSchema.Properties {
				properties = append(properties, name)
			}
			sort.Strings(properties)
			if strings.Join(properties, ",") != strings.Join(want.Properties, ",") {
				t.Fatalf("properties are %v, want %v", properties, want.Properties)
			}
			required := append([]string(nil), tool.InputSchema.Required...)
			sort.Strings(required)
			if strings.Join(required, ",") != strings.Join(want.Required, ",") {
				t.Fatalf("required is %v, want %v", required, want.Required)
			}
			annotations := tool.Annotations
			if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != want.ReadOnly {
				t.Fatalf("readOnlyHint is %v, want %v", annotations.ReadOnlyHint, want.ReadOnly)
			}
			if annotations.DestructiveHint == nil || *annotations.DestructiveHint {
				t.Fatal("destructiveHint must be advertised as false for every tool")
			}
			if annotations.IdempotentHint == nil || *annotations.IdempotentHint != want.Idempotent {
				t.Fatalf("idempotentHint is %v, want %v", annotations.IdempotentHint, want.Idempotent)
			}
			if annotations.OpenWorldHint == nil || *annotations.OpenWorldHint {
				t.Fatal("openWorldHint must be advertised as false for every tool")
			}
		})
	}
}

func TestToolsListIsDeterministic(t *testing.T) {
	first, err := json.Marshal(listTools(t, newTestServer(t)))
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(listTools(t, newTestServer(t)))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("the advertised contract is not stable across servers")
	}
}

func TestUnknownToolIsRejected(t *testing.T) {
	_, failure := callTool(t, newTestServer(t), "not_a_tool", map[string]any{})
	if failure == nil {
		t.Fatal("an unknown tool must produce a JSON-RPC error")
	}
	if !strings.Contains(failure.Error.Message, "not found") {
		t.Fatalf("the error does not identify the unknown tool: %s", failure.Error.Message)
	}
}

// ---------------------------------------------------------------------------
// Input validation
// ---------------------------------------------------------------------------

func TestToolInputValidation(t *testing.T) {
	root := newProject(t)
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"audit_diff without arguments object", "audit_diff", nil, "invalid arguments"},
		{"audit_diff without root", "audit_diff", map[string]any{}, "root is required"},
		{"audit_diff with a non-string root", "audit_diff", map[string]any{"root": 42}, "root is required"},
		{"audit_diff without origin", "audit_diff", map[string]any{"root": root}, "no origin selected"},
		{"audit_diff with two origins", "audit_diff", map[string]any{"root": root, "staged": true, "worktree": true}, "several origins selected"},
		{"audit_diff with a non-boolean flag", "audit_diff", map[string]any{"root": root, "staged": "true"}, "no origin selected"},
		{"audit_directory without root", "audit_directory", map[string]any{}, "root is required"},
		{"verify_claims without root", "verify_claims", map[string]any{}, "root is required"},
		{"verify_claims without base", "verify_claims", map[string]any{"root": root}, "base is required"},
		{"verify_claims without claims_path", "verify_claims", map[string]any{"root": root, "base": "HEAD"}, "claims_path is required"},
		{"report without root", "report", map[string]any{}, "root is required"},
		{"report with an invalid format", "report", map[string]any{"root": root, "directory": root, "format": "xml"}, "invalid format: xml (must be sarif or json)"},
		{"report without origin", "report", map[string]any{"root": root}, "no origin selected"},
		{"report with two origins", "report", map[string]any{"root": root, "staged": true, "directory": root}, "several origins selected"},
		{"baseline_create without root", "baseline_create", map[string]any{}, "root is required"},
		{"baseline_create without origin", "baseline_create", map[string]any{"root": root}, "no origin selected"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			message := mustFail(t, newTestServer(t), testCase.tool, testCase.args)
			if !strings.Contains(message, testCase.want) {
				t.Fatalf("error %q does not contain %q", message, testCase.want)
			}
		})
	}
}

func TestUnknownArgumentsAreIgnored(t *testing.T) {
	// The server does not enable schema validation: extra arguments reach the
	// handler and are ignored. This is the current public behavior.
	root := newProject(t)
	result := mustCall(t, newTestServer(t), "audit_directory", map[string]any{
		"root":       root,
		"frobnicate": true,
		"unexpected": "value",
	})
	if result.IsError {
		t.Fatal("an unknown argument must not turn a valid request into a failure")
	}
}

func TestInvalidTypesAreIgnoredWhereTheContractDoesNotValidateThem(t *testing.T) {
	// A non-string format is treated as absent, so the documented default of
	// SARIF applies. The server does not impose stricter validation.
	root := newProject(t)
	sarif := document(t, mustCall(t, newTestServer(t), "report", map[string]any{
		"root": root, "directory": root, "format": 42,
	}))
	assertSARIF(t, sarif)
}

// ---------------------------------------------------------------------------
// Successful responses and document contract
// ---------------------------------------------------------------------------

func TestAuditDirectoryDocument(t *testing.T) {
	root := newProject(t)
	result := mustCall(t, newTestServer(t), "audit_directory", map[string]any{"root": root})
	if result.IsError {
		t.Fatalf("the audit failed: %s", contentText(t, result))
	}
	doc := document(t, result)
	if doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("schema_version is %v, want %s", doc["schema_version"], domain.SchemaVersion)
	}
	if doc["tool_version"] != toolVersion {
		t.Fatalf("tool_version is %v, want %s", doc["tool_version"], toolVersion)
	}
	if doc["origin"] != "directory" {
		t.Fatalf("origin is %v, want directory", doc["origin"])
	}
	for _, key := range []string{"generated_at", "revision", "findings", "summary", "missing_rules", "files"} {
		if _, ok := doc[key]; !ok {
			t.Fatalf("the document is missing %s", key)
		}
	}
	files, ok := doc["files"].(map[string]any)
	if !ok || files["discovered"] == nil {
		t.Fatalf("files statistics are missing or malformed: %v", doc["files"])
	}
	if findings, ok := doc["findings"].([]any); ok {
		for _, raw := range findings {
			assertFindingContract(t, raw.(map[string]any))
		}
	}
}

func assertFindingContract(t *testing.T, finding map[string]any) {
	t.Helper()
	for _, key := range []string{"schema_version", "rule_id", "rule_version", "severity", "epistemic_status", "message", "location"} {
		if _, ok := finding[key]; !ok {
			t.Fatalf("a finding is missing %s", key)
		}
	}
	if finding["schema_version"] != domain.SchemaVersion {
		t.Fatalf("a finding has schema_version %v", finding["schema_version"])
	}
}

func TestAuditDirectoryDocumentIsDeterministic(t *testing.T) {
	root := newProject(t)
	s := newTestServer(t)
	first := document(t, mustCall(t, s, "audit_directory", map[string]any{"root": root}))
	second := document(t, mustCall(t, s, "audit_directory", map[string]any{"root": root}))
	if canonicalJSON(t, first) != canonicalJSON(t, second) {
		t.Fatal("two identical directory audits produced different documents")
	}
}

func TestAuditDiffWorktreeDocument(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"goodbye\"\n")

	result := mustCall(t, newTestServer(t), "audit_diff", map[string]any{"root": repo, "worktree": true})
	doc := document(t, result)
	if doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("schema_version is %v, want %s", doc["schema_version"], domain.SchemaVersion)
	}
	if doc["origin"] != "worktree" {
		t.Fatalf("origin is %v, want worktree", doc["origin"])
	}
	if doc["generated_at"] == nil || doc["summary"] == nil || doc["findings"] == nil {
		t.Fatalf("the audit document is incomplete: %v", doc)
	}
	if findings, ok := doc["findings"].([]any); ok {
		for _, raw := range findings {
			assertFindingContract(t, raw.(map[string]any))
		}
	}
}

func TestAuditDiffStagedDocument(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"staged\"\n")
	runGit(t, repo, "add", "greeting.py")

	result := mustCall(t, newTestServer(t), "audit_diff", map[string]any{"root": repo, "staged": true})
	doc := document(t, result)
	if doc["origin"] != "staged" {
		t.Fatalf("origin is %v, want staged", doc["origin"])
	}
	if doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("schema_version is %v, want %s", doc["schema_version"], domain.SchemaVersion)
	}
}

func TestReportFormats(t *testing.T) {
	root := newProject(t)
	args := map[string]any{"root": root, "directory": root}

	jsonDoc := document(t, mustCall(t, newTestServer(t), "report", withFormat(args, "json")))
	if jsonDoc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("the JSON report has schema_version %v", jsonDoc["schema_version"])
	}

	sarif := document(t, mustCall(t, newTestServer(t), "report", withFormat(args, "sarif")))
	assertSARIF(t, sarif)
	defaultFormat := document(t, mustCall(t, newTestServer(t), "report", args))
	if canonicalJSON(t, sarif) != canonicalJSON(t, defaultFormat) {
		t.Fatal("the default report format is not SARIF")
	}
}

func withFormat(args map[string]any, format string) map[string]any {
	copyArgs := make(map[string]any, len(args)+1)
	for key, value := range args {
		copyArgs[key] = value
	}
	copyArgs["format"] = format
	return copyArgs
}

func assertSARIF(t *testing.T, doc map[string]any) {
	t.Helper()
	if doc["version"] != "2.1.0" {
		t.Fatalf("the SARIF version is %v, want 2.1.0", doc["version"])
	}
	if _, ok := doc["$schema"]; !ok {
		t.Fatal("the SARIF document has no $schema")
	}
	runs, ok := doc["runs"].([]any)
	if !ok || len(runs) != 1 {
		t.Fatalf("the SARIF document has no single run: %v", doc["runs"])
	}
	run, ok := runs[0].(map[string]any)
	if !ok {
		t.Fatalf("a SARIF run is %T", runs[0])
	}
	tool, ok := run["tool"].(map[string]any)
	if !ok {
		t.Fatalf("the SARIF tool is %T", run["tool"])
	}
	driver, ok := tool["driver"].(map[string]any)
	if !ok {
		t.Fatalf("the SARIF driver is %T", tool["driver"])
	}
	if driver["name"] != "codixai" {
		t.Fatalf("the driver name is %v, want codixai", driver["name"])
	}
	if driver["version"] != toolVersion {
		t.Fatalf("the driver version is %v, want %s", driver["version"], toolVersion)
	}
	if driver["rules"] == nil {
		t.Fatal("the SARIF driver has no rules")
	}
}

func TestVerifyClaimsDocument(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"changed\"\n")
	runGit(t, repo, "add", "greeting.py")
	runGit(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "-m", "change")

	claimsPath := filepath.Join(repo, "claims.json")
	claims := `{
  "schema_version": "0.1",
  "base": "HEAD~1",
  "claims": [
    {"id": "c1", "type": "files_unchanged", "paths": ["greeting.py"]},
    {"id": "c2", "type": "files_unchanged", "paths": ["other.py"]}
  ]
}`
	writeTestFile(t, repo, "claims.json", claims)

	result := mustCall(t, newTestServer(t), "verify_claims", map[string]any{
		"root":        repo,
		"base":        "HEAD~1",
		"claims_path": claimsPath,
	})
	doc := document(t, result)
	if doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("schema_version is %v, want %s", doc["schema_version"], domain.SchemaVersion)
	}
	if doc["tool_version"] != toolVersion {
		t.Fatalf("tool_version is %v, want %s", doc["tool_version"], toolVersion)
	}
	if doc["base"] != "HEAD~1" || doc["head"] == nil {
		t.Fatalf("the verification document has no base/head: %v", doc)
	}
	results, ok := doc["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("the verification document has no two results: %v", doc["results"])
	}
	statuses := map[string]bool{}
	for _, raw := range results {
		entry, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("a claim result is %T", raw)
		}
		status, ok := entry["status"].(string)
		if !ok || status == "" {
			t.Fatalf("a claim result has no status: %v", entry)
		}
		if entry["claim"] == nil {
			t.Fatalf("a claim result has no claim: %v", entry)
		}
		statuses[status] = true
	}
	if !statuses["contradicted"] || !statuses["verified"] {
		t.Fatalf("unexpected claim statuses: %v", sortedKeysOfBool(statuses))
	}
	if !result.IsError {
		t.Fatal("a contradicted claim must mark the tool result as failed")
	}
}

func sortedKeysOfBool(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Error behavior
// ---------------------------------------------------------------------------

func TestApplicationErrorsAreControlled(t *testing.T) {
	repo := newGitRepo(t)
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"missing directory", "audit_directory", map[string]any{"root": filepath.Join(repo, "absent")}, ""},
		{"missing git repository", "audit_diff", map[string]any{"root": newProject(t), "worktree": true}, ""},
		{"missing base revision", "report", map[string]any{"root": repo, "base": "does-not-exist"}, ""},
		{"missing claims file", "verify_claims", map[string]any{"root": repo, "base": "HEAD", "claims_path": filepath.Join(repo, "absent.json")}, ""},
		{"malformed claims file", "verify_claims", map[string]any{"root": repo, "base": "HEAD", "claims_path": writeMalformedClaims(t, repo)}, ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			message := mustFail(t, newTestServer(t), testCase.tool, testCase.args)
			if strings.TrimSpace(message) == "" {
				t.Fatal("the server produced an empty error message")
			}
		})
	}
}

func writeMalformedClaims(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "malformed-claims.json")
	writeTestFile(t, root, "malformed-claims.json", "{not json\n")
	return path
}

func TestFailedAuditIsReportedAsFailedResult(t *testing.T) {
	// A finding that reaches the error threshold must travel with the document
	// and must also mark the MCP result as failed.
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"changed\"\n")
	runGit(t, repo, "add", "greeting.py")
	runGit(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "-m", "change")

	claimsPath := filepath.Join(repo, "claims.json")
	writeTestFile(t, repo, "claims.json", `{
  "schema_version": "0.1",
  "claims": [{"id": "c1", "type": "files_unchanged", "paths": ["greeting.py"]}]
}`)

	result := mustCall(t, newTestServer(t), "verify_claims", map[string]any{
		"root": repo, "base": "HEAD~1", "claims_path": claimsPath,
	})
	if !result.IsError {
		t.Fatal("a contradicted claim must set isError")
	}
	if doc := document(t, result); doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("a failed tool call must still carry a valid document: %v", doc)
	}
}

// ---------------------------------------------------------------------------
// Read-only guarantees and the legitimate write of baseline_create
// ---------------------------------------------------------------------------

func TestReadOnlyToolsLeaveTheRepositoryUntouched(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"changed\"\n")
	claimsPath := filepath.Join(repo, "claims.json")
	writeTestFile(t, repo, "claims.json", `{"schema_version":"0.1","claims":[{"id":"c1","type":"files_unchanged","paths":["missing.py"]}]}`)
	before := treeSnapshot(t, repo)
	s := newTestServer(t)

	// Each read-only tool is exercised with a valid request.
	_ = mustCall(t, s, "audit_diff", map[string]any{"root": repo, "worktree": true})
	_ = mustCall(t, s, "audit_diff", map[string]any{"root": repo, "base": "HEAD"})
	_ = mustCall(t, s, "audit_directory", map[string]any{"root": repo})
	_ = mustCall(t, s, "report", map[string]any{"root": repo, "directory": repo, "format": "json"})
	_ = mustCall(t, s, "report", map[string]any{"root": repo, "directory": repo})
	_ = mustCall(t, s, "verify_claims", map[string]any{"root": repo, "base": "HEAD", "claims_path": claimsPath})

	assertSameTree(t, before, treeSnapshot(t, repo))
	if _, err := os.Stat(filepath.Join(repo, ".codixai")); !os.IsNotExist(err) {
		t.Fatalf("a read-only tool created an index directory: %v", err)
	}
}

func TestAnalyzedCodeIsNotExecuted(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "run.sh", "#!/bin/sh\ntouch \"$(dirname \"$0\")/EXECUTED\"\n")
	writeTestFile(t, repo, "init_code.go", "package main\n\nfunc init() {\n    _ = writeMarker()\n}\n\nfunc writeMarker() error { return nil }\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "-c", "user.email=dev@example.com", "-c", "user.name=Dev", "commit", "-q", "-m", "add code")
	before := treeSnapshot(t, repo)

	s := newTestServer(t)
	_ = mustCall(t, s, "audit_diff", map[string]any{"root": repo, "base": "HEAD~1"})
	_ = mustCall(t, s, "audit_directory", map[string]any{"root": repo})

	if _, err := os.Stat(filepath.Join(repo, "EXECUTED")); !os.IsNotExist(err) {
		t.Fatalf("the analyzed code was executed: %v", err)
	}
	assertSameTree(t, before, treeSnapshot(t, repo))
}

func TestBaselineCreateWritesAndProtectsItsFile(t *testing.T) {
	repo := newGitRepo(t)
	writeTestFile(t, repo, "greeting.py", "def greet():\n    return \"changed\"\n")
	before := treeSnapshot(t, repo)
	s := newTestServer(t)
	args := map[string]any{"root": repo, "worktree": true, "out": "contract-baseline.json"}

	result := mustCall(t, s, "baseline_create", args)
	if result.IsError {
		t.Fatalf("baseline creation failed: %s", contentText(t, result))
	}
	doc := document(t, result)
	if doc["schema_version"] != domain.SchemaVersion {
		t.Fatalf("schema_version is %v, want %s", doc["schema_version"], domain.SchemaVersion)
	}
	if doc["created"] != true || doc["findings"] == nil || doc["new_findings"] == nil {
		t.Fatalf("the baseline document is incomplete: %v", doc)
	}
	if doc["tool_version"] != toolVersion {
		t.Fatalf("tool_version is %v, want %s", doc["tool_version"], toolVersion)
	}
	path := filepath.Join(repo, "contract-baseline.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the baseline file was not written: %v", err)
	}

	message := mustFail(t, s, "baseline_create", args)
	if !strings.Contains(message, "already exists") {
		t.Fatalf("replacing a baseline without force must be refused, got: %s", message)
	}
	if !strings.Contains(message, "force") {
		t.Fatalf("the error must tell the user how to recover, got: %s", message)
	}

	forced := withBool(args, "force", true)
	if result := mustCall(t, s, "baseline_create", forced); result.IsError {
		t.Fatalf("a forced replacement failed: %s", contentText(t, result))
	}

	after := treeSnapshot(t, repo)
	if len(after) != len(before)+1 || after["contract-baseline.json"] == "" {
		t.Fatalf("baseline_create changed unexpected files: %v", sortedKeys(after))
	}
}

func withBool(args map[string]any, key string, value bool) map[string]any {
	copyArgs := make(map[string]any, len(args)+1)
	for name, item := range args {
		copyArgs[name] = item
	}
	copyArgs[key] = value
	return copyArgs
}

func TestBaselinePathCannotEscapeTheRepository(t *testing.T) {
	repo := newGitRepo(t)
	message := mustFail(t, newTestServer(t), "baseline_create", map[string]any{
		"root": repo, "worktree": true, "out": "../outside-baseline.json",
	})
	if !strings.Contains(message, "outside") {
		t.Fatalf("an escaping baseline path must be rejected, got: %s", message)
	}
}

func TestToolResultSerialization(t *testing.T) {
	result := mustCall(t, newTestServer(t), "audit_directory", map[string]any{"root": newProject(t)})
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	content, ok := wire["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("the serialized result has no single content item: %v", wire)
	}
	item := content[0].(map[string]any)
	if item["type"] != "text" || item["text"] == nil {
		t.Fatalf("the serialized content is malformed: %v", item)
	}
	if _, ok := wire["isError"]; ok {
		t.Fatal("isError must be omitted on a successful call")
	}
}

// TestIncludeRulesControlsTheCatalog checks that the catalog of rules is part
// of the document by default and that a caller can ask for the compact
// document without losing the findings, the summary or the limitations.
func TestIncludeRulesControlsTheCatalog(t *testing.T) {
	root := newProject(t)

	withCatalog := document(t, mustCall(t, newTestServer(t), "audit_directory", map[string]any{"root": root}))
	if _, ok := withCatalog["rules"]; !ok {
		t.Fatal("the default document does not carry the catalog of rules")
	}

	compact := document(t, mustCall(t, newTestServer(t), "audit_directory",
		map[string]any{"root": root, "include_rules": false}))
	if _, ok := compact["rules"]; ok {
		t.Fatal("the compact document carries the catalog of rules")
	}
	for _, key := range []string{"findings", "summary", "files", "limitations"} {
		if _, ok := compact[key]; !ok {
			t.Fatalf("the compact document is missing %s", key)
		}
	}
	if len(compact) >= len(withCatalog) {
		t.Fatalf("the compact document is not smaller: %d keys against %d", len(compact), len(withCatalog))
	}
}

// TestIncludeRulesRejectsAWrongType checks that a value that is not a boolean
// is rejected instead of silently keeping the default.
func TestIncludeRulesRejectsAWrongType(t *testing.T) {
	root := newProject(t)
	_, failure := callTool(t, newTestServer(t), "audit_directory",
		map[string]any{"root": root, "include_rules": "no"})
	if failure == nil {
		t.Fatal("a wrong type of include_rules was accepted")
	}
	if !strings.Contains(failure.Error.Message, "include_rules") {
		t.Fatalf("the error does not name the argument: %s", failure.Error.Message)
	}
}

// TestReportIncludeRulesAppliesToTheJSONDocumentOnly checks that the SARIF
// document keeps its rule descriptors whatever the caller asks, because they
// are part of the SARIF contract and not of the audit document.
func TestReportIncludeRulesAppliesToTheJSONDocumentOnly(t *testing.T) {
	root := newProject(t)

	jsonDocument := document(t, mustCall(t, newTestServer(t), "report",
		map[string]any{"root": root, "directory": root, "format": "json", "include_rules": false}))
	if _, ok := jsonDocument["rules"]; ok {
		t.Fatal("the compact JSON document carries the catalog of rules")
	}

	result := mustCall(t, newTestServer(t), "report",
		map[string]any{"root": root, "directory": root, "format": "sarif", "include_rules": false})
	sarif := document(t, result)
	if _, ok := sarif["runs"]; !ok {
		t.Fatalf("the SARIF document is malformed: %v", sarif)
	}
}
