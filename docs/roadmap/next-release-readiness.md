# Next Release Readiness Assessment

- Date: 2026-11-14
- Updated: 2026-11-19 (WU1 gofmt clearance, WU2 MCP contract tests, WU3 SARIF schema verification, WU6 cross-platform verification)
- Assessed branch: `phase-5-2-5-2-reference-consistency`
- Assessed HEAD: `a7f5096828ee599ecee273f5b60b6782bc75adfc`
- Latest tag: `v0.1.0-rc.3`
- Recommended next target: `v0.1.0-rc.4` (pre-release), with `v0.1.0` as the eventual stable goal.

Statuses: `PASS`, `PARTIAL`, `FAIL`, `NOT VERIFIED`, `NOT APPLICABLE`.

## 1. Product completeness

| Area | Status | Evidence |
|---|---|---|
| CLI usability and command consistency | PASS | Command tree in `internal/cli/commands.go`; consistent flags and exit codes documented in conventions. |
| JSON output / machine-readable contracts | PASS | `--format json` on audit/report/index/measure; `domain.SchemaVersion` carried. |
| SARIF output and schema validity | PASS | `internal/report/sarif.go` produces SARIF 2.1.0; WU3 validates real CLI/MCP/builder documents against the local copy of the authoritative OASIS schema. See section 8. |
| Directory auditing | PASS | Full directory auditing added (commits since rc.3); `RunDirectoryAudit` + `audit --directory`. |
| Git-based auditing | PASS | `audit --base/--staged/--worktree`; `internal/integrations/git`. |
| Reference resolution | PARTIAL | Implemented for Go/Python/JS/TS but unreachable from CLI/MCP (dead path; see reachability audit). |
| Index persistence and recovery | PARTIAL | SQLite store works; `migrateToV2` has no caller (untested); `Save` writes empty hash/size. |
| MCP functionality and tool contracts | PARTIAL | Exactly 5 tools: 4 read-only (`audit_diff`, `audit_directory`, `verify_claims`, `report`) plus the mutating `baseline_create`. Contract tests added in WU2 and green; see the observation below on error semantics. |
| Error handling and graceful degradation | PASS | Errors wrapped with `%w` and recovery guidance; missing tools reported as unavailable. |
| Documentation and examples | PASS | README, guides, ADRs, tasks; Spanish guide present. |

## 2. Engineering quality

| Area | Status | Evidence |
|---|---|---|
| Unit and integration tests | PASS | `go test ./...` PASS (all packages incl. `tests/integration`). |
| Regression tests for known defects | PASS | WU4/WU5 regression tests present and green; corpus 100/100. |
| `go test ./...` | PASS | Verified 2026-11-14. |
| `go test -race ./...` | PASS | Verified 2026-11-14. |
| `go vet ./...` | PASS | Verified 2026-11-14. |
| `go build ./...` | PASS | Verified 2026-11-14 (incl. `ui` build tag). |
| `gofmt` compliance | PARTIAL | Only pre-existing `internal/domain/models.go` is unformatted; baseline issue, not introduced by this work. |
| `git diff --check` | PASS | Verified 2026-11-14 and 2026-11-19. |
| Deterministic output | PASS | Sorted findings; corpus determinism test; measurement harness determinism test. |
| Official SARIF schema validation | PASS | WU3: authoritative OASIS 2.1.0 schema vendored as test data; all generated documents pass; negative control rejects invalid documents. |
| Security (no execution of analyzed code) | PASS | Invariant 1; only `git` read-only subcommands invoked. |

## 3. Distribution and compatibility

