# Phase 5.2.6 — Cross-Language Reference Consistency: Closure Report

- Date: 2026-11-14
- Branch: `phase-5-2-5-2-reference-consistency`
- HEAD: `a7f5096828ee599ecee273f5b60b6782bc75adfc`
- Work-unit records: `odd/tasks/cross-language-reference-consistency.md` (WU1), `-wu2.md`, `-wu3.md`, `-wu4.md`, `-wu5.md`
- Deep audit report: `/tmp/codixai-recovery/manifest/wu5-deep-audit-report.md`

## Verdict

**`PHASE PARTIALLY COMPLETE`**

The audited reference-consistency work delivered its stated scope, and every check executed for this closure passes: the comparative corpus is **25 concepts / 100 subtests / 100 PASS**, 14 demonstrated defects were fixed with failing-test-first evidence, and the WU4/WU5 regression tests are present and green.

The phase cannot be declared complete for three reasons, all evidenced below:

1. **WU2 and WU3 remain `PARCIAL`** and their unexpanded areas were never closed or formally deferred by WU4 or WU5.
2. **PHP has no reference resolver and no comparative fixture**, while ADR 0018 announces PHP reference extraction and the specification's Phase 5 exit criterion asks for fixtures in five languages.
3. **ADR 0018 announces four downstream integrations** (`impact`, `api_change`, `architecture`, `claims`) that consume resolved references. No production code performs any of them.

None of these invalidates the delivered resolver work. They prevent an honest claim of phase closure.

The human-approved dispositions recorded in §13 resolve the documentation-level conflicts identified above but do not change this verdict.

## 1. Phase objective

Phase 5.2.6 audits whether the Go, Python, JavaScript and TypeScript reference resolvers satisfy a shared contract, and builds a comparative corpus plus evaluator that makes the invariants falsifiable (WU1 objective, `odd/tasks/cross-language-reference-consistency.md:3-4`).

The governing exit criterion is the specification's Phase 5 criterion (`docs/spec/CODIXAI_SPEC.md:703`):

> fixtures para los cinco lenguajes verifican las capacidades que se anuncian; las relaciones sin resolver se declaran como tales. No construir un grafo global más costoso sin casos de uso medibles.

## 2. Scope and completed work

| WU | Scope | Delivered |
| --- | --- | --- |
| WU1 | Comparative contract, taxonomy, resolution states, corpus and evaluator | Corpus 01–10 foundations (40 cases), `TestCrossLanguageReferenceCorpus`, 2 defect fixes |
| WU2 | Advanced shadowing, members, module ambiguity, TS spaces, computed dispatch | Concepts 11–17 (68 cases), 4 defect fixes |
| WU3 | Closures, typed receivers, inheritance, local-vs-external, cycles/reexports | 72 cases, 3 defect fixes, `external` evidence discipline |
| WU4 | Scope and binding completeness | 1 defect fix, zero new concepts |
| WU5 | Closure capture, reassignment, block scope, `global`/`nonlocal`, hoisting/TDZ, external audit | Concepts 19–25 (100 cases), 3 defect fixes, then a second audit round with 1 further defect fix and 10 new unit tests |

## 3. WU1–WU5 status matrix

| WU | Recorded status (verbatim) | Verified status | Evidence | Open gap |
| --- | --- | --- | --- | --- |
| WU1 | `**COMPLETADA** (with documented limitations and minimal defect corrections)` (`…consistency.md:171`) | Complete for the audited scope | Corpus 01–10, evaluator, 2 fixes | Cardinality not asserted by the suite |
| WU2 | `**PARCIAL** (… not every enumerated A-E minimum subcase added as a dedicated comparative concept …)` (`-wu2.md:104`) | Partial, confirmed | Concepts 11–17 | Typed/inferred receiver matrix, inheritance/override matrix, JS/TS cycle breadth, most TS advanced spaces |
| WU3 | `**PARCIAL** (… not all enumerated A–F minimum subcases added as dedicated comparative concepts)` (`-wu3.md:121`) | Partial, confirmed | 72 cases, 3 fixes | Same residual matrix; inheritance row recorded as “Partial (not expanded)” (`-wu3.md:24`) |
| WU4 | `**COMPLETADA** for the demonstrated-defect scope of WU4` (`-wu4.md:43`) | Complete for its scope | 1 fix + regression test | Zero corpus additions (declared) |
| WU5 | `**COMPLETE** — the original WU5 acceptance criteria are verified …` (`-wu5.md:136`) | Complete for its scope | Concepts 19–25, 100/100, 13 named tests exist | — |

