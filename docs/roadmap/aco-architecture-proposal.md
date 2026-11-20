# ACO — Agent Context Optimization: Architecture Proposal

- Date: 2026-11-14
- Status: **proposal — NOT IMPLEMENTED. Architecture decisions 0019–0021 approved 2026-11-14. ACO Phase 0 reachability audit and measurement harness implemented 2026-11-14; Phase 0 reachability (refs wrapper / MCP references tool) and Phase 1+ remain unimplemented.**
- Repository state: branch `phase-5-2-5-2-reference-consistency`, HEAD `a7f5096828ee599ecee273f5b60b6782bc75adfc`
- Related: [`phase-5-2-6-closure-report.md`](phase-5-2-6-closure-report.md), ADR [0019](../decisions/0019-context-package-as-application-read-model.md), [0020](../decisions/0020-reference-aware-impact-analysis.md), [0021](../decisions/0021-project-knowledge-reuse.md)

## Recommendation in one page

Codixai already owns the hard part of ACO: a deterministic structural index, four reference resolvers with auditable evidence, a versioned SQLite index, and a layering rule that lets CLI, MCP and UI share one application core. ACO does not need a new engine. It needs **three things the current code does not do**:

1. **Reachability** — reference resolution exists but no CLI command or MCP tool can call it (`RunResolveReferences` has no non-test caller).
2. **A read model** — nothing reads persisted references back (`LoadReferences` has no caller), and impact analysis consumes no index at all (`impact.NewAnalyzer(nil)` at its only call site).
3. **A selection layer** — there is no way to ask "what does *this task* need?" and no size budget anywhere in the codebase.

The proposal therefore reuses everything and adds one application use case plus one serializable document:

| Question | Recommendation |
| --- | --- |
| Core domain knows about ACO? | **No.** `domain.Reference`/`ReferenceEvidence`/`ResolutionStatus` are already the right vocabulary. |
| Context package type | **Application read model**, versioned document, not a domain invariant (ADR 0019). |
| Orchestration location | `internal/application/context` use cases, called by CLI, MCP and UI. |
| Persistence | Reuse the SQLite index; add a freshness check before reuse (ADR 0021). |
| Impact | Make it consume index + references with categorical confidence, not scores (ADR 0020). |
| Token measurement | Instrumented from ACO Phase 0, never retrofitted; agent-reported usage as primary, byte proxy as secondary. |
| New dependencies | **None.** |
| Remote services / vectors / microservices | **None.** Local-first is preserved. |

## 1. Problem

An AI coding agent working on a Codixai task today must explore the repository itself: read files, grep, infer call graphs, guess which tests matter. Codixai can already compute most of that deterministically, but it exposes none of it in a form an agent can consume cheaply.

The cost is not only tokens. It is **unverified context**: an agent that reconstructs the call graph from grep will confidently miss a caller that a resolver would have classified `ambiguous`.

ACO's goal is therefore not "shorter MCP responses". It is: **Codixai becomes the structural knowledge layer between the repository and the agent**, and the agent receives a small, evidence-backed, task-specific package with an explicit uncertainty section.

## 2. Evidence inventory

Everything below was verified against source on 2026-11-14. Items marked *absent* were confirmed by grep, not assumed.

### 2.1 Components that exist and can be reused

| Component | Responsibility | Evidence |
| --- | --- | --- |
| `internal/domain` | `Reference`, `SymbolRef`, `ReferenceEvidence`, `ReferenceKind`, `ResolutionStatus` | `domain/references.go:5-115` |
| `internal/index` | Deterministic walk + parse + symbol/dependency extraction | `index/index.go:81-140` |
| `index.Resolver` | One interface, four implementations | `index/resolver.go:11-16` |
| `index.ModuleResolver` | Module graph for JS/TS | `index/module_resolution.go:68-70` |
| `index/evidence.go` | Auditable evidence constructors | `index/evidence.go:12-129` |
| `index.NormalizeReferences` | Deterministic ordering | `index/reference_normalize.go:14` |
| `internal/storage` | SQLite index, schema v2, including `index_references` | `storage/index_store.go:19,48-53` |
| `internal/application` | Use cases, ports, layering guard | `application/ports.go:15-31`, `architecture_test.go:21-26` |
| `internal/report` | SARIF 2.1.0 and JSON documents | `report/sarif.go:41,63` |
| `internal/integrations/mcpserver` | 5 read-only tools, thin over application | `mcpserver/server.go:36,48,57,68,82` |
| `internal/cli` | Cobra commands including `index *` and `impact` | `cli/index_cmd.go:20-98`, `cli/commands.go:534` |

