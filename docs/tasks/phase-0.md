# Phase 0 — Executable foundations and technical decisions

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 0". This file turns them into atomic
tasks with executable acceptance checks.

**Phase exit criterion (specification):** a clean clone compiles, the tests
run without external services, and the dependency decisions are documented.
Creating empty directories is not enough.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P0-01 | Go module and CLI skeleton | DONE |
| P0-02 | `version`, `--help` and safe `init` | DONE |
| P0-03 | Domain model and `schema_version` | DONE |
| P0-04 | Read-only Git integration | DONE |
| P0-05 | Configuration of the analyzed repository | DONE |
| P0-06 | Presentation layer | DONE |
| P0-07 | Fixture corpus and profiles for five languages | DONE |
| P0-08 | Documentation and ADRs | DONE |
| P0-09 | Continuous integration | PARTIAL |
| P0-10 | Agent context package | DONE |
| P0-11 | Parsing strategy ADR | PENDING / blocked |
| P0-12 | Module path, repository URL and license | DONE |
| P0-13 | The architecture test of the application layer | DONE |

One item is not fully closed and is stated as such: the parsing strategy
ADR is **blocked** (P0-11 below). The module path, the repository URL and
the license (the MIT License) were confirmed by the maintainer on 2026-10-20
and applied.

P0-13 holds the requirement 8.7.1 of the specification: the application
layer does not depend on the presentation, and an architecture test
verifies it. It holds from the first phase, before the graphical interface
of Phase 7 exists.

---

### P0-01 — Go module and CLI skeleton

- Status: **DONE**
- Depends on: none

**Objective**
A Go module that builds a single binary and a command tree that can grow
without a rewrite.

**Sections to read**
4, 7, 9, 14 (Phase 0), 17.

**Deliverables**
- `go.mod` with `module codixai` and `go 1.27.0`, and no third party
  dependency.
- `cmd/codixai/main.go`, the process entry point.
- `internal/cli/framework.go`, the command framework, and
  `internal/cli/commands.go`, the whole command tree.

**Acceptance checks**
```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .   # prints nothing
```

**Out of scope**
Cobra and the Charm libraries, which are unavailable without network access.
The substitution is recorded in
[`docs/decisions/0003-cli-framework.md`](../decisions/0003-cli-framework.md).

---

### P0-02 — `version`, `--help` and safe `init`

- Status: **DONE**
- Depends on: P0-01

**Objective**
The three commands that a user needs to check that the tool works and to
configure a repository.

**Sections to read**
9, 10, 13, 14 (Phase 0), 18.3.

**Deliverables**
- `codixai version`, with the text and the JSON forms.
- `codixai --help`, `codixai help <command>` and `--help` on every command.
- `codixai init`, which creates `.codixai.yml` and never overwrites an
  existing file unless `--force` is given.
- The exit code convention, in `internal/cli/framework.go`.

**Acceptance checks**
```bash
go run ./cmd/codixai version
go run ./cmd/codixai --help
go run ./cmd/codixai init --root <empty-directory>
go run ./cmd/codixai init --root <empty-directory>   # reports that it exists, does not overwrite
```
The evidence in the repository is the test suite:
`TestHelpListsTheCommandsAndTheGlobalFlags`,
`TestVersionCommandPrintsTheVersionAndTheSchemaVersion`,
`TestInitCreatesTheConfigurationAndNeverOverwritesIt`,
`TestInitWithForceReplacesTheConfiguration`,
`TestUnknownCommandFailsWithTheInvalidArgumentsCode`.

**Out of scope**
The `index`, `status`, `overview`, `find`, `refs` and `deps` commands, which
belong to Phase 5.

---

### P0-03 — Domain model and `schema_version`

- Status: **DONE**
- Depends on: P0-01

**Objective**
Stable, transport agnostic models, and a versioned output contract.

**Sections to read**
0 (glossary), 1.2, 4, 7.2, 9.