WU2/WU3 partiality is **explained in-area but never reconciled at closure**. No later document accepts those areas as deferred, and the Engram-sync debt noted in WU1–WU3 simply disappears from WU4 and WU5.

## 4. Resolver capabilities by language

| Capability | Go | Python | JavaScript | TypeScript | PHP |
| --- | --- | --- | --- | --- | --- |
| Symbol extraction | `go/parser` + `go/ast` | Lexical extractor | Lexical extractor | Lexical extractor | File-level symbol only (`internal/index/index.go:207-215`) |
| Import / module resolution | `go/ast` imports | Absolute + relative module resolution | Module graph (`JSModuleResolver`) | Module graph + type/value spaces | `use` statements (lexical, analysis side only) |
| Reference extraction | AST + lexical | Lexical | Lexical | Lexical + signature/type scanning | **None** |
| Resolution states | resolved, ambiguous, unresolved, external, dynamic | same | same | same | n/a |
| Comparative corpus | 25 concepts | 25 concepts | 25 concepts | 25 concepts | **0** |
| Dedicated unit suite | `go_resolver_test.go` | `python_resolver_test.go` | `javascript_resolver_test.go` | `typescript_resolver_test.go` | **None** |

PHP is indexed only as one file-level symbol per file; the five-language index test passes on that symbol (`internal/index/index_five_lang_test.go:31-40`), which is a weaker guarantee than a reference capability.

## 5. Comparative corpus results

| Metric | Value |
| --- | --- |
| Concepts | 25 (`testdata/cross-language-reference/01_…`–`25_…`) |
| Language cases | 100 (`expect.json` per concept × 4 languages) |
| Case objects inside those files | 76 |
| Result (executed 2026-11-14) | **100/100 PASS** |
| Expected resolutions used | `dynamic` 29, `resolved` 23, `ambiguous` 10, `unresolved` 8, `external` 6 |
| Comparability labels | `approximately_equivalent` 41, `language_specific` 32, `strictly_equivalent` 14, `non_comparable` 13 |

Non-emission is encoded as an empty `cases` array plus a `comparability` label; there is no `non_emission` resolution state. The evaluator asserts `resolved` ⇒ target and no candidates, `ambiguous` ⇒ no target and ≥2 candidates, and leaves `unresolved`/`external`/`dynamic` to state plus evidence checks (`internal/index/cross_language_reference_audit_test.go:258-331`).

**Caveat:** the corpus test enumerates directories and logs counts (`:148-157`); it does not assert cardinality. The 100/100 figure is verified by execution, not protected by a regression guard.

## 6. Regression coverage

| Guard | What it protects |
| --- | --- |
| `TestCrossLanguageReferenceCorpus` | 25 concepts × 4 languages, state + evidence + determinism |
| `TestResolverQualityAuditCorpus` | 33 single-language cases (Go 8, Python 8, JS 8, TS 9), TP/FP/FN accounting |
| `TestTSResolverOneLineSignatureDoesNotLeakBodyAsType` | WU4 `indexTopLevel` ordering |
| `TestTSResolverOneLineSignatureParamsIgnoreBody` | WU5 `matchingCloseParen` |
| `TestPythonReferenceResolverRelativeMissingModuleNotExternal` | WU5 relative-import classification |
| `TestPythonReferenceResolverComprehensionTargetNotModule` | WU5 second-round comprehension fix |
| 10 further WU5 tests (4 Python, 3 JS, 3 TS total) | Verified-correct conservative behaviour |

All 13 named tests exist in `internal/index/*_test.go` and pass.

## 7. Confirmed corrections (14)