### 2.2 What is missing (verified absent)

| Missing | Verification |
| --- | --- |
| CLI/MCP access to reference resolution | `RunResolveReferences` has zero non-test callers |
| Reading persisted references | `LoadReferences`/`ListReferences` have zero non-test callers |
| Impact consuming the index | `AnalyzeImpact` never dereferences `a.index`; call site passes `nil` (`cli/commands.go:593`) |
| Application-layer impact use case | no file in `internal/application` mentions impact |
| Any size/token budget or pagination | only `LIMIT 1000` in `storage/index_store.go:543` |
| Context package type or document | zero hits for `ContextPackage` in Go and Markdown |
| Index freshness / invalidation | `Save` deletes all rows for the root and reinserts (`storage/index_store.go:288-290`); the only gate is `FileCount == 0` (`application/index.go:86`) |
| Ports for index, storage, impact | `ports.go` + `verify.go` declare only 5 ports |

### 2.3 Documentation drift that constrains the proposal

- ADR 0018 announces `impact`, `api_change`, `architecture` and `claims` consuming resolved references (`docs/decisions/0018-cross-language-references.md:156-160`). None exists.
- ADR 0018 announces PHP reference extraction and a `ReferenceExtractor` interface (`:22,27,153`). Neither exists.
- `docs/tasks/phase-5.md` lists P5-05/P5-06 as pending "requires network" (`:202,210`) although `internal/impact` and `codixai impact` exist.

ACO must not be designed on top of these claims. It is designed on the code.

Note: the ADR 0018 documentation drift was corrected on 2026-11-14 (see `docs/decisions/0018-cross-language-references.md`, implementation-status clarification). This proposal is designed on the code, not on the old claims.

## 3. Concern A — impact analysis and resolved references

**Status: confirmed, and stronger than reported.**

The impact analyzer is a status-based heuristic:

- `AnalyzeImpact` iterates changed files and classifies them by `FileChange.Status` (`impact/impact.go:45-113`).
- `findRelatedTests` derives candidate tests from path shape and never stats the file (`:115-140`).
- The analyzer holds an `*index.Index` field that is never read (`:35-37` vs `:45-98`).
- Its only production call site passes `nil` (`cli/commands.go:593`).
- `domain.Reference` is never referenced anywhere in `internal/impact`.

Consequence for ACO: **impact cannot currently contribute caller/callee context**, and any context package built today would inherit a heuristic that emits test paths that may not exist.

Design position: impact becomes reference-aware in two increments (ADR 0020):

1. **Index-aware** — resolve symbols from the persisted index instead of path shape.
2. **Reference-aware** — follow incoming references, and propagate the resolver's categorical state:
   - `resolved` → include as a confirmed dependency.
   - `ambiguous` → include with all candidates and an explicit uncertainty entry.
   - `dynamic` → include as a *possible* dependency, flagged, never as confirmed.
   - `unresolved` / `external` → include as a boundary marker, never as a local dependency.

No numeric confidence is introduced. The domain already expresses uncertainty categorically and the conventions forbid inventing precision.

## 4. Concern B — index reconstruction

**Status: confirmed for the resolution path; the rest of the product does not rebuild.**

| Entry point | Builds index? | Evidence |
| --- | --- | --- |
| `codixai index build` | Only when `--force` or the DB has no files | `application/index.go:80-94` |
| `index status/overview/find/refs/deps` | No — SQLite reads | `application/index.go:155-256` |
| `codixai impact` | No (and uses no index) | `cli/commands.go:574-596` |
| MCP (5 tools) | No | `mcpserver/server.go` |
| `RunResolveReferences` | **Always** — ignores any persisted index | `application/index.go:325-326` |

Additional facts:

- `Save` is delete-all-then-reinsert per root (`storage/index_store.go:288-290`); there is no incremental path.
- `Save` writes `hash=""` and `size=0` placeholders (`:331-333`), so content-based freshness is not possible without a schema change or a recomputation.
- References are persisted (`index_references`) and never read.
- `migrateToV2` has no caller (dead code), so schema v1→v2 migration is not exercised.

Consequence for ACO: a persistent knowledge model **already exists** (schema v2). ACO does not need new storage to start. It needs:

