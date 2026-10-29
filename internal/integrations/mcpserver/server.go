// Package mcpserver provides a local read-only MCP server using mcp-go.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// New builds a Codixai MCP server exposing read-only audit and verify tools.
func New(version string) *server.MCPServer {
	s := server.NewMCPServer("codixai", version, server.WithToolCapabilities(true))

	s.AddTool(mcp.NewTool("audit_diff",
		mcp.WithDescription("Run a Codixai audit over repository changes and return the analysis document."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository root")),
		mcp.WithString("base", mcp.Description("Base revision for base comparison")),
		mcp.WithBoolean("staged", mcp.Description("Audit staged changes")),
	), handleAuditDiff)

	s.AddTool(mcp.NewTool("verify_claims",
		mcp.WithDescription("Verify agent claims against the repository and return the verification document."),
		mcp.WithString("root", mcp.Required(), mcp.Description("Absolute path to the repository root")),
		mcp.WithString("base", mcp.Required(), mcp.Description("Base revision to compare against")),
		mcp.WithString("claims_path", mcp.Required(), mcp.Description("Path to the claims JSON file")),
	), handleVerifyClaims)

	return s
}

func handleAuditDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid arguments")
	}

	root := strings.TrimSpace(getString(args, "root"))
	if root == "" {
		return nil, fmt.Errorf("root is required")
	}

	staged := getBool(args, "staged")
	base := strings.TrimSpace(getString(args, "base"))

	source, err := git.New(root)
	if err != nil {
		return nil, err
	}

	origin := domain.OriginWorktree
	if staged {
		origin = domain.OriginStaged
	} else if base != "" {
		origin = domain.OriginBase
	}

	result, err := application.RunAudit(ctx, application.AuditRequest{
		Root:   root,
		Origin: origin,
		Base:   base,
		Source: source,
	})
	if err != nil {
		return nil, err
	}

	document, err := json.Marshal(result.Run)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(document)), nil
}

func handleVerifyClaims(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid arguments")
	}

	root := strings.TrimSpace(getString(args, "root"))
	if root == "" {
		return nil, fmt.Errorf("root is required")
	}
	base := strings.TrimSpace(getString(args, "base"))
	if base == "" {
		return nil, fmt.Errorf("base is required")
	}
	claimsPath := strings.TrimSpace(getString(args, "claims_path"))
	if claimsPath == "" {
		return nil, fmt.Errorf("claims_path is required")
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
		ToolVersion: "mcp",
	})
	if err != nil {
		return nil, err
	}

	document, err := json.Marshal(result.Document)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(document)), nil
}

func getString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(args map[string]any, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