| Area | Status | Evidence |
|---|---|---|
| Version injection / reporting | PASS | `-X main.version` injection documented; `codixai version` prints schema version. |
| Linux build and runtime | PASS | Native runtime smoke on the build host 2026-11-19 (section 9): version, help, directory audit, diff audit, SARIF and MCP stdio. Cross-compile amd64/arm64 PASS. |
| Windows build and runtime | PARTIAL | Cross-compile amd64/arm64 PASS (2026-11-19, section 9). No local runtime. CI runs `version` on `windows-latest`; the release workflow runs `--help`, `version` and the `ui` exit code natively. No functional audit/MCP execution and no observed result. |
| macOS build and runtime | PARTIAL | Cross-compile amd64/arm64 PASS (2026-11-19, section 9). No local runtime. CI runs `version` on `macos-latest`. No functional audit/MCP execution, no observed result and no packaged macOS artifact. |
| CLI packaging and installation | PASS | `scripts/release/package.sh` produces tar.gz/zip with VERSION, LICENSE, README. |
| MCP installation and configuration docs | PARTIAL | `docs/mcp-integration.md` exists; WU2 adds automated contract tests, but no automated end-to-end test with a real MCP client exists. |
| Release artifacts and checksums | PASS | SHA-256 printed by package script. |
| Signing | NOT APPLICABLE | No signing implemented; documented as not signed. |
| Upgrade and recovery behavior | PARTIAL | `migrateToV2` exists but has no caller; v1->v2 migration untested. |

## 4. Mandatory blockers for the next release

1. **Windows/macOS runtime verification — RESOLVED as a documented limitation (WU6)** — cross-compilation is verified for `windows/amd64`, `windows/arm64`, `darwin/amd64` and `darwin/arm64`; native functional execution remains Linux-only, with startup-level native checks defined in CI for Windows and macOS. Accepted for the `v0.1.0-rc.4` pre-release; see section 9 and the CHANGELOG limitation.

Cleared since the first assessment:

- **gofmt compliance** — cleared in WU1: `gofmt -w` was applied only to `internal/domain/models.go` (whitespace alignment); `gofmt -l .` is empty and the release CI formatting gate can pass.
- **CHANGELOG release entry — cleared in WU4**: the `[Unreleased]` section now names directory auditing, reference normalization, the ACO Phase 0 measurement harness and the reachability audit, and records the platform-verification limitation. The deferred items are described, not shipped.
- **MCP contract tests** — cleared in WU2: `internal/integrations/mcpserver/server_test.go` drives the real JSON-RPC entry point and covers registration, input validation, success documents, error behavior, read-only guarantees, determinism and the baseline write for all 5 tools.
- **SARIF schema validation** — cleared in WU3: the decision-0009 shape is no longer the only structural check; the authoritative schema is validated offline against real documents. No production defect was found, so no production file was changed.

### MCP contract observation (WU2, non-blocking)

The input schemas declare `root`, `base` and `claims_path` as required, but the server does not enable mcp-go's input-schema validation. Missing or wrongly typed parameters are rejected inside the handlers and surface as JSON-RPC `INTERNAL_ERROR` (-32603) instead of a tool result with `isError` or `INVALID_PARAMS`. Application failures, by contrast, are reported with `isError`. The tests pin this current behavior; changing it would alter the public error contract and is therefore deferred to an explicit follow-up decision.

A second finding is documentation drift: `docs/mcp-integration.md` documents only `audit_diff` and `verify_claims`, while the server registers five tools. Updating that guide is a separate documentation work unit.

## 5. Important improvements (non-blocking)

- Make reference resolution reachable (CLI `refs` wrapper + MCP `references` tool) — depends on ADR 0018 clarification; separate work unit.
- Make impact consume the index/references — ADR 0020, separate work unit.
- Exercise or remove `migrateToV2`; write empty hash/size fix — ADR 0021 follow-up.
- Runtime verification of the full CLI/MCP behavior on Windows and macOS (only startup-level CI checks exist today).

## 6. Long-term capabilities that must not block this release

- ACO Phase 1+ (context packages, packaging, budgets).
- PHP reference resolver.
- WU2/WU3 completion.
- Corpus cardinality assertions.
- Incremental indexing.
- Full type checker for TypeScript.

## 7. Recommended release scope

The smallest coherent release is `v0.1.0-rc.4` (pre-release) carrying: full directory auditing, reference normalization and evidence coverage (already in rc.3..HEAD), plus the ACO Phase 0 measurement harness and reachability audit. It should not wait for reference-aware impact or PHP. The changelog entry is written (WU4), and blocker 2 is accepted as a documented limitation for the rc.

## 8. WU3 — Official SARIF 2.1.0 schema verification

