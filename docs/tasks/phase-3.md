# Phase 3 — CI adoption: the baseline and the SARIF export

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 3". This file turns them into atomic tasks
with executable acceptance checks.

**Phase exit criterion (specification):** an example project can run Codixai
in CI, import the SARIF document in a compatible platform and detect only the
new findings according to the configured baseline.

**The phase is implemented.** The fingerprint, the baseline file, the
application of the baseline in the audit, the `baseline create` command, the
SARIF 2.1.0 export, the `report` command and the CI guide exist. The work is
committed in `7637d73` (the baseline and the fingerprint) and `440fb27` (the
SARIF export and the report command).

The identity of a finding is shared by the baseline and by the export: the
fingerprint of the decision
[`0008`](../decisions/0008-baseline-and-fingerprint.md) travels in
`partialFingerprints["codixaiFingerprint/v1"]` of the SARIF document, so a
platform that suppresses a result and a pipeline that applies a baseline use
the same value.

The two limits of the phase are declared and not hidden:

- **The SARIF document is not validated against the official JSON schema.**
  The development environment has no network access to fetch it. The
  structure that the decision 0009 fixes is validated by the tests, and the
  official schema is not. See "What is not verified".
- **No import into a real code scanning platform was exercised.** The
  document and the exit codes were checked against a temporary Git
  repository, not against a hosted platform.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P3-01 | The stable fingerprint of a finding | DONE |
| P3-02 | The versioned baseline and the audit integration | DONE |
| P3-03 | The `baseline create` command | DONE |
| P3-04 | The SARIF 2.1.0 export and the `report` command | DONE |
| P3-05 | The CI integration guide | DONE |

---

### P3-01 — The stable fingerprint of a finding

- Status: **DONE**
- Depends on: P1-01, P1-02

**Objective**
The identity of a finding is the rule, the path and a normalized context
taken from the diff, never the line number, so that a baseline survives the
movement of the code, which is the normal case in a repository with activity.

**Sections to read**
1.2, 4, 14 (Phase 3), 15; and the decision
[`0008`](../decisions/0008-baseline-and-fingerprint.md).

**Deliverables**
- `internal/baseline/baseline.go`: `Fingerprint`, `Context` and `Entries`.
  The fingerprint is the first 16 hexadecimal characters of the SHA-256 of
  `rule_id + 0x00 + path + 0x00 + normalized_context`. The normalization
  removes the leading and the trailing spaces and collapses every run of
  internal whitespace into a single space. The new side of the diff is read
  first, the old side only when the new side holds no line with that number,
  which is the case of a removed line.
- `internal/baseline/baseline.go`: the versioned document (`SchemaVersion`
  `0.1`, `created_at`, `tool_version`, `findings`), `Load`, `Save` and
  `Resolve`. The entries are sorted by their fingerprint and are recorded
  once. A missing file is an empty baseline; a malformed document names the
  file and the line of the problem; an unsupported version or an entry
  without a fingerprint or a rule identifier names the file, the entry and
  the flag that recreates the document. A path that escapes the root of the
  repository is rejected.
- The decision
  [`0008`](../decisions/0008-baseline-and-fingerprint.md).

**Acceptance checks**
```bash
go test ./internal/baseline/ -count=1   # ok, 18 tests
go vet ./...                            # exit 0
```
The evidence in the repository is the test suite:
`TestFingerprintIsStableAndHexadecimal`,
`TestFingerprintIgnoresTheLineNumber`,
`TestFingerprintChangesWhenTheLineChanges`,
`TestFingerprintWithoutContextIdentifiesTheRuleAndThePath`,
`TestContextNormalizesTheWhitespace`,
`TestContextReadsTheOldSideOfARemovedLine`,
`TestContextPrefersTheNewSideOfAModifiedLine`,
`TestFingerprintChangesWhenTheAddedLineChangesBesideARemovedLine`,
`TestContextIgnoresAnotherFile`, `TestEntriesAreSortedAndUnique`,
`TestSaveAndLoadRoundTrip`, `TestLoadAcceptsAMissingFile`,
`TestLoadRejectsAMalformedFile`,
`TestLoadRejectsAnotherSchemaVersion`,
`TestLoadRejectsAnEntryWithoutIdentity`,
`TestSaveRejectsAnotherSchemaVersion`,
`TestKnowsRequiresAnIdentifiableContext` and
`TestResolveKeepsTheBaselineInsideTheRepository`.

**Out of scope**
The line number as part of the identity, a hash of the whole file, the
message of the finding inside the identity, and the application of the
baseline inside the analysis engine. The reasons are recorded in the
decision 0008.

---

### P3-02 — The versioned baseline and the audit integration

