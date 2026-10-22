# 0010 — The distribution policy of the variants

- Status: **Accepted**
- Date: 2026-10-22

## Context

The specification of the version 0.6.0 defines two build variants of the
tool:

| Variant | Build | Requirements | Use |
| ------- | ----- | ------------ | --- |
| CLI only (the default) | `go build ./cmd/codixai` | Go toolchain. No Node.js, no Wails, no webview. | Terminals, CI, containers, `go install` |
| With the interface | the frontend is compiled and the binary is built with the `ui` tag | Node.js at build time, the native libraries of the webview at run time | Desktop use |

It also states, in section 8.2, that the variant with the interface holds
the whole CLI, that the code of the interface lives behind build tags so
that the CLI only variant does not import Wails, and that `go install` always
produces the CLI only variant because it cannot compile the frontend. The
pending decision 13 asks for the policy of the distribution of the variant
with the interface.

## Decision

**The published artifact is a single binary that holds the CLI and the
graphical interface.** The CLI only build remains the default of `go build`
and of `go install`, and it remains the artifact of the continuous
integration and of the containers.

Concretely:

- The release publishes one binary per platform, built with the `ui` tag.
  It is the same binary that answers `codixai audit`, `codixai verify` and
  `codixai ui`.
- The CLI only build is not published as a separate release artifact. It
  stays available for `go install`, for the pipelines and for the
  environments without a graphical system.
- The two builds come from the same source, and the continuous integration
  builds and tests both of them, so a variant that does not compile is
  detected before the release.
- The requirements of the system of the variant with the interface are
  documented in the release notes of every version: the runtime of WebView2
  in Windows and the libraries of WebKitGTK in Linux, confirmed in the
  decision of the spike 7.0.
- `codixai ui` in the CLI only build ends with a usage error that explains
  how to obtain the variant with the interface, as the specification
  requires in the section 8.2.

The decision is provisional in one respect: the packaging of the frontend
inside the binary, its size and its startup time are measured in the spike
7.0. If the measurement shows that a single binary is not viable, the
decision is revised with another ADR, and the specification is updated with
it.

## Consequences

Positive:

- A person installs one binary and gets the terminal and the desktop
  interface of the same tool, with the same version and the same rules.
- There is one artifact to download, to verify and to document.
- The CLI only variant keeps the properties that the pipelines need: no
  Node.js, no Wails, no webview.

Negative:

- The published binary depends on the native libraries of the webview, so
  its requirements are wider than the ones of the CLI only build.
- The build with the interface needs Node.js and the packaging of the
  frontend, so the release pipeline is more complex than a plain `go build`.
- A user that only wants the terminal still has to install the libraries of
  the webview, unless it uses `go install`.

## Alternatives considered

**Publish two artifacts, the CLI only one and the one with the interface.**
Rejected for now. It doubles the matrix of the releases and the
documentation, and the measurement of the spike 7.0 may show that the
difference does not justify it. The decision is revisited if the size or
the startup time of the combined binary is a problem.

**Publish only the CLI only binary and leave the interface to the sources.**
Rejected. The interface would have no release, and the decision of the
spike 7.0 would never reach a person.

**Publish the interface as a separate application that talks to the CLI.**
Rejected. It would duplicate the rules and the logic of the domain, and the
specification forbids an engine per interface.
