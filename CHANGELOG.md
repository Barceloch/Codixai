# Changelog

All notable changes to Codixai are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once a release is published.

**The project has not published a release yet.** The `0.1.0` entry below is
still unreleased and the versions in it describe the state of the default
branch.

## [Unreleased]

### Added

- **Module and CLI.** A Go module named `github.com/barceloch/codixai` that declares `go 1.27.0`
  and uses only the Go standard library, with no third-party dependency. The
  binary is built with `go build ./cmd/codixai`.
- **Command framework and command tree** in `internal/cli`: the commands
  `audit`, `verify`, `init`, `version` and `help`; the global flags `--root`,
  `--format`, `--no-color`, `--plain` and `--version`; the `audit` flags
  `--diff`, `--base`, `--staged`, `--worktree` and `--rules`; the `verify`
  flags `--base` and `--claims`; the `init` flag `--force`.
- **Exit code convention**: `0` success, `1` execution or configuration
  error, `2` invalid arguments, `3` findings that reach the configured
  `fail_on` threshold, or a claim of the verification that is
  `contradicted`.
- **Presentation layer** in `internal/cli/ui`: a centralized renderer, a
  table helper, `NO_COLOR` detection, terminal detection, and a plain mode
  that removes every decoration.
- **Versioned output contract.** Every JSON document carries
  `schema_version` (`0.1`); the tool version is stamped on the document.
- **Domain model** in `internal/domain`: `Finding`, `Evidence`, `Rule`,
  `AnalysisRun`, `Claim`, `ClaimResult`, `VerificationRun`, `Relation`,
  `Symbol`, `File`, `Repository`, `Diagnostic`, `Location`, `Severity`,
  `SupportLevel`, `EpistemicStatus` and `Resolution`.
- **Application layer** in `internal/application`: the audit use case and the
  `ChangeSource` and `DiffAnalyzer` ports.
- **Read-only Git integration** in `internal/integrations/git`: argument-slice
  invocation, environment scrubbing, `--no-ext-diff --no-textconv`,
  `-c core.fsmonitor=false`, `-c diff.external=`,
  `-c protocol.ext.allow=never`, `--no-optional-locks` on Git 2.15 and
  later, `--end-of-options` on Git 2.24 and later, base-ref validation and
  reachability checks, a unified diff parser and C-style path unquoting.
- **Configuration** in `internal/policies`: a loader for `.codixai.yml` (and
  `.codixai.yaml`) with a reduced YAML subset parser, documented defaults,
  validation with file and line in the error message, per-rule overrides,
  explicit suppressions and glob exclusion matching.
- **Language layer** in `internal/languages`: a registry, a shared
  line-level lexical scanner, and one package per language (`go`, `python`,
  `php`, `javascript`, `typescript`) with its profile, its standard library
  list and its manifest reader.
- **Analysis engine** in `internal/analysis` and the rule catalog in
  `internal/analysis/rules`.
- **Six rules**, all at the `syntactic` support level:
  `CDX-IMP-001` (error), `CDX-REF-001` (error, Go only),
  `CDX-TST-001` (warning), `CDX-SUP-001` (warning),
  `CDX-ERR-001` (warning) and `CDX-PLH-001` (warning).
- **Fixture corpus** in `testdata/rules`: 58 cases, with at least one
  positive and one negative case per rule and per supported language, each
  holding a diff and an expected result.
- **Integration tests** in `tests/integration`: an end-to-end audit over a
  temporary Git repository, the JSON output contract, the exit codes, the
  absence of modifications to the analyzed repository, and the absence of
  writes outside of it.
- **Documentation**: the README, the contributing guide, the code of conduct,
  the security policy, the changelog, the architecture overview, the
  implementation conventions, the CI integration guide, the task format with
  the Phase 0, Phase 1, Phase 2, Phase 3 and Phase 7 task lists, and ten
  architecture decision records.
- **Agent context package**: `AGENTS.md`, `opencode.json`, the read-only
  `codixai-reviewer` subagent and the `/task` and `/review` commands.
- **CI** in `.github/workflows/ci.yml`: a formatting job, `go vet`,
  `go test ./... -race`, a build matrix for Linux, macOS and Windows, and an
  explicit job for the support matrix validation.

### Added — Phase 2, the verification of claims

- **`codixai verify --base <ref> --claims <file>`.** The command compares the
  claims of an agent with the changes between the base revision and the head
  of the repository, and it reports one state per claim: `verified`,
  `contradicted` or `unresolved`. Both flags are required; a missing one is
  an invalid invocation with the exit code `2`.
