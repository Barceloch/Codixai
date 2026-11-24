# MCP Integration (Phase 4)

## Overview

Codixai exposes a local MCP server using `github.com/mark3labs/mcp-go`. The server reuses the existing application layer (`internal/application/audit.go`, `internal/application/verify.go`) without duplicating rule or domain logic.

The server registers five tools: four read-only (`audit_diff`, `audit_directory`, `verify_claims`, `report`) and one that writes a baseline file (`baseline_create`). Every document is produced by the same use case as the corresponding CLI command, with the tool version `mcp` stamped on it.

## Build

```bash
go build ./cmd/codixai-mcp-srv
```

## Transports

- **stdio (default)**: suitable for local agents (OpenCode, Claude, etc.)
- **Streamable HTTP**: `./codixai-mcp-srv -http :8080` for local HTTP-based integration

## Tools

The three tools that return an audit document accept an optional boolean
`include_rules`. It defaults to `true`, so a caller that reads the document as
before sees no change. The catalog of rules is static for a build: a caller
that does not need it sets `include_rules` to `false` and receives the same
document without the `rules` array, which keeps the context of the
conversation small. The findings, the summary, the limitations, the evidence
and the directory statistics are never omitted.

All tools take `root` as a required string. `root` is trimmed, converted to an absolute path when relative (relative paths are resolved against the working directory of the server process, not against another argument), and cleaned. The tools reject a missing `root`.

### `audit_diff`

Run an audit over repository changes.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the repository root"},
  "base": {"type": "string", "required": false, "description": "Base revision for base comparison"},
  "staged": {"type": "boolean", "required": false, "description": "Audit staged changes"},
  "worktree": {"type": "boolean", "required": false, "description": "Audit changes that are not committed yet"},
  "include_rules": {"type": "boolean", "required": false, "description": "Include the catalog of rules in the document (default: true)"}
}
```

**Behavior:**
- Exactly one origin must be selected: `base`, `staged` or `worktree`. Zero or several origins are an error; the tool does not default to the working tree.
- Uses `application.RunAudit` with `git.New(root)` as change source and `analysis.New` as analyzer.
- Returns the `domain.AnalysisRun` as JSON text content. `isError` is set when the findings reach the configured `audit.fail_on` threshold; the document is still returned.

**Mode:** read-only.

### `audit_directory`

Audit the current content of a directory without requiring a Git repository.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the directory root"},
  "include_rules": {"type": "boolean", "required": false, "description": "Include the catalog of rules in the document (default: true)"}
}
```

**Behavior:**
- Uses `application.RunDirectoryAudit` with `analysis.NewDirectoryAnalyzer`.
- Rules that need a diff are declared in `missing_rules` and are never counted as passed. Binary files, files over 10 MB, unsupported extensions and unreadable paths are skipped and named in `files.skipped_files`, each one with a `reason` of the catalog (`unsupported_language`, `binary`, `too_large`, `unreadable`, `outside_root`, `not_regular_file`); symbolic links that escape the root are rejected. Every path of the list is relative to the analyzed root, and a diagnostic never embeds an absolute path of the machine that runs the analysis. A directory excluded by `analysis.exclude` is not walked, so it is reported in `files.excluded_paths` instead.
- Returns the `domain.AnalysisRun` as JSON text content. The failure threshold is read from `audit.fail_on` of the repository configuration and is carried by `isError`.

**Mode:** read-only.

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
- `base` and `claims_path` are required and must be non-empty after trimming. A relative `claims_path` is resolved against the working directory of the server process.
- The claim file is validated before the comparison. A claim that declares a field no type of the catalog evaluates is rejected with an error that names the line and the field: `symbols` is such a field, because no claim type resolves a symbol, so a claim that declared it would be answered by a verdict that ignores what it asserts.
- Every claim is decided about the compared range. A test claim whose paths hold no changed file is `contradicted` when the range is not empty, because no test file of those paths was added or modified in the range, and `unresolved` when the range is empty, because there is no evidence at all.
- Uses `application.RunVerify` with `git.New(root)`, `claims.Loader{}`, and `claims.NewEvaluator(claims.WithRoot(root), claims.WithBase(base))`.
- Returns the verification document as JSON text content. `isError` is set when a claim is contradicted, or when an unresolved claim reaches the configured `verify.fail_on_unresolved` threshold; the document is still returned.

