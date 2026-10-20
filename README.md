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

**Experimental. This repository is at the Phase 0 / Phase 1 milestone, not a
finished product.**

What exists today:

- a Go module named `codixai` that builds a single binary;
- a CLI with the commands `audit`, `init`, `version` and `help`;
- a read-only Git integration with hardened invocation;
- a configuration file loader for `.codixai.yml`;
- six deterministic rules over five languages (Go, Python, PHP, JavaScript,
  TypeScript);
- a fixture corpus of 58 cases and end-to-end tests over a temporary
  repository.

What does **not** exist today:

- no persistent index, no `index`, `status`, `overview`, `find`, `refs` or
  `deps` commands;
- no `verify` command and no claim verification;
- no SARIF export, no baseline file support, no MCP server;
- no semantic resolution: every rule is at most `syntactic`;
- no released version and no published binary.

### License

**The license is PENDING maintainer confirmation.** The proposal is
Apache-2.0. No license is granted by this repository until the maintainer
confirms it and the `LICENSE` file is added. Do not assume any right to
redistribute the code until that file exists.

The maintainer was asked on 2026-10-20 and decided to keep the license
pending, so the repository is currently **not distributable**. The decision
is recorded in
[`docs/decisions/0004-module-path-repository-and-license.md`](docs/decisions/0004-module-path-repository-and-license.md).

## Why it exists

An agent can produce a large diff in seconds. Reviewing that diff well
requires answering questions that a diff alone does not answer:

- Does the change import something that is not declared in the manifest?
- Did the change delete or weaken a test?
- Did the change silence an analyzer instead of fixing the problem?
- Did the change leave an error unhandled or a stub in place?

Codixai answers a bounded version of that question deterministically, with
evidence, and states plainly what it could not check. It is a verifier, not a
chatbot: the same input always produces the same document, and every finding
points at a file, a line range and a rule.

## Install from source

The tool has no third-party dependencies: the whole program is written with
the Go standard library. A clone builds without network access.

Requirements: Go 1.27 or later (the module declares `go 1.27.0`).

```bash
git clone <repository-url> codixai   # the URL is PENDING maintainer confirmation
cd codixai
go build ./cmd/codixai
```

This produces the `codixai` binary in the current directory. You can also run
it without building:

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
codixai init                          # create .codixai.yml in the repository root
codixai version
```

`codixai init` never overwrites an existing configuration file; it reports
that the file exists and leaves it untouched. Pass `--force` to replace it.

### Exit codes

| Code | Meaning |
| ---- | ------- |
| `0` | The operation succeeded and no finding reaches the failure threshold. |
| `1` | Execution or configuration error. |
| `2` | Invalid arguments. |
| `3` | The audit completed and its findings reach the configured `fail_on` threshold. |

The default `audit.fail_on` is `error`. Because `go run` does not propagate
the exit code of the program it runs, use the built binary when you need to
observe code `3` in a script.

## Command surface

```text
codixai <command> [flags]

commands: audit, init, version, help

global flags:
  --root <value>    Root of the analyzed repository. Defaults to the current directory.
  --format <value>  Output format: text or json.
  --no-color        Disable the color output.
  --plain           Print the output without decoration.
  --version         Print the version of the tool and exit.

audit flags:
  --diff            Audit the changes of the repository. Required in this milestone.
  --base <value>    Base revision to compare the repository with HEAD.
  --staged          Audit the changes of the staging area.
  --worktree        Audit the changes that are not committed yet.
  --rules           Print the rule catalog and exit.

init flags:
  --force           Replace an existing configuration file.
```

Exactly one audit origin must be selected: `--base`, `--staged` or
`--worktree`.

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
- **No MCP, no SARIF, no baseline.** Those are planned for later phases and
  are not implemented.
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
| 2 | Claims, evidence and the verification contract | Not started |
| 3 | CI adoption: SARIF export and a versioned baseline | Not started |
| 4 | First agent integrations and MCP | Not started |
| 5 | Structural analysis and impact on demand | Not started |
| 6 | Wider rules and semantic analysis | Not started |
| 7 | A local API and a visual application | Not started |

The exit criterion of Phase 1 is not reached yet: the corpus of real diffs
produced by external agents, the false-positive report and the decision gate
that keeps, downgrades or discards each rule are still pending.

## Documentation

- [Product specification (v0.5.0)](docs/spec/CODIXAI_SPEC.md)
- [Architecture overview](docs/architecture/overview.md)
- [Implementation conventions](docs/guides/implementation-conventions.md)
- [Task format and result report](docs/tasks/README.md)
- [Phase 0 tasks](docs/tasks/phase-0.md) and [Phase 1 tasks](docs/tasks/phase-1.md)
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
channel. The repository URL and the contact address are **PENDING maintainer
confirmation**.

## Changelog

See [CHANGELOG.md](CHANGELOG.md). The project has not published a release
yet: the `0.1.0` entry is still unreleased.
