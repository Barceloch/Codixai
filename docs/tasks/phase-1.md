# Phase 1 — The diff verifier spike

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 1". This file turns them into atomic
tasks with executable acceptance checks.

**Phase exit criterion (specification):** the five languages are recognized
and have acceptance fixtures from this milestone; every rule activates only
in the languages where it has proven coverage; an initial evaluation of
precision, false positives, limitations and unsupported cases is published;
and the spike decides which rules are kept.

The phase is delivered **rule by rule**, not in a block: a rule and
language combination is published when it reaches its quality criterion.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P1-01 | `audit --diff` with the three origins | DONE |
| P1-02 | The six rules | DONE |
| P1-03 | The support matrix and its validation | DONE |
| P1-04 | The corpus of real diffs produced by agents | PENDING |
| P1-05 | The false positive report | PENDING |
| P1-06 | The decision gate of the phase | PENDING |
| P1-07 | `verify` with claims | PENDING |
| P1-08 | SARIF export | PENDING |
| P1-09 | Baseline of findings | PENDING |

The last three belong to the later phases of the specification but are
listed here because the specification forbids promising them before the
Phase 1 decision gate. They are **not implemented**.

---

### P1-01 — `audit --diff` with the three origins

- Status: **DONE**
- Depends on: P0-01, P0-02, P0-03, P0-04, P0-05

**Objective**
Audit the changes of a repository in read-only mode, without a checkout and
without a persistent index, from the three origins that an agent works
with.

**Sections to read**
9, 13, 14 (Phase 1), 1.2, 4.

**Deliverables**
- The `audit` command with `--diff`, `--base <ref>`, `--staged`,
  `--worktree` and `--rules`.
- The exact origin selection: one of the three, never zero and never two.
- The `application.RunAudit` use case, the `ChangeSource` and `DiffAnalyzer`
  ports, and the `analysis.Engine`.
- The declaration of the analyzed origin in the output.
- The policy application: the rule overrides, the suppressions and the
  `fail_on` threshold.

**Acceptance checks**
```bash
go test ./internal/application/
go test ./tests/integration/
go run ./cmd/codixai audit --diff --worktree
go run ./cmd/codixai audit --diff --staged
go run ./cmd/codixai audit --diff --base main
```
The evidence in the repository is the test suite: `TestRunAuditAppliesTheFailureThreshold`,
`TestRunAuditFailsOnAnErrorFinding`, `TestRunAuditAppliesTheRuleOverrides`,
`TestRunAuditAppliesTheSuppressions`, `TestRunAuditExcludesTheConfiguredPaths`,
`TestRunAuditRejectsAnIncompleteRequest`, `TestAuditReportsTheSuppressionAndThePlaceholder`,
`TestAuditJSONOutputIsACleanDocument`, `TestAuditWithoutFindingsSucceeds`,
`TestAuditWithoutOriginFailsWithTheInvalidArgumentsCode`,
`TestAuditDoesNotModifyTheRepository`, `TestBinaryReportsTheFindingsAndTheExitCode`
and `TestBinaryDoesNotWriteOutsideOfTheRepository`.

**Out of scope**
The persistent index, the checkout of the base revision, and the commands
`index`, `status`, `overview`, `find`, `refs` and `deps`.

---

### P1-02 — The six rules

- Status: **DONE**
- Depends on: P1-01, P0-07

**Objective**
Six deterministic rules, each with a stable identifier, a version, a
severity, a support matrix, evidence and a set of limitations.

**Sections to read**
0, 1.2, 5, 11, 14 (Phase 1), 15.

**Deliverables**
In `internal/analysis/rules/`:

| Rule | Severity | File |
| ---- | -------- | ---- |
| `CDX-IMP-001` | error | `imp.go` |
| `CDX-REF-001` | error | `ref.go` |
| `CDX-TST-001` | warning | `tst.go` |
| `CDX-SUP-001` | warning | `sup.go` |
| `CDX-ERR-001` | warning | `err.go` |
| `CDX-PLH-001` | warning | `plh.go` |

and the catalog in `catalog.go`, which publishes the support matrix, the
method, the evidence kind and the limitations per language.

**Acceptance checks**
```bash
go test ./internal/analysis/... 
go run ./cmd/codixai audit --rules
go run ./cmd/codixai --format json audit --rules
```
The evidence in the repository is the test suite: `TestFixtureCorpus`, which
runs the engine over the 58 cases of the corpus, `TestCatalogPublishesTheSupportMatrix`,
`TestLimitationsArePublishedPerLanguage` and
`TestRuleIdentifiersFollowTheConvention`.

**Out of scope**
The semantic resolution of the references outside Go, and any rule beyond
the six. The candidate rules of the specification that are not committed
stay uncommitted.

---

### P1-03 — The support matrix and its validation

- Status: **DONE**
- Depends on: P1-02, P0-07

