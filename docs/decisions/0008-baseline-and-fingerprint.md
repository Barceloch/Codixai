# 0008 — The baseline of findings and the stable fingerprint of a finding

- Status: **Accepted**
- Date: 2026-10-22

## Context

The specification gives Phase 3 a baseline that lives in the analyzed
repository, that is shared through CI, and that lets a pipeline fail on the
new findings without turning the existing debt into constant noise.

The hard part of that requirement is the identity of a finding. A baseline
that stores the line number of a finding dies the first time that a line
moves, which is exactly what happens in a real repository: the same
suppression is added, the file grows above it, and the baseline no longer
recognizes it. The specification asks for a **stable fingerprint**: the rule,
the path and a normalized context, never the line number alone.

Three states have to be told apart in the same run:

- a finding that is new: it is not in the baseline and it reaches the
  threshold;
- a finding that is known: it is in the baseline, it is not new, and it does
  not reach the threshold. This includes a finding whose line is not in the
  diff: the rule and the path identify it, so the baseline still recognizes
  it while they do not change.

## Decision

### The fingerprint

The fingerprint of a finding is the first 16 hexadecimal characters of the
SHA-256 of three values joined by a null byte:

```text
sha256(rule_id + 0x00 + path + 0x00 + normalized_context)
```

The **normalized context** is the text of the line the finding points at,
taken from the diff of the compared range, with the leading and trailing
spaces removed and every run of internal whitespace collapsed into a single
space. The normalization makes the fingerprint survive a reindentation and
the movement of the code around the finding, and it makes it change when the
code of the line really changes.

When the line of the finding is not part of the diff, the context is empty
and the fingerprint identifies the rule and the path only. The finding is
still reported, and the baseline still recognizes it as long as the file and
the rule do not change.

### The baseline file

```json
{
  "schema_version": "0.1",
  "created_at": "2026-10-22T10:00:00Z",
  "tool_version": "0.1.0",
  "findings": [
    {
      "fingerprint": "3f2a91c0d4b7e815",
      "rule_id": "CDX-SUP-001",
      "path": "internal/payment/service.go",
      "line": 41,
      "severity": "warning",
      "message": "The change adds an analyzer suppression."
    }
  ]
}
```

- The file is versioned with its own `schema_version`, `0.1`. It is a
  versioned document of the analyzed repository, and it is committed with it.
- The findings are sorted by their fingerprint, so the file is stable and
  its diffs are readable.
- The line, the severity and the message are informative. They are NOT part
  of the identity of the finding: they change when the code moves, and the
  baseline must survive that.
- The file is human readable and reviewable: a reviewer can see which
  findings are known and can remove the entries that must be reviewed
  again.

### The behavior of the audit

- The audit reads the baseline of the repository when the file exists. The
  path comes from the `audit.baseline` key of the configuration, and it
  defaults to `.codixai/baseline.json`.
- A finding whose fingerprint is in the baseline is **known**: it is not
  reported as a new finding and it does not reach the threshold of the
  command.
- The count of the known findings is written to the **standard error**. The
  standard output holds only the document of the new findings, so the
  document of the audit keeps its shape and its `schema_version` does not
  change. This is the same rule as the one of the suppressed findings.
- A finding whose line is not in the diff still has a fingerprint: the rule
  and the path identify it, and the baseline recognizes it while they do
  not change. The identity is coarser than the one of a finding with
  context, and it is reviewable in the baseline file.

### The command

```bash
codixai baseline create --base main
codixai baseline create --worktree
```

- The command needs one of the three origins, exactly like the audit.
- It writes the baseline of the path of the configuration, unless `--out`
  names another one.
- It never overwrites an existing baseline without `--force`, and it says
  what it would overwrite.
- The baseline that is being created is never applied to the run that
  creates it.

## Consequences

Positive:

- The baseline survives the movement of the code, which is the normal case
  in a living repository.
- The identity of a finding is computed from the diff, so the baseline does
  not need an index of the repository.
- The file is reviewable and its diffs are readable.

Negative:

- A finding whose line is not in the diff is identified by the rule and the
  path only, so it is recognized as long as the file does not change.
- The fingerprint changes when the code of the line changes, so a finding
  that was fixed and reintroduced is new. That is the correct behavior: it
  has to be reviewed again.
- The audit reads one more file of the analyzed repository. The path comes
  from the configuration and it is inside the root.

## Alternatives considered

**Store the line number as the identity of a finding.**
Rejected. The baseline would break with the first change that moves a line,
which is the normal case in a repository with activity.

**Store a hash of the whole file.**
Rejected. A change anywhere in the file would make every finding of the
file new, and the baseline would lose its meaning.

**Store the whole finding, message included, in the identity.**
Rejected. The message changes when the rule changes, and the rules change
with their versions. The identity must only hold what describes the defect
in the code.

**Apply the baseline inside the analysis engine.**
Rejected. The engine decides what it observed. The policy of the repository
decides what it does with it, in the application layer, like the severity
overrides and the suppressions.