1. A read path (`LoadReferences`) wired into a use case.
2. A freshness contract before reuse.
3. An explicit decision that "rebuild" remains the correct default until measurement proves otherwise.

Incremental indexing is **not** an ACO Phase 1 requirement. It becomes justified only when a measured task shows rebuild cost dominating.

## 5. Requirements → design

| Requirement | Design response |
| --- | --- |
| 4.1 Task-specific context | A `ContextRequest` selects a task kind + target (symbol, file, diff, finding, claim). Selection is deterministic. |
| 4.2 Progressive disclosure | Four levels over one selection; level is a request parameter, not a separate pipeline. |
| 4.3 Human control | Building a package is read-only. Handoff and any future write-capable operation require explicit approval recorded in the document. |
| 4.4 Deterministic vs reasoning | Codixai owns collection, selection, evidence and validation. It never chooses designs or resolves ambiguous requirements. |
| 4.5 Token savings | Instrumented harness from Phase 0; savings reported only with completeness checks alongside. |

### 5.1 Task taxonomy (Phase 0 deliverable, initial set)

| Task kind | Target | Primary selection |
| --- | --- | --- |
| `investigate_symbol` | symbol name + file | symbol, incoming/outgoing references, tests |
| `find_callers` | symbol | incoming references only |
| `analyze_change` | base ref or worktree | changed symbols, their references, affected tests |
| `investigate_finding` | rule id + location | finding, rule support, related symbols |
| `review_diff` | base ref | changed files, impacted symbols, claim risks |

Each task kind maps to one deterministic selection function. No free-form retrieval, no embeddings.

### 5.2 Context package model

A single versioned document, produced by the application layer:

```json
{
  "schema_version": "1",
  "package_id": "sha256:…",
  "generated_at": "2026-11-14T00:00:00Z",
  "task": { "kind": "investigate_symbol", "target": "ResolveReferences" },
  "level": 2,
  "repository": { "index_schema": 2, "head": "a7f5096", "fresh": true },
  "summary": {
    "symbols": 3, "references": 12, "tests": 2,
    "uncertainties": 1, "boundaries": 2
  },
  "symbols": [
    { "name": "…", "kind": "func", "file": "internal/index/resolver.go", "line": 11 }
  ],
  "references": [
    {
      "source": { "name": "…", "file": "…", "line": 42 },
      "target": { "name": "…", "file": "…", "line": 11 },
      "kind": "call",
      "resolution": "resolved",
      "evidence": [{ "kind": "ast_call", "ref": "…:42", "method": "go_ast" }]
    }
  ],
  "uncertainty": [
    { "kind": "ambiguous", "subject": "…", "candidates": ["…", "…"], "reason": "…" }
  ],
  "budget": { "estimated_tokens": 1180, "truncated": false, "omitted": [] }
}
```

### 5.2.1 Context package document versioning (Decision 3)

The context package carries its own `schema_version`, independent of the global project version, the database/schema version, the findings/report schema (`domain.SchemaVersion`), and MCP protocol versioning; version independence; additive backward-compatible evolution (new optional fields without a bump); consumers MUST tolerate unknown fields and treat absent optional fields as unspecified; a bump is required for removal/rename/meaning/type/structure change. Link to ADR 0019.

Rules that make it honest:

- `resolution` is copied from the resolver; ACO never upgrades a state.
- Every reference keeps its evidence; a reference without evidence is a defect.
- `uncertainty` is mandatory when any selected reference is `ambiguous`, `dynamic` or `unresolved`.
- `budget.truncated` is explicit, and `omitted` names what was cut. Silent truncation is forbidden.
- Ordering is deterministic (path, line, name) so the same repository produces the same bytes.

### 5.3 Progressive disclosure contract

| Level | Contains | Default budget | Never contains |
| --- | --- | --- | --- |
| 1 — Summary | task, counts, main components, confidence summary | ≤ 400 tokens | file contents |
| 2 — Structural | symbols, references, tests, impacted files | ≤ 2 000 tokens | excerpts |
| 3 — Evidence | locations, short excerpts, evidence entries, reasons | ≤ 6 000 tokens | full files |
| 4 — Deep | full relevant files/sections, extended chains | unbounded, explicit request | unrelated files |

Levels are cumulative and deterministic. An agent requests `level: 4` explicitly; nothing defaults to 4.

### 5.4 Human approval workflow

