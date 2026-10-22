# Codixai

Codixai is a local-first command line tool that audits the changes of a Git
repository against a deterministic catalog of rules. It is built for one
audience in particular: developers who review work produced by AI coding
agents.

The tool reads a diff, applies the rules that support the language of every
changed file, and reports each finding with its evidence, its support level,
its epistemic status and its known limitations.

**Codixai never executes the code of the analyzed repository, never invokes a
package manager and never accesses the network.**

## Status

**Experimental. This repository is at the Phase 0 / Phase 1 milestone, plus
the verification of claims of Phase 2 and the CI adoption of Phase 3. It is
not a finished product.**

What exists today:

- a Go module named `github.com/barceloch/codixai` that builds a single binary;
- a CLI with the commands `audit`, `report`, `baseline create`, `verify`,
  `init`, `version` and `help`;
- a read-only Git integration with hardened invocation;
- a configuration file loader for `.codixai.yml`;
- six deterministic rules over five languages (Go, Python, PHP, JavaScript,
  TypeScript);
- a fixture corpus of 58 cases and end-to-end tests over a temporary
  repository;
- the `verify` command: it compares the claims of an agent with the changes
  of the repository and reports one state per claim, `verified`,
  `contradicted` or `unresolved`;
- the claim contract of the decision
  [`0007`](docs/decisions/0007-claim-format-and-report-linking.md), with a
  corpus of cases in `testdata/claims`;
- a versioned baseline of the findings of a repository, with a stable
  fingerprint that survives the movement of the code, and the
  `baseline create` command that writes it;
- the `report` command and the SARIF 2.1.0 export of the new findings, with
  the fingerprint of the baseline in `partialFingerprints`.

What does **not** exist today:

- no persistent index, no symbol graph, no `index`, `status`, `overview`,
  `find`, `refs` or `deps` commands;
- no MCP server;
- no semantic resolution: every rule is at most `syntactic`;
- no graphical interface, no `codixai ui` command and no `ui` build variant;
- no verification of the behavior of a change: the `api_unchanged` and the
  `behavior_preserved` claims are always `unresolved`;
- no released version and no published binary.

### License

**The license is the MIT License.** It was confirmed by the maintainer on
2026-10-20 and the text is in [`LICENSE`](LICENSE). The decision is recorded
in
[`docs/decisions/0004-module-path-repository-and-license.md`](docs/decisions/0004-module-path-repository-and-license.md).

## Why it exists

An agent can produce a large diff in seconds. Reviewing that diff well
requires answering questions that a diff alone does not answer:

- Does the change import something that is not declared in the manifest?
- Did the change delete or weaken a test?
- Did the change silence an analyzer instead of fixing the problem?
- Did the change leave an error unhandled or a stub in place?
- Are the claims of the agent about the change true?

Codixai answers a bounded version of that question deterministically, with
evidence, and states plainly what it could not check. It is a verifier, not a
chatbot: the same input always produces the same document, and every finding
points at a file, a line range and a rule.

## Install from source

The tool has no third-party dependencies: the whole program is written with
the Go standard library. A clone builds without network access.

Requirements: Go 1.27 or later (the module declares `go 1.27.0`).

```bash
git clone https://github.com/Barceloch/Codixai.git
cd Codixai
go build ./cmd/codixai
```

The module path is `github.com/barceloch/codixai`, in lower case. The Go
toolchain matches the module path against the repository path in a case
sensitive way, and GitHub serves the repository under both spellings, so the
lower case path is the one that the toolchain resolves without surprises.

This produces the `codixai` binary in the current directory. The version of
the binary is injected at build time:

```bash
go build -ldflags "-X main.version=0.1.0" ./cmd/codixai
```

Without the flag, the binary reports `dev`. You can also run it without
building:

```bash
go run ./cmd/codixai --help
```

## Quick start

Audit the uncommitted changes of the repository in the current directory:

```bash
codixai audit --diff --worktree
```

The text report has this shape (the values below are illustrative, the
structure is the real one):