- Status: **DONE**
- Depends on: P3-01, P1-01

**Objective**
The audit reads the baseline of the repository, stops reporting the findings
that the baseline knows and keeps failing on the new ones, without changing
the shape of its document.

**Sections to read**
9, 10, 14 (Phase 3); and the decision
[`0008`](../decisions/0008-baseline-and-fingerprint.md).

**Deliverables**
- `internal/application/audit.go`: `applyBaseline`, applied after the policy.
  The path comes from the `audit.baseline` key of the configuration, which
  defaults to `.codixai/baseline.json` and is resolved against the root. A
  known finding is removed from the document and does not reach the
  threshold; it travels in `AuditResult.Known` and in
  `AuditResult.KnownFindings`.
- `AuditRequest.SkipBaseline`: the command that creates a baseline asks for
  the audit without the baseline, because a baseline never hides the findings
  it creates.
- `internal/cli/commands.go`: the count of the known findings is a diagnostic
  of the standard error (`N finding(s) known by the baseline, not reported`),
  next to the excluded files and the suppressed findings. The standard output
  keeps holding only the document.
- **The document of the audit keeps `schema_version` `0.1`.** The known
  findings are not part of the document, so no consumer visible field was
  added, removed or redefined.

**Acceptance checks**
```bash
go test ./...                 # 19 packages ok
go test ./tests/integration -count=1   # ok; the baseline and the report suites
```
The evidence in the repository is the test suite:
`TestRunAuditAppliesTheFailureThreshold`,
`TestRunAuditAppliesTheSuppressions` and the end-to-end
`TestBaselineCreateMakesTheKnownFindingsQuiet`, which asserts the exit code
`3` before the baseline, the exit code `0` and the count on the standard
error after it, the unchanged tree of the repository, the refusal to replace
an existing baseline without `--force`, and the exit code `3` again when the
line of a finding changes.

**Out of scope**
The storage of the baseline in a SQLite index: the baseline is a versioned
file of the analyzed repository and the index is a local, rebuildable cache
(specification, section 6.3). A baseline per origin, or a remote baseline.

---

### P3-03 — The `baseline create` command

- Status: **DONE**
- Depends on: P3-01, P3-02

**Objective**
`codixai baseline create` records the findings of a compared range in the
versioned file of the repository, with an explicit origin and an explicit
replacement.

**Sections to read**
9, 10, 14 (Phase 3), 15; and the decision
[`0008`](../decisions/0008-baseline-and-fingerprint.md).

**Deliverables**
- `internal/cli/baseline.go`: the `baseline` command and the `create`
  subcommand.
- The flags `--base <ref>`, `--staged` and `--worktree` (exactly one, like
  the audit), `--out <path>` and `--force`.
- The path is the `audit.baseline` key unless `--out` names another one; a
  relative path is resolved against the root and a path outside the root is
  rejected.
- The command never overwrites an existing baseline without `--force`, and
  the error names the file, the number of findings it records and the flag
  that replaces it. A baseline that cannot be read is replaced only with
  `--force`, and then every finding counts as new.
- The report of the run says how many findings were recorded and how many of
  them are new; in `--format json` the message goes to the standard error and
  the document to the standard output.
- The command writes one file of the repository and creates its directory
  when it is needed. It never modifies anything else.

**Acceptance checks**
```bash
go test ./tests/integration -count=1   # ok; the baseline suite is included
go test ./...                          # 19 packages ok
```
The evidence in the repository is the test suite:
`TestBaselineCreateMakesTheKnownFindingsQuiet`,
`TestBaselineCreateWithoutOriginFailsWithTheInvalidArgumentsCode` (exit code
`2` and an error that names `--base` and `--worktree`),
`TestBaselineSurvivesTheMovementOfTheCode`,
`TestBaselineFollowsTheConfiguredPath` and
`TestBaselineCreateForceReplacesAnUnreadableBaseline`.

**Out of scope**
`baseline show`, `baseline prune` and `baseline update`. The creation of the
baseline inside the pipeline, which would hide the review of the file. The
GPG signing of the baseline.

---

### P3-04 — The SARIF 2.1.0 export and the `report` command

- Status: **DONE**
- Depends on: P3-01, P1-01

**Objective**
`codixai report` writes the machine readable document of the same run that
the audit reports: SARIF 2.1.0 with the rules, the locations, the severity,
the epistemic status and the stable fingerprint of every new finding.

**Sections to read**
9, 14 (Phase 3), 15; and the decision
[`0009`](../decisions/0009-sarif-export.md).

**Deliverables**
- `internal/report/sarif.go`: one run, the driver `codixai` with its version
  and the catalog, `$schema`
  `https://json.schemastore.org/sarif-2.1.0.json` and `version` `2.1.0`.