| # | WU | Language | Defect | File |
| --- | --- | --- | --- | --- |
| 1 | WU1 | Python | Untyped local assignment shadowing resolved to an outer binding | `python_resolver.go` |
| 2 | WU1 | TypeScript | Parameter shadowing missing from value scope | `typescript_resolver_scan.go` |
| 3 | WU2 | Python | Local receiver shadowing a module binding | `python_resolver.go` |
| 4 | WU2 | Go | Local receiver shadowing an imported alias | `go_resolver.go` |
| 5 | WU2 | JavaScript | `resolveMemberCall` ignored a local receiver | `javascript_resolver.go` |
| 6 | WU2 | TypeScript | `const`/`let`/`var` missing from `valueNames`/`innerScopeShadowsValue` | `typescript_resolver_scan.go` |
| 7 | WU3 | Go | `localsAtPos` saw only the innermost scope's parameters | `go_resolver.go` |
| 8 | WU3 | Python | Unannotated parameters not treated as locals | `python_resolver.go` |
| 9 | WU3 | Go | Missing symbol in a known local package classified `external` | `go_resolver.go` |
| 10 | WU4 | TypeScript | `indexTopLevel` case ordering leaked one-line bodies into type space | `typescript_resolver_scan.go` |
| 11 | WU5 | Go | Captured locals resolved to package-level symbols | `go_resolver.go` |
| 12 | WU5 | TypeScript | `emitSignatureTypes` used the last `)` in the line | `typescript_resolver_scan.go` |
| 13 | WU5 | Python | Missing *relative* module classified `external` | `python_resolver.go` |
| 14 | WU5 | Python | Comprehension target resolved to a module-level function | `python_resolver.go` |

Root causes and failing-test-first evidence are recorded per WU in `odd/tasks/`. Corrections 1–10 and 11–13 are reported by their WU records; this closure re-verified that the code, the tests and the corpus expectations agree today.

## 8. Known limitations

**Go** — block scoping is flattened; `localsAtPos` merges all enclosing function scopes, so captured-local resolution is order-insensitive and deliberately conservative.

**Python** — comprehension targets are modelled line-locally, not as a full comprehension scope; multi-line and nested comprehension shadowing is not modelled; walrus targets are not modelled; wildcard imports never expand exports.

**JavaScript** — temporal visibility and initialization (TDZ) are not modelled; calls before a lexical declaration surface as `ambiguous` with candidates rather than as a precise TDZ error; `var` hoisting is preserved only as conservative abstention.

**TypeScript** — no type checker; interface/namespace declaration merging is unsupported and surfaces as `ambiguous` with all contributing declarations as candidates; `typeof value` type queries remain `unresolved`.

**All languages** — no cross-language resolution, no SSA, no interprocedural dataflow, no type inference.

## 9. Explicitly unsupported behaviour

These are decisions, not defects. They must stay unsupported until a task authorises them:

- Full comprehension scope model (multi-line, nested, walrus).
- TDZ / initialization-state modelling.
- TypeScript declaration merging.
- Wildcard import/export expansion.
- PHP reference resolution and PHP comparative fixtures.
- Cross-project or cross-repository resolution.
- Any global resolver, Tree-sitter migration or TypeScript Compiler API adoption.

Note: PHP reference resolution is outside the Phase 5.2.6 scope by approved decision (2026-11-14); the specification's five-language exit criterion remains a broader product goal. See §13.3.

## 10. Remaining risks

| Risk | Impact | Evidence |
| --- | --- | --- |
| Corpus cardinality is not asserted | A silently deleted concept would still pass | `cross_language_reference_audit_test.go:148-157` logs only |
| ADR 0018 over-claims PHP and four integrations | Consumers may trust capabilities that do not exist | `docs/decisions/0018-cross-language-references.md:13,22,27,153,156-160` vs code |
| Reference resolution is unreachable from CLI and MCP | The phase's product value is not delivered by any command | `RunResolveReferences` has no non-test caller |
| Persisted references are never read back | `index_references` is write-only in production | `LoadReferences`/`ListReferences` have no caller |
| WU2/WU3 deferrals are unaccepted | Closure criteria stay ambiguous | `-wu2.md:91-96`, `-wu3.md:109-113` |
| No ADR records the WU3/WU5 state-boundary decisions | `external` vs `unresolved`, comprehension targets and relative imports are only in work-unit notes | `docs/decisions/` stops at 0018 |
| `docs/tasks/phase-5.md` is stale | P5-05/P5-06 read as “pending, requires network” although `internal/impact` and the `impact` command exist | `docs/tasks/phase-5.md:159-160,202,210` |
| WU1–WU4 records are in Spanish | Violates the English-artifact convention; mixed status vocabulary (`COMPLETADA`/`PARCIAL`/`COMPLETE`) | `…consistency.md:171`, `-wu2.md:104`, `-wu3.md:121` |