```text
Codixai audit
  origin:            worktree
  revision:          head 7f1c0a4e1b2d3c4f5a6b7c8d9e0f1a2b3c4d5e6f
  root:              /path/to/repository
  files:             1 changed
  generated at:      2026-10-20T14:41:50Z
  tool:              codixai dev (schema 0.1)
  policy:            fail_on error

Findings (1)

  ERROR  CDX-IMP-001 v1  inferred
    path:              main.go:6
    message:           The import "github.com/acme/missing/pkg" is not declared in go.mod.
    evidence:          diff_hunk hunk:main.go:1
    support:           syntactic (manifest reading plus language specific import extraction)
    limitations:
      - Only the go.mod file of the repository root is read; ...
    recommendation:    Declare "github.com/acme/missing/pkg" in go.mod or remove the import.

Summary
  total:             1
  by severity:       error 1
  by rule:           CDX-IMP-001 1
  by language:       go 1
  by epistemic status: inferred 1

Limitations
  - The analysis is static and non semantic: ...
```

Other useful invocations:

```bash
codixai audit --diff --base main      # commits between the base and HEAD
codixai audit --diff --staged         # the staging area
codixai audit --rules                 # print the rule catalog and its support matrix
codixai --format json audit --diff --worktree   # a machine readable document
codixai report --format sarif --base main        # the SARIF document of the same run
codixai baseline create --base main   # record the current findings in the baseline
codixai init                          # create .codixai.yml in the repository root
codixai version
```

`codixai init` never overwrites an existing configuration file; it reports
that the file exists and leaves it untouched. Pass `--force` to replace it.

### Verify the claims of an agent

`codixai verify` compares the claims that an agent wrote about a change with
the real state of the repository. The claim file is a versioned JSON
document, `claims.json`:

```json
{
  "schema_version": "0.1",
  "base": "main",
  "claims": [
    { "id": "c1", "type": "tests_added", "paths": ["internal/payment/**"] },
    { "id": "c2", "type": "files_unchanged", "paths": ["internal/billing/**"] },
    { "id": "c3", "type": "behavior_preserved" }
  ]
}
```

Every claim needs a unique `id` and a `type` of the catalog:
`api_unchanged`, `behavior_preserved`, `dependency_added`,
`dependency_unchanged`, `files_unchanged`, `scope`, `tests_added`,
`tests_modified` and `tests_passed`. The `base` key is optional; when it is
present it has to match the base of the comparison, because the claims would
be about another revision.

```bash
codixai verify --base main --claims claims.json
```

The text report has this shape (the values below are illustrative, the
structure is the real one):

```text
Codixai verification
  base:              957dfec756a01a5fb0bfd04e0e1df33b65809a98
  head:              7ce93ad4bd3d11fd60fe67af4746dedc87f5a319
  generated at:      2026-10-22T00:15:28Z
  tool:              codixai dev (schema 0.1)
  policy:            fail_on_unresolved false

Claims (3)

  c1  tests_added  VERIFIED
    paths:             internal/payment/**
    message:           the test file(s) internal/payment/service_test.go were added or modified
    evidence:          diff_file internal/payment/service_test.go (the file appears in the compared change set)
    limitations:
      - The test files are recognized by the patterns of the language profile, so a test file with an unusual name is not recognized.

  c2  files_unchanged  VERIFIED
    paths:             internal/billing/**
    message:           no changed file matches the paths of the claim
    limitations:
      - The claim only holds for the compared range of revisions.

  c3  behavior_preserved  UNRESOLVED
    message:           static analysis does not decide the behavior of a change
    limitations:
      - Static analysis does not decide the behavior of a change.

Summary
  total:             3
  by state:          verified 2, contradicted 0, unresolved 1

Limitations
  - The verification is a static analysis of the compared range: Codixai did not run the code of the repository.
  - The test reports are external evidence: they were produced outside Codixai and Codixai cannot vouch for them.
  - An unresolved claim is not a failure: it means that the available evidence was not enough to decide.
  - The claims are checked against the whole compared range: the paths that the configuration excludes from the analysis are not excluded from the claims.
```

