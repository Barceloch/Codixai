# Phase 7 — The local graphical interface (Wails, Svelte and daisyUI)

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 7", and by the section 8. This file turns
them into atomic tasks with executable acceptance checks.

**The audit screen exists; the other screens do not.** The `ui` build
variant, the `codixai ui` command and the Wails v2 prototype are implemented
behind the `ui` build tag (the spike of P7-01 to P7-04). The audit screen of
P7-06 and the export of P7-07 are implemented with the Svelte 5, Vite,
Tailwind and daisyUI stack. The verification screen (P7-08), the structural
visualization (P7-09), the local API (P7-10) and the local server mode
(P7-05) are not implemented. The architecture that holds them exists: the
application layer is independent of the presentation, the results are
versioned documents, and `internal/application/architecture_test.go`
protects those two facts.

**Phase exit criterion (specification):** the screens call only the use
cases of `application`, with no analysis logic in the frontend; the output
shown coincides with the output of `--format json` for the same change,
checked automatically; both variants compile and are tested in CI; and the
verified platforms and the system requirements are documented.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P7-01 | The ADR of the version of Wails and a minimal prototype | DONE |
| P7-02 | The build with the `ui` tag in the three platforms of the CI | PENDING |
| P7-03 | The variant without the tag | DONE |
| P7-04 | The size and the startup time of both variants | DONE |
| P7-05 | The ADR of the local server mode | PENDING |
| P7-06 | The audit interface | DONE |
| P7-07 | The export of the reports from the interface | DONE |
| P7-08 | The verification interface | PENDING |
| P7-09 | The structural visualization | PENDING |
| P7-10 | The local API for external clients | PENDING |
| P7-11 | The equivalence between the interface and `--format json` | PARTIAL |

The spike 7.0 (P7-01 to P7-05) may be started after Phase 3: the risk of
this phase is in the packaging and in the distribution, not in the screens.
The other tasks wait for the use cases they expose.

---

## The rules of this phase

These rules are the ones of the section 17.9 of the specification. They
apply to every task of this file.

1. **Detect the version of Wails before writing any code.** The version is
   read from `go.mod` (`github.com/wailsapp/wails/v2` or
   `github.com/wailsapp/wails/v3`) and the matching command line tool
   follows it (`wails` for v2, `wails3` for v3). **The APIs of v2 and v3
   are incompatible and must never be mixed.** When in doubt, ask.
2. **`domain` and `application` never import Wails and never import
   anything of `ui/`.** The adapter only translates.
3. **The frontend uses the types generated from Go.** The TypeScript types
   are generated, never written by hand.
4. **`{@html}` is forbidden**, and so is any insertion of HTML built from
   the content of the repository. The lint rule of the frontend has to be
   enabled (for example `svelte/no-at-html-tags` of
   `eslint-plugin-svelte`).
5. **The frontend dependencies are installed with the frozen lockfile**
   (`pnpm install --frozen-lockfile`, the package manager of the decision
   [`0011`](../decisions/0011-wails-version-and-spike-frontend.md)).
   `pnpm install` is not run when it modifies the lockfile without a
   confirmation.
6. **Nobody claims that a screen "looks right" without evidence.** It is
   verified with automated tests and, when a visual judgement is needed, a
   person reviews it.

## The reference to load before writing the interface

The skill **`wails`** is installed in this repository and recorded in
`skills-lock.json` (`.agents/skills/wails/SKILL.md`). It is the reference of
this phase and it has to be loaded before any code of the interface is
written: it detects the version, it separates the APIs of v2 and v3, and it
carries the build commands of each version.

The documentation of Svelte, Tailwind and daisyUI also has to be consulted
in its current version: their configuration changes between the major
versions and it must not be assumed from memory.

## The stop conditions of this phase

The pending decisions 11 to 14 of the specification block this phase, and
the section 17.4.8 makes the agent stop and ask:

