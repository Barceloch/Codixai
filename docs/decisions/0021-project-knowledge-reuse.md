# ADR 0021: Project knowledge reuse behind an explicit freshness contract

- Status: proposed; freshness policy approved 2026-11-14 (Decision 5). Index reuse remains PROPOSED / NOT IMPLEMENTED
- Date: 2026-11-14
- Depends on: ADR 0017 (index schema v2), ADR 0019 (context package)

## Context

Reference resolution rebuilds the index unconditionally: `RunResolveReferences` calls `index.Build` at `internal/application/index.go:325-326` and ignores any persisted index. In contrast, `index status/overview/find/refs/deps` read SQLite and never build (`application/index.go:155-256`).

Persistence already exists and is richer than the read paths use:

- Schema v2 stores symbols, dependencies and references (`storage/index_store.go:48-53`).
- `Save` deletes every row for the root and reinserts (`:288-290`).
- `Save` writes `hash=""` and `size=0` placeholders (`:331-333`).
- `LoadReferences` and `ListReferences` have no production caller.
- The only freshness gate is `status.FileCount == 0` (`application/index.go:86`).
- `migrateToV2` has no caller, so the v1→v2 migration path is untested.

Agent Context Optimization needs reusable project knowledge. It does not need a new storage engine: the index is already the project knowledge model.

## Options

1. **Always rebuild.** Status quo. Simple and always correct, but the cost is paid per context request and cannot be measured away.
2. **Reuse with an explicit freshness contract.** Read the persisted index when a cheap freshness check passes; rebuild otherwise. Record the decision in the package (`repository.fresh`).
3. **Full incremental indexing.** Track per-file changes and update rows in place. Highest value at scale, highest risk, and currently unjustified by any measurement.

## Recommendation

**Option 2 for ACO Phase 1; Option 3 deferred until measurement justifies it.**

Freshness contract (pre-approval recommendation; superseded by the Approved freshness contract below):

- Compare the set of indexed files against the current worktree by path, size and modification time.
- If they match, reuse; if not, rebuild and record `fresh: false` in the package.
- A content hash is the correct long-term signal, but `Save` currently writes empty hashes, so enabling it requires a storage change and a migration decision. Treat it as a follow-up.
- Reuse never crosses roots: the existing root-scoped delete and the bounded-path invariant are preserved.
- The context package always states whether knowledge was reused or rebuilt.

## Consequences

- A new read path from the application layer into `storage` is required, plus wiring for `LoadReferences`.
- Correctness now depends on the freshness check. A stale index silently produces a stale package, so the check must be covered by tests that mutate one file and assert invalidation.
- Determinism is preserved: the same repository content produces the same package regardless of whether the index was reused or rebuilt.
- `migrateToV2` must be exercised or removed before reuse becomes the default, otherwise an old database can be read with a wrong shape.
- No new storage schema is introduced by this ADR.

## Approved freshness contract (2026-11-14)

- Content hash is authoritative for determining whether indexed knowledge corresponds to the current file content.
- File modification time and size may be used as performance optimizations or preliminary change indicators, but never as the ultimate guarantee of content identity.
- Freshness states: `fresh`, `stale`, `unknown`. The design must not present unverifiable knowledge as `fresh`; when freshness cannot be established, the state is `unknown` (and the safe action is rebuild or state `unknown`, never `fresh`).
- Future cache invalidation requirements: reuse never crosses roots (root-scoped delete preserved); freshness check must be covered by tests that mutate one file and assert invalidation; determinism preserved (same content → same package regardless of reuse vs rebuild).
- Incremental indexing remains DEFERRED until justified by measurements; rebuild remains an acceptable default.
- Link to ADR 0017 and ADR 0019.

## Open questions

- Is modification time trustworthy enough across editors and CI checkouts, or should content hashing be mandatory from the start? **Resolved 2026-11-14 (Decision 5): content hashing is authoritative; mtime/size are optimizations only.** (Recommendation: content hashing is authoritative; mtime/size are optimizations only.)
- Should the freshness check be exposed as `codixai index verify`, or stay internal?
- What is the acceptable staleness window when a build is already running (WAL concurrency)?