`--format json` writes the versioned document of the verification to the
standard output: `schema_version`, `tool_version`, `generated_at`, `base`,
`head`, `results`, `summary` and `limitations`. The diagnostics stay on the
standard error.

**Codixai never runs the tests of the analyzed repository.** A `tests_passed`
claim is only checked against a report that the user provides, in the JUnit
XML format (`.xml`) or in the `go test -json` format (`.json`), and only
when the report can be linked to the revision under analysis. A report
without a revision is never used: the claim stays `unresolved`.

The format of the claim file, the catalog of the types and the linking of a
report are fixed by the decision
[`0007`](docs/decisions/0007-claim-format-and-report-linking.md).

### The baseline of findings

A baseline is a versioned file of the analyzed repository that records the
findings the repository already has. It lets a pipeline fail on the new
findings without turning the existing debt into constant noise.

```bash
codixai baseline create --base main
```

The file is written at the `audit.baseline` key of the configuration, which
defaults to `.codixai/baseline.json`; `--out <path>` writes another one. The
command needs exactly one origin (`--base`, `--staged` or `--worktree`), the
same three as the audit, and it never replaces an existing baseline without
`--force`. The baseline that is being created is never applied to the run
that creates it.

The file has its own `schema_version` (`0.1`) and this shape (the values below
are illustrative, the structure is the real one):

```json
{
  "schema_version": "0.1",
  "created_at": "2026-10-22T10:00:00Z",
  "tool_version": "0.1.0",
  "findings": [
    {
      "fingerprint": "3f2a91c0d4b7e815",
      "rule_id": "CDX-SUP-001",
      "path": "internal/payment/service.go",
      "line": 41,
      "severity": "warning",
      "message": "The change adds an analyzer suppression."
    }
  ]
}
```

The findings are sorted by their fingerprint, so the file is stable and its
diffs are readable. The line, the severity and the message are informative:
they are not part of the identity of the finding.

**The identity is the fingerprint**: the first 16 hexadecimal characters of
`sha256(rule_id + 0x00 + path + 0x00 + normalized_context)`, where the context
is the text of the line the finding points at, taken from the diff, with the
leading and the trailing spaces removed and every run of internal whitespace
collapsed into a single space. The line number is never part of it, so a
finding that moves down a file stays known; it changes when the code of the
line really changes, so a finding that was fixed and reintroduced is new and
has to be reviewed again. When the line is not part of the compared range the
context is empty and the identity is the rule and the path.

The audit reads the baseline, removes the findings it knows and fails only on
the new ones. The standard output keeps holding only the document of the new
findings; the count of the known findings is a diagnostic of the standard
error:

```text
codixai: 2 finding(s) known by the baseline, not reported
```

**Commit the baseline and review it when it changes.** An entry that appears
is a finding that was accepted into the debt; an entry that disappears is a
finding that is not reported anymore. Both are decisions about the repository
and belong in the review of the change. The format, the identity and the
behavior of the audit are fixed by the decision
[`0008`](docs/decisions/0008-baseline-and-fingerprint.md).

### The SARIF export

`codixai report` writes the machine readable document of the same run that the
audit reports:

```bash
codixai report --format sarif --base main
```

`--format sarif` is the default of the command; `--format json` writes the
audit document instead, byte for byte the one that `codixai audit --format
json` writes. The command needs one origin, like the audit, and it shares the
exit code of the audit: `3` when a new finding reaches the `fail_on`
threshold.

The document is SARIF 2.1.0, with one run and the driver `codixai`:

- the rules of the catalog, sorted by identifier, with their description,
  their default level and their support;
- one result per new finding, with the rule and its index, the level, the
  message, the repository relative location and the line range;
- the severity mapping `error` → `error`, `warning` → `warning`, `info` →
  `note`; a severity outside the catalog becomes `none`, which is how SARIF
  says that a result fails nothing;