`docs/architecture/overview.md:297` (“There is no index”) was examined and is **not** a defect: it describes the audit path, which still does not use the index.

## 11. Validation evidence

**Newly executed for this closure (2026-11-14):**

| Command | Result |
| --- | --- |
| `go test ./internal/index -run TestCrossLanguageReferenceCorpus -count=1 -v` | PASS — 100 subtests enumerated, 100 PASS |
| `go test ./internal/index -run TestResolverQualityAuditCorpus -count=1 -v` | PASS — 33 cases |
| `git status --short` | 52 entries |
| `git diff --cached --name-only` | 0 |
| `git rev-parse HEAD` | `a7f5096828ee599ecee273f5b60b6782bc75adfc` |

**Historical, recorded by WU5 and not re-executed here** (no code changed since): `go build ./...`, `go vet ./...`, `go test ./...`, `go test -race ./...`, `gofmt -l`, `git diff --check` — all reported PASS.

**Not verified:** compilation of the `ui` build tag (needs the Wails toolchain); runtime behaviour of any CLI/MCP command beyond the two test runs above.

## 12. Final phase status

**`PHASE PARTIALLY COMPLETE`**

Closing conditions, in order of cost:

1. Accept or formally defer the WU2/WU3 residual matrix in a written decision.
2. Correct ADR 0018 so its claims match the implementation (PHP, `ReferenceExtractor`, downstream integrations).
3. Decide the PHP position: implement a PHP reference resolver with positive/negative fixtures, or narrow the announced capability and the Phase 5 exit claim to four languages.
4. Optional but cheap: assert corpus cardinality in the evaluator so 100/100 becomes falsifiable.

The reference-consistency work itself is complete and green. The phase is not closed because its own documentation and its specification exit criterion are broader than what was delivered.

## 13. Approved dispositions (2026-11-14)

### 13.1 Work-unit disposition

| WU | Disposition | Notes |
| --- | --- | --- |
| WU1 | IMPLEMENTED / VERIFIED (audited scope) | 10 concepts 01–10 foundations, evaluator, 2 fixes |
| WU2 | PARTIALLY_COMPLETE / DEFERRED | concepts 11–17; 4 fixes |
| WU3 | PARTIALLY_COMPLETE / DEFERRED | 72 cases; 3 fixes |
| WU4 | IMPLEMENTED (demonstrated-defect scope) | 1 fix |
| WU5 | IMPLEMENTED (defined scope) | concepts 19–25; 100/100 |

WU2 and WU3 are marked DEFERRED, not COMPLETE.

### 13.2 WU2 and WU3 formal deferral

Both WU2 and WU3 remain `PARTIALLY_COMPLETE` and are now formally deferred (not silently complete). Residual audit areas, grouped:

