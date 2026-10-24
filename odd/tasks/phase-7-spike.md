# Feature: Phase 7 spike — packaging, Wails v2 and the `ui` variant

Objective: deliver the spike 7.0 of the specification (section 14, "Fase 7",
items 7.0) — the Wails version decision (ADR 0011), a minimal prototype bound
to a read-only use case, the `ui` build tag discipline, and the size and
startup measurements of both variants.

Binding decisions:
- ADR 0011: Wails v2, module pinned to v2.17.0 (the installed CLI). Spike
  frontend is static vanilla assets, no Node.js. pnpm is the package manager
  of the later production stack.
- ADR 0010: the released artifact is a single binary CLI+UI; the CLI-only
  build stays the default of `go build` and `go install`.

Scope (tasks):

- [x] T1 — Pin `github.com/wailsapp/wails/v2 v2.17.0` in go.mod and create the
  top-level `ui/` adapter behind the `ui` build tag, with embedded vanilla
  assets, bound to the read-only audit use case (`RunAudit --worktree`).
  - Route: `go get github.com/wailsapp/wails/v2@v2.17.0` then `go mod tidy`;
    `ui/ui.go` (types, `App`, `Run`), `ui/window.go`
    (`//go:build ui && (production || dev)`, the `wails.Run` options and the
    embedded `ui/assets`), `ui/window_untagged.go` (the actionable error of a
    build without the Wails tags), `ui/assets/{index.html,app.js,styles.css}`.
  - Binding: `ui.App.Audit(ui.AuditRequest) (*ui.AuditView, error)` calls
    `application.RunAudit` through the `ui.AuditFunc` composed in
    `internal/cli/ui_command_ui.go`. The page calls
    `window.go.ui.App.Audit`; the v2 runtime scripts `/wails/ipc.js` and
    `/wails/runtime.js` are served by the Wails asset server.
  - Evidence: `go build -tags ui ./...` and
    `go build -tags "ui production webkit2_41" ./cmd/codixai` exit `0`;
    `go test -tags ui ./ui/` passes five adapter tests. The unique strings of
    `ui/assets` are present in the `ui` binary and absent from the default
    binary, which proves the embed.
- [x] T2 — The `codixai ui` command in both variants: default build answers a
  usage error explaining how to obtain the `ui` variant (exit 2); the `ui`
  build launches the window. Test-first on the error path.
  - Route: `internal/cli/ui_command.go` (`//go:build !ui`) and
    `internal/cli/ui_command_ui.go` (`//go:build ui`); the default message
    names `go build -tags "ui production" ./cmd/codixai` and the release
    binary.
  - Evidence: the test `tests/integration/ui_command_test.go` was written
    before the implementation. The RED was re-observed on 2026-10-24 by
    neutralizing the handler: `the exit code is 0, want 2`; restored, it
    passes. The built CLI answers `codixai ui` with the usage error and exit
    `2`.
- [x] T3 — P7-03: prove the default build compiles without Node.js, without
  Wails and without webview deps (`go build ./...` plus a dependency scan of
  the default binary/module graph).
  - Route: build-tag isolation (compiler-enforced) plus a manual scan; no
    test shells out to `go list`.
  - Evidence: `go list -deps ./cmd/codixai` → 138 packages, no
    `wails`/`webview`/`webkit`/`gtk`/`dbus` package; `CGO_ENABLED=0 go build
    ./cmd/codixai` exits `0`; the `ui production` graph reaches 302 packages
    and 28 `wailsapp/wails` packages.
- [x] T4 — P7-04: measure the size and startup time of both variants and
  record the real numbers.
  - Route: `stat -c%s` over the built binaries and the wall time of
    `codixai version` and of `CODIXAI_UI_AUTOQUIT=<duration> codixai ui`.
  - Evidence: CLI 6,637,046 B / 0.003 s; `ui` tag only 8,923,864 B (does not
    open the window); `ui production` 14,790,328 B with 0.303–0.404 s (0.1 s
    auto-quit), 1.185–1.269 s (1 s) and 2.265–2.288 s (2 s), i.e. about
    0.23–0.29 s of startup. The table lives in `docs/tasks/phase-7.md`,
    P7-04.
  - Finding: the first `wails build` attempt wrote the generated
    `wailsjs/runtime` wrappers into `ui/assets`, the `go:embed all:assets`
    directive embedded them and the binary grew by 24,576 B. `wailsjsdir`
    now points at `./ui/wailsjs` (outside the embed) and `ui/.gitignore`
    ignores it.
- [x] T5 — Update docs/tasks/phase-7.md (P7-01..P7-04 statuses with the real
  acceptance results) and CHANGELOG.md. README gets its own docs pass later.
  - Route: P7-01, P7-03 and P7-04 DONE with their result blocks; P7-02 stays
    PENDING with the note about the CI platform; P7-05..P7-11 untouched.
  - Evidence: the documents carry only observed commands and numbers.

**Toolchain ready (2026-10-23):** Wails CLI v2.17.0 installed via
`go install github.com/wailsapp/wails/v2/cmd/wails@latest`; module cache
populated; coherent pair rule of ADR 0011 satisfied.

**Known limitation (2026-10-23):** the `wails build` command line tool
compiles the main package at the project root and Codixai's entry point is
`cmd/codixai`, so the tool stops with `no Go files in /home/barcelo/Codixai`.
The spike builds with `go build` and the tags the tool would add
(`ui production`, plus `webkit2_41` where WebKitGTK 4.1 is the installed
library). `wails.json` remains as the description of the packaging and holds
no npm step.

**Known limitation (2026-10-24):** on this machine
`go build -tags "ui production"` (without `webkit2_41`) fails at the link
step: `libwebkit2gtk-4.0.so` needs `libjxl.so.0.11`, which is not installed,
and the `JxlDecoder*` symbols stay undefined. WebKitGTK 4.1 is installed and
links, so the window is built with `webkit2_41`.

Out of scope: P7-05 (local server mode ADR), P7-06..P7-11, the Svelte stack,
CI for Windows/macOS (no CI platform configured).

Verification per task: gofmt/build/vet/test plus the phase checks above.
