# Agent Diffs Corpus (P1-04)

This corpus contains **reproducible diffs representing common patterns produced by AI coding agents**. They are synthetic but representative of real agent behavior, created to evaluate rule precision and false positives when real agent diffs are not available.

Each case follows the same structure as `testdata/rules/`:
- `<name>.diff` — unified diff
- `<name>.expected.json` — expected rule IDs and count
- `repo/` (optional) — repository context (manifests, lockfiles)

## Provenance

These diffs are **synthetic but representative**, created by the maintainer to cover patterns observed in AI-assisted development:
- Missing imports after adding new library usage
- Test modifications (skip, remove assertions, add flaky tests)
- Suppression directives added to silence linters
- Error handling patterns (empty catch, swallowed errors)
- Placeholder implementations (TODO, FIXME, NotImplementedError)
- Reference to non-existent symbols

They are NOT real diffs from external agents (which would require network access and authorization). The distinction is documented per the specification.

## Corpus statistics

| Rule | Language | Positive cases | Negative cases |
|------|----------|----------------|----------------|
| CDX-IMP-001 | 5 | 5 | 5 |
| CDX-REF-001 | 1 (Go) | 2 | 3 |
| CDX-TST-001 | 5 | 5 | 5 |
| CDX-SUP-001 | 5 | 5 | 5 |
| CDX-ERR-001 | 5 | 5 | 5 |
| CDX-PLH-001 | 5 | 5 | 5 |

Total: 55 cases (agent diffs corpus).

Combined with the fixture corpus (`testdata/rules/`, 60 cases), the full evaluation set contains 115 cases.

## Limitations and interpretation

**This is a synthetic corpus.** The cases are designed by the maintainer to cover common patterns; they do not represent a random sample of real-world code changes.

- A precision or recall of 1.00 on this corpus **does not prove** the same performance on real projects.
- The corpus is small: most rule/language combinations have only 1–3 positive cases and 1–3 negative cases.
- The expected labels reflect the maintainer's intent for each synthetic case. They are not independently verified ground truth.
- The cases cover patterns *observed* in AI-assisted development, but they do not capture the full diversity of syntax, formatting, and edge cases found in production codebases.
- Results from this corpus should be reported as "evaluation on synthetic corpus" and not presented as a general certification of quality.

## Evaluation methodology

Run: `go test ./internal/analysis/ -run TestAgentDiffsCorpus`

Expected outcome per case: rule IDs and count matching `.expected.json`.
