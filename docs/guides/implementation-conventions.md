# Implementation conventions

This document holds the concrete conventions of the repository. It is
loaded on every session through the `instructions` field of `opencode.json`,
so it is kept short and practical.

The product contract is the specification
([`docs/spec/CODIXAI_SPEC.md`](../spec/CODIXAI_SPEC.md)). When a convention
and the specification disagree, the specification wins and the difference
has to be fixed in one of the two.

## Go style

- **English** for every identifier, comment, doc comment, user-facing string
  and document. The specification is in Spanish; the artifacts of the
  repository are in English.
- **Doc comment on every exported symbol**, starting with the name of the
  symbol.
- **A package comment on every package**, explaining its responsibility and
  its limits, not a summary of its functions.
- **Small functions with early returns.** A guard clause beats a deep
  nesting.
- **Receivers are named consistently** within a package, and a receiver is
  not used when the method does not need the state.
- **`gofmt` is not optional.** `gofmt -l .` must print nothing.
- **No global mutable state** outside the registries of the language and of
  the rule catalog, which are read only after the initialization.
- **No `panic`** outside of the initialization of a package that cannot
  recover. A malformed input is an error, not a crash.

## Error handling and messages

- Wrap with `%w` when the caller may need to inspect the cause, and always
  add the context that identifies the object: the file, the path, the flag,
  the rule.
- **An error message tells the user how to recover.** A message that only
  names the problem is a defect.

  ```go
  // Not enough: the user does not know what to do.
  fmt.Errorf("cannot read the configuration")

  // Enough: the user knows what failed and what the options are.
  fmt.Errorf("base ref %q does not exist in the repository: it must be a reachable commit (for example 'main', 'origin/main' or a commit hash)", ref)
  ```
- The errors of the configuration loader name the file and the line:
  `.codixai.yml: line 12: invalid audit.fail_on "fatal", expected one of error, warning, none`.
- The errors of the CLI are written to the standard error with the prefix
  `codixai:`. The standard output only ever holds the requested document.
- The internal errors of a rule do not stop the analysis. They become a
  `Diagnostic` attached to the run, and the analysis continues with the
  other files.

## The JSON output contract

- **Every document carries `schema_version`.** Its value is `0.1` today and
  it lives in `domain.SchemaVersion`. A change that adds, removes or
  redefines a consumer visible field increments it.
- **The standard output holds exactly one document.** No banner, no color,
  no progress, no spinner, no diagnostic. The diagnostics, the warnings
  about the excluded files and the count of the suppressed findings go to
  the standard error.
- **The document is deterministic.** The same input produces the same
  output, except for `generated_at`. The findings are sorted by path, line
  and rule identifier.
- **The document is valid JSON with a stable shape.** The fields of a
  finding are `schema_version`, `rule_id`, `rule_version`, `severity`,
  `epistemic_status`, `language`, `message`, `location`, `revision`,
  `evidence`, `support` and `recommendation`.
- **The severity, the epistemic status and the support level always have a
  textual representation.** The report never hides them behind a color.
- `--format json` never mixes the presentation with the document. The
  renderer is only used for the text format.

## The exit code convention

| Code | Meaning |
| ---- | ------- |
| `0` | The operation succeeded and no finding reaches the threshold. |
| `1` | Execution or configuration error. |
| `2` | Invalid arguments. |
| `3` | The audit completed and the findings reach the `fail_on` threshold. |

The convention is stable and documented. A new command does not invent a
code.

`fail_on` accepts `error` (the default), `warning` and `none`. The
comparison is categorical: the severity ranks are `info` < `warning` <
`error`, and they express an order, not a measurement.

## The fixture corpus

- The corpus lives in `testdata/rules/<RULE-ID>/<language>/<kind>/`, where
  `<kind>` is `positive` or `negative`.
- A case holds a `<name>.diff` and a `<name>.expected.json` with
  `expected_rule_ids` and `expected_count`.
- A case may hold a `repo/` directory with the manifests of the analyzed
  project, and the analysis is rooted there. When a directory holds more
  than one case, each case may bring its own repository in a directory
  named after it, for example `predeclared_identifiers.repo/`, which takes
  precedence over the shared `repo/`.
- **A rule needs a positive and a negative case in every language it
  claims to support.** The CI validates the matrix against the corpus.
- The fixtures are small and readable. A fixture that needs a long
  explanation to understand is a fixture that has to be simplified.
- The expected result lists the rule identifiers, not the messages. The
  messages change; the contract does not.
- The fixtures are the documentation of the behavior of the rule. When the
  behavior of a rule changes, the fixtures change with it and the version of
  the rule is incremented.

## The epistemic status discipline

`epistemic_status` is the state of a **finding**:

- `observed` — the finding was checked directly against the code or the
  diff.
- `inferred` — the finding comes from a heuristic.
- `unknown` — the tool could not determine the outcome.

The rules that follow from it:

- **Never call a heuristic a confirmed bug.** A message says what was
  observed, not what it means.
- **A rule that produces both states documents the difference** in its
  limitations, so the reader knows which finding is a fact and which one is
  a signal.
- **The status is a property of the check, not of the message.** A rule
  does not downgrade a finding to make the report look better.
- `resolution` (`confirmed`, `inferred`, `unresolved`) is a different
  vocabulary: it classifies a **relation** between symbols or files. A
  finding never carries a `resolution`, and a relation never carries an
  `epistemic_status`.

## The invariant of not executing the analyzed repository

**Codixai never executes the code of the analyzed repository.** It never
runs a test, a build, a script or an application. It never invokes a
package manager, and it never accesses the network.

Concretely:

- A dependency is considered declared only when a manifest of the repository
  says so. The manifests are read from disk with the parsers of the project.
- The tools that the ecosystem provides, such as `go list` or
  `go/packages`, are not used, because they may download modules and they
  may run the toolchain of the project. The Go analysis uses `go/parser` and
  `go/ast`, which only read the source text.
- The only executable program that Codixai starts is `git`, with read-only
  subcommands and with the hardening described in
  [`docs/decisions/0006-git-invocation-hardening.md`](../decisions/0006-git-invocation-hardening.md).
- A rule that needs a new external tool must justify it in an ADR, and the
  tool must be optional, detected, and reported as unavailable with a
  useful message.

## The Git conventions

- Git is called with an argument slice, never through a shell.
- The environment of the Git process is scrubbed, and the executable
  configuration of the analyzed repository is neutralized.
- A base ref is validated before it is used.
- The integration tests create their own temporary repository. They never
  touch the repository of the project.

## The commit conventions

- [Conventional Commits](https://www.conventionalcommits.org/): `feat:`,
  `fix:`, `docs:`, `refactor:`, `test:`, `build:`, `chore:`, and the
  optional scope, for example `feat(rules): add the placeholder rule`.
- The subject is imperative, in the past tense of the English imperative
  mood, and it does not end with a period.
- **No AI attribution.** No `Co-Authored-By` trailer, no "generated with"
  line, no tool name in the message. The commit describes the change, not
  who wrote it.
- One commit is one reviewable unit of work. The tests and the
  documentation of a change travel with it.

## The documentation language rule

- **The documentation of the repository is in English.** That includes the
  README, the ADRs, the guides, the task files, the doc comments and the
  messages of the CLI.
- The specification is in Spanish and is the contract. It is read by
  section, according to the map of section 17.3, and it is not modified
  except by an explicit decision of the maintainer.
- The documentation is focused and skimmable. A document that repeats the
  specification verbatim has no value: it points to the section of the
  specification instead.
- A document states what is true. What is not verified is written as not
  verified.
