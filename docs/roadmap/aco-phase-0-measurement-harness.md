# ACO Phase 0 — Measurement Harness

- Date: 2026-11-14
- Status: implemented (Phase 0 only)

## 1. Purpose

A minimal, deterministic, local-first harness that records and compares the token and context consumption of an AI coding agent with and without Codixai. It never calls a model provider, never transmits data and never executes analyzed code.

## 2. Command surface

- `codixai measure validate <file>` — validate a run record.
- `codixai measure compare --baseline <file> --codixai <file>` — compare two runs.
- `codixai measure tasks` — list registered measurement tasks.

Output follows the standard contract: the document goes to stdout, diagnostics to stderr. Use `--format json` for the machine-readable document.

## 3. Data model

Run records are JSON documents with `schema_version` (independent, `0.1`). Each record carries:

- `run_id`, `scenario` (`baseline` or `codixai`), `task_id`, `repository` (root, revision, state), `model`, `environment`.
- `metrics`: `input_tokens`, `output_tokens`, `total_tokens` (each with a `source`), `files_inspected`, `files_read_full`, `tool_calls`, `task_completed`, `task_correct`, `duration_ms`.
- `completeness`: optional reference counts.
- `recorded_at` (RFC3339).

Token sources: `provider_reported`, `tokenizer_estimate`, `byte_proxy`, `unknown`.

## 4. Token provenance rules

- `provider_reported` counts are the only values that may support a savings claim.
- `tokenizer_estimate` and `byte_proxy` are estimates and are never reported as savings.
- Missing counts are recorded as `unknown` (null value), never as zero.
- `ValidateRunRecord` rejects a value whose source is `unknown` but a value is present, and a missing value whose source is not `unknown`.

## 5. Comparability and savings

`Compare` requires the same `task_id`, repository revision and model in both runs; otherwise the runs are not comparable and no savings is computed. Both scenarios must be labelled correctly.

Savings (`Savings = (1 - Tc / Tb) * 100`) is computed only when both `total_tokens` are `provider_reported` with a positive baseline. Otherwise `savings` is null and the interpretation states why. Completeness is reported alongside any savings; a saving that drops a reference is a regression.

## 6. Byte proxy

`EstimateTokensFromBytes(n)` returns `floor(n / 4)`. It is a deterministic sizing proxy and must never be reported as a provider-reported token count. It is exposed for package sizing and budgets only.

## 7. Tasks

Five representative task kinds are registered (investigate_symbol, find_callers, analyze_change, investigate_finding, review_diff). Tasks are data, not code, so new tasks can be added to `internal/measurement/tasks.json` without redesigning the data model.

## 8. Limitations

- The harness does not call any model provider; token counts are supplied by the operator from a real run.
- No savings claim is valid without a comparable baseline recorded under equivalent conditions.
- Wall-clock time and tool-call counts depend on the operator's environment and are not normalized.
- The registered tasks are templates; no measured results are shipped.

## 9. How to add a task

Append an entry to `internal/measurement/tasks.json` with a unique `id`, a `kind`, a `description` and an optional `repository`. Re-run `codixai measure tasks`.
