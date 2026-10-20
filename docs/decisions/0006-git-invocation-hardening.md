# 0006 — Hardening of the Git invocation

- Status: **Accepted**
- Date: 2026-10-20

## Context

Codixai analyses a repository that it did not write and that it does not
trust. Section 13 of the specification is explicit about the Git
invocation:

> Invoke Git always with an argument list (never through shell strings) and
> neutralize the executable configuration of the repository, for example
> `--no-ext-diff`, `--no-textconv` and `-c core.fsmonitor=false`. Check the
> full list against the current Git documentation and cover it with tests.

The threat is not hypothetical. **A repository that defines
`diff.<driver>.command` in its own configuration makes a plain `git diff`
execute that command.** A repository can also define a textconv filter, a
filesystem monitor hook, a pager or an alternative transport. Each of them
is code that the analyzed repository controls, and running it would break
the invariant that Codixai never executes the code of the analyzed
repository.

The same section forbids executing the hooks of Git and requires the paths
to be validated against path traversal and against symbolic links that leave
the root.

## Decision

**Every invocation of Git goes through a single hardened path in
`internal/integrations/git`, and it is covered by tests.**

The rules of the invocation:

1. **Argument slice, never a shell.** `exec.CommandContext` is called with
   the binary and the argument slice. No string is ever handed to a shell,
   so no argument can be word split or expanded.
2. **Environment scrubbing.** The environment of the Git process is rebuilt
   from scratch. These variables are removed: `GIT_DIR`, `GIT_WORK_TREE`,
   `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`,
   `GIT_ALTERNATE_OBJECT_DIRECTORIES`, `GIT_CEILING_DIRECTORIES`,
   `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`, `GIT_CONFIG_COUNT`,
   `GIT_CONFIG_PARAMETERS`, `GIT_PAGER`, `GIT_EDITOR`, and the whole
   `GIT_CONFIG_KEY_*` and `GIT_CONFIG_VALUE_*` families. These are set:
   `LC_ALL=C`, `GIT_PAGER=cat`, `GIT_TERMINAL_PROMPT=0`,
   `GIT_OPTIONAL_LOCKS=0`, `GIT_ATTR_NOSYSTEM=1`, `GIT_ASKPASS=` and
   `GIT_SSH_COMMAND=`. The effect is deterministic output, no interactive
   prompt, no credential helper, no external transport, and no system wide
   attributes that could define a diff driver.
3. **The executable configuration of the repository is neutralized.** The
   diff is requested with `--no-ext-diff --no-textconv --no-color
   --find-renames --unified=3`, and the following configuration is forced:
   `-c core.pager=cat`, `-c core.fsmonitor=false`, `-c core.quotePath=true`,
   `-c diff.relative=false`, `-c diff.noprefix=false`,
   `-c diff.mnemonicPrefix=false`, `-c diff.external=`, `-c diff.renames=true`,
   `-c diff.context=3`, `-c diff.suppressBlankEmpty=false` and
   `-c protocol.ext.allow=never`. `--no-pager` is always passed.
4. **`--no-optional-locks` on Git 2.15 and later.** The version of the
   binary is detected once, and the flag is added when it is supported, so
   that a read-only command does not refresh the index on disk and leaves the
   repository byte for byte untouched. An unknown version is treated as an
   old one and only the universally supported flags are used.
5. **`--end-of-options` on Git 2.24 and later.** It is used with
   `rev-parse`, so that a reference can never be read as an option.
6. **Validation of the base ref.** A base ref may not start with `-`, may
   not contain whitespace or control characters, may not contain `..`, and
   must match the revision name character set. It is then verified with
   `rev-parse --verify --quiet <ref>^{commit}`, and the error message tells
   the user that the ref must be a reachable commit, with examples.
7. **The revision list is terminated.** The diff is called with `--` at the
   end, so a path can never be taken for a reference.
8. **Only read-only subcommands.** `diff` and `rev-parse`. Nothing that
   writes to the repository, to the index or to the object database is ever
   called.
9. **No hook is ever executed.** Codixai does not install a hook, does not
   run one, and does not configure one.
10. **Paths.** The root is resolved to an absolute path and must be a
    directory. The paths that reach the output are repository relative and
    cannot point outside the root. The files that the analysis reads are
    the ones the diff names, plus the manifests of the root.

The `diff.<driver>.command` observation is the reproduction that justifies
the whole decision: without `--no-ext-diff` and `-c diff.external=`, a
repository controlled configuration would reach the process table.

## Consequences

Positive:

- The invariant that Codixai never executes the code of the analyzed
  repository holds for the Git integration too, not only for the rules.
- The analyzed repository is not modified: the integration test
  `TestAuditDoesNotModifyTheRepository` checks it, and
  `TestBinaryDoesNotWriteOutsideOfTheRepository` checks that nothing is
  written outside of it.
- The diff output is deterministic: the locale is fixed, the pager is
  disabled, the quote path and the prefix options are forced.
- The behaviour degrades safely: on an old Git, only the flags that every
  Git understands are used.

Negative:

- The invocation is longer and more verbose than a plain `git diff`.
- Some repository specific configuration is ignored on purpose. A
  repository that relies on a custom diff driver will not be diffed the way
  its author configured it. That is the intended trade-off.
- The support of the optional flags depends on the version of the Git binary
  of the user, which means the exact behavior is not identical everywhere.

## Alternatives considered

**Use a Git library instead of the Git binary (go-git).**
Rejected for now. A library implementation removes the whole class of the
executable configuration problems, but it is a third party dependency, it
is unavailable in the current environment, and it does not read the
configuration of the repository the way Git does, which changes the results
in ways that have to be measured. It remains a candidate when the
dependency policy changes.

**Use `git diff` and trust the configuration of the repository.**
Rejected. It is exactly the vulnerability the specification names.

**Use `git -c diff.external= diff` without `--no-ext-diff`.**
Rejected. The `-c` only overrides the global value of `diff.external`; a
driver defined per path in the repository configuration would still be
selected. `--no-ext-diff` and `--no-textconv` are the flags that disable
the mechanism.

**Run the diff through a shell with a quoted command line.**
Rejected. A shell adds word splitting, glob expansion and a second
interpreter to the trust boundary.

**Copy the working tree into a temporary directory and diff there.**
Rejected. It would multiply the disk usage of every audit and would still
need the same hardening for the copy operation.

**Use `--no-optional-locks` unconditionally.**
Rejected. The flag does not exist before Git 2.15, and passing an unknown
flag makes Git fail. The version is detected instead.
