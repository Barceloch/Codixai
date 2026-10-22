# Phase 2 — Claims, evidence and the verification contract

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 2". This file turns them into atomic
tasks with executable acceptance checks.

**Phase exit criterion (specification):** the claims that are detectable in
the fixtures are verified, contradicted or left unresolved reproducibly; no
state is inferred without enough evidence.

**The phase is implemented.** The command, the contract, the evaluator, the
readers of the external reports, the policy and the corpus exist. The
exceptions are declared below and are not defects: they are the limits that
the decision
[`0007`](../decisions/0007-claim-format-and-report-linking.md) fixes on
purpose.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P2-01 | The claim contract and its decision | DONE |
| P2-02 | The catalog of the claim types | DONE |
| P2-03 | The evaluator of the claims | DONE |
| P2-04 | The readers of the external test reports and the linking rule | DONE |
| P2-05 | The `verify` command and its document | DONE |
| P2-06 | The exit codes and the policy | DONE |
| P2-07 | The corpus of the claims | DONE |
| P2-08 | The integration tests of the verification | DONE |
| P2-09 | The architecture test of the application layer | DONE |

---

### P2-01 — The claim contract and its decision

- Status: **DONE**
- Depends on: P0-03, P0-05

**Objective**
A versioned document that an agent writes to state what it asserts about a
change, with a strict validation: a typo in the file is an error, never an
unresolved claim that reads as a pass.

**Sections to read**
0 (glossary), 1.1 (pillar 4), 14 (Phase 2), 15.

**Deliverables**
- `docs/decisions/0007-claim-format-and-report-linking.md`: the format, the
  catalog, the linking of a report and the exit codes.
- `internal/claims/claims.go`: the loader and the validator. `schema_version`
  is required and is `0.1`; `base` is optional and has to match the base of
  the comparison; `claims` is required and must not be empty; every claim
  needs a unique `id` and a `type` of the catalog. An unknown field and an
  unknown type are rejected with a message that names the file and the line.
- `internal/domain/claims.go`: the model of the file, of the claim and of
  the result.

**Acceptance checks**
```bash
go test ./internal/claims/ -run 'TestParse|TestLoad|TestValidateBase'
go run ./cmd/codixai verify --help
```
The evidence in the repository is the test suite: `TestParseReadsAValidFile`,
`TestParseRejectsAMissingSchemaVersion`,
`TestParseRejectsAnUnsupportedSchemaVersion`, `TestParseRejectsMissingClaims`,
`TestParseRejectsEmptyClaims`, `TestParseRejectsAClaimWithoutIdentifier`,
`TestParseRejectsADuplicateIdentifier`, `TestParseRejectsAClaimWithoutType`,
`TestParseRejectsAnUnknownType`, `TestParseRejectsAnUnknownFieldOfTheFile`,
`TestParseRejectsAnUnknownFieldOfAClaim`,
`TestParseRejectsAPathThatIsNotRelative`,
`TestParseRejectsAPathThatEscapesTheRoot`,
`TestParseRejectsAReportThatIsNotRelative`,
`TestParseRejectsAClaimOfPathsWithoutPaths`, `TestLoadReadsAFile`,
`TestLoadReportsAMissingFile`, `TestValidateBaseRejectsAnotherBase`,
`TestValidateBaseAcceptsTheSameBase` and
`TestValidateBaseAcceptsAFileWithoutBase`.

**Out of scope**
The semantic analysis of the claims, and any type outside the catalog of
the decision 0007.

---

### P2-02 — The catalog of the claim types

- Status: **DONE**
- Depends on: P2-01

**Objective**
A closed catalog of the claim types, each one with its definition, its
known limits and the states it can reach.

**Sections to read**
14 (Phase 2), 15.

**Deliverables**
The catalog in `internal/claims/claims.go`, exposed by `claims.Catalog()` and
`claims.Types()`:

| Type | States it can reach |
| ---- | ------------------- |
| `files_unchanged` | `verified`, `contradicted`, `unresolved` |
| `scope` | `verified`, `contradicted`, `unresolved` |
| `tests_added` | `verified`, `contradicted`, `unresolved` |
| `tests_modified` | `verified`, `contradicted`, `unresolved` |
| `dependency_unchanged` | `verified`, `contradicted`, `unresolved` |
| `dependency_added` | `verified`, `contradicted`, `unresolved` |
| `api_unchanged` | `unresolved` only |
| `tests_passed` | `verified`, `contradicted`, `unresolved` |
| `behavior_preserved` | `unresolved` only |