- `partialFingerprints["codixaiFingerprint/v1"]`: the fingerprint of the
  baseline, so a platform that suppresses a known result and a pipeline that
  applies a baseline use the same identity;
- `runs[].properties.suppressedByBaseline`: the count of the findings that the
  baseline knows. They are not results, because they are not new.

**The document is structurally conformant to SARIF 2.1.0 and it has not been
validated against the official JSON schema**, because the development
environment has no network access to fetch it. The tests validate the
structure that the decision
[`0009`](docs/decisions/0009-sarif-export.md) fixes. Confirm the import once
on your platform before you make the step blocking.

The pipeline workflow — the origin flag, the exit codes, the baseline and the
import of the SARIF document — is described in
[`docs/guides/ci-integration.md`](docs/guides/ci-integration.md).

### Exit codes

| Code | Meaning |
| ---- | ------- |
| `0` | The operation succeeded and no finding reaches the failure threshold. |
| `1` | Execution or configuration error. |
| `2` | Invalid arguments. |
| `3` | The audit completed and its findings reach the configured `fail_on` threshold, or a claim of the verification is `contradicted`. |

The `report` command shares the codes of the audit: it is the same run with a
machine readable document. A refused replacement of an existing baseline
without `--force` is an execution error (`1`), not an invalid invocation.

The default `audit.fail_on` is `error`. A `verify` run fails with code `3`
when a claim is `contradicted`; an `unresolved` claim only fails when the
configuration of the repository sets `verify.fail_on_unresolved: true`, which
is not the default:

```yaml
verify:
  fail_on_unresolved: true
```

Because `go run` does not propagate the exit code of the program it runs,
use the built binary when you need to observe code `3` in a script.

## Command surface

```text
codixai <command> [flags]

commands: audit, report, baseline create, verify, init, version, help

global flags:
  --root <value>    Root of the analyzed repository. Defaults to the current directory.
  --format <value>  Output format: text or json, or sarif for the report command.
  --no-color        Disable the color output.
  --plain           Print the output without decoration.
  --version         Print the version of the tool and exit.

audit flags:
  --diff            Audit the changes of the repository. Required in this milestone.
  --base <value>    Base revision to compare the repository with HEAD.
  --staged          Audit the changes of the staging area.
  --worktree        Audit the changes that are not committed yet.
  --rules           Print the rule catalog and exit.

report flags:
  --base <value>    Base revision to compare the repository with HEAD.
  --staged          Report the findings of the staging area.
  --worktree        Report the findings that are not committed yet.

baseline create flags:
  --base <value>    Base revision to compare the repository with HEAD.
  --staged          Record the findings of the staging area.
  --worktree        Record the findings that are not committed yet.
  --out <value>     Path of the baseline file to write. Defaults to the audit.baseline key.
  --force           Replace an existing baseline file.

verify flags:
  --base <value>    Base revision to compare the repository with HEAD. Required.
  --claims <value>  Path of the claim file to verify. Required.

init flags:
  --force           Replace an existing configuration file.
```

Exactly one origin must be selected by `audit`, `report` and `baseline
create`: `--base`, `--staged` or `--worktree`. The `verify` command needs both
`--base` and `--claims`; a missing one is an invalid invocation and the exit
code is `2`.

## Rule catalog

The catalog is published by the tool itself:

```bash
codixai audit --rules
```

| Rule | Severity | Checks |
| ---- | -------- | ------ |
| `CDX-IMP-001` | error | An import added by the change refers to a dependency that is not declared in the manifests and does not belong to the standard library. |
| `CDX-REF-001` | error | A local symbol referenced by a changed Go file does not exist in the package directory of the file. |
| `CDX-TST-001` | warning | The change deletes, disables or weakens a test. |
| `CDX-SUP-001` | warning | The change adds a suppression directive that silences an analyzer. |
| `CDX-ERR-001` | warning | The change adds a pattern that discards an error or an empty handler. |
| `CDX-PLH-001` | warning | The change adds an obvious placeholder or stub. |

### Support matrix

