// Package mcpserver provides a local read-only MCP server using mcp-go.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/baseline"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
	"github.com/barceloch/codixai/internal/policies"
	"github.com/barceloch/codixai/internal/report"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// toolVersion is the version of the tool that the tools of this server report
// in the documents they produce. The server itself carries the version of the
// build; the documents name the version of the tool contract.
const toolVersion = "mcp"

// New builds a Codixai MCP server exposing audit and verify tools.
func New(version string) *server.MCPServer {
	s := server.NewMCPServer("codixai", version, server.WithToolCapabilities(true))

	readOnly := true
	once := true
	s.AddTool(mcp.NewTool("audit_diff",
		mcp.WithDescription("Run a Codixai audit over repository changes and return the analysis document."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository root")),
		mcp.WithString("base", mcp.Description("Base revision for base comparison")),
		mcp.WithBoolean("staged", mcp.Description("Audit staged changes")),
		mcp.WithBoolean("worktree", mcp.Description("Audit changes that are not committed yet")),
		mcp.WithBoolean("include_rules", mcp.Description("Include the catalog of rules in the document (default: true). Set it to false to keep the context small when you do not need the catalog: it is the same for every call of a build.")),
		mcp.WithReadOnlyHintAnnotation(readOnly),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(once),
		mcp.WithOpenWorldHintAnnotation(false),
	), handleAuditDiff)

	s.AddTool(mcp.NewTool("audit_directory",
		mcp.WithDescription("Run a Codixai audit over a directory without requiring a Git repository."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the directory root")),
		mcp.WithBoolean("include_rules", mcp.Description("Include the catalog of rules in the document (default: true). Set it to false to keep the context small when you do not need the catalog: it is the same for every call of a build.")),
		mcp.WithReadOnlyHintAnnotation(readOnly),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(once),
		mcp.WithOpenWorldHintAnnotation(false),
	), handleAuditDirectory)

	s.AddTool(mcp.NewTool("verify_claims",
		mcp.WithDescription("Verify agent claims against the repository and return the verification document."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository root")),
		mcp.WithString("base", mcp.Required(), mcp.Description("Base revision to compare against")),
		mcp.WithString("claims_path", mcp.Required(), mcp.Description("Path to the claims JSON file")),
		mcp.WithReadOnlyHintAnnotation(readOnly),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(once),
		mcp.WithOpenWorldHintAnnotation(false),
	), handleVerifyClaims)

	s.AddTool(mcp.NewTool("report",
		mcp.WithDescription("Export the audit results as SARIF or JSON document."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository or directory root")),
		mcp.WithString("format", mcp.Description("Output format: sarif or json (default: sarif)")),
		mcp.WithString("base", mcp.Description("Base revision for base comparison")),
		mcp.WithBoolean("staged", mcp.Description("Audit staged changes")),
		mcp.WithBoolean("worktree", mcp.Description("Audit working tree changes")),
		mcp.WithString("directory", mcp.Description("Analyze a directory without requiring a Git repository")),
		mcp.WithBoolean("include_rules", mcp.Description("Include the catalog of rules in the JSON document (default: true); the SARIF document always carries its rule descriptors")),
		mcp.WithReadOnlyHintAnnotation(readOnly),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(once),
		mcp.WithOpenWorldHintAnnotation(false),
	), handleReport)

	s.AddTool(mcp.NewTool("baseline_create",
		mcp.WithDescription("Create a baseline of findings for the repository. Writes one baseline file: it requires an origin and refuses to replace an existing baseline without force."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository root")),
		mcp.WithString("base", mcp.Description("Base revision for base comparison")),
		mcp.WithBoolean("staged", mcp.Description("Create baseline from staged changes")),
		mcp.WithBoolean("worktree", mcp.Description("Create baseline from working tree changes")),
		mcp.WithString("out", mcp.Description("Path of the baseline file to write (defaults to audit.baseline config)")),
		mcp.WithBoolean("force", mcp.Description("Replace an existing baseline file")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	), handleBaselineCreate)

	return s
}

// handleAuditDiff audits the changes of a Git repository.
func handleAuditDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(req)
	if err != nil {
		return nil, err
	}
	root, err := resolveServerPath(getString(args, "root"))
	if err != nil {
		return nil, err
	}
	origin, base, err := resolveDiffOrigin(getString(args, "base"), getBool(args, "staged"), getBool(args, "worktree"))
	if err != nil {
		return nil, err
	}
	source, err := git.New(root)
	if err != nil {
		return nil, err
	}
	result, err := application.RunAudit(ctx, application.AuditRequest{
		Root:     root,
		Origin:   origin,
		Base:     base,
		Source:   source,
		Analyzer: analysis.New(analysis.WithToolVersion(toolVersion)),
	})
	if err != nil {
		return nil, err
	}
	withRules, err := includeRules(args)
	if err != nil {
		return nil, err
	}
	if !withRules {
		result.Run.Rules = nil
	}
	document, err := json.Marshal(result.Run)
	if err != nil {
		return nil, err
	}
	return jsonResult(string(document), result.Failed), nil
}

// handleAuditDirectory audits every file of a directory without requiring Git.
func handleAuditDirectory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(req)
	if err != nil {
		return nil, err
	}
	root, err := resolveServerPath(getString(args, "root"))
	if err != nil {
		return nil, err
	}
	result, err := application.RunDirectoryAudit(ctx, application.DirectoryAuditRequest{
		Root:     root,
		Analyzer: analysis.NewDirectoryAnalyzer(analysis.WithToolVersion(toolVersion)),
	})
	if err != nil {
		return nil, err
	}
	withRules, err := includeRules(args)
	if err != nil {
		return nil, err
	}
	if !withRules {
		result.Run.Rules = nil
	}
	document, err := json.Marshal(result.Run)
	if err != nil {
		return nil, err
	}
	config, err := policies.Load(root)
	if err != nil {
		return nil, err
	}
	failed := runFailed(result.Run.Findings, config.Audit.FailOn)
	return jsonResult(string(document), failed), nil
}

// handleVerifyClaims verifies the claims of a file against a repository revision.
func handleVerifyClaims(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(req)
	if err != nil {
		return nil, err
	}
	root, err := resolveServerPath(getString(args, "root"))
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(getString(args, "base"))
	if base == "" {
		return nil, fmt.Errorf("base is required")
	}
	claimsPath := strings.TrimSpace(getString(args, "claims_path"))
	if claimsPath == "" {
		return nil, fmt.Errorf("claims_path is required")
	}
	if !filepath.IsAbs(claimsPath) {
		absolute, absErr := filepath.Abs(claimsPath)
		if absErr != nil {
			return nil, absErr
		}
		claimsPath = absolute
	}
	source, err := git.New(root)
	if err != nil {
		return nil, err
	}
	result, err := application.RunVerify(ctx, application.VerifyRequest{
		Root:        root,
		Base:        base,
		ClaimsPath:  claimsPath,
		Source:      source,
		Loader:      claims.Loader{},
		Evaluator:   claims.NewEvaluator(claims.WithRoot(root), claims.WithBase(base)),
		ToolVersion: toolVersion,
	})
	if err != nil {
		return nil, err
	}
	document, err := json.Marshal(result.Document)
	if err != nil {
		return nil, err
	}
	return jsonResult(string(document), result.Failed), nil
}

// handleReport exports the audit of a change set or of a directory as SARIF or
// as the audit document.
func handleReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(req)
	if err != nil {
		return nil, err
	}
	root, err := resolveServerPath(getString(args, "root"))
	if err != nil {
		return nil, err
	}
	format := strings.ToLower(strings.TrimSpace(getString(args, "format")))
	if format == "" {
		format = "sarif"
	}
	if format != "sarif" && format != "json" {
		return nil, fmt.Errorf("invalid format: %s (must be sarif or json)", format)
	}
	base := strings.TrimSpace(getString(args, "base"))
	staged := getBool(args, "staged")
	worktree := getBool(args, "worktree")
	directory := strings.TrimSpace(getString(args, "directory"))
	selected := 0
	if base != "" {
		selected++
	}
	if staged {
		selected++
	}
	if worktree {
		selected++
	}
	if directory != "" {
		selected++
	}
	if selected == 0 {
		return nil, fmt.Errorf("no origin selected: use one of base, staged, worktree or directory")
	}
	if selected > 1 {
		return nil, fmt.Errorf("several origins selected: use only one of base, staged, worktree or directory")
	}
	withRules, err := includeRules(args)
	if err != nil {
		return nil, err
	}

	var document []byte
	var failed bool

	if directory != "" {
		result, err := application.RunDirectoryAudit(ctx, application.DirectoryAuditRequest{
			Root:     directory,
			Analyzer: analysis.NewDirectoryAnalyzer(analysis.WithToolVersion(toolVersion)),
		})
		if err != nil {
			return nil, err
		}
		if format == "sarif" {
			document, err = json.Marshal(report.SARIFDirectory(result.Run, report.SARIFOptions{ToolVersion: toolVersion}))
		} else {
			if !withRules {
				result.Run.Rules = nil
			}
			document, err = json.Marshal(result.Run)
		}
		if err != nil {
			return nil, err
		}
		config, err := policies.Load(directory)
		if err != nil {
			return nil, err
		}
		failed = runFailed(result.Run.Findings, config.Audit.FailOn)
	} else {
		origin, baseRevision, err := resolveDiffOrigin(base, staged, worktree)
		if err != nil {
			return nil, err
		}
		source, err := git.New(root)
		if err != nil {
			return nil, err
		}
		result, err := application.RunAudit(ctx, application.AuditRequest{
			Root:     root,
			Origin:   origin,
			Base:     baseRevision,
			Source:   source,
			Analyzer: analysis.New(analysis.WithToolVersion(toolVersion)),
		})
		if err != nil {
			return nil, err
		}
		if format == "sarif" {
			document, err = json.Marshal(report.SARIF(result, report.SARIFOptions{ToolVersion: toolVersion}))
		} else {
			if !withRules {
				result.Run.Rules = nil
			}
			document, err = json.Marshal(result.Run)
		}
		if err != nil {
			return nil, err
		}
		failed = result.Failed
	}

	return jsonResult(string(document), failed), nil
}

// handleBaselineCreate writes the baseline of the findings of a change set.
func handleBaselineCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, err := toolArgs(req)
	if err != nil {
		return nil, err
	}
	root, err := resolveServerPath(getString(args, "root"))
	if err != nil {
		return nil, err
	}
	origin, base, err := resolveDiffOrigin(getString(args, "base"), getBool(args, "staged"), getBool(args, "worktree"))
	if err != nil {
		return nil, err
	}
	source, err := git.New(root)
	if err != nil {
		return nil, err
	}
	result, err := application.RunAudit(ctx, application.AuditRequest{
		Root:         root,
		Origin:       origin,
		Base:         base,
		Source:       source,
		Analyzer:     analysis.New(analysis.WithToolVersion(toolVersion)),
		SkipBaseline: true,
	})
	if err != nil {
		return nil, err
	}
	config, err := policies.Load(root)
	if err != nil {
		return nil, err
	}
	target := strings.TrimSpace(getString(args, "out"))
	if target == "" {
		target = config.Audit.Baseline
	}
	path, err := baseline.Resolve(root, target)
	if err != nil {
		return nil, err
	}
	previous, exists, err := readExistingBaseline(path)
	if err != nil {
		if !getBool(args, "force") {
			return nil, err
		}
		previous, exists = baseline.New(), true
	}
	if exists && !getBool(args, "force") {
		return nil, fmt.Errorf("the baseline %s already exists: it records %d finding(s) and it would be replaced. Use force to replace it", path, len(previous.Findings))
	}
	entries := baseline.Entries(result.Run.Findings, result.Changes)
	file := &baseline.File{
		SchemaVersion: baseline.SchemaVersion,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		ToolVersion:   toolVersion,
		Findings:      entries,
	}
	if err := baseline.Save(path, file); err != nil {
		return nil, err
	}
	newFindings := 0
	for _, entry := range entries {
		if !previous.Contains(entry.Fingerprint) {
			newFindings++
		}
	}
	document := map[string]any{
		"schema_version": domain.SchemaVersion,
		"tool_version":   toolVersion,
		"path":           path,
		"findings":       len(entries),
		"new_findings":   newFindings,
		"created":        true,
		"message":        fmt.Sprintf("%s created: %d finding(s) recorded, %d new", path, len(entries), newFindings),
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	return jsonResult(string(encoded), false), nil
}

// includeRules reads the optional include_rules argument.
//
// The catalog of rules is part of the audit document and it is included by
// default, so a caller that reads the document as before sees no change. The
// catalog is static for a build: a caller that does not need it asks for the
// compact document and keeps the context of the conversation small.
func includeRules(args map[string]any) (bool, error) {
	value, ok := args["include_rules"]
	if !ok {
		return true, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("include_rules must be a boolean")
	}
	return flag, nil
}

// toolArgs reads the arguments of a tool call.
func toolArgs(req mcp.CallToolRequest) (map[string]any, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid arguments")
	}
	return args, nil
}

// resolveServerPath resolves an absolute path of the analyzed root.
func resolveServerPath(value string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return "", fmt.Errorf("root is required")
	}
	if !filepath.IsAbs(path) {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = absolute
	}
	return filepath.Clean(path), nil
}

// resolveDiffOrigin selects the origin of a diff based audit.
//
// Exactly one of base, staged or worktree has to be selected: the audit of a
// repository without a selection is the audit of the changes that are not
// committed yet, and the CLI does not invent an origin for the caller.
func resolveDiffOrigin(base string, staged, worktree bool) (domain.ChangeOrigin, string, error) {
	selected := 0
	if strings.TrimSpace(base) != "" {
		selected++
	}
	if staged {
		selected++
	}
	if worktree {
		selected++
	}
	switch {
	case selected == 0:
		return "", "", fmt.Errorf("no origin selected: use one of base, staged or worktree")
	case selected > 1:
		return "", "", fmt.Errorf("several origins selected: use only one of base, staged or worktree")
	}
	if strings.TrimSpace(base) != "" {
		return domain.OriginBase, base, nil
	}
	if staged {
		return domain.OriginStaged, "", nil
	}
	return domain.OriginWorktree, "", nil
}

// readExistingBaseline reads the baseline that is already at a path.
//
// A missing file is not an error: the repository has no baseline yet. A
// malformed file is an error whose message names the problem.
func readExistingBaseline(path string) (*baseline.File, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return baseline.New(), false, nil
		}
		return nil, false, fmt.Errorf("cannot read the baseline %s: %w", path, err)
	}
	file, err := baseline.Load(path)
	if err != nil {
		return nil, true, err
	}
	return file, true, nil
}

