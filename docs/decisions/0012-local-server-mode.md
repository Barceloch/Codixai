# 0012 — The local server mode

- Status: **Decision pending** (pending decision 12)
- Date: 2026-10-24

## Context

The specification's Fase 7.4 mentions a local API for external clients, and the risk section of spike 7.0 identifies the packaging and distribution as the main risk, not the screens. The local server mode would add a browser on the loopback, exposing the audit interface programmatically.

This decision asks whether to add it, and under what security controls.

## Decision

**No local server mode.** The native window (Wails v2) remains the only delivery mechanism for the UI. No HTTP server is embedded in the binary, and no loopback port is opened.

### Reasons

- **Packaging risk.** Every opened port increases the binary's attack surface and the CI/test matrix. The spike 7.0 already measures size and startup; adding a server would add an uncontrolled dependency on port availability and OS networking.
- **Security model.** A loopback server requires a token, Host/Origin validation, and CORS management. Getting these right in a binary that must compile on Linux/macOS/Windows without a dedicated QA cycle is fragile. The architecture already protects the read-only use case through the Wails bridge + file dialogs.
- **Use case absence.** No consumer in the specification or in the project's stated needs justifies exposing an API surface beyond the native window + file dialogs. The `codixai verify` command already serves the same programmatic purpose (claims file + repository diff).
- **Maintenance burden.** A server mode would need its own tests, its own security review, its own documentation, and its own deprecation path. The project's stability hinges on the two variants (CLI-only and CLI+UI) staying simple.

### If a local server mode is ever needed

A future decision would have to define:

- **Loopback binding.** `127.0.0.1:<random_port>` with a deterministic allocation strategy.
- **Per-session token.** A random JWT or opaque string, persisted only for the lifetime of the binary process, never written to disk.
- **Host/Origin validation.** Only requests with `Host: 127.0.0.1` and `Origin: http://127.0.0.1:<port>` are accepted; any other origin is refused with `403`.
- **No CORS.** The server explicitly disables CORS headers; the bridge only ever calls from the same window.
- **Read-only exposure.** The server only serves GET endpoints with the audit/summary data; no POST/DELETE/PUT that modifies state. All state changes go through the Wails bridge + the CLI commands.
- **Expiration.** The server shuts down when the binary exits; no stale port lingering.

However, this remains a hypothetical path: the current decision is to omit the mode entirely.

## Consequences

Positive:

- No extra surface to test, document, or maintain in CI.
- No port conflicts, no CORS surprises, no token leakage risk.
- The binary stays focused on the two build variants (CLI-only and CLI+UI).
- Security is inherited from the Wails bridge + file dialogs, which are already tested.

Negative:

- If a consumer later needs a programmatic interface beyond the CLI, they must use `codixai verify` with a claims file, or wrap the binary's exit code.
- The decision can be revisited in a future ADR if a validated use case appears.