`none` means the rule does not apply to that language. `syntactic` means the
rule works on the syntax of the changed lines. `resolved` means the rule
relies on semantically resolved relations; no rule reaches that level today.

| Rule | go | javascript | php | python | typescript |
| ---- | -- | ---------- | --- | ------ | ---------- |
| `CDX-IMP-001` | syntactic | syntactic | syntactic | syntactic | syntactic |
| `CDX-REF-001` | syntactic | none | none | none | none |
| `CDX-TST-001` | syntactic | syntactic | syntactic | syntactic | syntactic |
| `CDX-SUP-001` | syntactic | syntactic | syntactic | syntactic | syntactic |
| `CDX-ERR-001` | syntactic | syntactic | syntactic | syntactic | syntactic |
| `CDX-PLH-001` | syntactic | syntactic | syntactic | syntactic | syntactic |

A rule may only declare support for a language that has positive and negative
fixtures. The CI validates the matrix against the fixture corpus.

## Limitations

These are real limitations of the current implementation, not caveats about
the roadmap:

- **Syntactic level only.** Four of the five languages are read with a shared
  line-level lexical scanner. Go uses `go/parser` and `go/ast`. Nothing is
  resolved semantically, so no rule is declared `resolved`.
- **No index.** `audit --diff` compares Git blobs directly. There is no
  persistent index, no symbol graph and no cross-file resolution.
- **`CDX-REF-001` produces false positives.** The Go reference rule works on
  the package directory of the changed file, so an identifier declared in
  another package of the module is reported as missing. The predeclared
  identifiers of the language and the packages of the module are resolved.
  The finding is a signal to review, not a confirmed defect.
- **`CDX-TST-001` reports conditional skips.** A test that is skipped when a
  tool is missing is a legitimate practice, and the rule cannot tell it
  apart from a test that is disabled on purpose.
- **Manifests are read with reduced parsers.** `go.mod`, `requirements.txt`,
  `pyproject.toml`, `composer.json` and `package.json` are read directly, with
  simplified parsing, and only the root of the repository is considered.
- **Only the added lines are analyzed** by most rules. A defect that the
  change does not touch is out of scope.
- **Two claim types can never be decided.** `api_unchanged` is always
  `unresolved`, because the public API is not extracted yet, and
  `behavior_preserved` is always `unresolved`, because static analysis does
  not decide the behavior of a change.
- **A test report is only used when it can be linked.** A `tests_passed`
  claim needs a JUnit XML or a `go test -json` report that declares the
  revision under analysis, either in the claim (`report_revision`) or in the
  report itself. A report without it leaves the claim `unresolved`, even
  when the report is honest. Codixai never produces the report itself and
  never runs the tests of the repository.
- **The claims ignore the excluded paths.** The `scope` and the
  `files_unchanged` claims are checked against the whole compared range: a
  path that the configuration excludes from the analysis is not excluded
  from the claims.
- **The baseline is one versioned file of the repository.** There is no
  baseline per origin, no remote baseline and no `baseline show`, `prune` or
  `update` command. A baseline that is not committed is not shared by the
  pipeline. A finding whose line is not part of the compared range is
  identified by the rule and the path only, which is coarser than the
  identity of a finding with a context.
- **The SARIF export is not validated against the official JSON schema.**
  The document is structurally conformant to SARIF 2.1.0 and the structure is
  validated by the tests, but the environment has no network access to fetch
  the schema. No import into a code scanning platform was exercised; confirm
  it once on your platform.
- **The rule catalog publishes no human readable name**, so the SARIF
  descriptors carry the rule identifier in `name` and the description in both
  `shortDescription` and `fullDescription`.
- **No MCP server.** It belongs to Phase 4 and it is not implemented.
- **No graphical interface.** The `codixai ui` command, the Wails adapter
  and the Svelte frontend belong to Phase 7 and do not exist. The
  architecture that would allow them is already in place and is verified by
  a test.
- **No cross-platform verification.** Only Linux was exercised in the
  development environment. macOS and Windows builds are defined in CI but
  their results have not been observed yet.
