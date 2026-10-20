# 0003 — The CLI framework

- Status: **Accepted (provisional)**
- Date: 2026-10-20

## Context

The specification names **Cobra** as the framework of the CLI (section 6.1),
together with the Charm libraries (Lip Gloss, Glamour, Bubbles) for the
presentation and Chroma as a candidate for the syntax highlighting.

The development environment has **no network access**, so no module can be
fetched: Cobra, Lip Gloss, Glamour, Bubbles and Chroma are all unavailable
today. The specification also states that the presentation layer of Phase 1
only needs a minimum — centralized styles, `--no-color`, `NO_COLOR`,
`--plain`, clean JSON and actionable errors — and that Glamour, Chroma and
Bubbles are incorporated when they add measurable value and do not block the
first milestone.

The decision was therefore between importing a framework that could not be
used yet, or writing the minimum that the milestone needs while keeping the
migration path open.

## Decision

**Ship a small internal command framework in `internal/cli`, with the whole
command tree isolated in `internal/cli/commands.go`, so that the migration to
Cobra is mechanical.**

- The framework lives in `internal/cli/framework.go`: a `Command` type with
  `Use`, `Short`, `Long`, `Example`, `RunE`, `Flags` and `PersistentFlags`,
  a `flag.FlagSet` per command, and a `dispatch` method.
- The whole tree is built in one function, `NewRootCommand`, and every
  command declares its flags and its handler in one constructor:
  `newAuditCommand`, `newInitCommand`, `newVersionCommand` and
  `newHelpCommand`.
- The global flags are declared in `root.PersistentFlags` and are accepted
  before and after the name of the command, so `codixai --format json audit`
  and `codixai audit --format json` behave the same way.
- The exit codes of the specification are mapped in one place, in
  `cli.Execute`: `0` success, `1` execution or configuration error, `2`
  invalid arguments, `3` findings that reach the threshold.
- The presentation lives in `internal/cli/ui`, separately from the use cases
  and from the domain model, with a centralized renderer, a table helper, the
  `NO_COLOR` support and the plain mode.

The migration to Cobra, when it happens, is a rewrite of the framework file
and of the command constructors, not of the use cases, of the rules or of
the output contract.

## Consequences

Positive:

- The tool builds and runs with no dependency at all, which is what the
  environment allows and what keeps the build reproducible offline.
- The command surface, the flags and the handlers can be read in one file.
- The exit code convention and the stdout/stderr split are explicit and
  tested.
- The presentation layer is separated from the domain, so the rules and the
  use cases never depend on a terminal.

Negative:

- The framework does not offer the features of Cobra: no shell completion, no
  command groups, no generated documentation, no richer help templates. The
  help output is hand written and stable, but it is not extensible.
- The specification exit criterion for Phase 0 that names the CLI stack is
  **not met**: the stack is not Cobra. The decision is provisional for that
  reason.
- The migration is a real cost, and it is deferred to Phase 1.5.

## Alternatives considered

**Import Cobra now and accept that the build fails without network.**
Rejected. A project that does not build in its own development environment
cannot be reviewed, tested or verified, and the milestone requires a
compilable clean clone.

**Copy a minimal part of Cobra.**
Rejected. A vendored copy of a third party framework is a fork with a
maintenance burden, a license obligation and no upstream support, for a
surface that is not needed yet.

**Write the CLI as a plain `main` with a `switch` on the first argument.**
Rejected. The command tree, the help output and the flag parsing are part of
the product contract of the specification, and the tests of the framework
already exercise them.

**Adopt Cobra in Phase 1.5, together with Lip Gloss and Glamour.**
Accepted as the plan. The swap is a Phase 1.5 task, and it is listed as such
in the task list.
