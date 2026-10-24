# 0011 — The version of Wails and the frontend of the spike

- Status: **Accepted**
- Date: 2026-10-22

## Context

The specification defers to this phase the version of Wails (pending decision
11) and orders a packaging spike (7.0) whose deliverable is that decision,
a minimal prototype bound to a real read-only use case, the measurement of
both build variants, and the check that the default build stays free of
Node.js, Wails and the webview.

The evidence available at the moment of deciding:

- the Wails command line tool **v2.17.0** is installed and `wails doctor`
  passes on the development machine (Manjaro Linux, webview deps present);
- the module and the CLI are now a coherent pair at v2.17.0;
- Wails **v3 not stable** (beta per the consulted docs of the specification), its command line tool is not installed, and its APIs
  are incompatible with v2 (the mixing of both is a known LLM failure mode,
  which is why the phase rules forbid it);
- the module cache holds the Wails module at v2.17.0;
  fetched at spike time;
- the local proxy was observed stall-cutting `proxy.golang.org` and the npm
  registry at spike start, while the Go toolchain later resolved both;
- for the frontend stack the maintainer chose **pnpm** as the package
  manager (2026-10-22).

## Decision

### The version of Wails (pending decision 11)

**Wails v2.** The `ui` build pins the Go module to
`github.com/wailsapp/wails/v2 v2.17.0`, the exact version of the installed
command line tool, so the ADR and the toolchain stay a coherent pair. The
v2 CLI remains the one used to build and to doctor the system.

The upgrade of the pair (module and CLI together) is its own maintenance
decision and is out of the scope of this spike.

Wails v3 is rejected: not stable, not installed, incompatible with v2, and it
would couple the release to an API that may change under it.

### The frontend of the spike

The minimal prototype embeds a **static frontend, plain HTML and JavaScript,
without Node.js**: the assets are embedded with `go:embed` in the `ui`
package. This is deliberate: the risk of the spike is packaging and
distribution, so the prototype must weigh and start the real binary without
adding a frontend toolchain to the measurement.

The production stack of the interface (Svelte, Vite, Tailwind CSS and
daisyUI, with **pnpm** and `pnpm install --frozen-lockfile`) is NOT part of
the spike. Its exact versions are pinned when the audit interface of P7-06
starts, per section 8, and only once the npm registry is reachable.
`{@html}` stays forbidden from the first line of the interface.

## Consequences

Positive:

- The decision rests on the toolchain that actually exists and is verified
  by `wails doctor`.
- The spike measures the real cost of the binary without the noise of a
  frontend build.
- The chosen stack for the later interface is declared now: pnpm, exact
  versions at P7-06.

Negative:

- The prototype's frontend is throwaway; the audit interface of P7-06 does
  not reuse its assets.
- The entry point of the repository (`cmd/codixai`) is not compatible with
  the default `wails build` command, so the UI variant must be built with
  `go build` and explicit tags; `wails.json` is present only to disable the
  default frontend steps.
