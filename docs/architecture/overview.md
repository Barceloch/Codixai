# Architecture overview

This document describes the layers of Codixai, the direction of the
dependencies, the data flow of an audit and of a verification, the domain
model, the extension points, and the known limits of the current
implementation.

The product contract is the specification
([`docs/spec/CODIXAI_SPEC.md`](../spec/CODIXAI_SPEC.md), v0.6.0). This
document describes what the code actually does today.

## Layers

```text
cmd/codixai                 process entry point, calls cli.Execute
  |
internal/cli                command tree, flag parsing, exit codes, rendering
  |
internal/application        use cases and the ports they need
  |                         (ChangeSource, DiffAnalyzer, ClaimLoader,
  |                          ClaimEvaluator)
  |
internal/analysis           the analysis engine
internal/analysis/rules     the rule catalog
internal/claims             the claim contract, the evaluator and the
  |                         readers of the external test reports
internal/languages          the language registry, the lexical scanner, the
  |                         language packages
internal/claims             the claim contract, the evaluator and the
  |                         readers of the external test reports
  +-------------------------+
  |
internal/domain             the models, with no dependency on anything else
internal/glob               the glob matcher of the repository paths
internal/policies           the configuration of the analyzed repository
internal/integrations/git   the read-only Git provider
```

`internal/policies` is a leaf like `internal/domain`: it holds the
configuration model and its loader, and it does not depend on the use cases.
`internal/glob` is a leaf too: it holds the glob matcher, it never touches
the filesystem, and it exists so that the exclusion patterns of the
configuration and the paths of the claims share one syntax without
depending on each other.

### The layers that do not exist yet

The local graphical interface belongs to Phase 7 and is not implemented. The
place it will have is already decided by the specification:

```text
internal/ui/                the Wails adapter, behind the `ui` build tag
ui/embed.go                 embeds the compiled assets, behind the `ui` build tag
ui/frontend/                Svelte + Vite + Tailwind + daisyUI, compiled to
                            ui/frontend/dist
```

The build tag is what keeps the default variant free of Node.js, of Wails
and of the webview dependencies, and it is what keeps the CLI usable in a
terminal, in CI and in a container.

## The dependency rule

- **`internal/domain` imports nothing from the project.** Not the CLI, not
  the storage, not the providers, not the analysis engine. It only uses the Go
  standard library. The models are the contract that the CLI, the future MCP
  server and any other consumer share.
- **`internal/application` declares the ports.** `ChangeSource` collects the
  changes of a repository; `DiffAnalyzer` applies the rules to a change set;
  `ClaimLoader` reads a claim file and checks its base; `ClaimEvaluator`
  computes the result of every claim. The use cases `RunAudit` and
  `RunVerify` receive the implementations as parameters, so they never
  depend on a provider.
- **The providers implement the ports.** `internal/integrations/git.Source`
  implements `ChangeSource` and asserts it at compile time with
  `var _ application.ChangeSource = (*Source)(nil)`.
  `internal/analysis.Engine` implements `DiffAnalyzer` with the same
  assertion. `internal/claims.Loader` implements `ClaimLoader` and
  `internal/claims.Evaluator` implements `ClaimEvaluator`.
- **`internal/cli` is the composition root.** It wires the use case to the
  providers, applies the policy of the repository and renders the result. It
  never implements a rule.
- **`internal/languages` and `internal/analysis` produce structured
  information.** The analysis engine asks the registry for the language of a
  file, asks the language package for its profile, and applies the rules.

The direction never goes back up: the domain knows nothing about the CLI, and
the analysis engine knows nothing about the command line. A future web
application or MCP server would call the same use cases.

### The direction of the dependencies of the future interface

```text
ui  ->  application  ->  domain
```

The interface would call the same use cases and would return the same
versioned documents that `--format json` returns today, so the two
presentations have to show the same result for the same change; an
automated test will check it (the task P7-11 of
[`docs/tasks/phase-7.md`](../tasks/phase-7.md)). The adapter only
translates: it holds no rule and no analysis logic. The requirement is
verified from the first phase by
`internal/application/architecture_test.go`, which fails when the
application layer imports the CLI, the user interface or a desktop toolkit,
or when the domain imports any package of the project.

## The requirements of the section 8.7 of the specification

The specification requires five things from Phase 0. This is how each one
is met today.

