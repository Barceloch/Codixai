# ADR 0017: Structural index storage engine v2

- Status: accepted
- Date: 2026-11-07

## Context

Phase 5.1 introduced a structural index with SQLite persistence (schema v1). Phase 5.2 adds cross-language reference resolution, requiring storage of references between symbols.

The original schema v1 (ADR 0002 deferred, ADR 0014 accepted) stores:
- Files
- Symbols
- Dependencies (imports)
- Build metadata

Phase 5.2 requires adding:
- References between symbols (calls, type references, imports, inheritance)
- Reference metadata (kind, resolution status, evidence, candidates)

## Decision

**Extend the SQLite schema to v2** with a new `index_references` table and related indexes. The schema remains compatible with v1 databases through migration.

### Schema v2 Changes

```sql
-- New table for references
CREATE TABLE IF NOT EXISTS index_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE,
    source_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL,
    target_file_id INTEGER REFERENCES index_files(id) ON DELETE SET NULL,
    target_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL,
    referenced_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    language TEXT NOT NULL,
    resolution TEXT NOT NULL,  -- resolved, ambiguous, unresolved, external, dynamic
    source_line INTEGER NOT NULL,
    source_column INTEGER,
    evidence TEXT,  -- JSON array of evidence objects
    candidates TEXT,  -- JSON array of candidate symbol IDs
    created_at INTEGER NOT NULL
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_references_source_file ON index_references(source_file_id);
CREATE INDEX IF NOT EXISTS idx_references_source_symbol ON index_references(source_symbol_id);
CREATE INDEX IF NOT EXISTS idx_references_target_symbol ON index_references(target_symbol_id);
CREATE INDEX IF NOT EXISTS idx_references_resolution ON index_references(resolution);
CREATE INDEX IF NOT EXISTS idx_references_language ON index_references(language);
CREATE INDEX IF NOT EXISTS idx_references_name ON index_references(referenced_name);
```

### Migration Strategy

1. **Version detection**: Read `index_metadata.schema_version` (default 1 if missing)
2. **Migration v1 → v2**:
   - Add `index_references` table
   - Add `referenced_name`, `kind`, `resolution`, `evidence`, `candidates` columns
   - Update `schema_version` metadata to 2
3. **Rollback safety**: Migration is additive (new table + columns), no data loss
4. **Version check**: Reject databases with schema_version > current (fail fast)

### Storage Interface Extensions

```go
type IndexStore interface {
    // ... existing methods ...

    SaveReferences(ctx context.Context, refs []Reference, rootPath string) error
    LoadReferences(ctx context.Context, rootPath string) ([]Reference, error)
    ListReferences(ctx context.Context, rootPath string, filter ReferenceFilter) ([]Reference, error)
}
```

### Evidence Storage

Evidence is stored as JSON array in `evidence` column:
```json
[
  {"kind": "ast_call", "ref": "main.go:42:10", "detail": "Call to Foo()", "method": "ast"},
  {"kind": "import_declaration", "ref": "import:github.com/foo/bar", "detail": "Import of bar", "method": "import_graph"}
]
```

Candidates stored as JSON array of symbol IDs:
```json
[123, 456, 789]
```

### FTS5 Consideration

FTS5 is **not enabled** for references in this version. The current query patterns (by file, by symbol, by resolution, by language) are well-served by B-tree indexes. FTS5 can be added in a future version if full-text search on reference names/evidence becomes a requirement.

### Migration Implementation

- `migrateV1toV2(ctx, tx)` — runs in transaction
- Idempotent: checks for table/column existence before adding
- Updates `index_metadata` schema_version to 2
- Preserves all v1 data (files, symbols, dependencies, builds)

### Compatibility

- **Forward**: v1 databases auto-migrate on first open
- **Backward**: v2 databases cannot be opened by v1 code (schema_version check)
- **Forward-compatible reads**: v1 code can read v2 databases if it ignores unknown tables/columns (but will fail schema_version check)

### Consequences

Positive:
- Minimal schema change, additive only
- Preserves all v1 data
- Supports full reference resolution data model
- Transactional migration

Negative:
- Schema version bump requires migration on first use
- New columns in existing tables not needed (additive table only)

## Alternatives Considered

**Separate reference database**: Rejected — adds complexity, transaction boundaries across databases are hard.

**JSON blob in index_files**: Rejected — loses queryability, no FK integrity.

**Separate reference file (JSON/msgpack)**: Rejected — loses transactional consistency with index.

## Compliance

- Domain imports nothing (invariant 2)
- Application layer does not depend on presentation (invariant 3)
- Results are versioned, serializable documents
- Error messages tell user how to recover
- No execution of analyzed repository code (invariant 1)
- Honest epistemic status on every finding (invariant 6)