// runFailed reports whether the findings of a run reach the configured
// failure threshold.
func runFailed(findings []domain.Finding, failOn string) bool {
	threshold := severityRank(domain.SeverityError)
	switch strings.ToLower(strings.TrimSpace(failOn)) {
	case policies.FailOnWarning:
		threshold = severityRank(domain.SeverityWarning)
	case policies.FailOnNone:
		threshold = 0
	}
	if threshold <= 0 {
		return false
	}
	for _, finding := range findings {
		if severityRank(finding.Severity) >= threshold {
			return true
		}
	}
	return false
}

// severityRank orders the severities.
func severityRank(severity domain.Severity) int {
	switch severity {
	case domain.SeverityInfo:
		return 1
	case domain.SeverityWarning:
		return 2
	case domain.SeverityError:
		return 3
	default:
		return 0
	}
}

// jsonResult wraps the JSON text of a document in a tool result.
//
// When the run reached the failure threshold the result is reported as failed:
// the document travels with the content and the caller reads the findings, the
// failure is also carried by the MCP result.
func jsonResult(text string, failed bool) *mcp.CallToolResult {
	result := mcp.NewToolResultText(text)
	result.IsError = failed
	return result
}

func getString(args map[string]any, key string) string {
	if value, ok := args[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return ""
}

func getBool(args map[string]any, key string) bool {
	if value, ok := args[key]; ok {
		if flag, ok := value.(bool); ok {
			return flag
		}
	}
	return false
}