| Requirement | How it is met today |
| ----------- | -------------------- |
| 8.7.1 The application layer depends on neither the CLI nor the UI, and an architecture test verifies it. | The application layer declares the ports and receives the providers as parameters. `internal/application/architecture_test.go` parses the imports of the package and of the domain and fails on a forbidden one. |
| 8.7.2 The results are versioned, serializable DTOs. | Every document carries `schema_version` (`0.1`) and is produced with `encoding/json`. The models live in `internal/domain` and the documents of the use cases in `internal/application`. |
| 8.7.3 The long operations receive a `context` and report progress. | The use cases take a `context.Context` and check it with `ctx.Err()` between the steps of a run. The progress phases are not reported yet: the CLI has no progress output, and the phases of the interface will be the ones of the CLI. |
| 8.7.4 Every file access is bounded to the selected root. | The root is resolved once by the CLI. The paths of a claim and the path of a report are relative to the root and cannot escape it: the loader rejects an absolute path, a Windows drive and a `..` segment. The Git provider is invoked with the root as its working directory. The claim file itself is a file the user names explicitly with `--claims`. |
| 8.7.5 The UI specific code lives behind build tags. | There is no UI code yet. The phase 7 tasks will put the Wails adapter and the embedded assets behind the `ui` build tag, so the default variant compiles without Node.js, without Wails and without a webview. |

## The data flow of an audit

```text
codixai audit --diff --worktree
  |
  1. cli.runAudit
  |     resolves the root, validates the origin, checks that the root is a
  |     Git repository
  |
  2. application.RunAudit
  |     loads the policy of the repository (.codixai.yml, or the defaults)
  |     asks the ChangeSource for the changes
  |     removes the excluded paths
  |
  3. git.Source.Changes
  |     builds the argument slice, runs git, parses the unified diff into a
  |     domain.ChangeSet
  |
  4. analysis.Engine.Analyze
  |     for every changed file: detect the language, build the diff oriented
  |     view (the lexical segments of the added and removed lines), run the
  |     rules that support the language
  |
  5. application.applyPolicy
  |     applies the rule overrides, removes the suppressed findings, rebuilds
  |     the summary, decides whether the threshold is reached
  |
  6. cli.writeAuditReport or the JSON encoder
  |     writes the document to the standard output, the diagnostics to the
  |     standard error, and returns the exit code
```

The policy is applied after the analysis on purpose: the analyzer knows the
rules and the policy knows the decisions of the repository.

## The data flow of a verification

```text
codixai verify --base <ref> --claims claims.json
  |
  1. cli.runVerify
  |     rejects a missing --base or --claims with the invalid arguments code,
  |     resolves the root and checks that the root is a Git repository
  |
  2. application.RunVerify
  |     loads the policy of the repository (.codixai.yml, or the defaults)
  |     loads the claim file and checks that its base is the base of the
  |     comparison
  |     asks the ChangeSource for the changes and for the head
  |
  3. claims.Evaluator
  |     for every claim: computes the state from the changed files, from the
  |     manifests of the root and from the external report
  |     sorts the results by the identifier of the claim
  |
  4. cli.writeVerifyReport or the JSON encoder
        writes the versioned document to the standard output, the diagnostics
        to the standard error, and returns the exit code
```

The evaluator never executes anything: it reads the manifests and the
reports of the analyzed repository as text. A report is external evidence
and the evidence of the result says so.

## The domain model

The models live in `internal/domain`. The ones that matter in this milestone:

- **`Rule`** — a deterministic check with a stable identifier `CDX-<AREA>-<NNN>`,
  a version, an area, a description, a default severity, a support matrix by
  language, the method, the known limitations and the kind of evidence it
  produces.
- **`Finding`** — the result of a rule on a change: the rule identifier and
  version, the severity, the epistemic status, the language, the message, the
  location, the revision, the evidence, the support (level, method and
  limitations) and an optional recommendation.
- **`Evidence`** — a verifiable reference: a kind (`diff_hunk`,
  `source_range`, `manifest_entry`), a stable reference and an optional
  clarification.
- **`AnalysisRun`** — the document of one analysis: the schema version, the
  tool version, the instant, the origin of the changes, the revision, the
  findings, the summary, the rules and the limitations.
- **`ChangeSet` and `FileChange`** — the changes of a repository, with the
  hunks and the lines of every file.
- **`Claim` and `ClaimResult`** — the model of the verification contract of
  Phase 2: the assertion, its state (`verified`, `contradicted` or
  `unresolved`), the message that says what was observed, the evidence and
  the limitations of the check.
- **`Relation`** — a link between symbols or files, with its method and its
  resolution.

### Two different kinds of uncertainty

The domain model keeps two distinct vocabularies, and mixing them would
break the contract of the output:

- **`epistemic_status` classifies a finding.** It answers "how do I know
  this?".
  - `observed` — the finding was checked directly against the code or the
    diff.
  - `inferred` — the finding was derived by a heuristic.
  - `unknown` — the tool could not determine the outcome.
- **`resolution` classifies a relation.** It answers "how well is this link
  known?".
  - `confirmed`, `inferred`, `unresolved`.

A finding never carries a `resolution`, and a relation never carries an
`epistemic_status`. The values share the word `inferred` because the two
vocabularies were merged in the specification v0.5.0, but they answer
different questions.