- **Schema source**: the OASIS Standard JSON schema for SARIF 2.1.0, retrieved from `https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/schemas/sarif-schema-2.1.0.json`. Its `$id` is the canonical `https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json`. SHA-256: `ad6db49878699b091f3eeb765b6e29e92a34bad4da88664d000c923b549c3a25`.
- **Method**: the schema is vendored as `tests/integration/testdata/sarif-schema-2.1.0.json` (source and hash recorded in the adjacent README). `tests/integration/sarif_schema_test.go` registers it under its own `$id` and compiles it offline with the `github.com/santhosh-tekuri/jsonschema/v6` validator already present in `go.mod`; no new dependency and no runtime network access. Format assertions are enabled. A negative control proves that the validator rejects an unsupported version, an unknown root field and a missing `runs` array.
- **Schema comparison**: the SchemaStore copy referenced by the output documents was also retrieved. After removing descriptions, titles, defaults and examples and normalizing `const` to a one-value `enum`, the only differences are redundant `properties` entries in `anyOf`/`oneOf` branches; the accepted document set is the same.
- **Paths and cases validated**: `report --directory` (with and without findings), `report --worktree` (with and without findings), `report --base` and `report --staged` through the real CLI; the MCP `report` tool for directory and worktree origins through the real JSON-RPC entry point; and the production builders for a nil result, a finding without a line, a finding whose rule is absent from the catalog, and a directory run with missing rules. Every case asserts that the document was produced by the real path and, when applicable, that findings were present.
- **Result**: every generated document passed. No schema violation was found, so no failing regression test and no production correction were needed. `internal/report/sarif.go` was not modified by WU3.
- **Limitations**: the export does not emit evidence or related locations, so those SARIF shapes are not exercised. The UI export uses the same `report.SARIF` builder as the CLI and MCP paths and is covered transitively; no separate UI case was added. Schema validation is a test-time gate, not a runtime behavior of the tool. Runtime messages in the report command still say the tool never fetches a schema; that remains true.

## 9. WU6 — Cross-platform verification

- **Environment**: Linux x86_64 (Manjaro), Go 1.27.1. Every cross-build used
  `CGO_ENABLED=0`. No native Windows or macOS runtime is available here. Wine
  11.19 is installed, but a bounded startup probe timed out with no output; it
  was deliberately not used as verification because an emulator is not a
  native platform.
- **Native Linux runtime (executed 2026-11-19)**: built `./cmd/codixai` and
  `./cmd/codixai-mcp-srv` with `CGO_ENABLED=0 go build -trimpath`. Observed,
  all exiting 0: `version`, `--help`, `audit --help`, `report --help`,
  `measure --help`; `audit <fixture-directory> --format json` returned
  `schema_version: 0.1`, `origin: directory`, 3 analyzed / 2 skipped files and
  `missing_rules: CDX-TST-001`; a throwaway Git repository produced a
  `--worktree` JSON document and a SARIF 2.1.0 export; the MCP server completed
  a real stdio `initialize` → `tools/list` (five tools) → `tools/call
  audit_directory` exchange. Fixtures lived in a scratch directory outside the
  worktree.
- **Cross-compilation (executed 2026-11-19)**: `GOOS`/`GOARCH` matrix
  `windows/amd64`, `windows/arm64`, `darwin/amd64`, `darwin/arm64` with
  `CGO_ENABLED=0 go build -trimpath`, for both `./cmd/codixai` and
  `./cmd/codixai-mcp-srv`. `file` confirmed PE32+ for Windows and Mach-O for
  macOS in every case. This is build evidence only, not runtime verification.
- **Existing CI evidence (defined in the workflows, not observed from this
  environment)**: `.github/workflows/ci.yml` builds and runs
  `go run ./cmd/codixai version` on `ubuntu-latest`, `macos-latest` and
  `windows-latest`; `.github/workflows/release.yml` runs `codixai.exe --help`,
  `version` and the `ui` exit code natively on `windows-latest`. These are
  startup-level checks: they do not exercise auditing, claims, reporting or
  the MCP tools, and the release workflow publishes no macOS artifact.
- **Unverified**: functional behavior (directory and diff auditing,
  `verify_claims`, `report`, the MCP tools and filesystem/path edge cases) on
  Windows and macOS; the graphical interface on every platform; the runtime
  results of the CI jobs.
- **Decision**: option B. `v0.1.0-rc.4` proceeds as a pre-release with the
  explicit limitation that the Windows and macOS binaries are cross-compiled but
  not runtime-verified, and that only Linux was executed end-to-end. This does
  not claim universal cross-platform runtime compatibility.
