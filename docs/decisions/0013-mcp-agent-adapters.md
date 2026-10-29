# ADR 0013: MCP agent adapters

- Status: accepted
- Date: 2026-10-24

## Context

Codixai needs to integrate with agents via MCP (Phase 4). The specification requires that MCP and adapters do not duplicate rules or domain logic, and at least one integration must invoke the same verification engine as the CLI.

## Decision

Codixai will provide a **local read-only MCP server** as a first-class integration (`cmd/codixai-mcp-srv`, package `internal/integrations/mcpserver`) rather than agent-specific adapters. The server exposes tools that delegate to the application layer:

- `audit_diff`: calls `application.RunAudit` with `git.New` as the change source
- `verify_claims`: calls `application.RunVerify` with `git.New`, `claims.Loader`, and `claims.NewEvaluator`

Rationale:
- **Single engine**: same use cases as CLI/UI, no rule duplication
- **Read-only by construction**: tools only read and return documents; no mutations
- **Transport agnostic**: supports stdio and streamable HTTP for local use
- **Minimal surface**: no LSP/remote execution, respects "no telemetry, no code upload" (invariant 10)
- **Agent agnostic**: any MCP client (OpenCode, Claude, Cursor, etc.) can use it without Codixai-specific adapter code

## Consequences

- Adding agent-specific adapters is not necessary if the MCP server meets requirements
- Future capabilities (index/search) from Phases 5-6 should also be exposed via MCP tools that call application services
- The server must remain read-only and must not import CLI/presentation layers
- Packaging: MCP server can be built as a separate binary or integrated; for now separate binary is simple and avoids expanding default build surface

## Compliance

- Domain imports nothing (invariant 2) — MCP server imports application/integrations/domain only
- Application does not depend on presentation (invariant 3) — respected
- No execution of analyzed repository code (invariant 1)
- Results are versioned documents from domain/application layer
