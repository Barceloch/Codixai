# Feature: Release v0.1.0-rc.2 — precompiled binaries

Objective: ship downloadable, precompiled artefacts of Codixai v0.1.0-rc.2
for Linux amd64 and Windows amd64 (CLI and UI), with checksums, a reproducible
GitHub Actions pipeline, and a documented download/installation section.

## Problem

`v0.1.0-rc.1` has no published artefacts, and the source at `HEAD` cannot
produce a UI binary: `frontend/src/App.svelte` declares `cancel` twice, the
verify tab references bindings that were never generated (`VerifyAsync`,
`ChooseFile`, `onVerifyEvent`), and the Go adapter of the verify screen emits
no event and requires a Go callback that JavaScript cannot pass.

## Constraints

- Never execute the code of an analysed repository (Codixai invariant 1).
- The UI variant is built with `-tags "ui production webkit2_41"` on Linux and
  `-tags "ui production"` on Windows.
- The version of the CLI is injected with `-X main.version=0.1.0-rc.2`.
- The artefacts must come from the workflow, not from this machine's state.
- Nothing of `v0.1.0-rc.1` is rewritten; the tag stays untouched.

## Tasks

- [x] T1 — Complete the Go adapter of the verify screen (event driven, no Go
  callback) and add the missing file dialog binding.
  - Route: delegated writer (2 non-trivial files).
  - Files: `ui/ui.go`, `ui/ui_test.go`, `internal/cli/ui_command_ui.go`.
  - Acceptance: `go test -tags ui ./ui/`, `go build -tags "ui production
    webkit2_41" ./cmd/codixai`. All 9 new tests pass with `-race`.

- [x] T2 — Regenerate the Wails bindings and repair the frontend.
  - Route: delegated writer.
  - Files: `frontend/src/lib/wailsjs/**` (generated), `frontend/src/lib/bridge.ts`,
    `frontend/src/App.svelte`, `frontend/src/lib/components/VerifyForm.svelte`,
    `frontend/src/lib/components/AuditForm.svelte`,
    `frontend/src/lib/components/FindingDetail.svelte`,
    `frontend/src/lib/i18n.svelte.ts`.
  - Acceptance: `pnpm --dir frontend build` (0 errors), `pnpm --dir frontend
    run check` (0 errors, 6 pre-existing warnings), `go build ./...`.

- [x] T3 — Reproducible packaging script and release workflow.
  - Route: inline (new files, understood design).
  - Files: `scripts/release/package.sh`, `.github/workflows/release.yml`,
    `scripts/release/visual-check.mjs`.
  - Acceptance: the script packages the four artefacts, the checksums verify,
    the contents are exactly the expected files and no forbidden file travels
    in them. The workflow YAML is valid and every embedded bash block parses.

- [x] T4 — Documentation: `README.md` "Download and Installation",
  `CHANGELOG.md`, ADR `0016` for the artifact layout, Spanish guide
  `docs/guides/codixai-usage.es.md`.

- [x] T5 — Verification: full Go checks, real UI launch on Linux, CLI smoke
  test, package and checksum verification, WCAG AA contrast check on the served
  CSS.

- [ ] T6 — Publish: commit, annotated tag `v0.1.0-rc.2`, release with assets.

## Delivery

- Strategy: `single-pr` is not applicable (no PR): the release is one branch
  (`feat/phase-0-1-verifier`) plus an annotated tag.
- Forecast: below the 400 authored changed line budget except for the release
  workflow and documentation.

## Route per task

- T1 Go adapter: **delegated direct** (2 non-trivial files, `ui/ui.go` and
  `ui/ui_test.go`, plus `internal/cli/ui_command_ui.go`).
- T2 Frontend: **delegated direct** (4 files: `App.svelte`, `bridge.ts`,
  `VerifyForm.svelte`, `i18n.svelte.ts`, plus the two components that did not
  build).
- T3 Packaging script and release workflow: **inline** (new files, understood
  design).
- T4 Documentation: **inline** (README, CHANGELOG, ADR 0016, Spanish guide).
- T5 Verification: **inline** plus the runner jobs of the workflow.
- T6 Publish: **inline** (commit, annotated tag, release).

## Acceptance criteria

- `go test ./... -race` passes.
- `gofmt -l .` prints nothing in the files this release touches (18 pre-existing
  unformatted files are untouched and are not part of this release).
- `pnpm --dir frontend build` and `pnpm --dir frontend run check` pass with 0
  errors.
- The Linux UI binary starts under Xvfb and quits by itself with
  `CODIXAI_UI_AUTOQUIT=4s`.
- The four packages are produced by `scripts/release/package.sh`, their
  checksums verify, their contents are exactly the expected files and no
  forbidden file travels in them.
- A developer without Go can download a package, follow the README and run
  Codixai.

## Evidence log

- Inspection: `go test ./...` passes at `aec990f`. `pnpm run build` fails
  (`Identifier 'cancel' has already been declared`). `svelte-check`: 29 errors.
  `go build -tags "ui production webkit2_41"` passes; the window starts and
  quits with `CODIXAI_UI_AUTOQUIT=2s` (exit 0) on Manjaro.
- Release state: repository is public, default branch
  `feat/phase-0-1-verifier`, zero workflow runs, zero releases, tag
  `v0.1.0-rc.1` present and untouched.
- WCAG AA contrast on the served CSS: 8/8 checks pass (worst 6.78:1 against a
  4.5:1 minimum). No `text-transform: uppercase` on the body and no
  `Lenguaje`/`LANGUAGE` label in the stylesheet.