```
human selects task + target
  → codixai context build --task … --level 2      (read-only, no approval needed)
  → package shown as a document (JSON) or a human summary (text)
  → human reviews uncertainty and omissions
  → human approves handoff                          (explicit flag / explicit MCP call)
  → agent receives the approved package
  → human decides whether to accept any change
```

Codixai never modifies code in ACO. Approval gates the handoff, not the analysis, because the analysis is read-only and deterministic. If a later phase adds a write-capable operation, it requires its own approval gate and its own ADR.

### 5.5 Token measurement strategy

Definitions:

- `T_b` — tokens consumed by the baseline workflow: the agent explores the repository without Codixai.
- `T_c` — tokens consumed with the Codixai package, **including** the package itself and any follow-up exploration.
- `Savings = (1 − T_c / T_b) × 100`.

Measurement protocol (no fabricated numbers; all values must come from a recorded run):

1. Fix a task set of 5 representative tasks from Codixai's own repository (one per task kind).
2. Record for each task: agent-reported input/output tokens, number of tool calls, wall-clock time.
3. Run each task twice: baseline (no package) and with a package, same model and same prompt.
4. Record **completeness** for each run: did the agent identify every reference that the resolver knows about? A package that omits a critical dependency is a failure regardless of savings.
5. Report per task and in aggregate: `T_b`, `T_c`, savings, completeness, time-to-package, reuse count.

Token counting rules:

- Primary source: the agent runtime's reported usage. It is the only number that reflects the real cost.
- Secondary source: a deterministic byte proxy (`bytes / 4`) used only for package sizing and budgets, never reported as a savings result.
- No new dependency is introduced for tokenization. If exact counting becomes a requirement, it is a separate ADR.

Guardrail: **savings are never reported without completeness.** A 70% saving that drops a caller is a regression.

## 6. Architecture

### 6.1 Layering

```
cli ─┐
mcp ─┼─→ internal/application/context ─→ internal/index ─→ internal/languages/*
ui  ─┘            │                     └→ internal/storage
                  ├→ internal/impact
                  └→ internal/domain (read-only vocabulary)
```

The application layer keeps declaring ports where a provider could vary. ACO's first iteration reads the index and storage directly, exactly as `RunResolveReferences` does today; introducing ports for them is a separate decision, not a prerequisite.

### 6.2 New components (minimum viable)

| Component | Purpose | Why it cannot be avoided |
| --- | --- | --- |
| `internal/application/context.go` | Use cases: build, inspect, approve | CLI/MCP/UI must share one implementation |
| `ContextPackage` + `ContextRequest` types | Versioned read model | No existing type can express it |
| Selection functions per task kind | Deterministic selection | The core of "task-specific" |
| Budget estimator | Level budgets, truncation reporting | Requirement 4.5 |
| Index freshness check | Reuse safety | Reuse without freshness is a correctness bug |

Explicitly **not** created: a new index engine, a new storage schema for ACO, a vector store, an agent framework, a prompt library, a retrieval service.

### 6.3 Where each concern lives

| Concern | Location |
| --- | --- |
| Symbol/reference data | `internal/domain` (unchanged) |
| Index and resolution | `internal/index` (unchanged) |
| Persistence | `internal/storage` (read path added) |
| Selection, packaging, budgeting | `internal/application/context` |
| Impact | `internal/impact` (reference-aware) |
| Transport | `internal/cli`, `internal/integrations/mcpserver`, `internal/ui` |
| Serialization | application document type, JSON |

### 6.4 MCP surface (proposed, additive)

| Tool | Input | Output |
| --- | --- | --- |
| `context_build` | `root`, `task`, `target`, `level` | context package document |
| `context_inspect` | `package_id` or inline package | summary + uncertainty only |
| `references` | `root`, `symbol`, `file` | references with evidence |
| `impact` | `root`, `base` | reference-aware impact result |

All remain read-only. Existing five tools keep their contracts.

### 6.5 CLI surface (proposed, additive)

```
codixai context build --task investigate_symbol --symbol X --level 2 --root .
codixai context build --task analyze_change --base main --level 1
codixai context inspect --package <file|->
codixai refs --symbol X            # thin wrapper over resolution
codixai impact --base main         # existing command, upgraded
```

`context build` writes its document to stdout only, per the existing output contract.

### 6.6 UI

The UI consumes the same application use cases through `internal/ui` behind the `ui` build tag. No ACO logic in the frontend; the package is rendered from the same document.

## 7. Roadmap