**Acceptance checks**
```bash
go test ./internal/claims/ -run TestCatalogHoldsEveryType
go run ./cmd/codixai verify --help   # prints the identifiers of the catalog
```

**Out of scope**
`api_unchanged` and `behavior_preserved` are not implemented as decidable
types: they are always `unresolved`, as the decision 0007 states. The
extraction of the public API belongs to Phase 5 or Phase 6.

---

### P2-03 — The evaluator of the claims

- Status: **DONE**
- Depends on: P2-01, P2-02

**Objective**
Compute the state of every claim from the compared range, from the
manifests of the repository and from the external reports, with the evidence
and the limitations of the check.

**Sections to read**
1.2, 4, 14 (Phase 2), 15.

**Deliverables**
- `internal/claims/evaluate.go`: the evaluator, the evidence of a result
  (`diff_file` for a changed file, `external_report` for a report) and the
  limitations of the type of the claim.
- `internal/application/verify.go`: the use case `RunVerify`, the ports
  `ClaimLoader` and `ClaimEvaluator`, the versioned document of the
  verification and the summary by state.

**Acceptance checks**
```bash
go test ./internal/claims/ -run TestEvaluate
go test ./internal/application/
```

**Out of scope**
Running the tests of the analyzed repository, and any inference about the
intent of the change that is not declared.

---

### P2-04 — The readers of the external test reports and the linking rule

- Status: **DONE**
- Depends on: P2-01, P2-03

**Objective**
Read a test report produced outside Codixai, link it to the revision under
analysis and never present it as observed by the tool.

**Sections to read**
13, 14 (Phase 2), 15.

**Deliverables**
- `internal/claims/report.go`: the reader of the JUnit XML format (`.xml`)
  and of the JSON stream of `go test -json` (`.json`), both read with the
  standard library, and the linking rule.
- The linking rule: a report is linked when the claim declares a
  `report_revision` equal to the head of the compared range, or when the
  report itself declares that revision (a `revision` or a `commit` property
  of a JUnit XML document, a `revision` field of a `go test -json` event).
- An unsupported format is `unresolved`, never a guess.

**Acceptance checks**
```bash
go test ./internal/claims/ -run 'TestParseJUnitXML|TestParseGoTestJSON|TestParseReport|TestLinkReport'
go test ./tests/integration/ -run 'TestVerifyLinksAnExternalReportToTheRevision|TestVerifyLeavesAnUnlinkedReportUnresolved'
```

**Out of scope**
The TAP format and every other format. The pending decision 10 of the
specification still lists the accepted report formats as open; the decision
0007 fixes the two formats of this milestone.

---

### P2-05 — The `verify` command and its document

- Status: **DONE**
- Depends on: P2-03, P2-04

**Objective**
`codixai verify --base <ref> --claims <file>`: compare the claims with the
changes between the base and the head, and report one state per claim with
its evidence and its limitations.

**Sections to read**
9, 14 (Phase 2), 15.

**Deliverables**
- `internal/cli/verify.go`: the command, the flags `--base` and `--claims`,
  the text report and the JSON encoder.
- The versioned document: `schema_version`, `tool_version`, `generated_at`,
  `base`, `head`, `results`, `summary` and `limitations`. The results are
  sorted by the identifier of the claim.
- The limitations of the verification are part of the document.

**Acceptance checks**
```bash
go run ./cmd/codixai verify --help
go run ./cmd/codixai --format json verify --base <ref> --claims <file>
```
The evidence in the repository is the test suite:
`TestVerifyJSONOutputIsACleanDocument` and
`TestVerifyContradictsTheFalseClaimAndVerifiesTheTrueOne`.

**Out of scope**
The `--staged` and the `--worktree` origins of the verification. The
verification compares the base with the head.

---

### P2-06 — The exit codes and the policy

- Status: **DONE**
- Depends on: P2-05

**Objective**
Map the result of the verification to the exit code convention, and let the
repository decide whether an unresolved claim is a failure.

**Sections to read**
9 (códigos de salida), 10, 14 (Phase 2).

**Deliverables**
- `0`: every claim is `verified` or `unresolved`.
- `3`: at least one claim is `contradicted`.
- `2`: a missing `--base` or `--claims`, or an invalid invocation.
- `1`: an execution or configuration error.
- The configuration key `verify.fail_on_unresolved`, in `internal/policies`:
  it is `false` by default, and an unresolved claim only fails when it is
  `true`.

