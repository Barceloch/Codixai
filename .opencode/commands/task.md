---
description: Start a task of Codixai. Reads the task, confirms its scope and its acceptance checks, and reports the plan before the implementation.
agent: build
---

Execute the following task of Codixai: **$ARGUMENTS**

If `$ARGUMENTS` is empty, ask which task has to be executed and stop.

## Steps

1. **Locate the task.** The tasks live in `docs/tasks/`. If the identifier
   is unknown, list the tasks of the phase and ask which one to execute.
2. **Read the task** and extract: the objective, the dependencies, the
   sections of the specification to read, the deliverables, the acceptance
   checks and the out of scope.
3. **Check the dependencies.** If a dependency is pending, say so and stop.
4. **Read the sections of the specification** that the task names, and only
   those. The specification is `docs/spec/CODIXAI_SPEC.md`.
5. **Read the conventions** in `docs/guides/implementation-conventions.md`.
6. **Inspect the existing code and the tests** before proposing anything.
   The tests are the specification of the current behavior.
7. **State the plan**: the files to touch, the checks to run, and what is
   explicitly out of scope.

For a task of the graphical interface (Phase 7), load the skill `wails` of
this repository before writing any code, and follow the rules of the
section 17.9 of the specification.

## Stop conditions

Stop and ask, instead of choosing, when the task depends on a pending
decision (the security contact, the supported platforms, the minimum Go
version, the CI platform, the report formats, the version of Wails, the
languages of the interface), when the specification and the task contradict
each other, when a new dependency or an access to the network would be
needed, when the change would modify `schema_version` or another fixed
contract, or when something cannot be verified in this environment.

A task of the graphical interface that requires choosing the version of
Wails, adding an npm dependency or changing the distribution policy stops
and asks as well.

## Rules

- Implement the smallest change that satisfies the task.
- Run the acceptance checks of the task and the general checks:
  `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test ./...`.
- Never execute the code of the analyzed repository, never invoke a package
  manager, never access the network.
- Never claim that a test passed without running it.
- Report with the format of `docs/tasks/README.md`: the changes, the
  commands executed with their real result, what was not verified, and the
  open questions.
