# CI integration

Codixai is meant to run in a pipeline: it reads the changes of a repository,
applies a deterministic rule catalog and fails only when the policy says so.
This guide covers the three things a pipeline needs: the origin flag, the
exit codes, and the two ways of sharing the result — the baseline and the
SARIF document.

The tool is read-only. It never runs the tests or the build of the analyzed
repository, never invokes a package manager and never accesses the network.
It starts `git` with read-only subcommands and nothing else.

**There is no published binary yet.** Build the tool from source in the
pipeline:

```bash
go build -ldflags "-X main.version=$GIT_COMMIT" -o ./bin/codixai ./cmd/codixai
```

Use the built binary, not `go run`: `go run` does not propagate the exit code
of the program it runs, and the exit code is the contract of the pipeline.

## One origin flag, always

Every diff command — `audit`, `report` and `baseline create` — needs exactly
one origin:

| Flag | The changes it selects |
| ---- | ---------------------- |
| `--base <ref>` | The commits between `<ref>` and `HEAD` |
| `--staged` | The staging area |
| `--worktree` | The changes that are not committed yet |

Zero origins or two origins are an invalid invocation, and the exit code is
`2`. In a pipeline the origin is `--base origin/main` (the base branch) after
a checkout that fetches it. The base ref has to be reachable: a ref that does
not exist is an execution error, not an empty run.

## The exit codes

| Code | Meaning |
| ---- | ------- |
| `0` | The operation succeeded and no finding reaches the `fail_on` threshold. |
| `1` | Execution or configuration error. |
| `2` | Invalid arguments. |
| `3` | The audit completed and a new finding reaches the threshold — or, for `verify`, a claim is `contradicted`. |

`audit` and `report` share the codes: `report` is the same run with a machine
readable document. `fail_on` accepts `error` (the default), `warning` and
`none`.

`verify` uses the same convention with one difference: `3` means that a claim
is `contradicted`. An `unresolved` claim is not a failure unless the
repository sets `verify.fail_on_unresolved: true`.

The document goes to the standard output and nothing else does. The
diagnostics go to the standard error: the excluded files, the suppressed
findings, the findings that the baseline knows and the failure threshold.

## The baseline workflow

A baseline is a versioned file of the analyzed repository that records the
findings the repository already has. It lets the pipeline fail on the new
findings without turning the existing debt into constant noise.

**1. Create it once, on a branch.**

```bash
codixai baseline create --base main
```

The file is written at the `audit.baseline` key of the configuration, which
defaults to `.codixai/baseline.json`. `--out <path>` writes another one and a
relative path is resolved against the root. The command refuses to replace an
existing baseline without `--force`, and it says how many findings the file
it would replace holds.

**2. Commit it with the repository.**

```bash
git add .codixai/baseline.json
```

The file has its own `schema_version` (`0.1`) and its findings are sorted by
their fingerprint, so its diffs are readable.

**3. Review every change of the file.**

An entry that appears is a finding that was accepted into the debt. An entry
that disappears is a finding that is not reported anymore. Both are decisions
about the repository and belong in the review of the change, like the code.
Do not refresh the baseline inside the pipeline: a baseline that updates
itself defeats its purpose.

**4. Audit in the pipeline.**

```bash
codixai audit --diff --base origin/main
```

The audit reads the baseline, removes the findings it knows and fails with
`3` only when a new finding reaches the threshold. The count of the known
findings is a diagnostic:

```text
codixai: 2 finding(s) known by the baseline, not reported
```

The document of the audit keeps its shape and its `schema_version`: the
known findings are not in it.

**5. Refresh it deliberately.**

```bash
codixai baseline create --base main --force
```

Then review the diff of the file and commit it.

### What the fingerprint survives

The identity of a finding is `sha256(rule_id + 0x00 + path + 0x00 +
normalized_context)`, cut to 16 hexadecimal characters. The line number is
not part of it, so a finding that moves down a file stays known. It changes
when the code of the line changes, so a finding that was fixed and
reintroduced is new and has to be reviewed again.

When the line of a finding is not part of the compared range, the context is
empty and the identity is the rule and the path; the baseline recognizes it
as long as they do not change. The rule is the decision
[`0008`](../decisions/0008-baseline-and-fingerprint.md).

## The SARIF document

```bash
codixai report --format sarif --base origin/main > codixai.sarif
```

`--format sarif` is the default of the `report` command; `--format json`
writes the audit document instead, byte for byte the one that `audit --format
json` writes. The exit code is the exit code of the audit, so the step fails
with `3` when a new finding reaches the threshold — the same run, one
document for the machine.

What the document carries:

- one run, the driver `codixai` with its version and the rules of the
  catalog, sorted by identifier;
- one result per new finding: `ruleId`, `ruleIndex`, `level`, `message`, the
  repository relative location and the line range;
- the severity mapping `error` → `error`, `warning` → `warning`,
  `info` → `note`;
- `partialFingerprints["codixaiFingerprint/v1"]`: the same fingerprint the
  baseline uses, so a platform can recognize a known result;
- `runs[].properties.suppressedByBaseline`: the count of the findings the
  baseline knows. They are not results, because they are not new.

The mapping and the shape are fixed by the decision
[`0009`](../decisions/0009-sarif-export.md).

### The limitation, stated

**The document is structurally conformant to SARIF 2.1.0, and it has not been
validated against the official JSON schema.** The development environment has
no network access to fetch
`https://json.schemastore.org/sarif-2.1.0.json`. The tests validate the
version, the run, the driver, the rules, the results, the locations and the
fingerprints, not the schema.

Confirm the import once on your platform before you make the step blocking.
If the platform rejects the document, report it: the shape of the export is a
contract and a rejection is a defect.

## An example pipeline

The sketch uses the step syntax of GitHub Actions; the steps translate to any
provider and only the upload action changes.

```yaml
steps:
  - name: Checkout
    run: git fetch --no-tags origin main && git checkout "$GIT_COMMIT"

  - name: Build Codixai
    run: go build -o ./bin/codixai ./cmd/codixai

  - name: Audit the new findings
    run: ./bin/codixai audit --diff --base origin/main

  - name: Write the SARIF document
    if: always()
    run: ./bin/codixai report --format sarif --base origin/main > codixai.sarif

  - name: Upload the SARIF document
    if: always()
    # The action of the code scanning platform of your choice.
    # It reads codixai.sarif and imports the rules, the results and the
    # fingerprints.
    run: echo "upload codixai.sarif with the action of your platform"
```

Notes on the sketch:

- The upload step runs with `always()` on purpose: the document of a failing
  audit is the one the platform has to import.
- The audit step is the one that decides the exit code of the pipeline. The
  report step repeats the same run, so it also fails with `3` when there is a
  new finding. Keep only one of the two as the blocking step if the pipeline
  reports the same failure twice.
- Do not create the baseline in the pipeline. It is created on a branch,
  reviewed and committed.

## The verification in the pipeline

The claims of an agent are checked with the same convention:

```bash
codixai verify --base origin/main --claims claims.json
```

The command fails with `3` when a claim is `contradicted`. An `unresolved`
claim does not fail by default, because the absence of evidence is not
evidence of a false claim. Codixai does not run the tests of the analyzed
repository: a `tests_passed` claim is only checked against a report that the
pipeline provides and that declares the revision under analysis.
