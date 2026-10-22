# 0007 — The claim format, the claim catalog and the linking of a test report

- Status: **Accepted**
- Date: 2026-10-20

## Context

The specification gives Phase 2 one capability: `codixai verify --base <ref>
--claims <file>`. It compares the explicit assertions of an agent with the
real state of the repository, and it returns for every assertion one of
three states:

- `verified` — the repository confirms the assertion.
- `contradicted` — the repository shows that the assertion is false.
- `unresolved` — the evidence is not enough to decide.

The specification fixes the principles and leaves the format open:

- The format is versioned with `schema_version`.
- The initial catalog of claim types is **illustrative** and subject to an
  ADR (section 14, Phase 2).
- Codixai does not run tests. It can read a report produced elsewhere and
  link it to the revision under analysis. A report that cannot be linked
  leaves the claim `unresolved`.
- `behavior_preserved` is not verifiable by static analysis and stays
  `unresolved` unless concrete evidence is provided.
- The exit code `3` is produced by a contradicted claim. An unresolved
  claim only produces it when the policy says so.

Three risks shape this decision:

1. **A false verification is worse than no verification.** An agent that
   asks "did I break the API?" deserves an honest `unresolved`, not a
   cheerful `verified` produced by a check that never looked at anything.
2. **The catalog must not grow by intuition.** Every claim type needs a
   definition, an evaluation that can be computed from the diff, and a
   documented limitation.
3. **The format is a contract.** An agent writes the claim file, so the
   shape must be stable, documented and versioned from the first release.

## Decision

### The claim file

```json
{
  "schema_version": "0.1",
  "base": "main",
  "claims": [
    { "id": "c1", "type": "tests_added", "paths": ["internal/payment/**"] },
    { "id": "c2", "type": "files_unchanged", "paths": ["internal/billing/**"] },
    { "id": "c3", "type": "tests_passed", "report": "reports/go-test.json", "report_revision": "9f2c1ab" }
  ]
}
```

- `schema_version` is required and is `0.1`. A file with another value is
  rejected with a message that names the file, the line and the supported
  values. A file without it is rejected too: the format is versioned on
  purpose.
- `base` is optional. When it is present it must match the base of the
  comparison, otherwise the claim file is rejected, because the claims would
  be about another revision.
- `claims` is required and must not be empty.
- Every claim needs an `id`, which must be unique inside the file, and a
  `type`, which must be one of the types of the catalog. An unknown type is
  rejected with a message that lists the supported types. It is never
  silently ignored: an unknown claim is an error, not an unresolved claim,
  because a typo in a claim file must not read as a pass.
- `paths` are globs of repository relative paths, with the same syntax as
  the exclusion patterns of the configuration.
- `report` is the path of a test report, relative to the root of the
  repository.
- `report_revision` is the revision the report belongs to.

### The catalog

| Type | `verified` | `contradicted` | `unresolved` |
| ---- | ---------- | -------------- | ------------ |
| `files_unchanged` | No changed file matches the paths. | A changed file matches the paths. | The diff is empty, so there is nothing to compare. |
| `scope` | Every changed file matches the paths. | A changed file is outside the paths. | The diff is empty. |
| `tests_added` | A test file that matches the paths was added or modified. | No test file that matches the paths changed. | The paths hold no test file of the language. |
| `tests_modified` | A test file that matches the paths was modified. | No test file that matches the paths was modified. | The paths hold no test file of the language. |
| `dependency_unchanged` | No manifest or lockfile changed. | A manifest or lockfile changed. | The repository has no manifest of any language. |
| `dependency_added` | The declared dependencies appear in the changed manifests. | A manifest changed and the declared dependencies are absent. | The repository has no manifest, or the claim declares no dependency. |
| `api_unchanged` | — | — | Always. The public API is not extracted yet. |
| `tests_passed` | The report is linked to the revision and reports a success. | The report is linked to the revision and reports a failure. | There is no report, or the report cannot be linked to the revision. |
| `behavior_preserved` | — | — | Always. Static analysis does not decide the behavior of a change. |

A claim that cannot be evaluated never becomes `verified`. The three states
are decided by the evidence that exists, and the absence of evidence is
always `unresolved`.

The patterns of the test files are the ones of the language profiles, so
`tests_added` uses the same knowledge as `CDX-TST-001`.

### The linking of a test report

A report is **linked** to the revision under analysis when one of these
holds:

- the claim declares `report_revision` and it equals the head of the
  compared range, or
- the report itself declares the revision: a JUnit XML file with a
  `revision` or a `commit` property, or a `go test -json` stream with a
  `revision` field in one of its events.

An unlinked report leaves the claim `unresolved`, with the reason in the
message. A report that is linked and reports a failure produces a
`contradicted` claim.

Two formats are supported in this milestone, both read with the standard
library: JUnit XML and the JSON stream of `go test -json`. A report in
another format is `unresolved`, never a guess.

### The exit code

- `0`: every claim is `verified` or `unresolved`.
- `3`: at least one claim is `contradicted`.
- An `unresolved` claim only produces `3` when the configuration sets
  `verify.fail_on_unresolved: true`. The default is `false`, because an
  unresolved claim is usually a limitation of the analysis, not a failure.

### The results

Every result carries the claim, the state, a message that says what was
observed, the evidence that supports the state and the limitations of the
check. The document of a verification is versioned, sorted by the id of the
claim, and holds the base and the head of the compared range.

## Consequences

Positive:

- An agent receives an honest answer per claim, and an honest `unresolved`
  when the analysis cannot decide.
- The catalog is small, explicit and computable from the diff.
- The format is stable and versioned, so an agent can depend on it.
- A report is never used as if Codixai had produced it. The provenance is
  external and visible in the evidence.

Negative:

- `api_unchanged` and `behavior_preserved` are always `unresolved`. The
  catalog promises less than it could promise with a semantic analysis.
- The linking of a report depends on the report declaring its revision. A
  report without it is `unresolved`, even when it is honest.
- A claim file with a typo is an error, not an unresolved claim. The strict
  behavior is intentional, but it rejects files that a lenient reader would
  accept.

## Alternatives considered

**Let the catalog grow with every new check.**
Rejected. A catalog without a definition and a documented limitation becomes
a list of guesses. The catalog grows when a check can be computed and
described, not when it is convenient.

**Report `unresolved` claims as `verified` when no finding contradicts
them.**
Rejected. The absence of a finding never proves an assertion. It is the
central principle of the specification (section 1.2).

**Run the tests of the analyzed repository to check `tests_passed`.**
Rejected. It breaks the invariant that Codixai never executes the code of
the analyzed repository, and it would need the toolchain of the project.

**Accept a report without linking it to the revision.**
Rejected. A report of another revision is evidence about another state of
the repository, and using it would produce a false verification.

**Version the claim format with the version of the tool.**
Rejected. The format has its own lifecycle, and the version of the tool
changes far more often than the format.
