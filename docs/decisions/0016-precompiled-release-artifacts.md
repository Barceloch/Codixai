# 0016 — The layout of the precompiled release artifacts

- Status: **Accepted**
- Date: 2026-10-30

## Context

The specification asks, in section 8.2, that the variant with the interface
holds the whole CLI and that `go install` always produces the CLI only
variant. ADR 0010 decided that the published artifact is a single binary per
platform built with the `ui` tag, and that the CLI only build is not published
as a separate release artifact.

This release needs precompiled binaries for developers who do not have Go,
and it needs to be honest about what was executed. The user asked for four
separate packages (`_cli` and `_ui` per platform), which is not the same as one
binary per platform.

## Decision

**The release publishes four packages per version, not one binary per platform.**

| Package | Contents | Runtime requirements |
| ------- | -------- | -------------------- |
| `codixai_<version>_linux_amd64_cli.tar.gz` | Static CLI, `CGO_ENABLED=0` | None |
| `codixai_<version>_linux_amd64_ui.tar.gz` | CLI + window, `-tags "ui production webkit2_41"` | GTK 3, WebKitGTK 4.1, glibc 2.39+ |
| `codixai_<version>_windows_amd64_cli.zip` | Static CLI, `CGO_ENABLED=0` | None |
| `codixai_<version>_windows_amd64_ui.zip` | CLI + window, `-tags "ui production"` | WebView2 runtime |

Consequences:

- The CLI package exists because a static binary has no runtime requirements
  and serves the servers, the containers and the pipelines. It is not the
  default of `go build`: the default of `go build` and of `go install` is still
  the CLI only build, and the release workflow builds the CLI package with
  `CGO_ENABLED=0` from the same source.
- The UI package holds the window and the CLI. It is not a second binary: it
  is the same `./cmd/codixai` source built with the `ui` tag, so the audit,
  verify, report, baseline, init, index and impact commands are byte for byte
  the same program. The `ui` command exists in the UI package and answers with
  a usage error in the CLI package.
- Every package holds the binary, the licence, a `VERSION` file, a `README.md`
  and a `README.es.md`. No source, no VCS data, no local file, no `.env` and
  no log travels in a package.
- The archives are deterministic: the timestamps are frozen to 1980-01-01 and
  the owner is 0, so packaging the same bytes twice produces the same archive
  and the same SHA-256.
- `SHA256SUMS` covers all four packages and is published with them.
- The release notes state, per package, whether it was compiled, executed or
  not executed by the workflow. A binary that was compiled but not executed is
  never presented as working.

## Why not one binary per platform

ADR 0010 assumed the UI binary would work everywhere the CLI works. The
reality is different: the Linux UI links against the system libraries, so a
binary built on a rolling release does not start on Debian or Ubuntu. A single
binary would either be unusable on Linux or would silently ship a broken
package. Splitting the packages keeps the CLI usable everywhere and makes the
runtime requirements explicit per package.

## Consequences

Positive:

- A developer without Go can install the CLI on any Linux or Windows machine.
- A developer with a graphical session can install the UI on the supported
  distributions.
- The requirements of the UI are documented per package and per platform, so
  nobody installs a binary that does not start.
- The packages are reproducible from the workflow, not from this machine.

Negative:

- Two packages per platform instead of one.
- The CLI package duplicates the build of the UI package.

## Verification status

The release notes and the Spanish guide state, per package, whether it was
compiled, executed or not executed by the workflow. The Linux CLI and Linux UI
are executed by the workflow on the runner; the Windows CLI is executed; the
Windows UI is compiled but not executed because the runner has no interactive
desktop.