**Deliverables**
- `internal/domain/models.go`: `Finding`, `Evidence`, `Rule`, `AnalysisRun`,
  `Claim`, `ClaimResult`, `VerificationRun`, `Relation`, `Symbol`, `File`,
  `Repository`, `Diagnostic`, `Location`, `Severity`, `SupportLevel`,
  `EpistemicStatus` and `Resolution`.
- `domain.SchemaVersion`, currently `0.1`.
- The distinction between `epistemic_status` (finding) and `resolution`
  (relation).

**Acceptance checks**
```bash
go test ./internal/analysis/ -run 'TestRunIsSerializable|TestRunIsDeterministic'
go test ./internal/analysis/rules/ -run TestRuleIdentifiersFollowTheConvention
```

**Out of scope**
The `verify` command and the claim contract of Phase 2. The models exist;
the use case does not.

---

### P0-04 — Read-only Git integration

- Status: **DONE**
- Depends on: P0-03

**Objective**
Collect the changes of a repository without executing anything that the
repository controls, and without modifying it.

**Sections to read**
9, 13, 14 (Phase 1), 4.

**Deliverables**
- `internal/integrations/git/source.go`: the hardened invocation, the base
  ref validation, the environment scrubbing, the version detection.
- `internal/integrations/git/diff.go`: the unified diff parser and the C
  style path unquoting.
- `internal/integrations/git/language.go`: the language of a path.
- The implementation of the `application.ChangeSource` port.

**Acceptance checks**
```bash
go test ./internal/integrations/git/
go test ./tests/integration/ -run 'TestAuditDoesNotModifyTheRepository|TestBinaryDoesNotWriteOutsideOfTheRepository'
```
The evidence in the repository is the test suite of the package:
`TestBaseArgsNeutralizeRepositoryConfiguration`, `TestDiffArgsAreReadOnly`,
`TestDiffArgsRejectInvalidBaseRef`, `TestGitEnvRemovesRedirectingVariables`,
`TestSourceInvocationsAreReadOnly`,
`TestSourceDoesNotRunRepositoryCommands`, `TestUnquotePath` and the ten
`TestParseUnifiedDiff*` tests.

**Out of scope**
The index and the comparison of the blobs of two revisions, which belong to
Phase 5.

---

### P0-05 — Configuration of the analyzed repository

- Status: **DONE**
- Depends on: P0-01

**Objective**
A documented, validated configuration file with defaults, per rule
overrides and explicit suppressions.

**Sections to read**
10, 11, 13, 4.

**Deliverables**
- `internal/policies/policies.go`: the model, the defaults and the loader.
- `internal/policies/yaml.go`: the reduced YAML subset parser.
- `internal/policies/glob.go`: the exclusion matcher.
- `.codixai.yml` and `.codixai.yaml` are both accepted.

**Acceptance checks**
```bash
go test ./internal/policies/
```
The evidence in the repository is the test suite: `TestLoadWithoutFileReturnsDefaults`,
`TestLoadValidFile`, `TestLoadMalformedFile`, `TestRuleOverrides`,
`TestSuppressions`, `TestExcludedUsesTheConfiguredPatterns`,
`TestLoadAlternativeFileName` and the six `TestParseYAML*` tests.

**Out of scope**
The baseline file, which belongs to Phase 3. The `audit.baseline` key is
parsed and validated, but no file is read.

---

### P0-06 — Presentation layer

- Status: **DONE**
- Depends on: P0-01, P0-03

**Objective**
A centralized, consistent and accessible terminal output, with a clean JSON
alternative.

**Sections to read**
9, 10, 4, 14 (Phase 0).

**Deliverables**
- `internal/cli/ui/renderer.go`: the centralized styles, the color
  detection and the plain mode.
- `internal/cli/ui/table.go`: the table helper.
- `internal/cli/report.go`: the text report of the audit and of the rule
  catalog.
- `--no-color`, the `NO_COLOR` environment variable, `--plain` and the
  automatic deactivation of the color when the output is not a terminal.