- **Decision 11 — the version of Wails** (v2 stable, or v3 alpha with the
  typed services and the server mode) and its update policy. The decision
  is taken in P7-01 and recorded in an ADR.
- **Decision 12 — the local server mode** (a browser on the loopback in
  addition to the native window) and its security controls. The decision is
  taken in P7-05.
- **Decision 13 — the distribution policy** of the variant with the
  interface. **Resolved** by the decision
  [`0010`](../decisions/0010-distribution-policy-of-the-variants.md): the
  published artifact is a single binary that holds the CLI and the graphical
  interface, built with the `ui` tag; the CLI only build stays the default of
  `go build` and of `go install` and is not published as a separate artifact.
  **The platforms that the variant with the interface supports are still
  pending**: the decision 0010 fixes the policy and not the matrix of the
  supported platforms, which still has to be confirmed before the release.
- **Decision 14 — the languages of the interface** (Spanish, English) and
  the translation policy.

**Stop and ask** when a task of this phase requires choosing the version of
Wails, adding an npm dependency or changing the distribution policy. The
distribution policy itself is no longer an open choice: the decision
[`0010`](../decisions/0010-distribution-policy-of-the-variants.md) fixes it,
and the tasks of this file follow it. The platforms of the variant with the
interface, the requirements of the system per platform and the size and the
startup time of the combined binary are still pending, and they are what the
spike 7.0 has to measure.

---

### P7-01 — The ADR of the version of Wails and a minimal prototype

- Status: **DONE** (2026-10-23)
- Depends on: P2-05 (the use cases the interface calls)

**Objective**
Decide the version of Wails, with the evidence of a minimal prototype bound
to a real read-only use case, and record the decision in an ADR.

**Sections to read**
6.1, 8.1, 8.2, 8.3, 14 (Phase 7, spike 7.0), 17.9.

**Deliverables**
- An ADR in `docs/decisions/`, named after the decision of the version of
  Wails: the decision, the reasons, the consequences, the update policy and
  the requirements of the system per platform.
- A prototype: a native window that calls one read-only use case of
  `application` and renders its versioned document.
- The `ui/` package, behind the `ui` build tag.

**Acceptance checks**
```bash
go build -tags ui ./cmd/codixai
go build ./cmd/codixai            # the default variant, without the tag
go test ./internal/application/ -run TestApplicationDoesNotDependOnThePresentation
```
The prototype itself needs a graphical environment. The machine of the
spike has one (a Wayland session), so the window was opened with the
measurement mode; the automated verification of a screen is still manual
and it is declared as such.

**Out of scope**
The screens of 7.1, the server mode of P7-05 and the platforms of the
variant with the interface. The distribution policy itself is fixed by the
decision
[`0010`](../decisions/0010-distribution-policy-of-the-variants.md) and is
not a choice of this task.

**Result (2026-10-23, re-verified 2026-10-24)**

- The decision is recorded in
  [`0011`](../decisions/0011-wails-version-and-spike-frontend.md): Wails v2,
  the module pinned to `v2.17.0`, the same version as the installed command
  line tool. The spike frontend is static vanilla HTML and JavaScript
  embedded with `go:embed` from `ui/assets`; there is no Node.js in the
  build.
- The prototype lives in `ui/` behind the `ui` build tag. `ui.App.Audit`
  calls `application.RunAudit` with the origin that the window selects
  (worktree or base revision) and returns the versioned document, which the
  page renders with `textContent`. The binding surface is
  `window.go.ui.App`.
- `go build -tags ui ./...` and `go build -tags ui ./cmd/codixai` exit `0`.
  The window is opened by a binary that also carries the Wails build tags:
  `go build -tags "ui production webkit2_41" ./cmd/codixai` exits `0` (see
  P7-04).
- The adapter has its own tests behind the same tag: `go test -tags ui
  ./ui/` runs five cases that check the validation of the request (empty
  root, two origins, no origin) and that the view carries the document and
  the origin of the use case.
