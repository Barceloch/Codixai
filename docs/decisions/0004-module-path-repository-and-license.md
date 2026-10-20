# 0004 — Module path, repository URL and license

- Status: **Accepted for the module path and the repository URL / pending maintainer confirmation for the license**
- Date: 2026-10-20

## Context

The specification asks for a definitive repository URL, a owning
organization, a confirmed license and a minimum Go version, and it lists
them as **pending decisions that require confirmation** (the closing list of
the specification, decisions 1, 2, 3 and 4). It also states that the
decisions that block a package with a definitive name or license are the
ones that may block the creation of an executable base, and that the parsing
strategy is the one that blocks the rules of Phase 1.

None of these confirmations has been obtained. Inventing a URL, an
organization or a license would be a fabrication that the repository would
then have to contradict.

The current state:

- `go.mod` declares `module github.com/Barceloch/Codixai` and `go 1.27.0`.
  The module path is the confirmed repository URL, so the program can be
  built, tested and imported by another module.
- The repository has a `LICENSE` file pending and a confirmed remote at
  `https://github.com/Barceloch/Codixai`.
- The documentation refers to the license as pending, with Apache-2.0 as
  the proposal, and never claims that a license is granted.

## Decision

**Use the module path `github.com/Barceloch/Codixai`, keep the license
pending, and record exactly what changes when the maintainer confirms each
decision.**

The license status is stated the same way in every document: the license is
**PENDING maintainer confirmation**, the proposal is Apache-2.0, and the
`LICENSE` file will be added before the first public release. No document
claims that a license is granted.

### The checklist of the confirmation

When the maintainer confirms the decisions, these files change, and only
these:

| Decision | File | Change | State |
| -------- | ---- | ------ | ----- |
| Module path | `go.mod` | `module codixai` becomes the definitive path. | **Applied**: `module github.com/Barceloch/Codixai`. |
| Module path | every `.go` file under `internal/` and `cmd/` | The import paths change from `codixai/internal/...` to `<confirmed-domain>/codixai/internal/...`. There are no import paths outside the module. | **Applied**: `github.com/Barceloch/Codixai/internal/...`. |
| Repository URL | `README.md` | The clone instruction in "Install from source" is completed with the confirmed URL. | **Applied**. |
| Repository URL | `CONTRIBUTING.md` | The clone instruction is completed with the confirmed URL. | **Applied**. |
| Repository URL | `SECURITY.md` | The private advisory URL is completed. | **Applied**. |
| Repository URL | `docs/decisions/0004-module-path-repository-and-license.md` | The status changes from *Provisional* to *Accepted*, and this table is marked as applied. | **Applied**. |
| License | `LICENSE` | The file is added with the confirmed license text, and its copyright holder. |
| License | `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md` | The "pending" statements are replaced by the confirmed license. |
| License | `CODE_OF_CONDUCT.md` | The pending attribution note is resolved together with the license. |
| Security contact | `SECURITY.md` | The private advisory URL and the contact address are completed. |
| Security contact | `README.md` | The security section points at the confirmed channel. |
| Platforms | `README.md`, `SECURITY.md` | The list of the platforms that are actually built and tested is published, once the builds exist. |

Nothing else depends on the URL, the organization or the license: no source
file mentions them, and no test asserts them.

### The confirmation log

| Date | Decision | Answer |
| ---- | -------- | ------ |
| 2026-10-20 | Repository URL and organization | The maintainer confirmed `https://github.com/Barceloch/Codixai`. The module path is `github.com/Barceloch/Codixai`. |
| 2026-10-20 | License | Asked: Apache-2.0 (the proposal), MIT, or keep it pending. Answer: **keep it pending**. No `LICENSE` file is added, and the repository is not distributable until the license is confirmed. |

## Consequences

Positive:

- The repository is buildable and testable today, without a name that was
  invented.
- The module path is the confirmed URL, so the project can be imported by
  another module and installed with `go install`.
- The confirmations become a mechanical, reviewable change instead of a
  redesign.
- The documentation never overstates the state of the project.

Negative:

- The repository cannot be distributed as a package with a definitive license.
- The publication of the first release is blocked on the confirmations.

## Alternatives considered

**Invent a plausible URL and organization, for example `github.com/codixai/codixai`.**
Rejected. The specification forbids inventing a URL, and a fabricated URL in
the documentation is a lie that a reader would act on.

**Use a vanity import path with a `replace` directive, for example
`codixai.dev/codixai`.**
Rejected. It is still an invented domain, and it adds a redirect obligation
to whoever ends up owning it.

**Publish without a license.**
Rejected. A repository without a license grants nothing, and the
specification lists the license as a required public file.

**Choose the license now, as the agent.**
Rejected. The specification lists the license confirmation as a decision
that requires the maintainer, and the agent must stop and ask instead of
choosing.
