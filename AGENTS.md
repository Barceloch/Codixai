# AGENTS.md

Invariants, workflow and stop conditions for the agents that work on
Codixai. The conventions are in
`docs/guides/implementation-conventions.md` and the task format is in
`docs/tasks/README.md`. Both are loaded on every session.

## Invariants

1. **Never execute the code of the analyzed repository.** No test runner, no
   build, no script, no package manager, no network. The only executable
   program Codixai starts is `git`, with read-only subcommands and with the
   hardening of `docs/decisions/0006-git-invocation-hardening.md`.
2. **The domain imports nothing.** `internal/domain` does not depend on the
   CLI, on the providers or on the analysis engine. The application layer
   declares the ports; the providers implement them.
3. **The application layer does not depend on the presentation.**
   `internal/application` imports neither the CLI nor the user interface nor
   a desktop toolkit; `internal/application/architecture_test.go` verifies
   it. The direction is `ui -> application -> domain`.
4. **Everything is deterministic.** The same input produces the same
   document. Sort whatever reaches an output.
5. **The results are versioned, serializable documents.** Every document
   carries `schema_version`; a change of a consumer visible field
   increments it.
6. **The epistemic status is honest.** A heuristic is `inferred`, never a
   confirmed bug. The absence of findings never proves that a change is
   correct. A claim that cannot be decided is `unresolved`, never
   `verified`.
7. **A rule may only claim support for a language that has positive and
   negative fixtures.** The CI validates the matrix against the corpus.
8. **The file access is bounded to the selected root.** Every path is
   resolved against the root of the analyzed repository; a path that
   escapes it is rejected.
9. **The code of a future interface lives behind build tags.** The Wails
   adapter and the compiled frontend belong to the `ui` variant: the
   default build needs neither Node.js nor Wails nor a webview.
10. **No telemetry, no code upload, no remote execution**, without an
    explicit review by the maintainer.
11. **English in the artifacts**: identifiers, comments, strings, documents
    and commit messages. The specification is in Spanish and is the
    contract.
12. **The standard output holds only the requested document.** The
    diagnostics go to the standard error.

## Workflow

1. **Read the task.** The active task is in `docs/tasks/`, and it names the
   sections of the specification to read. Read only those sections.
2. **Inspect the code and the tests** before proposing anything. The tests
   are the specification of the current behavior.
3. **Confirm the scope** and the acceptance checks. If the task is
   ambiguous, apply the stop conditions.
4. **Implement the smallest change** that satisfies the task. Nothing
   outside the scope.
5. **Run the checks** of the task and the general ones: `gofmt -l .` (it
   must print nothing), `go build ./...`, `go vet ./...` and `go test
   ./...`. When the task includes the self check, run `go run ./cmd/codixai
   audit --diff --worktree` over the change.
6. **Review the diff**: dependencies, scope, conventions, secrets.
7. **Report** with the format of `docs/tasks/README.md`: the changes, the
   commands executed with their real result, what was not verified, and the
   open questions.

## Stop conditions

Stop and ask, instead of choosing, when:

1. The task depends on a pending decision: the security contact, the
   platforms, the minimum Go version, the CI platform, the report formats,
   the version of Wails, the languages of the interface.
2. The specification or the task is ambiguous, or two of them contradict
   each other. The precedence is: sections 1.2 and 4, then the text of the
   current phase, then the examples.
3. An unforeseen dependency, an access to the network or the execution of
   the analyzed repository is required.
4. The change modifies an output format or a fixed contract, in particular
   `schema_version`.
5. A test fails without a clear cause after two reasonably different
   attempts. Report the failure; do not relax the test.
6. Declaring the support of a rule in a language would require skipping the
   fixtures.
7. Something cannot be verified in the environment, for example compiling on
   Windows or macOS. Declare it as not verified, never as correct.
8. A task of the graphical interface requires choosing the version of
   Wails, adding an npm dependency or changing the distribution policy
   (the pending decisions 11 to 13 of the specification). Stop and ask.

## Commands

```bash
gofmt -l .                                              # must be empty
go build ./... && go vet ./... && go test ./...
go test ./... -race
go test ./internal/analysis/ -run TestSupportMatrixMatchesFixtures
go test ./internal/claims/ -run TestClaimCorpus
go run ./cmd/codixai --help
go run ./cmd/codixai audit --diff --worktree
go run ./cmd/codixai audit --rules
go run ./cmd/codixai verify --help
```

## Rules for the report

- **Never claim that a test passed without running it.** A command that was
  not executed is not listed as executed.
- **Never invent a command, an API, a metric, a real diff or a capability
  of an external tool.** Never weaken, disable or delete a test to make the
  suite pass.
- **Mark what could not be verified as not verified.**
- **A change to the specification is a decision of the maintainer.** The
  agent updates the specification and records the decision in an ADR, or it
  asks.