- **The claim contract and its decision** in
  [`docs/decisions/0007-claim-format-and-report-linking.md`](docs/decisions/0007-claim-format-and-report-linking.md).
  The claim file is a versioned document: `schema_version` is required and is
  `0.1`, `base` is optional and has to match the base of the comparison, and
  every claim needs a unique `id` and a `type` of the catalog. An unknown
  type or an unknown field is an error, never an unresolved claim.
- **The claim catalog** in `internal/claims`: `files_unchanged`, `scope`,
  `tests_added`, `tests_modified`, `dependency_unchanged`,
  `dependency_added`, `api_unchanged`, `tests_passed` and
  `behavior_preserved`. The evaluator computes every state from the diff, the
  manifests of the repository and the external reports.
- **The readers of the external test reports** in `internal/claims`: JUnit
  XML (`.xml`) and the JSON stream of `go test -json` (`.json`), read with the
  standard library. A report is linked to the revision under analysis when
  the claim declares the `report_revision` or when the report declares the
  revision itself. An unlinked report leaves the claim `unresolved`, and the
  evidence always states that the provenance of the report is external.
  **Codixai never runs the tests of the analyzed repository.**
- **The verification document**: a versioned JSON document with
  `schema_version`, `tool_version`, `generated_at`, `base`, `head`,
  `results`, `summary` and `limitations`, sorted by the identifier of the
  claim.
- **The exit code policy**: `3` when a claim is `contradicted`; an
  `unresolved` claim only fails when the configuration of the repository sets
  `verify.fail_on_unresolved: true`, which is not the default.
- **The corpus of the claims** in `testdata/claims`: one case per type and
  per state the type can reach, with a diff, a claim file and the expected
  result. The corpus is synthetic: the diffs are written by hand and are not
  diffs produced by external agents.
- **The architecture test** `internal/application/architecture_test.go`: it
  verifies that the application layer imports neither the command line nor
  the user interface nor a desktop toolkit, and that the domain imports no
  package of the project. It is the requirement 8.7.1 of the specification,
  and it holds from the first phase.

### Added — Phase 3, the baseline and the SARIF export

- **The stable fingerprint of a finding** in `internal/baseline`: the first 16
  hexadecimal characters of
  `sha256(rule_id + 0x00 + path + 0x00 + normalized_context)`. The context is
  the text of the line the finding points at, taken from the diff, with the
  leading and the trailing spaces removed and every run of internal whitespace
  collapsed into a single space. The new side of the diff is read first and
  the old side only for a removed line. The line number is never part of the
  identity, so the baseline survives the movement of the code.
- **The versioned baseline** in `internal/baseline`: its own `schema_version`
  (`0.1`), `created_at`, `tool_version` and the findings sorted by their
  fingerprint. A missing file is an empty baseline; a malformed document, an
  unsupported version and an entry without an identity are errors that name
  the file, the line and the way to recover. A path that escapes the root of
  the repository is rejected.
- **The baseline in the audit**: the audit reads the baseline of the
  `audit.baseline` key of the configuration (`.codixai/baseline.json` by
  default), removes the findings it knows and fails only on the new ones. The
  count of the known findings is a diagnostic of the standard error. **The
  document of the audit keeps `schema_version` `0.1`**: the known findings are
  not part of it.
- **`codixai baseline create`**: it records the findings of a compared range in
  the versioned file of the repository. One origin (`--base`, `--staged` or
  `--worktree`), `--out <path>`, and `--force` to replace an existing
  baseline, which is never replaced silently. The baseline that is being
  created is never applied to the run that creates it.
- **`codixai report`** and **the SARIF 2.1.0 export** in `internal/report`:
  one run, the driver `codixai`, the rules of the catalog sorted by identifier
  and one result per new finding. The severity mapping is `error` → `error`,
  `warning` → `warning`, `info` → `note`, and a severity outside the catalog
  becomes `none`. The fingerprint of the baseline travels in
  `partialFingerprints["codixaiFingerprint/v1"]` and the findings that the
  baseline knows are counted in `runs[].properties.suppressedByBaseline`.
  `--format sarif` is the default of the command and `--format json` writes
  the audit document, byte for byte. The command shares the exit code of the
  audit.
- **The CI integration guide** in
  [`docs/guides/ci-integration.md`](docs/guides/ci-integration.md): the origin
  flag, the exit codes, the baseline workflow and the import of the SARIF
  document.
- **Three architecture decision records**:
  [`0008`](docs/decisions/0008-baseline-and-fingerprint.md) (the baseline and
  the fingerprint), [`0009`](docs/decisions/0009-sarif-export.md) (the SARIF
  export) and
  [`0010`](docs/decisions/0010-distribution-policy-of-the-variants.md) (the
  distribution policy of the variants).

### Changed

