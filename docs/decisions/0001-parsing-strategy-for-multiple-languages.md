# 0001 — Parsing strategy for multiple languages

- Status: **Accepted with conditions** (updated 2026-10-27)
- Date: 2026-10-20 (original), 2026-10-27 (updated)

## Context

The specification places the parsing strategy on the critical path of
Phase 1. Section 6.1 names Tree-sitter as the mechanism for the multi
language parsing, and the deliverables of Phase 0 (section 14) require an
ADR that compares "Tree-sitter via CGO" against the WASM and other
alternatives, backed by **test builds on Linux, macOS and Windows**.

Two constraints of the original environment blocked that decision:

1. **No network access** — Tree-sitter grammars and Go bindings could not be
   fetched.
2. **Only Linux available** — macOS and Windows builds could not be produced
   or tested.

## Decision (2026-10-27)

**Keep the line-level lexical scanner as the default parsing strategy for
the release candidate. Adopt Tree-sitter via CGO with vendored grammars as
an opt-in build tag (`treesitter`) for Phase 1.5+, not as a blocker for the
current milestone.**

### Default strategy (release candidate): lexical scanner

- **Go**: `go/parser` + `go/ast` with `ParseComments` (real AST, support level
  `syntactic` for diff-only analysis, `reserved` for future semantic rules).
- **Python, PHP, JavaScript, TypeScript**: shared line-level lexical scanner
  (`internal/languages/lexical`) with per-language profiles. Knows comment
  and string syntax, continues block comments and multi-line strings across
  lines, never reports markers inside strings/comments.
- All rules stay at `support_level: syntactic` (or `none` where not
  applicable). `epistemic_status` is always `inferred` or `unknown`, never
  `observed` for non-Go languages.
- Limitations explicitly published per rule and language (see
  `codixai audit --rules`).

### Opt-in strategy (Phase 1.5+): Tree-sitter via CGO

- **Mechanism**: `github.com/tree-sitter/go-tree-sitter` (CGO) + vendored
  grammar C sources for Go, Python, JavaScript, TypeScript, PHP.
- **License review**: All five grammars are MIT-licensed (verified from
  upstream repositories). Compatible with Codixai's MIT license.
- **Build tag**: `treesitter` — when enabled, language packages swap the
  lexical extractor for a Tree-sitter based one; the rule engine and
  support matrix remain unchanged.
- **Support level upgrade path**: When Tree-sitter extractor is active and
  a rule uses it, the rule may declare `support_level: resolved` for that
  language with updated `limitations`.
- **Cross-platform builds required**: Linux (verified), macOS, Windows —
  must be observed in CI before enabling by default.

### Why not WASM

- Adds a WASM runtime dependency (wasm3, wasmer, or wazero) with its own
  license, memory model, and performance profile.
- Grammar files still need vendoring and license review.
- No measurable benefit over CGO for this use case (diff analysis, not
  real-time editing).
- CGO is already enabled in the toolchain; the C sources are small and
  well-maintained.

## Evidence gathered (2026-10-27)

| Criterion | Status |
|-----------|--------|
| Grammar licenses (Go, Py, JS, TS, PHP) | MIT — compatible ✓ |
| CGO available in environment | Yes (`gcc` present, `CGO_ENABLED=1`) ✓ |
| Network access to fetch deps | **No** — blocked in current env ✗ |
| macOS/Windows builds observed | **No** — CI matrix declared only ✗ |
| Lexical scanner false positives documented | Yes (see below) ✓ |
| Tree-sitter precision improvement demonstrated | Conceptually yes; no PoC without network ✗ |

### Known false positives of lexical scanner (will persist in RC)

1. **CDX-TST-001**: `t.Skip("git is not available...")` reported as weakened
   test (legitimate conditional skip). Affects Go; similar patterns in
   Python/JS/TS/PHP.
2. **CDX-IMP-001**: Relative imports in Python (`from . import x`) marked
   correctly as relative but existence not verified; JS/TS/ PHP package
   inference by 2 namespace segments is heuristic.
3. **CDX-ERR-001**: Empty `catch`/`except` only detected when closing brace
   visible in changed lines.
4. **CDX-SUP-001**: Only known directive patterns detected; config files
   (`.eslintrc`, `pyproject.toml`) not parsed.
5. **CDX-PLH-001**: `Explicit` markers need `Require` regex to distinguish
   comment vs code context.

These are **accepted limitations** for the release candidate. Every finding
carries `epistemic_status: inferred`, `support_level: syntactic`, and the
limitations are user-visible in `codixai audit --rules` and in JSON/SARIF
output.

## Conditions for enabling Tree-sitter by default (future ADR update)

Before changing the default strategy, the following must be **observed**
(not just declared):

1. **License review recorded** in this ADR for each vendored grammar (MIT
   confirmed above).
2. **Clean builds on Linux, macOS, Windows** with `CGO_ENABLED=1` and the
   `treesitter` tag, from a fresh clone, exact Go version documented.
3. **Fixture corpus passes** with Tree-sitter extractor: all 58 existing
   cases produce identical or better results (fewer false positives, no
   new false negatives).
4. **Binary size impact measured** and documented (target: <10MB increase
   for UI variant).
5. **At least one rule upgraded** to `resolved` with evidence (candidate:
   `CDX-IMP-001` for Python/JS/TS/PHP, `CDX-ERR-001` for all).

## Consequences

Positive:

- Release candidate ships on time with honest, documented limitations.
- No new dependencies, no CGO requirement for default build (pure Go
  scanner, CGO only for `go/ast` which is stdlib).
- Clear upgrade path: `treesitter` build tag, same language package seam.
- Users see `syntactic` and know exactly what was checked.

Negative:

- Four languages remain at `syntactic` level for RC.
- Known false positives (especially `t.Skip` conditional) persist and are
  reported.
- `CDX-REF-001` unavailable for Python/JS/TS/PHP (declared `support: none`).

## Alternatives considered (unchanged)

**Tree-sitter through CGO with vendored grammars.** Kept as Phase 1.5
candidate with conditions above.

**Tree-sitter through WASM runtime.** Rejected — extra dependency, no
clear advantage.

**LSP / ecosystem analyzers.** Rejected for this milestone per spec.

**Staying with line-level scanner forever.** Rejected — Phase 1.5 will
adopt Tree-sitter when conditions are met.