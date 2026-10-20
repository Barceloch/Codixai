# Contributing to Codixai

Thank you for taking the time to contribute. This document describes how to
set up the project, what has to pass before a change is accepted, and how to
add a language or a rule.

## Current project state

Codixai is an experimental project at the Phase 0 / Phase 1 milestone. The
honest summary is in the [README](README.md#status). Two facts matter for a
contributor:

- The module is `codixai` and the program uses **only the Go standard
  library**. There is no `go.sum` and no third-party dependency.
- The module path, the repository URL and the license are **pending maintainer
  confirmation**. Do not change them on your own: see
  [`docs/decisions/0004-module-path-repository-and-license.md`](docs/decisions/0004-module-path-repository-and-license.md).

## Development setup

Requirements:

- Go 1.27 or later. The module declares `go 1.27.0`.
- `git`, because the integration tests create a temporary repository.

No network access is needed today. The build environment has none, and the
project was written to work in that condition.

```bash
git clone <repository-url> codixai
cd codixai
go build ./...
go test ./...
```

## The commands that must pass

Run all four before opening a pull request:

```bash
gofmt -l .          # must print nothing
go build ./...
go vet ./...
go test ./...
```

CI runs the same checks, plus `go test ./... -race` and a build matrix for
Linux, macOS and Windows.

## Code conventions

- **English.** Identifiers, comments, doc comments, user-facing strings and
  documentation are in English. The specification is written in Spanish, but
  the artifacts of the repository are in English.
- **Doc comments on every exported symbol.** Start with the name of the
  symbol, as `go doc` expects.
- **Small functions, early returns.** Prefer a guard clause over a deep
  nesting.
- **Errors carry the context.** Wrap with `%w` and name the file, the path or
  the flag that is wrong. An error message must tell the user how to recover.
- **No new dependencies without a written justification** in the pull request
  description: why it is needed, which license it carries and how it is
  maintained. A dependency that needs the network also needs a decision from
  the maintainer.
- **Deterministic output.** The same input must always produce the same
  document. Sort anything that reaches an output.
- **Never execute the analyzed repository.** No package manager, no test
  runner, no build of the analyzed project, no network. The rules read text.

The full list lives in
[`docs/guides/implementation-conventions.md`](docs/guides/implementation-conventions.md).

## How to add a language

A language support is a package under `internal/languages/` plus a registry
entry. The steps:

1. **Create the package** `internal/languages/<name>/` with a `Profile()`
   function that returns a `lexical.Profile`:
   - `ID` and `Name`;
   - `Extensions` (lower case, with the dot);
   - `LineComments`, `BlockComments`, `Strings`, `RawPrefixes`,
     `RegexLiterals`;
   - the markers used by the rules: `Suppressions`, `Skips`, `Assertions`,
     `Placeholders`, `Swallows`;
   - `TestPaths`, the glob patterns of the test files.
2. **Declare the standard library** in a `stdlib` map and expose an
   `IsStdlib` function, as `internal/languages/go/golang.go` and
   `internal/languages/python/python.go` do.
3. **Add a manifest reader** in the same package, for example
   `internal/languages/python/manifest.go`. It reads the manifest of the
   language directly from disk. It must not run a package manager.
4. **Register the language** in `internal/languages/registry/registry.go`, and
   add its identifier to `internal/domain/models.go` and to
   `internal/policies.SupportedLanguages`.
5. **Add fixtures**: at least one positive and one negative case per rule
   that the language will support, under
   `testdata/rules/<RULE-ID>/<language>/{positive,negative}/`.
6. **Update the support matrix** of every affected rule in
   `internal/analysis/rules/catalog.go`. A rule may only declare support for
   a language that has both fixture kinds; CI enforces it.

## How to add a rule

A rule is a deterministic check with a stable identity. The steps:

1. **Choose the identifier** with the convention `CDX-<AREA>-<NNN>`, where the
   area is a short uppercase token (`IMP`, `REF`, `TST`, `SUP`, `ERR`, `PLH`).
   An identifier is never reused with another meaning.
2. **Declare the contract** in `internal/analysis/rules/catalog.go`:
   - `ID`, `Version` (start at `1`, increment it when the behavior changes),
     `Area`;
   - `Description`, in English, describing what the rule checks;
   - `Severity` (`info`, `warning` or `error`);
   - `EvidenceKind`, the kind of evidence the rule produces;
   - `Method`, how the rule checks the code;
   - `Languages`, the support matrix: `none`, `syntactic` or `resolved`;
   - `Limitations`, per language, the known limits of the rule.
3. **Implement the check** in a new file of `internal/analysis/rules/`. The
   check receives a `*Context` and returns findings. It must not execute
   anything and must not touch the network.
4. **Classify the epistemic status honestly.** A finding that was read
   directly from the code or the diff is `observed`. A finding produced by a
   heuristic is `inferred`. Never present a heuristic signal as a confirmed
   bug.
5. **Add fixtures**: a `positive` case and a `negative` case per language
   that the rule supports, under `testdata/rules/<RULE-ID>/<language>/`.
   Each case holds a `<name>.diff` and a `<name>.expected.json` with
   `expected_rule_ids` and `expected_count`. A case may hold a `repo/`
   directory with the manifests of the analyzed project.
6. **Add a test** in the package of the rule when the rule has a behavior
   that the fixtures do not cover.
7. **Do not raise the support level** above `syntactic` in this milestone.
   `resolved` requires semantic resolution that does not exist yet.

The support matrix is validated against the corpus by
`TestSupportMatrixMatchesFixtures` in `internal/analysis/engine_test.go`, and
the identifier convention by `TestRuleIdentifiersFollowTheConvention` in
`internal/analysis/rules/catalog_test.go`.

## Architecture decision records

Any significant architectural change needs an ADR in `docs/decisions/`, named
`NNNN-<slug>.md`, with the sections **Status**, **Context**, **Decision**,
**Consequences** and **Alternatives considered**, and a date. Create the
next number in the sequence; do not renumber the existing files.

An ADR is required when a change:

- introduces a dependency or a toolchain requirement;
- changes the JSON output contract or `schema_version`;
- changes the layer boundaries or the dependency direction;
- changes the security posture of the Git integration;
- decides a question that the specification lists as pending.

## Pull request process

1. Open an issue first for anything larger than a bug fix, so the design can
   be discussed before the code is written.
2. Work on a branch. Do not commit directly to the default branch.
3. Keep the change reviewable: one task should fit in a single review. Split
   unrelated changes into separate pull requests.
4. Use [Conventional Commits](https://www.conventionalcommits.org/) for the
   commit messages. Do not add AI attribution, `Co-Authored-By` trailers or
   generated-by markers to the commits.
5. Fill in the pull request template, including the commands you ran and
   their real result.
6. Make sure the four checks pass locally before requesting a review.
7. Expect review comments about scope and conventions. A change that goes
   beyond the scope of the task is a defect, not a bonus.

## No telemetry, no code upload, no remote execution

Contributions that add any of the following are **not accepted** without an
explicit review by the maintainer:

- telemetry, analytics, crash reporting or usage measurement;
- sending the code of the analyzed repository, or any part of it, to a remote
  service;
- executing code of the analyzed repository, invoking a package manager or
  running a build of the analyzed project;
- remote execution of any kind, including a hidden update check;
- loading an executable plugin automatically.

If a contribution needs one of these, it must be opt-in, documented, and
explicitly approved. The default posture of the project is local-first:
indexing and analysis work without an account, a remote service or an AI
model.

## Reporting a bug

Include the version (`codixai version`), the exact command, the repository
layout needed to reproduce, the expected result and the actual result. When
possible, attach a minimal fixture. Never include secrets, tokens or private
code: a reduced reproduction is enough.

## Code of conduct

The project adopts the [Contributor Covenant 2.1](CODE_OF_CONDUCT.md).
