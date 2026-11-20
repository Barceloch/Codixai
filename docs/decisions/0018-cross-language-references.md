# ADR 0018: Cross-language reference resolution architecture

- Status: accepted
- Date: 2026-11-07

## Context

Phase 5.2 requires cross-language reference resolution to identify relationships between symbols across files and languages. The specification requires:

1. Identify references between symbols (calls, type references, imports, inheritance)
2. Distinguish observed relationships from inferred ones
3. Preserve evidence and location information
4. Support five languages: Go, Python, PHP, JavaScript, TypeScript
5. Integrate with existing index, impact, api_change, and architecture packages

## Decision

Codixai will implement a **three-layer architecture** for cross-language reference resolution:

### 1. Parsing Layer (Language-specific extractors)

Each language package provides a `ReferenceExtractor` that extracts syntactic occurrences of references from source code:

- **Go**: Uses `go/ast` to extract call expressions, method calls, type references, imports
- **Python**: Lexical scanner with regex patterns for calls, attribute access, imports
- **JavaScript/TypeScript**: Lexical scanner with segment-aware string extraction
- **PHP**: Lexical scanner for function calls, method calls, static calls

Each extractor returns **syntactic occurrences** (ReferenceOccurrence) with:
- Source location (file, line, column)
- Referenced name
- Reference kind (call, method, type, import, etc.)
- Container symbol (when determinable)
- Language

The extractor does **not** resolve references — it only reports what the syntax shows.

### 2. Resolution Layer (ReferenceResolver)

A language-agnostic `ReferenceResolver` consumes syntactic occurrences and the structural index to produce resolved references:

1. Builds a symbol table from the structural index (SymbolRef)
2. For each syntactic occurrence:
   - Attempts to match against imported symbols (via imports/dependencies)
   - Attempts to match against local declarations (same file/package)
   - Attempts to match against package-level declarations (via import graph)
3. Produces `Reference` objects with:
   - Source SymbolRef
   - Target SymbolRef (when resolved)
   - ResolutionStatus (resolved, ambiguous, unresolved, external, dynamic)
   - Evidence chain

The resolver is language-agnostic but uses language-specific symbol table builders.

### 3. Evidence Classification Layer

Each Reference carries an Evidence chain with explicit epistemic status:

- **observed**: Direct syntactic evidence (e.g., AST node for call, import declaration)
- **inferred**: Heuristic match (e.g., name match via import path, heuristic module resolution)
- **unknown**: Insufficient information

ResolutionStatus is distinct from EpistemicStatus:
- **ResolutionStatus**: resolved, ambiguous, unresolved, external, dynamic
- **EpistemicStatus**: observed, inferred, unknown (applies to each evidence item)

## Domain Model

### Reference

Represents a resolved reference between symbols.

```go
type Reference struct {
    ID            string
    Source        SymbolRef
    Target        *SymbolRef        // nil when unresolved
    Kind          ReferenceKind
    Resolution    ResolutionStatus
    Candidates    []SymbolRef       // for ambiguous
    Evidence      []Evidence
    Language      string
}
```

### SymbolRef

Identifies a symbol with enough information for resolution:

```go
type SymbolRef struct {
    Name       string
    Kind       string
    File       string
    Line       int
    Column     int
    Language   string
    Container  string       // package, class, namespace
    Signature  string
    Exported   bool         // true/false/unknown
}
```

### ReferenceKind

```go
type ReferenceKind string

const (
    ReferenceCall        ReferenceKind = "call"
    ReferenceMethod      ReferenceKind = "method"
    ReferenceType        ReferenceKind = "type"
    ReferenceImport      ReferenceKind = "import"
    ReferenceInheritance ReferenceKind = "inheritance"
    ReferenceVariable    ReferenceKind = "variable"
    ReferenceField       ReferenceKind = "field"
)
```

### ResolutionStatus

```go
type ResolutionStatus string

const (
    ResolutionResolved    ResolutionStatus = "resolved"
    ResolutionAmbiguous   ResolutionStatus = "ambiguous"
    ResolutionUnresolved  ResolutionStatus = "unresolved"
    ResolutionExternal    ResolutionStatus = "external"
    ResolutionDynamic     ResolutionStatus = "dynamic"
)
```

### Evidence (extends domain.Evidence)

```go
type ReferenceEvidence struct {
    Kind        string  // "ast_call", "lexical_call_pattern", "import_declaration", "module_resolution", "heuristic_candidate"
    Ref         string  // e.g., "file.go:42:10", "import:github.com/foo/bar"
    Detail      string
    Method      string  // "ast", "lexical_pattern", "import_graph", "heuristic"
}
```