- The assets are embedded in the `ui` binary: the unique strings of
  `ui/assets/index.html` and `ui/assets/app.js` are found in the binary
  built with the tag and not in the default binary.
- The window was opened in the development machine and closed by the
  measurement mode: `CODIXAI_UI_AUTOQUIT=1s ./codixai ui --root .` exited
  with the code `0` in 1.185 s to 1.269 s (three runs of the clean build).
  The automated verification of a screen is still manual and it is declared
  as such.

---

### P7-02 — The build with the `ui` tag in the three platforms of the CI

- Status: **PENDING**
- Depends on: P7-01

**Note (2026-10-23)**
The CI job for the `ui` variant is not added: the repository has no CI
platform configured for the builds of Windows and macOS, which is a pending
decision of the maintainer. The local build of the variant on Linux is
verified in P7-01 and P7-04; a declared job without observed results is not
evidence, so this task stays pending until the platform is decided.

**Objective**
Compile the variant with the interface in Linux, macOS and Windows from the
CI, and document the requirements of the system of each platform.

**Sections to read**
8.2, 8.5, 14 (Phase 7, spike 7.0), 16 (publicación).

**Deliverables**
- A job of `.github/workflows/ci.yml` that builds the `ui` variant in the
  three platforms.
- The documented requirements: the WebView2 runtime in Windows and the
  WebKitGTK libraries in Linux, confirmed in the ADR of P7-01.
- The frontend build with `npm ci`, only in the job that needs it.

**Acceptance checks**
```bash
go build -tags ui ./cmd/codixai
```
The builds of macOS and Windows run in the CI. **A declared job is not
evidence**: the results have to be observed before the phase can be closed.

**Out of scope**
The publishing of the releases, which depends on the variant with the
interface that this phase produces: the decision
[`0010`](../decisions/0010-distribution-policy-of-the-variants.md) publishes
one binary per platform, built with the `ui` tag, and it cannot be published
before that binary exists.

---

### P7-03 — The variant without the tag

- Status: **DONE** (2026-10-23)
- Depends on: P7-01

**Objective**
Prove that the default variant compiles and runs without Node.js, without
Wails and without the dependencies of a webview.

**Sections to read**
8.2, 14 (Phase 7, spike 7.0).

**Deliverables**
- The UI code, the embedded assets and the Wails dependency, all behind the
  `ui` build tag.
- A CI job that builds the default variant in an environment without
  Node.js.

**Acceptance checks**
```bash
go build ./cmd/codixai
go run ./cmd/codixai --help
go list -deps ./cmd/codixai | grep -i wails   # no result
```

**Out of scope**
The `codixai ui` command of the variant with the interface.

**Result (2026-10-23)**

The isolation is enforced by the compiler: every file of `ui/` carries
`//go:build ui`, so a file of the default build that imported the package
would not compile. The module graph was also inspected, with the real
output:

```console
$ go list -deps ./cmd/codixai | wc -l
138
$ go list -deps ./cmd/codixai | grep -i wails
$ go list -deps ./cmd/codixai | grep -Ei 'webview|webkit|gtk|go-ole|dbus'
$ CGO_ENABLED=0 go build -o /tmp/codixai-nocgo ./cmd/codixai
$ go build ./cmd/codixai
$ go list -deps -tags "ui production webkit2_41" ./cmd/codixai | wc -l
302
$ go list -deps -tags "ui production webkit2_41" ./cmd/codixai | grep -c wailsapp/wails
28
```

The two greps printed nothing: no Wails package and no webview, WebKitGTK,
GTK or D-Bus package is reachable from the default binary. The build with
`CGO_ENABLED=0` succeeds, so the default variant does not need a C
toolchain. The same scan over the `ui` variant reaches 302 packages and
includes `github.com/wailsapp/wails/v2/...`, which is the contrast that
proves the tag is doing the work. The default binary answers `codixai ui`
with the usage error of the CLI only variant, covered by
`tests/integration/ui_command_test.go`. The `Build` job of the CI
(`.github/workflows/ci.yml`) already builds this variant in Linux, macOS
and Windows without installing Node.js.