**Objective**
Every rule publishes, for each of the five languages, its support level,
and the matrix is checked against the fixture corpus.

**Sections to read**
5, 11, 15.

**Deliverables**
- The `Languages` and the `Limitations` maps of every rule.
- The publication of the matrix by `codixai audit --rules`, in text and in
  JSON.
- `TestSupportMatrixMatchesFixtures` in `internal/analysis/engine_test.go`.
- An explicit CI job that runs that test.

**Acceptance checks**
```bash
go test ./internal/analysis/ -run TestSupportMatrixMatchesFixtures
go test ./internal/analysis/rules/ -run TestCatalogPublishesTheSupportMatrix
go run ./cmd/codixai audit --rules
```

**Out of scope**
The support level `resolved`, which no rule reaches in this milestone.

---

### P1-04 — The corpus of real diffs produced by agents

- Status: **PENDING** — blocked on an external input
- Depends on: P1-01

**Objective**
A versioned collection of real or reproducible diffs produced by AI
agents, with positive and negative cases, that feeds the precision
evaluation of the phase.

**Sections to read**
14 (Phase 1), 15, 17.8.

**Deliverables**
A directory of diffs, with the same shape as the fixture corpus of
`testdata/rules/`, plus the metadata of their origin.

**Acceptance checks**
Not executable yet. The task needs diffs that were really produced by
external agents; they cannot be fabricated.

**Out of scope**
The fixtures of `testdata/rules/`, which are synthetic and already exist.

**Why it is pending.** The specification asks for "a versioned collection of
real or reproducible diffs produced by agents". The synthetic corpus
fulfills the fixture requirement but not the corpus requirement, and the
difference must not be blurred in the documentation.

---

### P1-05 — The false positive report

- Status: **PENDING** — blocked on P1-04 and on a pending decision
- Depends on: P1-04

**Objective**
Publish an initial evaluation of the precision, of the false positives, of
the limitations and of the unsupported cases, per rule and per language.

**Sections to read**
14 (Phase 1), 15, 18.1.

**Deliverables**
A report in `docs/`, with the measured numbers and their reproduction
command.

**Acceptance checks**
Not executable yet. It needs P1-04, and it needs the thresholds of the
pending decision 7 of the specification, which are not defined.

**Out of scope**
Any precision number that was not measured. The report must not contain an
estimate presented as a measurement.

**Why it is pending.** The specification lists the thresholds of precision
and of false positives per rule and per language as a **pending decision**
(decision 7). Until the maintainer defines them, the report cannot declare
a rule good or bad.

---

### P1-06 — The decision gate of the phase

- Status: **PENDING** — blocked on P1-05
- Depends on: P1-05, P1-04

**Objective**
Before the start of Phase 2, decide with data which rules are kept, which
are downgraded to `inferred` and which are discarded.

**Sections to read**
14 (Phase 1, puerta de decisión), 1.2.

**Deliverables**
A decision recorded in an ADR, with the data that supports it.

**Acceptance checks**
Not executable yet. The gate also requires diffs reviewed by people outside
the project and the opinion of external users, which cannot be produced
inside the repository.

**Out of scope**
Starting Phase 2 before the gate is closed.

---

### P1-07 — `verify` with claims

- Status: **PENDING** — belongs to Phase 2
- Depends on: P1-06

**Objective**
`codixai verify --base <ref> --claims <file>`: contrast the claims of an
agent with the diff and with the results of the rules, with the states
`verified`, `contradicted` and `unresolved`.

**Sections to read**
1.1 (pillar 4), 14 (Phase 2), 15.

**Deliverables**
The use case, the versioned schema of the claims and the command.

**Acceptance checks**
Not executable yet. The command does not exist: the `Claim` and the
`ClaimResult` models exist in the domain, but there is no use case and no
command.

**Out of scope**
Running the tests of the analyzed repository. A `tests_passed` claim is
only contrasted against an external report that is bound to the revision.

---

### P1-08 — SARIF export

- Status: **PENDING** — belongs to Phase 3
- Depends on: P1-06

**Objective**
Export the findings as SARIF, with a consistent rule and location
representation, and validate the output.

**Sections to read**
9, 14 (Phase 3), 15.

**Acceptance checks**
Not executable yet. The export does not exist. The `--format` flag of the
CLI only accepts `text` and `json` today.

---

### P1-09 — Baseline of findings

- Status: **PENDING** — belongs to Phase 3
- Depends on: P1-08

**Objective**
A versioned baseline, shared through CI, that distinguishes the new
findings from the existing ones, with a stable fingerprint that survives
the displacement of the code.

**Sections to read**
10, 14 (Phase 3), 15.

**Acceptance checks**
Not executable yet. The `audit.baseline` key of the configuration is parsed
and validated, but no baseline file is read and none is written.

**Out of scope**
The storage of the baseline in the SQLite index: the baseline is a versioned
file of the analyzed repository, and the index is a local, rebuildable
cache.