**Acceptance checks**
```bash
go test ./internal/policies/ -run TestVerify
go test ./tests/integration/ -run 'TestVerifyExitsWithZeroWhenNoClaimIsContradicted|TestVerifyRequiresTheBaseAndTheClaims'
```

**Out of scope**
Any other configuration key of the verification.

---

### P2-07 — The corpus of the claims

- Status: **DONE**
- Depends on: P2-03, P2-04

**Objective**
A versioned corpus that covers every type of the catalog with every state
the type can reach, so that the behavior of the evaluator is documented and
reproducible.

**Sections to read**
15.

**Deliverables**
`testdata/claims/<case>/`, with a `change.diff`, a `claims.json`, an
`expected.json` that holds the expected state of every claim, and an
optional `repo/` directory with the manifests or the reports of the case.
`TestClaimCorpus` runs the cases and checks the coverage: every type needs
an `unresolved` case, and every type that can be decided also needs a
`verified` and a `contradicted` case.

**Acceptance checks**
```bash
go test ./internal/claims/ -run TestClaimCorpus
```

**Out of scope**
The corpus of real diffs produced by external agents, which is the task
P1-04 of Phase 1 and is still pending. **The corpus of the claims is
synthetic**: the diffs were written by hand and are not diffs produced by
agents. The two corpora must not be confused.

---

### P2-08 — The integration tests of the verification

- Status: **DONE**
- Depends on: P2-05, P2-06

**Objective**
Prove, over a real temporary Git repository, that the command behaves as the
contract says.

**Sections to read**
15, 18.2.

**Deliverables**
`tests/integration/verify_test.go`:
`TestVerifyContradictsTheFalseClaimAndVerifiesTheTrueOne`,
`TestVerifyJSONOutputIsACleanDocument`,
`TestVerifyExitsWithZeroWhenNoClaimIsContradicted`,
`TestVerifyRequiresTheBaseAndTheClaims`,
`TestVerifyLinksAnExternalReportToTheRevision`,
`TestVerifyLeavesAnUnlinkedReportUnresolved` and
`TestVerifyDoesNotModifyTheRepository`.

**Acceptance checks**
```bash
go test ./tests/integration/
```

**Out of scope**
The tests of the macOS and the Windows builds. Only Linux was exercised in
the development environment.

---

### P2-09 — The architecture test of the application layer

- Status: **DONE**
- Depends on: P0-13 (the same deliverable, declared in Phase 0)

**Objective**
Verify, from the first phase, that the application layer depends on neither
the presentation nor a desktop toolkit, and that the domain depends on
nothing of the project.

**Sections to read**
7.1, 8.7.1.

**Deliverables**
`internal/application/architecture_test.go`:
`TestApplicationDoesNotDependOnThePresentation` and
`TestDomainImportsNothingOfTheProject`.

**Acceptance checks**
```bash
go test ./internal/application/ -run TestApplicationDoesNotDependOnThePresentation
go test ./internal/application/ -run TestDomainImportsNothingOfTheProject
```

**Out of scope**
The user interface itself, which belongs to Phase 7.

---

## The self check of the phase

The change of the phase was audited with the tool itself, over the range
that holds it:

```bash
codixai audit --diff --base bb5b1bd
```

The run reports **0 findings over 109 changed files**. The absence of
findings does not prove that the change is correct, but no rule of the
catalog is violated by the phase.

## What is not done

These are the honest limits of the phase. They are not tasks in progress.

- **`api_unchanged` and `behavior_preserved` are always `unresolved`.** The
  public API is not extracted yet, and static analysis does not decide the
  behavior of a change. They never become `verified`.
- **The `scope` and the `files_unchanged` claims ignore the excluded
  paths.** The claims are checked against the whole compared range: a path
  that the configuration excludes from the analysis is not excluded from the
  claims. The limitation is published in the document of every
  verification.
- **The corpus of the claims is synthetic.** The diffs were written by hand.
  The corpus of real diffs produced by external agents is P1-04, and it is
  still pending.
- **The report formats are limited to JUnit XML and `go test -json`.** Any
  other format leaves the claim `unresolved`.
- **The verification has no progress output** and no cancellation beyond the
  `context` of the process.
- **The verification does not use the findings of the rules.** The claims
  are contrasted with the diff, with the manifests and with the external
  reports. The specification mentions "the diff and the results of the
  rules"; only the first is implemented.
