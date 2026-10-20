# Tasks

This directory holds the executable work of the project. The specification
fixes the scope and the exit criteria of each phase; the tasks here fix the
atomic steps and their acceptance checks.

- [Phase 0 — Executable foundations](phase-0.md)
- [Phase 1 — The diff verifier spike](phase-1.md)

## The format of a task

Every task has a **stable identifier** and the following sections. A task
that is missing one of them is not ready to be executed.

```markdown
### P0-03 — Title of the task

- Status: DONE | PARTIAL | PENDING
- Depends on: P0-01, P0-02

**Objective**
One or two sentences: what changes in the repository when the task is done.

**Sections to read**
The sections of the specification that the task has to respect, by number.

**Deliverables**
The files, the packages and the commands that the task produces.

**Acceptance checks**
The commands that can be executed to prove the task is done, and what their
result has to be.

**Out of scope**
What the task explicitly does not do, so that nobody widens it.
```

The rules:

- **The identifier is stable.** A task is never renumbered. A task that is
  abandoned keeps its identifier and says so.
- **The dependencies are explicit.** A task that depends on a pending task
  cannot be started.
- **The acceptance checks are executable.** They are commands, not
  descriptions. "The code works" is not an acceptance check.
- **The out of scope is explicit.** The scope of a task is the deliverable,
  and everything else belongs to another task.
- **A task should fit in one review by one person in one session.**

## The format of the result report

The agent that executes a task reports its result with the following
structure. The report is the evidence of the task, and it is written with
the real results, not with the expected ones.

```markdown
## Result of <task id>

**Changes**
The files that were added, changed or deleted, with one line per file
about what it does.

**Commands executed**
The commands that were run, with their real result. A command that was not
run is not listed as if it had been.

**Not verified**
What could not be checked in this environment, and why. This section is
never empty by politeness: it is empty only when everything relevant was
actually verified.

**Open questions**
The decisions that the agent could not take alone, with the section of the
specification that requires them to be confirmed.
```

The rules:

- **A command that was not executed is never reported as executed.**
- **A test that was not run is never reported as passing.**
- **A finding that the agent believes to be a false positive is reported
  with its evidence**, not suppressed silently.
- **The result of the self check** (`codixai audit --diff --worktree` over
  the change of the agent) is part of the report when the task includes it.

## The stop conditions

The agent **stops and asks** instead of choosing, when:

1. The task depends on a pending decision: the URL of the repository, the
   license, the security contact, the supported platforms, the minimum Go
   version, the CI platform, the report formats.
2. The specification or the task is ambiguous, or two of them contradict
   each other. The precedence is: the design principles (sections 1.2 and
   4), then the text of the current phase (section 14), then the examples.
3. An unforeseen dependency, an access to the network or the execution of
   the code of the analyzed repository is required.
4. The change modifies an output format or an already fixed contract, in
   particular `schema_version`.
5. A test fails without a clear cause after two reasonably different
   attempts. The failure is reported; the test is not relaxed.
6. Declaring the support of a rule in a language would require skipping the
   fixtures.
7. Something cannot be verified in the environment, for example compiling on
   Windows or on macOS. It is declared as not verified, never as correct.

The values the agent may use without asking: the installed version of Go,
recorded in an ADR, and the English technical identifiers, comments and
commit messages. Everything else in the list of the pending decisions
requires a confirmation.

## The definition of ready and the definition of done

A task is **ready** when it has an objective, dependencies, the list of the
sections to read, deliverables, executable acceptance checks and an explicit
out of scope.

A task is **done** when:

- the acceptance checks were executed and their real result was reported;
- nothing was added outside the scope;
- the new decisions have an ADR;
- the diff was reviewed for dependencies, scope, conventions and secrets.

A task is not done because the code was written. It is not done because the
directories exist.
