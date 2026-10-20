# 0001 — Parsing strategy for multiple languages

- Status: **Proposed / blocked**
- Date: 2026-10-20

## Context

The specification places the parsing strategy on the critical path of
Phase 1. Section 6.1 names Tree-sitter as the mechanism for the multi
language parsing, and the deliverables of Phase 0 (section 14) require an
ADR that compares "Tree-sitter via CGO" against the WASM and other
alternatives, backed by **test builds on Linux, macOS and Windows**.

Two constraints of the current environment block that decision:

1. **There is no network access.** The Tree-sitter grammars and their Go
   bindings cannot be fetched, and neither can the C toolchain that a CGO
   build needs in a reproducible way. The project was therefore written
   with the Go standard library only, with no `go.sum` and no third party
   dependency.
2. **Only Linux is available here.** macOS and Windows builds cannot be
   produced or tested in the development environment, so the cross platform
   evidence that the specification requires does not exist.

The decision therefore cannot be closed today. What can be done is to fix
the seam that the decision will fill, and to state plainly what is missing.

## Decision

**Keep one `parser` seam per language, and keep every rule at the
`syntactic` support level until a real parser exists.**

- **Go** uses the standard toolchain: `go/parser` and `go/ast` with
  `ParseComments`, in `internal/languages/go`. The parser reads the source
  text that the analysis already holds. No module is downloaded and no
  external tool is executed.
- **Python, PHP, JavaScript and TypeScript** use the shared line level
  lexical scanner of `internal/languages/lexical`, driven by the profile of
  each language. The scanner knows the comment and string syntax of the
  language, keeps only the minimal state needed to continue a block comment
  or a multi line string across lines, and never reports a marker that
  lives inside a string or a comment.
- The scanner does **not** build a syntax tree, does **not** resolve names
  and does **not** evaluate expressions. Everything derived from it is
  syntactic evidence.
- Consequently, no rule is declared `resolved` in this milestone. The
  support matrix of every rule is `syntactic` or `none`, and the limitation
  is published with the rule.

**The candidate for Phase 1.5 is Tree-sitter, with the integration mechanism
still pending:** CGO with vendored grammars, or a WASM runtime. Three
questions remain open and must be answered before the decision is closed:

1. The CGO versus WASM choice, with its consequences for the build, the
   binary distribution and the cross platform support.
2. The **license review of the grammars**, which is a precondition: each
   grammar that would be vendored must be reviewed and its license
   recorded before it enters the repository.
3. The **build evidence on the three declared platforms**: Linux, macOS and
   Windows, from a clean clone, with the exact Go version documented.

**The required cross platform evidence is NOT available yet.** No macOS or
Windows build of this project has been produced or observed. The CI
workflow declares a matrix for the three platforms, but a declared matrix is
not evidence.

## Consequences

Positive:

- The tool builds and runs offline, with no dependency and no vendored
  grammar, which is what the current environment requires.
- The seam is explicit. A language package exposes a profile and a set of
  extraction functions; replacing the scanner with Tree-sitter changes the
  implementation of the language package and not the rule engine.
- The support level published with every rule is honest. A user reading
  `codixai audit --rules` sees `syntactic` and never has to guess whether a
  finding was semantically resolved.
- The Go path already shows what a real parser buys: `CDX-REF-001` can
  exist for Go alone, because `go/ast` provides a real declaration set.

Negative:

- Four of the five languages have no tree, so the rules are limited to
  markers that a line level scan can justify. The rules are more prone to
  false positives and to false negatives than a tree based implementation.
- `CDX-REF-001` is only declared for Go. In the other four languages the
  equivalent of the specification rule 1b stays `unresolved`, as the
  specification itself requires.
- The specification exit criterion for Phase 0 that names the parsing
  strategy as decided is **not met**. The decision is proposed, not
  accepted.

## Alternatives considered

**Tree-sitter through CGO with vendored grammars.**
Rejected for now, kept as the leading candidate. It gives a real tree for
every language, which would lift the support level of the rules from
`syntactic` to `resolved` for the cases the grammar covers. The costs:
CGO makes the build depend on a C toolchain, which is exactly what a
reproducible multi platform release must minimize; the grammars must be
vendored, so their licenses must be reviewed and their size must be
accepted; the binary becomes substantially larger. The decision needs the
license review and the three platform builds.

**Tree-sitter through a WASM runtime.**
Kept as the second candidate. It removes the C toolchain requirement and
keeps the grammars in a data file, which may simplify the cross platform
build. The costs: a WASM runtime is itself a dependency with its own
license, its own memory model and its own performance profile, and the
grammar file still has to be reviewed and vendored. The decision needs the
same evidence as the CGO option, plus a measurement of the performance of
the scanner on a realistic diff.

**Language servers (LSP).**
Rejected for this milestone. An LSP server is a long lived process that has
to be started, configured and trusted, it is specific to one language and
one editor ecosystem, and it does not fit the "no execution of the analyzed
repository" constraint, since a server may load project dependencies. The
specification places LSP in a later phase.

**Ecosystem analyzers (ast-grep, Semgrep, and the linters of each
language).**
Rejected for this milestone. They would need to be installed on the machine
of the user, they may download or execute project dependencies, and they
would make the result depend on the environment instead of on the
repository. The specification allows them as optional integrations, with an
explicit availability check and a useful failure message.

**Staying with the line level scanner forever.**
Rejected. The scanner is a milestone solution, not a final one: it cannot
justify a reference rule, and it is the main source of the false positives
that the Phase 1 decision gate will have to measure.