The proof is a manual command and not a unit test on purpose: a test that
shelled out to `go list` would depend on the toolchain of the machine,
while the fact that matters — the default build never compiles `ui/` — is
already guaranteed by the build tags.

---

### P7-04 — The size and the startup time of both variants

- Status: **DONE** (2026-10-23)
- Depends on: P7-02, P7-03

**Objective**
Measure the size of the binary and the time of the startup of both
variants, and publish the numbers with the command that reproduces them.

The measurement is what keeps the decision
[`0010`](../decisions/0010-distribution-policy-of-the-variants.md)
provisional: the published artifact is a single binary with the CLI and the
interface, and a measurement that shows it is not viable revises the decision
with another ADR.

**Sections to read**
14 (Phase 7, spike 7.0), 15.

**Deliverables**
A report in `docs/` with the measured numbers, the platform, the method and
the command that reproduces the measurement. The report is the result block
of this task.

**Acceptance checks**
```bash
go build -o /tmp/codixai-default ./cmd/codixai
go build -tags "ui production webkit2_41" -o /tmp/codixai-ui ./cmd/codixai
stat -c%s /tmp/codixai-default /tmp/codixai-ui
```

**Out of scope**
Any number that was not measured. A declared figure is not a measurement.

**Result (2026-10-23, re-measured 2026-10-24)**

Platform of the measurement: Linux amd64, Go `1.27.0`, Wails `v2.17.0`,
WebKitGTK 4.1 (`webkit2_41`), Wayland session. The binaries are built from
the same source with the commands of the table, with a clean `ui/assets`
(see the fourth limitation below). Only the Linux build is measured: the
dependency on P7-02 is satisfied locally and the CI matrix of that task is
still pending.

| Variant | Build command | Size | Startup |
| ------- | ------------- | ---- | ------- |
| CLI only | `go build -o codixai ./cmd/codixai` | 6,637,046 B (6.33 MiB) | 0.003 s (`codixai version`, 5 runs) |
| `ui` tag only | `go build -tags ui -o codixai ./cmd/codixai` | 8,923,864 B (8.51 MiB) | not applicable: it compiles, but the window reports the missing Wails tags |
| `ui` production | `go build -tags "ui production webkit2_41" -o codixai ./cmd/codixai` | 14,790,328 B (14.10 MiB) | 0.303 s – 0.404 s with a 0.1 s auto-quit; 1.185 s – 1.269 s with 1 s; 2.265 s – 2.288 s with 2 s (≈0.23 s – 0.29 s of startup) |

The startup of the window is measured with the auto-quit of the prototype:
`OnStartup` schedules `runtime.Quit` after the duration of
`CODIXAI_UI_AUTOQUIT`, so the wall time of the process is the startup plus
the duration. The medians of the 1 s and the 2 s runs are 1.253 s and
2.278 s, which give a slope of 1.03 and an intercept of about 0.23 s; the
0.1 s runs (median 0.383 s) put the startup near 0.29 s. The window opened
in all nine runs and the process exited with the code `0`.

Reproduction:

```bash
go build -o /tmp/codixai-default ./cmd/codixai
go build -tags "ui production webkit2_41" -o /tmp/codixai-ui ./cmd/codixai
stat -c%s /tmp/codixai-default /tmp/codixai-ui
TIMEFORMAT=%R; time /tmp/codixai-default version
TIMEFORMAT=%R; time CODIXAI_UI_AUTOQUIT=1s /tmp/codixai-ui ui --root .
```