- **The specification is adopted in its version 0.6.0.** It adds the local
  graphical interface (Wails, Svelte and daisyUI) as Phase 7, the
  requirements that hold from Phase 0, the rules of the section 17.9 and the
  pending decisions 11 to 14. The architecture of the project already
  satisfies the requirements of the section 8.7; the interface itself does
  not exist.
- **The license is the MIT License**, confirmed by the maintainer on
  2026-10-20 and recorded in the ADR 0004. The `LICENSE` file holds its
  verbatim text.
- **The module path is `github.com/barceloch/codixai`**, in lower case, and
  the repository is `https://github.com/Barceloch/Codixai`.
- **The distribution policy of the variants is fixed** by the decision
  [`0010`](docs/decisions/0010-distribution-policy-of-the-variants.md): the
  published artifact is a single binary that holds the CLI and the graphical
  interface, built with the `ui` tag, while the CLI only build stays the
  default of `go build` and of `go install`. The decision is provisional on
  the measurement of the size and the startup time of the combined binary,
  and the supported platforms of the variant with the interface are still
  pending. Nothing of the interface is published yet.

### Known limitations

- The analysis is syntactic and non-semantic. No rule is declared
  `resolved`, and no relation is resolved across packages or modules.
- `CDX-REF-001` reports identifiers that are declared in another package of
  the module as missing local symbols. It is a signal, not a confirmed
  defect. The predeclared identifiers of the language and the packages of
  the module are resolved since the version 2 of the rule.
- `CDX-TST-001` reports a test that is skipped at run time, including the
  conditional skips that skip a test when a tool is missing. It is a signal,
  not a confirmed defect.
- The manifests are read with reduced parsers and only the root of the
  repository is considered. Workspace modules and vendored dependencies are
  not resolved.
- Most rules only inspect the lines that the change adds. A defect that the
  change does not touch is out of scope.
- There is no persistent index, no symbol graph, no `index`, `status`,
  `overview`, `find`, `refs` or `deps` command.
- The baseline is one versioned file of the repository: there is no baseline
  per origin, no remote baseline and no `baseline show`, `prune` or `update`
  command. A finding whose line is not part of the compared range is
  identified by the rule and the path only, which is coarser than the
  identity of a finding with a context.
- The SARIF document is structurally conformant to SARIF 2.1.0 and it is not
  validated against the official JSON schema: the environment has no network
  access to fetch it. No import into a code scanning platform was exercised.
- The rule catalog publishes no human readable name, so the SARIF descriptors
  carry the rule identifier in `name` and the same text in
  `shortDescription` and `fullDescription`.
- There is no MCP server.
- The `api_unchanged` and the `behavior_preserved` claims are always
  `unresolved`: the public API is not extracted yet, and static analysis does
  not decide the behavior of a change. They are part of the catalog on
  purpose, and they never become `verified`.
- A `tests_passed` claim is only decided by a report that declares the
  revision under analysis. A report without a revision leaves the claim
  `unresolved`, even when it is honest. The reports are external evidence:
  Codixai did not produce them and cannot vouch for them.
- The `scope` and the `files_unchanged` claims are checked against the whole
  compared range: the paths that the configuration excludes from the analysis
  are not excluded from the claims.
- The corpus of the claims is synthetic. The diffs were written by hand and
  are not diffs produced by external agents.
- There is no graphical interface: no `codixai ui` command, no Wails adapter
  and no Svelte frontend. The interface belongs to Phase 7, and the `ui`
  build variant does not exist yet.
- Only Linux was exercised in the development environment. The macOS and
  Windows builds are defined in CI but their results have not been observed.
- No release has been published: there is no tag, no packaged binary and no
  measured performance or precision figure.
- The security contact address is pending maintainer confirmation. The
  repository URL, the module path and the license are confirmed: the
  repository is `https://github.com/Barceloch/Codixai`, the module path is
  `github.com/barceloch/codixai` and the license is the MIT License.

## [0.1.0 - Unreleased]

The first public milestone. It groups the executable foundations of Phase 0
and the diff verifier spike of Phase 1. It has not been tagged yet.

### Added

- The `audit --diff` command with the three origins `--base <ref>`,
  `--staged` and `--worktree`, in read-only mode, without a checkout and
  without a persistent index.
- The five target languages with their acceptance fixtures: Go, Python, PHP,
  JavaScript and TypeScript.
- The support matrix published by `codixai audit --rules` and validated
  against the fixture corpus in CI.
- The JSON output contract with `schema_version`, a clean standard output
  and the diagnostics on the standard error.

### Known limitations

See the [Unreleased](#unreleased) section above: the corpus of real diffs
produced by external agents, the false-positive report and the Phase 1
decision gate are still pending.