- The severity mapping: `error` stays `error`, `warning` stays `warning`,
  `info` becomes `note`, and a severity outside the catalog becomes `none`,
  which is how SARIF says that a result fails nothing. The export never
  escalates a severity it does not know.
- `results[].partialFingerprints["codixaiFingerprint/v1"]`: the fingerprint
  of the decision 0008, computed from the path as the baseline stores it, so
  the two identities are the same value.
- `runs[].properties.suppressedByBaseline`: the count of the findings that
  the baseline knows. They are not results, because they are not new.
- `results[].properties`: `epistemic_status`, `support_level` and
  `rule_version`. `rules[].properties.support`: the level, the method and the
  limitations.
- The document is deterministic: the rules are sorted by their identifier and
  the results by path, line, rule and message. A rule that a result
  references and the catalog of the run does not describe is published with a
  minimal entry, so `ruleIndex` always points into the array. A finding
  without a line omits `region`, because SARIF has no line zero.
- `internal/cli/report_command.go`: the `report` command with `--base`,
  `--staged` and `--worktree`; `--format sarif` (the default of the command)
  writes the SARIF document and `--format json` writes the audit document,
  byte for byte the one that `audit --format json` writes. An explicit
  `--format text` is a usage error: the command has no text document. The
  exit code is the exit code of the audit.
- The decision
  [`0009`](../decisions/0009-sarif-export.md).

**Acceptance checks**
```bash
go test ./internal/report/ -count=1    # ok, 11 tests
go test ./tests/integration -count=1   # ok; the report suite is included
go test ./... -race                    # ok
```
The evidence in the repository is the test suite:
`TestDocumentDeclaresTheVersionAndTheSchema`,
`TestDriverDescribesTheTool`, `TestDriverFallsBackToTheVersionOfTheRun`,
`TestRulesAreSortedAndDescribeTheCatalog`, `TestResultMapsTheFinding`,
`TestLevelMapping`, `TestKnownFindingsTravelInTheRunProperties`,
`TestResultsAreSortedByPathLineAndRule`, `TestDocumentIsDeterministic`,
`TestFindingWithoutALineOmitsTheRegion`,
`TestResultWhoseRuleIsMissingGetsAPublishedEntry`,
`TestDocumentHasTheRequiredStructure`,
`TestReportWritesTheSarifDocumentOfTheFindings`,
`TestReportAppliesTheBaselineToTheDocument`,
`TestReportJSONIsTheAuditDocument` and
`TestReportWithoutOriginFailsWithTheInvalidArgumentsCode`.

**Out of scope**
The validation against the official JSON schema of SARIF, which needs network
access. SARIF 2.0.0. The upload to any platform. Including the known findings
as results.

---

### P3-05 — The CI integration guide

- Status: **DONE**
- Depends on: P3-02, P3-03, P3-04

**Objective**
A short guide that a pipeline author can follow: the origin flag, the exit
code convention, the baseline workflow and the import of the SARIF document,
with the limits of the export stated.

**Sections to read**
9, 14 (Phase 3), 16.

**Deliverables**
- `docs/guides/ci-integration.md`: the one origin flag, the exit codes of the
  audit and of the verification, the baseline workflow (`baseline create
  --base main` once, commit the file, review it when it changes, then
  `audit --diff --base origin/main`), an example pipeline and the honest
  limitation of the SARIF import.
- The pointer from the `README.md` and the entries of the `CHANGELOG.md`.

**Acceptance checks**
```bash
go test ./tests/integration -count=1   # ok; it proves the documented workflow
```
The document itself has no executable check: it describes the commands above
and their exit codes. The claims it makes about a code scanning platform are
not verified in this environment and are written as such.

**Out of scope**
A pipeline for a specific CI provider, the credentials of a platform, the
validation of the SARIF document against the official schema, and an example
project inside this repository.

---

## What is not verified

These are the honest limits of the phase. They are not tasks in progress.

- **The SARIF document is not validated against the official JSON schema.**
  The environment has no network access to fetch
  `https://json.schemastore.org/sarif-2.1.0.json`. The tests validate the
  structure that the decision 0009 fixes: the version, the run, the driver,
  the rules, the results, the locations and the fingerprints.
- **No import into a code scanning platform was exercised.** The guide states
  it, and a pipeline author should confirm the import once on their platform.
- **No example project runs the tool in a hosted CI.** The end-to-end proof
  is the temporary Git repository of `tests/integration`, not a hosted
  pipeline. The CI of this repository does not run `baseline create` nor
  `report`.
- **Only Linux was exercised.** The macOS and Windows builds are declared in
  the CI configuration and their results have not been observed.