Both measured binaries are plain `go build` output, and two consecutive
builds of each variant are byte for byte identical. With the
`-ldflags "-w -s"` that `wails build` passes in production, the same sources
give 4,403,360 B for the CLI only variant and 10,212,224 B for the
`ui production` variant (9.74 MiB, 2.32 times the CLI only binary).

Four limitations of the packaging, observed and not hidden:

- **Wails v2 needs its own build tags.** The variant does not run with the
  `ui` tag alone: Wails gates its application layer behind `production` (or
  `dev`). `ui/window_untagged.go` replaces the cryptic message of the
  toolkit with the command that adds the tags.
- **The `wails build` command line tool cannot build this layout.** Wails
  v2 compiles the main package at the root of the project, and the entry
  point of Codixai is `cmd/codixai`. The observed error is
  `no Go files in /home/barcelo/Codixai`. The spike therefore builds with
  `go build` and the same tags that the tool would pass
  (`desktop,production,ui`). `wails.json` stays as the description of the
  packaging, with `frontend:install` and `frontend:build` empty: Wails v2
  treats an empty command as "skip", and there is no npm step.
- **WebKitGTK 4.0 does not link on this machine.** `go build -tags "ui
  production"` (without `webkit2_41`) fails at the link step with
  `libjxl.so.0.11, necesario para libwebkit2gtk-4.0.so, no se ha encontrado`
  and the `JxlDecoder*` symbols of that library undefined. The `webkit2_41`
  tag links WebKitGTK 4.1, which is installed and links, so the table uses
  it. The release notes of the variant have to name the WebKitGTK library
  the binary was built against.
- **The generated Wails wrappers must stay out of the embedded assets.**
  The first `wails build` attempt wrote `wailsjs/runtime/*` to the
  `wailsjsdir` of `wails.json`, which pointed at `ui/assets/wailsjs`; the
  `go:embed all:assets` directive embedded those files and the measured
  binary grew by 24,576 B (14,814,904 B instead of 14,790,328 B). The
  directory was removed and `wailsjsdir` now points at `./ui/wailsjs`,
  outside the embed, with a `ui/.gitignore` entry. The static frontend never
  imports the generated wrappers: it loads `/wails/ipc.js` and
  `/wails/runtime.js` from the asset server.

---

### P7-05 — The ADR of the local server mode

- Status: **PENDING**
- Depends on: P7-01

**Objective**
Decide whether a local server mode (a browser on the loopback) is added to
the native window, and define its security controls.

**Sections to read**
8.1, 8.5, 14 (Phase 7, spike 7.0).

**Deliverables**
An ADR in `docs/decisions/`, named after the local server mode: the
decision, the loopback binding, the random port, the per session token, the
validation of `Host` and `Origin`, the absence of an open CORS and the
read-only nature of the exposure.

**Acceptance checks**
Not executable today: the mode does not exist. The acceptance check of this
task is the ADR and the confirmation of the decision 12.

**Out of scope**
The implementation of the server mode, which belongs to the task that
follows the decision.

---

### P7-06 — The audit interface

- Status: **DONE** (2026-10-24)
- Depends on: P7-01, P1-01

**Objective**
The first visual milestone: select a repository, run an audit as a
cancellable job with real progress, and read the findings with their
evidence.

**Sections to read**
8.3, 8.4, 8.5, 9, 14 (Phase 7, 7.1).

**Deliverables**
- The selection of the repository, with a native dialog or `--root`.
- The three origins of the audit (`--base`, `--staged`, `--worktree`) as
  cancellable jobs, with a progress that represents real phases.
- The list of the findings, with a filter and an order by rule, severity,
  epistemic status, language and path.
- The detail of a finding: the evidence, the hunk of the diff, the support
  level of the rule, the limitations and the recommendation.
- The light and the dark theme, the accessibility and the texts prepared for
  a translation.

**Acceptance checks**
```bash
pnpm --dir frontend check
pnpm --dir frontend lint
pnpm --dir frontend build
go build -tags ui ./cmd/codixai
go test -tags ui ./ui/
```

