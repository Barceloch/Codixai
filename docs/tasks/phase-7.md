# Phase 7 — The local graphical interface (Wails, Svelte and daisyUI)

The scope and the exit criteria of this phase are fixed by the
specification, section 14, "Fase 7", and by the section 8. This file turns
them into atomic tasks with executable acceptance checks.

**Nothing of this phase is implemented.** The interface belongs to Phase 7,
the `codixai ui` command does not exist, and the `ui` build variant does not
exist. What exists today is the architecture that will hold it: the
application layer is independent of the presentation, the results are
versioned documents, and `internal/application/architecture_test.go` protects
those two facts.

**Phase exit criterion (specification):** the screens call only the use
cases of `application`, with no analysis logic in the frontend; the output
shown coincides with the output of `--format json` for the same change,
checked automatically; both variants compile and are tested in CI; and the
verified platforms and the system requirements are documented.

## Summary

| ID | Task | Status |
| -- | ---- | ------ |
| P7-01 | The ADR of the version of Wails and a minimal prototype | PENDING |
| P7-02 | The build with the `ui` tag in the three platforms of the CI | PENDING |
| P7-03 | The variant without the tag | PENDING |
| P7-04 | The size and the startup time of both variants | PENDING |
| P7-05 | The ADR of the local server mode | PENDING |
| P7-06 | The audit interface | PENDING |
| P7-07 | The export of the reports from the interface | PENDING |
| P7-08 | The verification interface | PENDING |
| P7-09 | The structural visualization | PENDING |
| P7-10 | The local API for external clients | PENDING |
| P7-11 | The equivalence between the interface and `--format json` | PENDING |

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
5. **The frontend dependencies are installed with `npm ci`**, ideally with
   `--ignore-scripts`. `npm install` is not run when it modifies the
   lockfile without a confirmation.
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
  interface (a single binary with the CLI and the interface, against the
  CLI only as the default) and the platforms it supports.
- **Decision 14 — the languages of the interface** (Spanish, English) and
  the translation policy.

**Stop and ask** when a task of this phase requires choosing the version of
Wails, adding an npm dependency or changing the distribution policy.

---

### P7-01 — The ADR of the version of Wails and a minimal prototype

- Status: **PENDING**
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
- The `internal/ui/` package, behind the `ui` build tag.

**Acceptance checks**
```bash
go build -tags ui ./cmd/codixai
go build ./cmd/codixai            # the default variant, without the tag
go test ./internal/application/ -run TestApplicationDoesNotDependOnThePresentation
```
The prototype itself needs a graphical environment, which the development
environment does not have: the verification of the window is a manual one
and it is declared as such.

**Out of scope**
The screens of 7.1, the server mode of P7-05 and the distribution policy of
the decision 13.

---

### P7-02 — The build with the `ui` tag in the three platforms of the CI

- Status: **PENDING**
- Depends on: P7-01

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
The publishing of the releases, which belongs to Phase 3.

---

### P7-03 — The variant without the tag

- Status: **PENDING**
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

---

### P7-04 — The size and the startup time of both variants

- Status: **PENDING**
- Depends on: P7-02, P7-03

**Objective**
Measure the size of the binary and the time of the startup of both
variants, and publish the numbers with the command that reproduces them.

**Sections to read**
14 (Phase 7, spike 7.0), 15.

**Deliverables**
A report in `docs/` with the measured numbers, the platform, the method and
the command that reproduces the measurement.

**Acceptance checks**
Not executable today: the variant with the interface does not exist. The
acceptance check of this task is the report with its reproduction command.

**Out of scope**
Any number that was not measured. A declared figure is not a measurement.

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

- Status: **PENDING**
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
go test ./ui/frontend/            # the component and the flow tests
go build -tags ui ./cmd/codixai
```

**Out of scope**
The export of the reports (P7-07), the verification (P7-08) and the
visualization (P7-09).

---

### P7-07 — The export of the reports from the interface

- Status: **PENDING**
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
go test ./ui/frontend/
```

**Out of scope**
The SARIF export of the CLI, which belongs to Phase 3 (P1-08) and does not
exist yet.

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

- Status: **PENDING**
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
go test ./internal/ui/ ./tests/integration/
```

**Out of scope**
A comparison of the screenshots: the contract is the document, not the
rendering.
