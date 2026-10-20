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
  `audit`, `init`, `version` and `help`; the global flags `--root`,
  `--format`, `--no-color`, `--plain` and `--version`; the `audit` flags
  `--diff`, `--base`, `--staged`, `--worktree` and `--rules`; the `init` flag
  `--force`.
- **Exit code convention**: `0` success, `1` execution or configuration
  error, `2` invalid arguments, `3` findings that reach the configured
  `fail_on` threshold.
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
  implementation conventions, the task format with the Phase 0 and Phase 1
  task lists, and six architecture decision records.
- **Agent context package**: `AGENTS.md`, `opencode.json`, the read-only
  `codixai-reviewer` subagent and the `/task` and `/review` commands.
- **CI** in `.github/workflows/ci.yml`: a formatting job, `go vet`,
  `go test ./... -race`, a build matrix for Linux, macOS and Windows, and an
  explicit job for the support matrix validation.

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
- There is no `verify` command, no claim verification, no SARIF export, no
  baseline support and no MCP server.
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