**Out of scope**
The export of the reports (P7-07), the verification (P7-08) and the
visualization (P7-09).

**Result (2026-10-24)**

- The frontend lives in `frontend/`: Svelte 5.57.0, Vite 8.3.0, TypeScript
  5.9.3, Tailwind CSS 4.3.3 and daisyUI 5.7.42, with exact versions pinned
  in `package.json` and `pnpm-lock.yaml` committed. The package manager is
  pnpm, as the decision
  [`0011`](../decisions/0011-wails-version-and-spike-frontend.md) fixes.
  `wails.json` builds it with `pnpm install --frozen-lockfile` and
  `pnpm run build`.
- The compiled frontend is embedded by the `frontend` package with
  `//go:embed all:dist`. The directory keeps a `dist/.gitkeep` so that the
  Go build works before the first `pnpm run build`; the Vite configuration
  restores the placeholder after every build.
- The TypeScript types are generated from the Go adapter, never written by
  hand: `go run -tags "ui bindings" ./cmd/codixai ui --root .` writes
  `frontend/src/lib/wailsjs`. The bridge (`frontend/src/lib/bridge.ts`) only
  wraps the generated functions.
- The screen selects the repository (native dialog with
  `ChooseRepository`, or the `--root` of the invocation), the three origins
  (`worktree`, `staged`, `base`), and runs the audit with `AuditAsync`. The
  phases are the real ones of the use case (`validating`, `collecting`,
  `collected`, `analyzing`, `analyzed`, `done`), reported by wrappers in
  `internal/cli/ui_command_ui.go`, and `CancelAudit` stops the run through
  the context.
- The list filters and sorts by rule, severity, epistemic status, language
  and path (`frontend/src/lib/findings.ts`). The detail shows the evidence,
  the hunk of the change set, the support level, the limitations and the
  recommendation.
- The texts are English and Spanish, persisted in `localStorage`
  (`i18n.svelte.ts`); the theme is light, dark or the system preference,
  persisted and applied before the first paint (`theme.svelte.ts` and the
  inline script of `index.html`).
- `svelte/no-at-html-tags` is an error in `frontend/eslint.config.js`, and
  `{@html}` is not used.
- Real results: `pnpm --dir frontend check` reported 0 errors and 0
  warnings; `pnpm --dir frontend lint` exited 0; `pnpm --dir frontend build`
  wrote `dist/index.html` and two assets; `go build -tags ui ./cmd/codixai`
  and `go test -tags ui ./ui/` exited 0. The `ui production` binary built
  with `webkit2_41` opened the window with
  `CODIXAI_UI_AUTOQUIT=3s ./codixai ui --root .` and exited with the code
  `0`; the standard error only carried GTK and WebKit warnings of the
  environment.
- **Not verified:** the visual review of the window. The layout, the
  contrast of both themes and the keyboard navigation were not reviewed by
  a person. The default WebKitGTK 4.0 link also fails in this environment
  because the system library is missing `libjxl.so.0.11`; the `webkit2_41`
  link works.

---

### P7-07 — The export of the reports from the interface

- Status: **DONE** (2026-10-24)
- Depends on: P7-06, P1-08

**Objective**
Export the findings as JSON and as SARIF through a native save dialog.

**Sections to read**
8.3, 8.5, 14 (Phase 7, 7.1).

**Deliverables**
The export in the interface, writing only to the path the person chooses,
never to the analyzed repository.

**Acceptance checks**
```bash
go test -tags ui ./ui/ ./tests/integration/
go build -tags ui ./cmd/codixai
```

**Out of scope**
The SARIF export of the CLI, which belongs to Phase 3 (P1-08).

**Result (2026-10-24)**

- `ui.App.Export` writes the report of a finished run to the path that the
  person chooses in the native save dialog. It reuses the result of the run,
  so it exports exactly the document that the screen shows.
