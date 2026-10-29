# ADR 0014: Structural indexing storage engine

- Status: accepted
- Date: 2026-10-24

## Context

Phase 5 requires structural analysis and on-demand impact capabilities. The specification requires:
1. Extending the index ONLY to improve rules or proven use cases
2. Fixtures for the 5 supported languages verify advertised capabilities  
3. Unresolved relationships declared as `unresolved`
4. Do NOT build a costly global graph without measurable use cases

Need to choose storage/indexing engine for lightweight structural indexation.

## Decision

Codixai will use a **lightweight, pure-Go in-memory index** with optional SQLite caching, NOT an external or heavy database engine.

Key design:
- **No global graph**: store symbols (file, line, symbol, kind, imports) per file in structured data
- **Pure Go / AST parsers**: use `go/parser` for Go, lexical profiles for Python/JS/TS/PHP
- **No toolchain invocation**: no `go list`, no `npm`, no language toolchains (Invariant 1)
- **Unresolved status**: unresolvable relationships marked `unresolved`
- **Cross-platform**: zero CGO; pure Go guarantees cross-platform execution

## Consequences

- Index creation is fast and on-demand
- Simple queries (`status`, `overview`, `find`, `refs`, `deps`) execute in-memory
- No network, external binary, or database daemon required
