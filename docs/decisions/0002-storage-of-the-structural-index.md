# 0002 — Storage of the structural index

- Status: **Deferred**
- Date: 2026-10-20

## Context

The specification places SQLite as the internal index of Codixai (section
6.3) and requires the driver and the FTS5 availability to be decided
through an ADR with test builds (Phase 0 deliverables, section 14).

The same section is explicit about the timing: the decision **may be
postponed until before Phase 5**, because `audit --diff` does not need a
persistent index. The Phase 1 spike compares the Git blobs of the base and
of the head directly, with no checkout and no database.

The milestone that exists today therefore has no storage layer at all. The
`audit` use case reads the diff, applies the rules and returns a document.
There is no `.codixai/` directory, no cache and no database file.

Deferring the decision is a deliberate decision, and this ADR records it so
that it is not an omission.

## Decision

**Defer the storage decision until before Phase 5. `audit --diff` keeps
working without a persistent index, and the index is a local, rebuildable
cache, never a source of truth.**

The decision criteria that will be applied when it is taken:

1. **The driver.** The comparison is between the pure Go driver
   `modernc.org/sqlite` and the CGO driver `mattn/go-sqlite3`.
   - `modernc.org/sqlite`: no CGO, so the cross platform build stays simple
     and the binary is a pure Go artifact, at the cost of a larger binary
     and a lower raw performance. It also requires network access to fetch
     the module, which the current environment does not have.
   - `mattn/go-sqlite3`: the reference implementation and the best
     performance, but it requires a C toolchain in every build environment,
     which is a direct conflict with the reproducible multi platform
     release that the specification requires.
2. **FTS5.** The full text search of Phase 5 needs FTS5. The pure Go driver
   historically had a partial FTS5 support, so the availability of FTS5 has
   to be **verified with a test build**, not assumed. If FTS5 is not
   available in the chosen driver, the search falls back to a table scan
   with an explicit limitation, or the decision moves to another driver.
3. **The build evidence.** The same three platforms as the parsing
   decision: Linux, macOS and Windows, from a clean clone, with the exact
   Go version documented.
4. **The license review.** The driver and the SQLite amalgamation carry
   their own licenses and must be reviewed before they enter the repository.
5. **The schema.** The index must be rebuildable from the repository, must
   store the schema version and the version of the tool that created it, and
   must be able to detect and recover from a corrupt or incompatible index.

## Consequences

Positive:

- The milestone stays dependency free and builds offline.
- The diff comparison, which is the whole value of the current milestone, is
  not delayed by a database.
- The index, when it exists, will be designed for the use cases that Phase 5
  actually needs, instead of for the guesses of Phase 0.

Negative:

- The Phase 0 deliverable that names the storage ADR as "decided" is
  **not met**: this ADR is deferred, not accepted.
- Repeated audits of the same repository repeat the work. The cost is
  accepted while the corpus of real diffs is still small.
- The structural index, the symbol extraction and the relation resolution
  that the specification places in Phase 5 all wait for this decision.

## Alternatives considered

**Adopt SQLite now, in Phase 0.**
Rejected. It would add a dependency and a CGO or vendored-amalgamation
question to a milestone whose only job is to prove that the diff verifier
works. The specification explicitly allows the deferral.

**Keep the index in memory or in a plain file, without SQLite.**
Rejected as the final solution. A memory index dies with the process and
would force every consumer to re-parse the repository. A plain file format
would be a second storage engine to maintain, with no search capability and
no transactional guarantee.

**Use an embedded document store or an embedded key value store.**
Rejected for now. They are simpler to embed, but the specification names
SQLite and FTS5, and the relation queries of Phase 5 need a query language
with joins. The decision can be revisited if the SQLite driver choice turns
out to be unacceptable.

**Keep the deferral open indefinitely.**
Rejected. The decision has a deadline: before the start of Phase 5, the
criteria above are applied and this ADR is rewritten with a status of
Accepted or Rejected.
