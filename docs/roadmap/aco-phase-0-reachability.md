# ACO Phase 0 — Reachability Audit

- Date: 2026-11-14
- Status: Phase 0 reachability audit (read-only; no implementation change to reachability)
- Repository state: branch `phase-5-2-5-2-reference-consistency`, HEAD `a7f5096828ee599ecee273f5b60b6782bc75adfc`

## 1. Method

Reachability was determined by reading the command registration (`internal/cli/commands.go`), the MCP tool registration (`internal/integrations/mcpserver/server.go`), the application use cases (`internal/application/`), and grepping for callers of each capability. A capability is "reachable" only when a non-test caller exists through CLI or MCP.

## 2. Reachability matrix

| Capability | Entry point | CLI | MCP | Uses persisted index | Status |
|---|---|---|---|---|---|
| Diff audit | `codixai audit` / `audit_diff` | yes | yes | no | IMPLEMENTED / REACHABLE |
| Directory audit | `codixai audit --directory` / `audit_directory` | yes | yes | no | IMPLEMENTED / REACHABLE |
| Report (SARIF/JSON) | `codixai report` / `report` | yes | yes | no | IMPLEMENTED / REACHABLE |
| Baseline create | `codixai baseline create` / `baseline_create` | yes | yes | no | IMPLEMENTED / REACHABLE (mutating) |
| Verify claims | `codixai verify` / `verify_claims` | yes | yes | no | IMPLEMENTED / REACHABLE |
| Index build | `codixai index build` | yes | no | writes | IMPLEMENTED / REACHABLE (CLI only) |
| Index status/overview/find/refs/deps | `codixai index *` | yes | no | reads | IMPLEMENTED / REACHABLE (CLI only) |
| Index diff (blobs) | `codixai index diff` | yes | no | no | IMPLEMENTED / REACHABLE (CLI only) |
| Impact analysis | `codixai impact` | yes | no | no (passes nil) | IMPLEMENTED / PARTIALLY REACHABLE |
| Reference resolution | `application.RunResolveReferences` | no | no | writes refs (unread) | IMPLEMENTED / NOT REACHABLE |
| Persisted references (load/list) | `storage.LoadReferences` / `ListReferences` | no | no | reads | IMPLEMENTED / NOT REACHABLE |
| UI | `codixai ui` | yes | no | no | IMPLEMENTED / REACHABLE (CLI only) |
| Init | `codixai init` | yes | no | no | IMPLEMENTED / REACHABLE (CLI only) |
| Measurement harness | `codixai measure *` | yes | no | no | IMPLEMENTED / REACHABLE (this phase) |

## 3. Disconnected components and dead paths

1. **`application.RunResolveReferences`** (`internal/application/index.go:313`) has no non-test caller. It builds an in-memory index, runs the four resolvers, normalizes, and persists via `store.Save` + `store.SaveReferences`. Nothing invokes it in production. References are written and never read.
2. **`storage.IndexStore.LoadReferences`** (`:780`) and **`ListReferences`** (`:899`) have no non-test callers. `ListReferences` delegates to `LoadReferences`. The persisted `index_references` table is write-only in production.
3. **`impact.Analyzer`** (`internal/impact/impact.go`) holds an `*index.Index` field that is never read; `AnalyzeImpact` produces findings from `FileChange.Status` and path shape only. `runImpact` (`internal/cli/commands.go:593`) passes `impact.NewAnalyzer(nil)`. `AnalyzeImpactFromDiff` (`:154`) builds an index that is still unused. Impact is therefore a status-based heuristic with no caller/callee information.
4. **`storage.migrateToV2`** (`:178`) has no caller; the v1->v2 migration path is untested.
5. **`storage.Save`** writes `hash=""` and `size=0` placeholders (`:331-333`), so content-based freshness is not currently possible.
6. **`application.RunIndexDiff`** (`internal/application/index.go:282`) returns "not implemented".
7. **`index.ModuleResolver` and `index.Resolver`** are reachable only through `RunResolveReferences`, hence transitively unreachable.

## 4. Data and reuse summary

- Index build and read paths (`application.RunIndex`, `RunIndexStatus`, `RunIndexFind`, `RunIndexRefs`, `RunIndexDeps`) are reachable from the CLI and reuse the persisted SQLite store. They do not rebuild unless forced.
- Reference resolution and persisted references are implemented but unreachable; reuse requires a thin CLI/MCP wrapper (not implemented in this phase).
- Impact is reachable from the CLI but does not consume the index or references; making it reference-aware requires the integration described in ADR 0020 (deferred).
- The measurement harness added in this phase is reachable from the CLI (`codixai measure *`).

## 5. Evidence

- Caller search: `grep -rn "RunResolveReferences"` (only definition); `grep -rn "LoadReferences\|ListReferences"` (only definitions and the internal delegation); `grep -rn "migrateToV2"` (only definition).
- `internal/cli/commands.go:593` passes `impact.NewAnalyzer(nil)`.
- `internal/impact/impact.go:35-37,45-98` never dereferences `a.index`.
- `internal/storage/index_store.go:331-333` writes empty hash/size.

## 6. Conclusion

Phase 0 confirms that Codixai's structural index and audit/verify/report capabilities are reachable, while reference resolution, persisted references and reference-aware impact are implemented but not reachable. No architectural redesign was performed. Making reference resolution reachable (a `refs` wrapper and an MCP `references` tool) and making impact reference-aware are documented dependencies for later work units, not part of this phase.