**Mode:** read-only.

### `report`

Export the audit of a change set or of a directory as SARIF 2.1.0 or as the audit document.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the repository or directory root"},
  "format": {"type": "string", "required": false, "description": "Output format: sarif or json (default: sarif)"},
  "base": {"type": "string", "required": false, "description": "Base revision for base comparison"},
  "staged": {"type": "boolean", "required": false, "description": "Audit staged changes"},
  "worktree": {"type": "boolean", "required": false, "description": "Audit working tree changes"},
  "directory": {"type": "string", "required": false, "description": "Analyze a directory without requiring a Git repository"},
  "include_rules": {"type": "boolean", "required": false, "description": "Include the catalog of rules in the JSON document (default: true)"}
}
```

**Behavior:**
- Exactly one origin must be selected: `base`, `staged`, `worktree` or `directory`. Zero or several origins are an error.
- `format` is case insensitive; `sarif` is the default and `json` writes the audit document. Any other value is an error.
- `include_rules` applies to the JSON document only. A SARIF document always carries its rule descriptors in `runs[].tool.driver.rules`, because they are part of the SARIF contract.
- With `directory`, the audit is rooted at that path and uses `report.SARIFDirectory` for SARIF. With a diff origin, the audit is rooted at `root` and uses `report.SARIF`.
- Returns the selected document as JSON text content. `isError` follows the failure threshold of the selected audit.

**Mode:** read-only.

### `baseline_create`

Create a baseline of findings for a repository. This is the only tool that writes.

**Input schema:**
```json
{
  "root": {"type": "string", "required": true, "description": "Absolute path to the repository root"},
  "base": {"type": "string", "required": false, "description": "Base revision for base comparison"},
  "staged": {"type": "boolean", "required": false, "description": "Create baseline from staged changes"},
  "worktree": {"type": "boolean", "required": false, "description": "Create baseline from working tree changes"},
  "out": {"type": "string", "required": false, "description": "Path of the baseline file to write (defaults to audit.baseline config)"},
  "force": {"type": "boolean", "required": false, "description": "Replace an existing baseline file"}
}
```

**Behavior:**
- Exactly one diff origin must be selected: `base`, `staged` or `worktree`. Directory analysis is not supported.
- The audit runs with baseline application disabled, so the recorded findings are the findings of the selected range.
- `out` defaults to `audit.baseline` of the repository configuration. The resolved path must stay inside `root`.
- An existing baseline is never replaced silently: the tool returns an error naming the file and the number of findings it records unless `force` is set.
- The baseline file is versioned (`schema_version`, `created_at`, `tool_version`, sorted findings). The tool returns a JSON summary with `schema_version`, `tool_version`, `path`, `findings`, `new_findings`, `created` and `message`. `isError` is not set by this tool.

**Mode:** mutating (writes one baseline file); annotated as non-destructive and not idempotent.

## Architecture constraints

- **Mixed access**: four tools are read-only; `baseline_create` writes exactly one baseline file inside the repository root.
- **No duplication**: reuses `internal/application`, `internal/claims`, `internal/integrations/git`, `internal/analysis` and `internal/report`.
- **Deterministic output**: same input produces the same document, except for `generated_at` and the `created_at` of a baseline.
- **Schema versioning**: documents include `schema_version` as produced by the application layer.
- **Bounded file access**: paths are resolved against the repository root by the underlying use cases; a path that escapes it is rejected.
- **No execution of analyzed code, no network access**: the server invokes only read-only `git` subcommands through the application layer.

## Limits

- Large diffs and claims files follow the same limits as the CLI.
- Errors use two channels. A handler or validation failure is returned as a JSON-RPC error; an audit or verification whose findings reach the failure threshold returns its document with `isError` set. mcp-go input-schema validation is not enabled in this server, so missing or wrongly typed arguments are rejected inside the handlers and currently surface as JSON-RPC `INTERNAL_ERROR` (-32603) rather than `INVALID_PARAMS`.
- Only the five tools listed above are exposed. Reference resolution and reference-aware impact are not available through MCP.
- The catalog of rules is a large, static part of the audit document: measured on the fixtures of the QA, it is 54% of an `audit_diff` response with findings and 80% of an `audit_directory` response without them. `include_rules: false` removes it and keeps every other field.

## References

- Specification: `docs/spec/CODIXAI_SPEC.md` §8.1, §8.5, Phase 4
- ADRs: 0007 (claims), 0013 (agent adapters), existing decisions in `docs/decisions/`
- Implementation: `internal/integrations/mcpserver/`, `cmd/codixai-mcp-srv/`
- Contract tests: `internal/integrations/mcpserver/server_test.go`
- Integration parity: `tests/integration/mcp_parity_test.go`

## Limits and Error Classification (P4-05)

### Supported languages
Codixai supports 5 languages as per fixtures: Go, Python, JavaScript, TypeScript, PHP. Reference resolution is implemented for Go, Python, JavaScript and TypeScript only; PHP has no reference resolver and no comparative fixtures.

### Known limits
| Limit | Status | Notes |
|-------|--------|-------|
| Languages not in fixture corpus | `unresolved` | Claims/rules for unsupported languages are marked `unresolved`, never `verified` |
| Ranges without resolvable evidence | `unresolved` | Cannot be decided → `unresolved`, not guessed |
| Heuristic findings | `inferred` | Never called confirmed bugs; epistemic status preserved |
| External tools (LSP/ecosystem) | Not used | Invariant 1: no execution of analyzed repo, no package manager, no network |
| Network access | Disabled | No telemetry, no code upload (invariant 10) |
| MCP input-schema validation | Not enabled | Handlers validate; structural errors surface as JSON-RPC `INTERNAL_ERROR` (-32603) |
| Reference resolution / impact via MCP | Not exposed | No MCP tool for either capability |

### Error classification
| Error type | Category | Behavior |
|------------|----------|----------|
| Missing required args (root/base/claims_path) | Structural | Rejected in the handler; surfaced as a JSON-RPC error (currently -32603) |
| Wrongly typed args | Structural | Rejected in the handler; surfaced as a JSON-RPC error (currently -32603) |
| No origin / several origins selected | Structural | JSON-RPC error from the origin check of `audit_diff`, `report` or `baseline_create` |
| Invalid format in `report` | Structural | JSON-RPC error naming the accepted formats |
| Invalid git repo / missing base ref | Structural | Returned as an error from `git.New` / application layer |
| Missing claims file / invalid claims JSON | Structural | Returned as an error from `claims.Loader` |
| Existing baseline without `force` | Structural | Error naming the file and its recorded finding count |
| Undecidable claim / insufficient evidence | Semantic/Unresolved | Returned in document as claim state `unresolved` (not an error) |
| Contradicted claim | Semantic | Returned in document as claim state `contradicted`; may set `isError` |
| Findings at the `fail_on` threshold | Semantic | Document returned with `isError` set; the document is not withheld |
| Rule internal error | Semantic | Becomes diagnostic on run, analysis continues (per invariants) |

### Size expectations
| Artifact | Expected | Notes |
|----------|----------|-------|
| MCP server binary | ~10-15 MB | Similar to CLI, no extra deps |
| Tool response (audit) | Varies by diff size | JSON document with findings |
| Tool response (verify) | Small (per claims) | Claims + results |
| Tool response (baseline_create) | Small | Summary with path and finding counts |