## Resolution Strategies by Language

| Language | Strategy | Resolved When |
|----------|----------|---------------|
| **Go** | AST + package resolution | Direct import, same package, explicit import alias |
| **Python** | Import graph + module resolution | Absolute import, explicit from-import, resolved relative import |
| **JavaScript** | Module graph + dynamic import tracking | Static import, resolved require() |
| **TypeScript** | Module graph + type reference tracking | Static import, type reference |
| **PHP** | Use statement + PSR-4 heuristic | Use statement, static call with namespace |

Unresolved cases are marked with appropriate ResolutionStatus and Evidence explaining why.

## Integration with Existing Packages

- **index**: Provides structural index (SymbolRef table, import graph)
- **impact**: Uses resolved references for caller/callee analysis
- **api_change**: Uses resolved references for consumer impact
- **architecture**: Uses resolved references for coupling/cohesion
- **claims**: api_unchanged, behavior_preserved use resolved references

### Implementation status (verified 2026-11-14)

This list records design intent, not delivered integration. Verified against the code:

- `index` provides the structural index as described.
- `impact`, `api_change`, `architecture` and `claims` do **not** consume resolved references today. `internal/impact` never reads its index field, its only call site passes `nil`, and no production code outside `internal/index` and `internal/storage` references `domain.Reference`.
- PHP reference extraction (see *Parsing Layer*) and the `ReferenceExtractor` interface are **not implemented**. `internal/index` contains Go, Python, JavaScript and TypeScript resolvers only, and the comparative corpus has no PHP case.

Follow-up decisions are tracked in `docs/roadmap/phase-5-2-6-closure-report.md` and ADR 0020.

#### Scope clarification (approved 2026-11-14)

- Four audited comparative resolvers: Go, Python, JavaScript, TypeScript. PHP reference resolution is **NOT IMPLEMENTED** and is **not** part of Phase 5.2.6.
- PHP is a separate future development item: its own resolver implementation, language-specific fixtures, regression tests, quality audit, and acceptance criteria. Do not imply PHP reference resolution is currently implemented.
- The broader product language vision (five languages) does not equal implemented resolver coverage. The comparative audit covers four languages.
- The four announced downstream integrations (`impact`, `api_change`, `architecture`, `claims`) remain **PENDING / NOT IMPLEMENTED** unless implementation evidence proves otherwise. `internal/impact` holds an `*index.Index` it never reads and its only production call site passes `nil`; `domain.Reference` has no non-test caller outside `internal/index` and `internal/storage`.
- Cross-reference: closure report §13.3 and §13.4.
## Known Limitations

1. **Non-Go languages**: Resolution is heuristic (`inferred`), never `observed` for cross-file
2. **Dynamic references**: `require(var)`, `import()`, `eval()`, `getattr()` marked `dynamic`
3. **Ambiguous imports**: Multiple candidates marked `ambiguous` with candidate list
4. **Dynamic imports**: `import()`, `require(var)`, `importlib.import_module()` marked `dynamic`
5. **Relative imports**: Python/JS relative imports unresolved if not resolvable from index
6. **Dynamic dispatch**: Interface calls, virtual methods marked `unresolved` without concrete implementation evidence

## Future Compatibility with Tree-sitter

The architecture is designed to accept Tree-sitter extractors as a drop-in replacement for the lexical scanner. The `ReferenceExtractor` interface remains the same; only the implementation changes. When Tree-sitter is enabled via build tag, the language packages swap the lexical extractor for a Tree-sitter based one that produces richer `ReferenceOccurrence` data (exact column, exact node type, etc.).

## Consequences

Positive:
- Clean separation of parsing, resolution, and evidence
- Language-agnostic resolver core
- Honest epistemic status on every reference
- No CGO requirement for default build
- Tree-sitter opt-in path preserved

Negative:
- Non-Go cross-file resolution remains `inferred`/`unresolved`
- Dynamic features not resolved
- Relative imports may stay `unresolved`

## Alternatives Considered

**Single-pass resolver per language**: Rejected — couples parsing and resolution, harder to test and swap.

**Global call graph**: Rejected — too expensive, not needed for current use cases.

**LSP integration**: Rejected — requires external toolchain, violates Invariant 1.

## Compliance

- Domain imports nothing (invariant 2)
- Application layer does not depend on presentation (invariant 3)
- Results are versioned, serializable documents
- Error messages tell user how to recover
- No execution of analyzed code (invariant 1)
- Honest epistemic status on every finding (invariant 6)