The rule that follows from this: a heuristic signal is never published as a
confirmed bug. `CDX-ERR-001` reports an empty handler as `observed` because
the empty body is visible in the diff, and reports a swallowed error as a
signal to review the intent of the change.

## The support matrix

Every rule publishes, for each of the five languages, one of three levels:

- `none` — the rule does not apply to the language.
- `syntactic` — the rule works on the syntax of the changed lines. This is
  the maximum level reachable in this milestone.
- `resolved` — the rule relies on semantically resolved relations. No rule
  reaches it today.

The matrix is published by `codixai audit --rules` and validated against the
fixture corpus by `TestSupportMatrixMatchesFixtures` in
`internal/analysis/engine_test.go`: a rule may not declare support for a
language that has no positive and negative fixtures.

## How to add a language

1. Create `internal/languages/<name>/` with a `Profile()` function that
   returns a `lexical.Profile`: the identifier, the name, the extensions,
   the comment and string syntax, the raw string prefixes, the markers of
   the rules (suppressions, skips, assertions, placeholders, swallows) and
   the glob patterns of the test files.
2. Declare the standard library of the language and expose an `IsStdlib`
   function.
3. Add a manifest reader in the same package. It reads the manifest from
   disk and never runs a package manager.
4. Register the language in `internal/languages/registry/registry.go`, and
   add its identifier to `internal/domain/models.go` and to
   `policies.SupportedLanguages`.
5. Add a positive and a negative fixture per supported rule under
   `testdata/rules/<RULE-ID>/<language>/`.
6. Update the support matrix of the affected rules.

## How to add a rule

1. Choose the identifier `CDX-<AREA>-<NNN>`. An identifier is never reused
   with another meaning.
2. Declare the contract in `internal/analysis/rules/catalog.go`: version,
   area, description, default severity, evidence kind, method, support
   matrix and the limitations per language.
3. Implement the check in `internal/analysis/rules/`. It receives a
   `*Context` and returns findings. It executes nothing.
4. Classify the epistemic status of every finding honestly.
5. Add a positive and a negative fixture per supported language.
6. Keep the support level at `syntactic` until a semantic resolver exists.

## Known limits of the current implementation

- **Everything is syntactic.** Go uses `go/parser` and `go/ast`; the other
  four languages use the shared line-level scanner of
  `internal/languages/lexical`. The scanner does not build a tree and does
  not resolve names.
- **The scope of the Go reference rule is the package directory** of the
  changed file, not the module, so identifiers declared in another package
  are reported as missing.
- **The manifests are read with reduced parsers**, and only the root of the
  repository is considered.
- **Most rules only read the lines that the change adds.** A deleted test
  is an exception, because the rule compares the removed and the added
  lines.
- **There is no index.** The audit compares the blobs that Git already
  holds; it does not build a structure of the repository.
- **Two claim types can never be decided.** `api_unchanged` and
  `behavior_preserved` are always `unresolved`: the public API is not
  extracted yet, and static analysis does not decide the behavior of a
  change. The catalog keeps them because the decision 0007 declares them.
- **A report is only used when it declares the revision.** A `tests_passed`
  claim needs a JUnit XML or a `go test -json` report linked to the head of
  the compared range, either through the `report_revision` of the claim or
  through the revision the report declares. Otherwise the claim is
  `unresolved`. Codixai never runs the tests of the repository.
- **The claims ignore the excluded paths.** The `scope` and the
  `files_unchanged` claims are checked against the whole compared range: the
  paths that the configuration excludes from the analysis are not excluded
  from the claims.
- **The presentation layer is minimal.** It has a centralized renderer, a
  table, a plain mode and the `NO_COLOR` support, and nothing else. The
  specification allows this minimum in Phase 1.
- **There is no graphical interface.** The Wails adapter, the Svelte frontend
  and the `codixai ui` command belong to Phase 7 and do not exist. The
  requirements of the section 8.7 of the specification are met today, and
  the architecture test protects them.
- **The parsing strategy is not settled.** See
  [`docs/decisions/0001-parsing-strategy-for-multiple-languages.md`](../decisions/0001-parsing-strategy-for-multiple-languages.md):
  the decision is blocked, and the cross-platform build evidence that the
  specification requires is not available yet.

## Note on the OpenCode configuration format

The `opencode.json` of this repository uses the top-level `agent` key, with
`description`, `mode`, `permission` and `prompt` per entry, and the
top-level `instructions` array with repository-relative paths. This format
was validated against the installed **OpenCode v2.0.32** and against the
documentation published at <https://opencode.ai/docs/config/>.

The specification records this as an assumption that must be re-checked when
OpenCode is upgraded (section 17.6): the format has changed between
generations of the tool, so a future upgrade may require adjusting the
package. Adjust the package, not the specification, and record the change in
an ADR.