- `ui.EncodeExport` produces the same bytes as the CLI: the JSON document
  with the encoder and the indentation of `internal/cli`, and the SARIF
  document with `report.SARIF`. The function is exported so the parity test
  exercises the exact encoder of the export.
- A path inside the audited repository is rejected before writing: the
  tool never writes into the repository that it reads.
- The screen offers the two formats once a run finishes, and reports the
  written path or the cancellation of the dialog.
- Real results: `go test -tags ui -count=1 ./ui/ ./tests/integration/`
  exited 0; the parity test compares the normalized JSON document and the
  SARIF bytes byte for byte. `go build -tags "ui production webkit2_41"
  ./cmd/codixai` exited 0 and produced the binary.
- **Not verified:** the native save dialog itself. The adapter is tested
  through the encoder and the guard, but the dialog needs a graphical
  session and was not exercised.

---

### P7-08 — The verification interface

- Status: **PENDING**
- Depends on: P7-01, P2-05

**Objective**
Load a claim file and the contributed reports, and present the results
`verified`, `contradicted` and `unresolved` with their evidence.

**Sections to read**
8.3, 8.4, 14 (Phase 7, 7.2).

**Deliverables**
- The load of a claim file and of the reports, through a native dialog.
- The presentation of the three states, with the evidence and the
  limitations of every result.

**Acceptance checks**
```bash
go test ./ui/frontend/
```

**Out of scope**
Any state that the CLI does not produce. The interface does not decide a
claim: it shows what the use case decided.

---

### P7-09 — The structural visualization

- Status: **PENDING**
- Depends on: P7-06, P1-04 (Phase 5), Phase 6

**Objective**
Bounded graphs of the dependencies, the impact of a change and the maps of
the modules, with the unresolved relations visually distinguishable.

**Sections to read**
8.3, 8.4, 14 (Phase 7, 7.3).

**Deliverables**
The graphs, with limits, filters and pagination. An unbounded graph is never
returned by default.

**Acceptance checks**
```bash
go test ./ui/frontend/
```

**Out of scope**
The graph itself: the use cases of the structural analysis belong to Phase 5
and to Phase 6 and do not exist yet.

---

### P7-10 — The local API for external clients

- Status: **PENDING**
- Depends on: P7-05

**Objective**
A local API for external clients, **only if a use case justifies it**, with
the same security rules as the section 8.5.

**Sections to read**
8.5, 14 (Phase 7, 7.4).

**Deliverables**
The API and its documentation, with the controls of the decision 12.

**Acceptance checks**
Not executable today: the task is conditional on a validated use case, and
the use case has to be written down first.

**Out of scope**
Any remote exposure. The API is local and read-only.

---

### P7-11 — The equivalence between the interface and `--format json`

- Status: **PARTIAL** (2026-10-24)
- Depends on: P7-06, P7-08

**Objective**
Prove automatically that the interface shows the same results as
`--format json` for the same change.

**Sections to read**
8.3, 14 (Phase 7, criterio de salida), 18.

**Deliverables**
A test that runs the same use case through the adapter of the interface and
through the JSON encoder of the CLI, and compares the two documents.

**Acceptance checks**
```bash
go test -tags ui ./ui/ ./tests/integration/
```

**Out of scope**
A comparison of the screenshots: the contract is the document, not the
rendering.

**Result (2026-10-24)**

- `tests/integration/ui_parity_test.go` runs the same change through
  `cli.UIAuditFunc()` (the composition of the window) and through the JSON
  encoder of the CLI, and compares the two documents after normalizing
  `generated_at`. The SARIF export is compared byte for byte.
- The audit half of the criterion is covered. The task is **PARTIAL**
  because it depends on P7-08, and the verification screen does not exist:
  there is no verification document to compare yet.
- Real result: `go test -tags ui -count=1 ./ui/ ./tests/integration/`
  exited 0.
