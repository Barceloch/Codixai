# ADR 0020: Reference-aware impact analysis with categorical confidence

- Status: proposed; versioning policy approved 2026-11-14 (Decision 4). Reference-aware impact design remains PROPOSED / NOT IMPLEMENTED
- Date: 2026-11-14
- Depends on: ADR 0018 (reference resolution), ADR 0019 (context package)

## Context

`internal/impact` currently classifies changed files by `FileChange.Status` and derives likely tests from path shape (`impact/impact.go:45-140`). It holds an `*index.Index` that it never reads, and its only production call site passes `nil` (`cli/commands.go:593`). `domain.Reference` is never referenced in the package.

The Phase 5.2.6 audit delivered resolvers that classify every reference as `resolved`, `ambiguous`, `unresolved`, `external` or `dynamic`, each with evidence. Those states are exactly the information impact analysis lacks.

`AGENTS.md` invariant 6 requires honest epistemic status, and the implementation conventions forbid inventing numeric confidence. Any propagation must therefore stay categorical.

## Options

1. **Keep the status-based heuristic.** Cheapest, but it cannot answer "who calls this function?" and emits test paths that may not exist.
2. **Index-aware only.** Resolve symbols and files from the index. Better, still no caller information.
3. **Index + reference-aware with categorical propagation.** Follow incoming references and carry the resolver's state into the impact result.

## Recommendation

**Option 3, in two increments.**

- Increment 1: impact consumes the persisted index instead of path shape.
- Increment 2: impact follows incoming references and propagates state:
  - `resolved` → confirmed dependency;
  - `ambiguous` → all candidates plus an explicit uncertainty entry;
  - `dynamic` → possible dependency, flagged, never confirmed;
  - `unresolved` / `external` → boundary marker, never a local dependency.
- The application layer gains an impact use case so CLI, MCP and UI share it.
- `impact.ImpactResult` grows additively. Any change to an existing field or its meaning requires a `schema_version` decision and a new ADR.

## Consequences

- Impact results become larger and more honest; consumers must handle the uncertainty list.
- `codixai impact` stops passing `nil`, which changes its output. This is a user-visible change and must be documented.
- Impact inherits every resolver limitation (TDZ, declaration merging, comprehension breadth). Those appear as uncertainty, not as missing data.
- No dataflow or runtime test selection is introduced.

## Approved versioning policy (2026-11-14)

- Additive, backward-compatible optional impact fields may be introduced without a schema version bump.
- A version bump is required when a change: breaks existing consumers; changes the meaning of existing fields; removes or renames existing fields; alters established interpretation or semantics incompatibly.
- Categorical reference states are preserved: `resolved`, `ambiguous`, `dynamic`, `unresolved`, `external`. No numeric confidence scores are introduced or permitted.
- Ambiguous and dynamic references must never be silently converted into confirmed dependencies; they appear as uncertainty or as boundary markers.
- Link to ADR 0018 and the closure report §13.4.

## Open questions

- Should `ImpactResult` remain in `internal/impact`, or move to the application layer as a read model?
- How should an `ambiguous` reference affect the "likely failing tests" recommendation: include the test, or only list it as uncertain?
- Does the existing `impact` command contract allow additive fields without a `schema_version` bump? **Resolved 2026-11-14 (Decision 4): yes, additive fields without a bump.** (Recommendation: yes, per invariant 5, but confirm.)