Phase order is adjusted on evidence: **measurement is instrumented in Phase 0**, not last, and **reachability is Phase 0**, because a resolution engine that nothing can call cannot be measured.

### ACO Phase 0 — Reachability and measurement harness

- **Objective:** make reference resolution reachable, and establish the measurement harness before any optimization.
- **Dependencies:** none.
- **Components:** `application/context.go` (skeleton), CLI `refs` wrapper, MCP `references` tool, measurement protocol document.
- **Files likely to change:** `internal/application/index.go` (or a new `context.go`), `internal/cli/index_cmd.go`, `internal/integrations/mcpserver/server.go`, `docs/mcp-integration.md`.
- **Tests:** application-level resolution test reachable from CLI handler; MCP tool contract test; determinism test (same input → same bytes).
- **Risks:** exposing resolution surfaces existing resolver limitations to users. Mitigation: evidence and uncertainty are always included.
- **Acceptance:** `codixai refs` returns references with evidence for all four languages; MCP `references` returns the same document; baseline task set recorded with real token counts.
- **Exclusions:** no packaging, no budgets, no impact changes.

### ACO Phase 1 — Structural context packages

- **Objective:** produce a level-1/level-2 package for the five task kinds.
- **Dependencies:** Phase 0.
- **Components:** `ContextRequest`, `ContextPackage`, selection functions, freshness check, deterministic serialization.
- **Files likely to change:** `internal/application/context.go`, `internal/storage/index_store.go` (read path), `internal/cli/context_cmd.go`, `internal/integrations/mcpserver/server.go`.
- **Tests:** golden documents per task kind; determinism; freshness invalidation; omission reporting.
- **Risks:** selection that looks complete but misses a dependency. Mitigation: completeness check against the resolver's full reference set.
- **Acceptance:** each task kind produces a deterministic package; every included reference carries evidence; `uncertainty` is populated whenever a selected reference is not `resolved`.
- **Exclusions:** no level 3/4 excerpts, no impact integration.

### ACO Phase 2 — Reference-aware impact

- **Objective:** make impact consume the index and resolved references with categorical confidence.
- **Dependencies:** Phase 1.
- **Components:** reference-aware `impact.Analyzer`, application impact use case, MCP `impact` tool.
- **Files likely to change:** `internal/impact/impact.go`, `internal/application/` (new use case), `internal/cli/commands.go`, `internal/integrations/mcpserver/server.go`.
- **Tests:** caller discovery via resolved references; ambiguous references produce uncertainty, not a confirmed dependency; dynamic references flagged; determinism.
- **Risks:** impact output contract change. Mitigation: additive fields only; a breaking change requires an ADR and a `schema_version` decision.
- **Acceptance:** a change to a function lists its resolved callers; ambiguous callers appear as uncertainty; `codixai impact` no longer passes `nil`.
- **Exclusions:** no dataflow, no runtime test selection guarantees.

### ACO Phase 3 — Context packaging and budgets

- **Objective:** levels 3 and 4, budgets, truncation reporting, completeness validation.
- **Dependencies:** Phase 2.
- **Components:** excerpt extractor (bounded file reads), budget estimator, omission report.
- **Files likely to change:** `internal/application/context.go`, `internal/source` (bounded reads).
- **Tests:** budget never exceeded; truncation always reported; level 4 requires explicit request; excerpt bounds respected.
- **Risks:** excerpts leaking secrets or exceeding file-access bounds. Mitigation: reuse the existing root-bounded reader and exclusion rules.
- **Acceptance:** a level-3 package stays within budget and names everything it omitted.
- **Exclusions:** no embedding-based selection.

### ACO Phase 4 — MCP and CLI integration

- **Objective:** human-controlled handoff, structured outputs, backward compatibility.
- **Dependencies:** Phase 3.
- **Components:** `context_build`/`context_inspect` tools, `context` CLI command, approval field.
- **Files likely to change:** `internal/integrations/mcpserver/server.go`, `internal/cli/context_cmd.go`, `docs/mcp-integration.md`.
- **Tests:** MCP contract tests; approval required for handoff; existing tools unchanged.
- **Risks:** tool proliferation. Mitigation: four additive tools maximum; no per-task tools.
- **Acceptance:** an agent can request a package and receive a versioned document; a human can approve handoff explicitly.
- **Exclusions:** no write-capable tools.

### ACO Phase 5 — Measurement and optimization