- **No published release.** There is no tagged version, no packaged binary and
  no measured performance or precision figure.

The absence of findings does not prove that a change is correct, safe or
complete.

## Roadmap

The phases follow the product specification
([`docs/spec/CODIXAI_SPEC.md`](docs/spec/CODIXAI_SPEC.md)).

| Phase | Scope | State |
| ----- | ----- | ----- |
| 0 | Executable foundations and technical decisions | In progress, mostly done |
| 1 | The diff verifier spike: the six rules over five languages | In progress, partially done |
| 2 | Claims, evidence and the verification contract | Implemented: the `verify` command, the claim contract and the corpus |
| 3 | CI adoption: SARIF export and a versioned baseline | Implemented: the stable fingerprint, the baseline, the `report` command and the SARIF 2.1.0 export |
| 4 | First agent integrations and MCP | Not started |
| 5 | Structural analysis and impact on demand | Not started |
| 6 | Wider rules and semantic analysis | Not started |
| 7 | The local graphical interface (Wails, Svelte and daisyUI) | Not started |

The exit criterion of Phase 1 is not reached yet: the corpus of real diffs
produced by external agents, the false-positive report and the decision gate
that keeps, downgrades or discards each rule are still pending.

Phase 2 is implemented, with the exceptions that the decision
[`0007`](docs/decisions/0007-claim-format-and-report-linking.md) declares
on purpose: the `api_unchanged` and the `behavior_preserved` claims are
always `unresolved`.

Phase 3 is implemented: the stable fingerprint, the versioned baseline, the
`baseline create` command, the SARIF 2.1.0 export and the `report` command
exist, with the decisions
[`0008`](docs/decisions/0008-baseline-and-fingerprint.md) and
[`0009`](docs/decisions/0009-sarif-export.md). Two parts of its exit
criterion are not verified: the document is not validated against the
official JSON schema of SARIF, and no import into a hosted code scanning
platform was exercised. The workflow is described in
[`docs/guides/ci-integration.md`](docs/guides/ci-integration.md).

Phase 7 is the local graphical interface: a native window built with Wails
and a frontend built with Svelte, Tailwind and daisyUI, launched with
`codixai ui` and shipped in the same binary as the CLI. Its spike 7.0
(the version of Wails, the packaging, the builds per platform and the
distribution) may be started after Phase 3, because the risk is in the
packaging and not in the screens. None of it exists today.

## Documentation

- [Product specification (v0.6.0)](docs/spec/CODIXAI_SPEC.md)
- [Architecture overview](docs/architecture/overview.md)
- [Implementation conventions](docs/guides/implementation-conventions.md)
- [CI integration](docs/guides/ci-integration.md)
- [Task format and result report](docs/tasks/README.md)
- [Phase 0 tasks](docs/tasks/phase-0.md), [Phase 1 tasks](docs/tasks/phase-1.md),
  [Phase 2 tasks](docs/tasks/phase-2.md),
  [Phase 3 tasks](docs/tasks/phase-3.md) and
  [Phase 7 tasks](docs/tasks/phase-7.md)
- [Architecture decision records](docs/decisions/)
- [Contributing](CONTRIBUTING.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). It describes the development setup,
the commands that must pass, how to add a language and how to add a rule.

The short version: the project has no third-party dependencies today, every
rule needs positive and negative fixtures, every significant architectural
change needs an ADR, and contributions that add telemetry, code upload or
remote execution are not accepted.

## Security

Read [SECURITY.md](SECURITY.md). In short: Codixai is read-only, it does not
execute the code of the analyzed repository, it does not invoke package
managers, it does not access the network, and it neutralizes the executable
configuration a repository can define for Git.

The private security advisory of the GitHub repository is the reporting
channel: <https://github.com/Barceloch/Codixai/security/advisories/new>. The
contact address is **PENDING maintainer confirmation**.

## Changelog

See [CHANGELOG.md](CHANGELOG.md). The project has not published a release
yet: the `0.1.0` entry is still unreleased.