**Acceptance checks**
```bash
go test ./internal/cli/... 
go run ./cmd/codixai --no-color audit --rules
go run ./cmd/codixai --format json audit --rules | head -5
```
The evidence in the repository is the test suite:
`TestRendererDisablesColorWhenTheDestinationIsNotATerminal`,
`TestRendererDisablesColorWithTheNoColorFlag`,
`TestRendererDisablesColorWithTheNoColorEnvironmentVariable`,
`TestPlainModeRemovesEveryDecoration`, `TestNoColorRemovesEveryEscapeSequence`,
`TestSeverityLabelIsAlwaysTextual`, `TestTableAlignsTheColumns`,
`TestJSONOutputHoldsOnlyTheDocument` and
`TestNoColorEnvironmentVariableRemovesTheEscapeSequences`.

**Out of scope**
Lip Gloss, Glamour, Bubbles and Chroma, and any interactive TUI. The
specification allows a minimum in Phase 1.

---

### P0-07 — Fixture corpus and profiles for five languages

- Status: **DONE**
- Depends on: P0-03, P0-04

**Objective**
The five target languages are recognized from the first milestone, with a
profile, a manifest reader and acceptance fixtures.

**Sections to read**
5, 7, 11, 15.

**Deliverables**
- `internal/languages/registry/registry.go`: the registry.
- `internal/languages/lexical/`: the shared line level scanner and the
  profile type.
- `internal/languages/{go,python,php,javascript,typescript}/`: the profile,
  the standard library list and the manifest reader of each language.
- `testdata/rules/`: 58 cases, with at least one positive and one negative
  case per rule and per supported language.

**Acceptance checks**
```bash
go test ./internal/languages/...
go test ./internal/analysis/ -run TestFixtureCorpus
go run ./cmd/codixai audit --rules
```
The evidence in the repository is the test suite: `TestDetect`, `TestLanguages`,
`TestProfileFor`, `TestProfileForPath` in the registry, the five `TestScanner*` tests and the
`TestMatchMarkerRequiresBothPatterns` and `TestMatchPath` tests in the
scanner, and the per language tests such as `TestReadManifest`,
`TestExtractImports`, `TestEmptyHandlers` and
`TestFindUndefinedReferences`.

**Out of scope**
The semantic level 2 and the level 3 of the specification. The support
level of every rule stays `syntactic`.

---

### P0-08 — Documentation and ADRs

- Status: **DONE**
- Depends on: P0-01

**Objective**
The public files that the specification requires, and the decisions that
shape the project.

**Sections to read**
16, 6, 14 (Phase 0), 17.

**Deliverables**
- `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`,
  `CHANGELOG.md`.
- `docs/architecture/overview.md`.
- `docs/guides/implementation-conventions.md`.
- `docs/decisions/0001-...` through `docs/decisions/0006-...`.
- The issue and the pull request templates.

**Acceptance checks**
```bash
go test ./...
ls docs/decisions/
```
Every path that the documents reference has to exist.

**Out of scope**
The `LICENSE` file, which is pending the confirmation of the maintainer.

---

### P0-09 — Continuous integration

- Status: **PARTIAL**
- Depends on: P0-01, P0-03, P0-07

**Objective**
A CI that compiles, vets, tests and formats the project without
credentials and without external services.

**Sections to read**
15, 16, 18.2.

**Deliverables**
- `.github/workflows/ci.yml`: the formatting job, `go vet`, `go test ./...
  -race`, the build matrix for Linux, macOS and Windows, and the explicit
  job for the support matrix validation.

**Acceptance checks**
```bash
gofmt -l .
go vet ./...
go test ./... -race
go test ./internal/analysis/ -run TestSupportMatrixMatchesFixtures
```
These commands have been executed locally on Linux. **The macOS and the
Windows jobs of the matrix have not been observed**: the workflow declares
them, but a declared job is not evidence.

**Out of scope**
The publishing of the releases, the reproducible binaries and the SARIF
upload, which belong to Phase 3.

---

### P0-10 — Agent context package

- Status: **DONE**
- Depends on: P0-08

**Objective**
A package that an agent can load to work on the project: the invariants,
the conventions, the task format and a read-only reviewer.

