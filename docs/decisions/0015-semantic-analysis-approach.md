# ADR 0015: Semantic analysis approach for Phase 6

- Status: accepted
- Date: 2026-10-24

## Context

Phase 6 requires semantic analysis capabilities including:
1. Public API change detection and consumer impact
2. Code similarity detection for potential reimplementations
3. Language-specific resolvers with ecosystem tools (LSP, gofmt, pylint, etc.)
4. Architecture analysis and configurable conventions
5. Optional IA-based explanations based on evidence

## Decision

Codixai will implement **semantic analysis capabilities using pure-Go parsers and syntactic markers**, without requiring external toolchains or language-specific runtimes. This approach respects the project invariants:

- **Invariant 1**: Never execute analyzed repository code
- **Invariant 6**: Honest epistemic status on every finding
- **Invariant 10**: No telemetry, no code upload, no remote execution

## Implementation approach

### 1. API Change Detection (P6-01)
- Use existing rule catalog and change detection infrastructure
- Compare symbol signatures across revisions
- Report breaking changes (removed functions, changed signatures)
- Mark relationships as `unresolved` when cannot be determined syntactically

### 2. Code Similarity (P6-02)
- Extract symbol signatures (name, parameters, return type structure)
- Compare signatures across files using string similarity
- Report similarity percentages and file pairs
- Mark as `unresolved` when similarity cannot be determined

### 3. Language-Specific Resolvers (P6-03)
- **Go**: Use `golang.FindUndefinedReferences` and `golang.Parse` for import analysis
- **Python**: Use lexical profile markers and import detection
- **JS/TS**: Use lexical profile markers  
- **PHP**: Use lexical profile markers
- All resolvers mark unresolvable relationships as `unresolved`

### 4. Architecture Analysis (P6-04)
- Use existing dependency tracking from audit/verify use cases
- Report cycle detection based on import graphs
- Mark conventions respected/not-respected

### 5. Optional IA Explanations (P6-05)
- Provide brief evidence-based explanations (max 2 sentences)
- Separate deterministic findings from IA explanations
- Never replace deterministic analysis

## Consequences

- All analysis remains purely syntactic/structural
- No external tool execution (respects Invariant 1)
- Epistemic status preserved on every finding
- Cross-platform, zero network dependencies for core analysis
- Results versioned with `schema_version`

## Compliance

- Domain imports nothing (invariant 2)
- Application layer does not depend on presentation (invariant 3)
- Results are versioned, serializable documents
- Error messages tell user how to recover
