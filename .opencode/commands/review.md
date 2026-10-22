---
description: Review the current change of Codixai. Read-only review of the diff against the active task, the specification and the conventions.
agent: codixai-reviewer
---

Review the current change of the repository: **$ARGUMENTS**

If `$ARGUMENTS` is empty, review the whole working tree change.

## Steps

1. **Establish the scope of the change.** Read the diff of the working tree
   and the diff of the branch, when there is one. Never modify the
   repository.
2. **Find the task** that the change belongs to, in `docs/tasks/`. If there
   is no task, say so: a change without a task has no defined scope, and
   that is itself the first finding.
3. **Read the sections of the specification** that the task names.
4. **Read `docs/guides/implementation-conventions.md`** and the invariants
   of `AGENTS.md`.
5. **Check the change** against:
   - the scope of the task: anything outside it is a finding;
   - the layer boundaries: `internal/domain` imports nothing, the
     application declares the ports, the providers implement them, and the
     application layer imports neither the command line nor the user
     interface nor a desktop toolkit;
   - the output contract: `schema_version`, a clean standard output, the
     documented exit codes;
   - the epistemic status: a heuristic is never presented as a confirmed
     bug, and a claim that cannot be decided is `unresolved`, never
     `verified`;
   - the claim contract: an unknown type is an error, a report that cannot
     be linked to the revision leaves the claim `unresolved`, and Codixai
     never runs the tests of the analyzed repository;
   - the support matrix: a rule may only claim a language that has positive
     and negative fixtures;
   - the invariants of not executing the analyzed repository, of no
     telemetry, of no upload of code and of no remote execution;
   - the evidence: every claim of the change has to be backed by something
     observable.
6. **Report the findings.**

## Output

For each finding: the file and the line, what is wrong, why it matters, and
the smallest correction. Order the findings by severity and separate the
confirmed problems from the open questions.

State explicitly what was **not** verified. Never claim that a test passed
without having run it.

A change of the graphical interface (Phase 7) is reviewed against the rules
of the section 17.9 of the specification: one version of Wails only, no
import of Wails in the domain nor in the application, generated types in the
frontend, no `{@html}`, and `npm ci` for the dependencies of the frontend.
