# Security Policy

Codixai is a local-first tool that reads a repository and reports findings.
This document describes what the tool does and does not do with the code it
reads, what a reporter should include, and where to report a problem.

## Supported versions

The project has not published a release yet. The `0.1.0` entry in the
[changelog](CHANGELOG.md) is unreleased. Security fixes land on the default
branch; there is no release branch to patch yet.

## Threat model summary

### What Codixai does

- Reads the diff of a Git repository and the text of the changed files.
- Reads the manifests of the analyzed repository (`go.mod`,
  `requirements.txt`, `pyproject.toml`, `composer.json`, `package.json`) with
  its own parsers.
- Writes one document to the standard output and diagnostics to the standard
  error.

### What Codixai never does

- **It never executes the code of the analyzed repository.** No test runner,
  no build, no script, no application is started. The only executable
  program Codixai starts is `git`, and only with read-only subcommands.
- **It never invokes a package manager.** No `npm install`, no
  `composer install`, no `pip install`, no `go get`. A dependency is
  considered declared only when the manifest of the repository says so.
- **It never accesses the network.** There is no HTTP client in the binary
  and no update check.
- **It never writes to the analyzed repository.** The commands are read-only
  by default. `codixai init` creates `.codixai.yml`, and it refuses to
  overwrite an existing file unless `--force` is given explicitly.
- **It never installs hooks.** Codixai does not write a Git hook, a pre-commit
  hook or any other file in `.git/`.

### Git invocation hardening

A repository can define executable configuration for Git: an external diff
driver, a textconv filter, an fsmonitor hook, a pager, or an alternative
transport. A plain `git diff` would run some of them. Codixai neutralizes
that configuration on every invocation:

- Git is always called with an argument slice, never through a shell, so no
  argument is ever interpreted by a shell.
- The environment is scrubbed: `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`,
  `GIT_OBJECT_DIRECTORY`, `GIT_ALTERNATE_OBJECT_DIRECTORIES`,
  `GIT_CEILING_DIRECTORIES`, `GIT_EXTERNAL_DIFF`, `GIT_DIFF_OPTS`,
  `GIT_CONFIG_COUNT`, `GIT_CONFIG_PARAMETERS`, `GIT_PAGER`, `GIT_EDITOR`,
  and the `GIT_CONFIG_KEY_*` / `GIT_CONFIG_VALUE_*` families are removed.
  `LC_ALL=C`, `GIT_TERMINAL_PROMPT=0`, `GIT_OPTIONAL_LOCKS=0`,
  `GIT_ATTR_NOSYSTEM=1`, `GIT_ASKPASS=` and `GIT_SSH_COMMAND=` are set.
- The diff is requested with `--no-ext-diff --no-textconv --no-color`, so an
  external diff driver or a textconv filter defined by the repository is
  never executed.
- `-c core.fsmonitor=false` disables the filesystem monitor hook, and
  `-c diff.external=` clears the external diff command.
- `-c protocol.ext.allow=never` disables the external transport helpers.
- `--no-pager` is always passed, and `core.pager=cat` is set.
- `--no-optional-locks` is used on Git 2.15 and later, so a read-only command
  does not refresh the index on disk.
- `--end-of-options` is used on Git 2.24 and later for `rev-parse`, so a
  reference can never be read as an option.
- A base ref is validated before use: it may not start with `-`, may not
  contain whitespace or control characters, may not contain `..`, and must
  match the revision name character set. The ref is then verified to exist
  as a reachable commit, with an actionable error when it does not.
- The diff is terminated with `--`, so a path can never be taken for a
  reference.

The reproducible observation behind this hardening: a repository that
defines `diff.<driver>.command` in its own configuration makes a plain
`git diff` execute that command. Codixai never lets the analyzed repository
reach the process table.

### Paths and symlinks

- The analyzed root is resolved to an absolute path and must be a directory.
- Every path that reaches the output is a repository-relative path, and the
  path of a finding cannot point outside the analyzed repository.
- The analysis only reads files that the diff itself names, plus the
  manifests of the repository root.
- Paths excluded by the configuration (`vendor/**`, `node_modules/**`,
  `dist/**`, `build/**`, `.git/**`, `coverage/**` by default) are removed
  before the analysis and are reported on the standard error.

### Secrets

- Codixai does not send anything anywhere, so there is no exfiltration path
  in the product.
- The output contains file paths, line numbers, rule identifiers and the
  message of the finding. A finding message quotes the identifier that
  triggered the rule, which can be a fragment of the changed line. Treat the
  JSON document as sensitive when the repository contains secrets: it is
  derived from your code.
- The standard output is the only channel that carries the document.
  Diagnostics, warnings about excluded files and suppressed findings go to
  the standard error, so a redirected standard output stays clean.

### Limits

- The analysis is read-only and bounded by the size of the diff.
- The number of files, the size of the output and the runtime are bounded by
  the resources of the machine that runs it. Codixai does not implement
  configurable resource limits yet.

### Known risks

- Parsers and external tools have their own vulnerabilities. The parsers used
  today are the Go standard library and the standard library scanner of the
  project itself, but any parser added later must be kept up to date.
- The rules are syntactic heuristics. A finding is a signal, not a proof of a
  defect. Do not treat the presence or the absence of findings as a verdict
  on the safety of a change.

## Reporting a vulnerability

**The reporting channel is the private security advisory of the GitHub
repository, once its URL is confirmed. The contact address is PENDING
maintainer confirmation. No email address is published here yet, because
none has been confirmed.**

Until the channel is confirmed, open a regular issue that describes the
problem at a high level, without a working exploit, and ask the maintainer
for the private channel.

### What to include

- The version of Codixai (`codixai version`) and the version of Git.
- The platform and the operating system.
- A description of the problem and of the impact you expect.
- The steps to reproduce, with a reduced repository when possible. Do not
  send a private repository.
- Whether the issue has already been reported.

### What not to include

- Credentials, tokens, keys or the content of a private repository.
- A proof-of-concept that exfiltrates data or damages the machine.

### What to expect

- An acknowledgement of the report.
- An assessment of the impact and of the fix.
- A fix and, when the maintainer decides it is appropriate, a credit in the
  changelog.

## Non-security bugs

Functional bugs belong in the issue tracker, not in the security advisory.
See [CONTRIBUTING.md](CONTRIBUTING.md) for the information a bug report
should carry.
