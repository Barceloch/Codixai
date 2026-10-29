# MCP Integration (Phase 4)

## Overview

Codixai exposes a local read-only MCP server using `github.com/mark3labs/mcp-go`. The server reuses the existing application layer (`internal/application/audit.go`, `internal/application/verify.go`) without duplicating rule or domain logic.

## Build

```bash
go build ./cmd/codixai-mcp-srv
```

## Transports

- **stdio (default)**: suitable for local agents (OpenCode, Claude, etc.)
- **Streamable HTTP**: `./codixai-mcp-srv -http :8080` for local HTTP-based integration

## Tools

### `audit_diff`

Run an audit over repository changes.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the repository root"},
  "base": {"type": "string", "required": false, "description": "Base revision for base comparison"},
  "staged": {"type": "boolean", "required": false, "description": "Audit staged changes"}
}
```

**Behavior:**
- Uses `application.RunAudit` with `git.New(root)` as change source
- Origin resolved as `staged` → `domain.OriginStaged`, `base` set → `domain.OriginBase`, else `domain.OriginWorktree`
- Returns the `domain.AnalysisRun` as JSON text content

### `verify_claims`

Verify agent claims against the repository.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the repository root"},
  "base": {"type": "string", "required": true, "description": "Base revision to compare against"},
  "claims_path": {"type": "string", "required": true, "description": "Path to the claims JSON file"}
}
```

**Behavior:**
- Uses `application.RunVerify` with `git.New(root)`, `claims.Loader{}`, and `claims.NewEvaluator(...)`
- Returns the verification document as JSON text content

## Architecture constraints

- **Read-only**: no repository mutations, no test execution, no network access
- **No duplication**: reuses `internal/application`, `internal/claims`, `internal/integrations/git`
- **Deterministic output**: same input produces same JSON document (except runtime metadata)
- **Schema versioning**: documents include `schema_version` as produced by the application layer
- **Bounded file access**: paths resolved against repository root by underlying use cases

## Limits

- Large diffs/claims files follow the same limits as the CLI
- Errors are returned as tool errors with structured messages
- Only read operations are exposed

## References

- Specification: `docs/spec/CODIXAI_SPEC.md` §8.1, §8.5, Phase 4
- ADRs: 0007 (claims), existing decisions in `docs/decisions/`
- Implementation: `internal/integrations/mcpserver/`, `cmd/codixai-mcp-srv/`

## Limits and Error Classification (P4-05)

### Supported languages
Codixai supports 5 languages as per fixtures: Go, Python, JavaScript, TypeScript, PHP.

### Known limits
| Limit | Status | Notes |
|-------|--------|-------|
| Languages not in fixture corpus | `unresolved` | Claims/rules for unsupported languages are marked `unresolved`, never `verified` |
| Ranges without resolvable evidence | `unresolved` | Cannot be decided → `unresolved`, not guessed |
| Heuristic findings | `inferred` | Never called confirmed bugs; epistemic status preserved |
| External tools (LSP/ecosystem) | Not used | Invariant 1: no execution of analyzed repo, no package manager, no network |
| Network access | Disabled | No telemetry, no code upload (invariant 10) |

### Error classification
| Error type | Category | Behavior |
|------------|----------|----------|
| Missing required args (root/base/claims_path) | Structural | Tool returns error immediately |
| Invalid git repo / missing base ref | Structural | Returned as error from `git.New` / application layer |
| Missing claims file / invalid claims JSON | Structural | Returned as error from `claims.Loader` |
| Undecidable claim / insufficient evidence | Semantic/Unresolved | Returned in document as claim state `unresolved` (not an error) |
| Contradicted claim | Semantic | Returned in document as claim state `contradicted` |
| Rule internal error | Semantic | Becomes diagnostic on run, analysis continues (per invariants) |

### Size expectations
| Artifact | Expected | Notes |
|----------|----------|-------|
| MCP server binary | ~10-15 MB | Similar to CLI, no extra deps |
| Tool response (audit) | Varies by diff size | JSON document with findings |
| Tool response (verify) | Small (per claims) | Claims + results |