- **Objective:** report savings with completeness, and decide whether index reuse/incremental work is justified.
- **Dependencies:** Phase 4.
- **Components:** measurement harness, benchmark fixtures, reporting.
- **Files likely to change:** `internal/application/context.go` (metrics), `docs/roadmap/` (results).
- **Tests:** benchmark fixtures are deterministic; regression benchmark fails on savings loss beyond a declared threshold.
- **Risks:** optimizing tokens at the expense of correctness. Mitigation: completeness is a gate, not a metric.
- **Acceptance:** per-task and aggregate savings reported with completeness and time-to-package; reuse measured.
- **Exclusions:** no incremental indexing unless the measurement justifies it.

### 7.1 Approved decisions reflected in this roadmap (2026-11-14)

- Independent context schema versioning (Decision 3).
- Backward-compatible impact evolution: additive optional fields need no bump (Decision 4).
- Hash-based freshness (Decision 5).
- Progressive disclosure (§5.3).
- Human approval (§5.4).
- Token-savings measurement with completeness gate (§5.5).
- Corpus quality as a prerequisite for trustworthy context (link to closure report §13.5).

### 7.2 Separate future work item: PHP reference resolver

PHP is outside Phase 5.2.6 and outside the ACO phase sequence; it is an independent future development item with its own resolver, fixtures, regression tests, quality audit and acceptance criteria; it is not an ACO phase and does not expand the four-language comparative scope.

## 8. Decisions

### 8.1 ADR-backed (created with this proposal)

| ADR | Decision |
| --- | --- |
| 0018 | Cross-language reference resolution architecture; implementation-status clarification (2026-11-14) |
| 0019 | Context package is an application read model, not a domain concept; it carries its own `schema_version` — approved 2026-11-14 (decision); implementation NOT IMPLEMENTED |
| 0020 | Impact analysis becomes reference-aware with categorical confidence propagation — approved 2026-11-14 (decision); implementation NOT IMPLEMENTED |
| 0021 | Project knowledge is reused from the existing SQLite index behind an explicit freshness contract — approved 2026-11-14 (decision); implementation NOT IMPLEMENTED |

### 8.2 Still open (no ADR yet)

| Decision | Why it is deferred |
| --- | --- |
| Progressive disclosure contract as a public, versioned contract | Needs the Phase 1 golden documents before freezing |
| Token-budget enforcement (hard fail vs soft truncate) | Needs real measurements |
| Evidence/confidence propagation across languages | Needs Phase 2 results; legitimate language differences must survive |
| MCP/CLI shared orchestration shape | Needs Phase 4; current 5-tool surface is stable |
| Human approval representation | Depends on whether any write-capable operation is ever approved |
| Whether `domain.SchemaVersion` changes | Adding a new document type should not bump the findings schema; confirm with maintainer |

## 9. Risks, assumptions, open questions

**Assumptions**

- The existing index is deterministic and complete enough for selection (verified for symbol/dependency extraction; reference completeness is language-dependent and documented).
- Agents can consume a versioned JSON document without a new protocol.

**Risks**

| Risk | Mitigation |
| --- | --- |
| A package looks authoritative while a resolver abstained | `uncertainty` is mandatory; evidence is always attached |
| Impact output becomes a de-facto contract | Additive fields only; ADR before any breaking change |
| Freshness check cost exceeds rebuild cost | Measure both in Phase 5; keep rebuild as default |
| Tool proliferation in MCP | Four additive tools maximum |
| Documentation drift (ADR 0018) misleads implementers | Corrected in the Phase 5.2.6 closure |
| Corpus cardinality not asserted | Trustworthy context depends on corpus quality; mitigation: implement the contract in closure report §13.5 before ACO Phase 1 |

**Open questions**

1. Should the context package be persisted, or regenerated per request? (Recommendation: regenerated; `package_id` is a content hash, not a stored record.)
2. What is the maximum acceptable package size for level 2 in practice? Needs measurement.
3. Does ACO need a `--json-schema` style discovery command for agents? Deferred.

## 10. What this proposal does not do

- It does not implement ACO.
- It does not change `internal/**`, tests, domain types, `schema_version`, CLI or MCP behaviour.
- It does not add dependencies, caches, persistence or incremental indexing.
- It does not claim any token savings. No benchmark has been run.
- It does not declare Phase 5.2.6 complete; see the closure report.
- It does not authorize ACO Phase 0. Phase 0 requires explicit human authorization.