- A. Closures/capture: closure capture semantics, nested parameter shadow cases beyond 11, local alias/init-to-other-symbol/limited-visibility forms; capture vs shadowing, captured var changes, post-declaration/hoisting-sensitive refs, multi-level capture chains, captured method/function usage, uncaptured outer refs in edge scopes.
- B. Typed receivers: known/inferred/annotated receiver success paths, inherited/overridden, instance-vs-static, property-vs-method/field-vs-method, imported receiver; declared/constructor/assigned-known/unknown receiver success/abstention matrix, inherited/overridden, static-vs-instance, property-vs-method/field-vs-method, imported receiver, reassignment/alias/chained members.
- C. Inheritance/overrides: inherited/overridden dispatch, homonym methods, base-type refs, `super`, partial hierarchies.
- D. Module collisions/external-vs-local: alias/relative/wildcard/namespace/duplicate local modules/external-vs-local collision/cycles/missing-local-module; local-vs-external name collision, truly external dependency with missing symbol, nonexistent module vs known package, relative-vs-absolute collision, builtin-vs-local, duplicate local modules.
- E. JS/TS cycles and reexports: named/wildcard/namespace reexport chains, multi-hop cycles, missing reexport target, two paths to same/different symbol, type-only/value/mixed reexport, circular partial resolution.
- F. TS type/value spaces: same-name across spaces, class as type+value, export type, namespace, wrong-space refs, extends/implements; class as type+value, same-name different spaces, type parameter shadowing, export type/reexport type-only, extends/implements breadth, wrong-space misuse, `typeof` in type context.
- G. Computed/dynamic dispatch: `obj["method"]()`, `registry[key]()`, `factory()()`, `getObject().method()`, chained/func-value forms.

Deferral rationale: the residual areas were not shown to contain a shared-invariant defect, and no consumer or task currently requires them; the closure audit accepts the deferral rather than fabricating completed work.

Reopening criteria: an authorized task names the residual area; a demonstrated shared-invariant defect in a residual area with failing-test-first evidence; a consumer requiring the capability; positive+negative fixtures per language claimed; full validation gates green.

Evidence required for closure: comparative corpus concepts per residual area, passing subtests, regression tests, `gofmt -l .` (empty), `go vet ./...`, `go test ./...`, `go test -race ./internal/index`, `git diff --check`, English record.

### 13.3 PHP scope clarification

Phase 5.2.6 comparative audit covers Go, Python, JavaScript and TypeScript only. PHP reference resolution is NOT IMPLEMENTED and is NOT part of Phase 5.2.6. PHP is a separate future development item with its own resolver implementation, language-specific fixtures, regression tests, quality audit and acceptance criteria. The broader five-language product vision (ADR 0018, spec Phase 5 exit criterion) is not equivalent to implemented resolver coverage. Do not imply five-language coverage.

### 13.4 ADR 0018 implementation-status correction

The four announced downstream integrations (`impact`, `api_change`, `architecture`, `claims`) remain PENDING / NOT IMPLEMENTED. `internal/impact` holds an `*index.Index` it never reads; its only production call site passes `nil`; `domain.Reference` has no non-test caller outside `internal/index` and `internal/storage`. ADR 0018's implementation-status clarification section (updated 2026-11-14) is the authoritative record.

### 13.5 Comparative corpus cardinality contract (Decision 6)

Baseline: 25 concepts × 4 languages = 100 cases. Required automated validation must detect: missing concepts; unexpected concepts; missing language fixtures; duplicate cases; invalid fixture data; unexpected total case counts. A result such as 90/90 must not be reported as equivalent to 100/100. Note that the current evaluator enumerates directories and logs counts (`internal/index/cross_language_reference_audit_test.go:148-157`) and does not assert cardinality; the contract is documented here for a separate authorized implementation task.

Acceptance criteria (future task): the test fails when a concept directory is removed; fails when an unexpected concept directory is added; fails when a language fixture is missing; fails on duplicate case identifiers; fails on invalid fixture data (schema/expect mismatch); fails when the total count differs from 100; passes at exactly 25 concepts × 4 languages = 100 cases. Do not modify tests or fixtures in this task.

### 13.6 Follow-up work not authorized by this document

List: PHP reference resolver; corpus cardinality assertion implementation; ACO Phase 0 and subsequent phases; context package types; freshness hashing/cache invalidation/incremental indexing; impact integration. State explicitly that these are future implementation tasks requiring separate authorization.

### 13.7 Effect on the closing conditions (§12)

Condition 1 formally deferred by approval; condition 2 corrected by ADR 0018 clarification; condition 3 narrowed to four languages by the approved PHP scope; condition 4 remains a future task with the contract in §13.5. The verdict remains `PHASE PARTIALLY COMPLETE`.