**Sections to read**
17, 17.2, 17.3, 17.6.

**Deliverables**
- `AGENTS.md`, at most about 100 lines.
- `opencode.json`, with the `instructions` array and the `agent` entry of
  the reviewer.
- `.opencode/agents/codixai-reviewer.md`.
- `.opencode/commands/task.md` and `.opencode/commands/review.md`.
- `docs/tasks/README.md`, `docs/tasks/phase-0.md` and
  `docs/tasks/phase-1.md`.

**Acceptance checks**
```bash
opencode --version          # the format was validated against v2.0.32
wc -l AGENTS.md
```
The discovery of the agents and of the commands, and the loading of
`AGENTS.md`, have to be re-checked when OpenCode is upgraded (section
17.6).

**Out of scope**
The MCP integration, which belongs to Phase 4.

---

### P0-11 — Parsing strategy ADR (blocked)

- Status: **PENDING / blocked**
- Depends on: none

**Objective**
Close the decision that the specification places on the critical path of
Phase 1: Tree-sitter through CGO, against the WASM runtime and the other
alternatives, with test builds on Linux, macOS and Windows.

**Sections to read**
5, 6.1, 6.2, 13, 14 (Phase 0), 14 (Phase 1).

**Deliverables**
- `docs/decisions/0001-parsing-strategy-for-multiple-languages.md`,
  currently with the status *Proposed / blocked*.

**Acceptance checks**
Not executable today. The decision needs:
1. the license review of the grammars that would be vendored;
2. the choice between CGO and WASM, with its consequences;
3. the build evidence on Linux, macOS and Windows.

**Out of scope**
The implementation of Tree-sitter, which is a Phase 1.5 task.

**Why it is blocked.** The development environment has no network access,
so the grammars and the bindings cannot be fetched, and only Linux is
available, so the macOS and the Windows builds cannot be produced. The
decision must not be closed with a plausible argument in place of the
required evidence.

---

### P0-12 — Module path, repository URL and license (done)

- Status: **DONE**
- Depends on: none

**Objective**
Confirm the module path, the repository URL, the organization and the
license (Apache-2.0 is the proposal) and apply the checklist of
[`docs/decisions/0004-module-path-repository-and-license.md`](../decisions/0004-module-path-repository-and-license.md).

**Acceptance checks**

```bash
grep '^module ' go.mod                           # module github.com/barceloch/codixai
grep -rn '"codixai/internal' --include='*.go' . # no result
grep -n 'github.com/barceloch/codixai' README.md CONTRIBUTING.md
test -f LICENSE                                  # the MIT License
go build ./... && go vet ./... && go test ./...
```

The maintainer confirmed the module path, the repository URL and the license
(MIT) on 2026-10-20. The security contact **address** is still pending and
belongs to the same decision list.

**Out of scope**
Publishing a release.

---

### P0-13 — The architecture test of the application layer

- Status: **DONE**
- Depends on: P0-01, P0-10

**Objective**
Verify, from the first phase, that the application layer depends on neither
the command line nor the user interface nor a desktop toolkit, and that the
domain depends on nothing of the project. The requirement is the 8.7.1 of
the specification, and it holds before the graphical interface of Phase 7
exists.

**Sections to read**
7.1, 8.7.1.

**Deliverables**
- `internal/application/architecture_test.go`: the test parses the imports of
  the files of `internal/application` and of `internal/domain` and fails on
  a forbidden one.
- `TestApplicationDoesNotDependOnThePresentation`: the application layer
  imports neither `internal/cli`, nor `internal/ui`, nor `cmd`, nor
  `github.com/wailsapp/wails`.
- `TestDomainImportsNothingOfTheProject`: the domain imports no package of
  the project.

**Acceptance checks**
```bash
go test ./internal/application/ -run TestApplicationDoesNotDependOnThePresentation
go test ./internal/application/ -run TestDomainImportsNothingOfTheProject
```

**Out of scope**
The user interface itself, which belongs to Phase 7 and does not exist. The
test forbids an import of a package that does not exist yet, so it does not
need it to be written.
