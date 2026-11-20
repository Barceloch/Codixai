# ADR 0019: The context package is an application read model

- Status: accepted (2026-11-14) — decision approved; implementation NOT IMPLEMENTED
- Date: 2026-11-14
- Depends on: ADR 0018 (reference resolution), ADR 0017 (index schema v2)

## Context

Agent Context Optimization (ACO) needs a serializable document that describes the relevant symbols, references, evidence, uncertainty and budgets for one task.

The repository has a clear precedent: every serializable document already lives in the application layer — `application.VerificationDocument` (`internal/application/verify.go:57`) and `application.AuditResult` (`internal/application/audit.go:45`). `domain` holds only vocabulary (`domain/references.go`). `domain` is constrained by `AGENTS.md` invariant 2 (the domain imports nothing) and by `internal/application/architecture_test.go:58`, which forbids project-internal imports.

The question is where a *consumer-facing projection* belongs. A context package is derived, task-shaped and disposable. It is not an invariant of the repository model, and it must never influence how references are resolved.

## Options

1. **Domain type.** `domain.ContextPackage`. Simple, but it pulls a consumer concern (prompt budgeting, disclosure levels) into the vocabulary that all providers must satisfy.
2. **Application read model.** A type in `internal/application/context` that composes domain vocabulary. CLI, MCP and UI share it; the domain stays unchanged.
3. **New top-level package.** `internal/context`. Clear ownership, but it duplicates the role of the application layer and invites a second orchestration path.

## Recommendation

**Option 2 — application read model.**

- The package composes `domain.Reference`, `domain.SymbolRef` and `domain.ReferenceEvidence`; it introduces no new domain concept.
- `domain` continues to know nothing about ACO.
- The document carries its own `schema_version`, independent of `domain.SchemaVersion`. The findings schema must not be bumped because a new, unrelated document type appeared.
- CLI, MCP and a future UI call the same use case and serialize the same type.

## Consequences

- A new document contract exists and needs the same discipline as the findings contract: versioning, determinism, stable field names.
- The application layer grows one use case family (`context build/inspect`) and no new provider abstraction in the first iteration.
- `schema_version` for the package must be fixed before ACO Phase 1 ships, and any breaking change to it follows invariant 5.
- If a future consumer needs the package inside `domain`, that is a separate ADR with evidence.

## Approved versioning contract (2026-11-14)

- The context package carries its own `schema_version`, independent of: the global project version; the database schema version; report/findings schema versions (`domain.SchemaVersion`); MCP protocol versioning.
- Independence: a change to the findings schema does not change the context package schema, and vice versa.
- Compatibility: additive optional fields may be added without a version bump; consumers MUST tolerate unknown fields and treat absent optional fields as unspecified.
- Breaking version change required when: a field is removed or renamed; the meaning or type of an existing field changes; the structure of an existing field changes incompatibly; established interpretation or semantics change incompatibly.
- Domain independence: the context package is an application read model; `domain` imports nothing and knows nothing about ACO; no ACO vocabulary is added to `domain`.
- Link to ADR 0018 and ADR 0017.

## Open questions

- Should the package be persisted, or regenerated per request? (Current recommendation: regenerated; `package_id` is a content hash.)
- Does the package need a `schema_version` distinct from the findings schema, or should the project adopt one global document version? **Resolved 2026-11-14 (Decision 3): distinct `schema_version`, approved.** (Recommendation: distinct, but this is a maintainer decision under `AGENTS.md` stop condition 4.)
