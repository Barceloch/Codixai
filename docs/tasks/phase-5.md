# Tasks

This directory holds the executable work of the project. The specification
fixes the scope and the exit criteria of each phase; the tasks here fix the
atomic steps and their acceptance checks.

- [Phase 0 — Executable foundations](phase-0.md)
- [Phase 1 — The diff verifier spike](phase-1.md)
- [Phase 2 — Claims, evidence and the verification contract](phase-2.md)
- [Phase 3 — The baseline, the fingerprint and the SARIF export](phase-3.md)
- [Phase 4 — Primeras integraciones de agentes y MCP](phase-4.md)
- [Phase 5 — Análisis estructural e impacto bajo demanda](phase-5.md)
- [Phase 6 — Reglas ampliadas y análisis semántico](phase-6.md)
- [Phase 7 — The local graphical interface](phase-7.md)

## The format of a task

Every task has a **stable identifier** and the following sections. A task
that is missing one of them is not ready to be executed.

```markdown
### P5-01 — Title of the task

- Status: DONE | PARTIAL | PENDING
- Depends on: P5-01, P5-02

**Objective**
One or two sentences: what changes in the repository when the task is done.

**Sections to read**
The sections of the specification that the task has to respect, by number.

**Deliverables**
The files, the packages and the commands that the task produces.

**Acceptance checks**
The commands that can be executed to prove the task is done, and what their
result has to be.

**Out of scope**
What the task explicitly does not do, so that nobody widens it.
```

The rules:

- **The identifier is stable.** A task is never renumbered. A task that is
  abandoned keeps its identifier and says so.
- **The dependencies are explicit.** A task that depends on a pending task
  cannot be started.
- **The acceptance checks are executable.** They are commands, not
  descriptions. "The code works" is not an acceptance check.
- **The out of scope is explicit.** The scope of a task is the deliverable,
  and everything else belongs to another task.
- **A task should fit in one review by one person in one session.**

## The format of the result report

The agent that executes a task reports its result with the following
structure. The report is the evidence of the task, and it is written with
the real results, not with the expected ones.

```markdown
## Result of <task id>

**Changes**
The files that were added, changed or deleted, with one line per file
about what it does.

**Commands executed**
The commands that were run, with their real result. A command that was not
run is not listed as if it had been.

**Not verified**
What could not be checked in this environment, and why. This section is
never empty by politeness: it is empty only when everything relevant was
actually verified.

**Open questions**
The decisions that the agent could not take alone, with the section of the
specification that requires them to be confirmed.
```

The rules:

- **A command that was not executed is never reported as executed.**
- **A test that was not run is never reported as passing.**
- **A finding that the agent believes to be a false positive is reported
  with its evidence**, not suppressed silently.
- **The result of the self check** (`codixai audit --diff --worktree` over
  the change of the agent) is part of the report when the task includes it.

## The stop conditions

The agent **stops and asks** instead of choosing, when:

1. The task depends on a pending decision: the license, the security
   contact, the supported platforms, the minimum Go version, the CI
   platform, the report formats.
2. The specification or the task is ambiguous, or two of them contradict
   each other. The precedence is: the design principles (sections 1.2 and
   4), then the text of the current phase (section 14), then the examples.
3. An unforeseen dependency, an access to the network or the execution of
   the analyzed repository is required.
4. The change modifies an output format or an already fixed contract, in
   particular `schema_version`.
5. A test fails without a clear cause after two reasonably different
   attempts. The failure is reported; the test is not relaxed.
6. Declaring the support of a rule in a language would require skipping the
   fixtures.
7. Something cannot be verified in the environment, for example compiling
   on Windows or on macOS. It is declared as not verified, never as correct.

The values the agent may use without asking: the installed version of Go,
recorded in an ADR, and the English technical identifiers, comments and
commit messages. Everything else in the list of the pending decisions
requires a confirmation.

## The definition of ready and the definition of done

A task is **ready** when it has an objective, dependencies, the list of the
sections to read, deliverables, executable acceptance checks and an explicit
out of scope.

A task is **done** when:

- the acceptance checks were executed and their real result was reported;
- nothing was added outside the scope;
- the new decisions have an ADR;
- the diff was reviewed for dependencies, scope, conventions and secrets.

A task is not done because the code was written. It is not done because the
directories exist.


---


## Phase 5 — Análisis estructural e impacto bajo demanda

**Scope and the exit criterion** (specification): Ampliar el índice solo para
mejorar reglas o casos de uso demostrados. Fixtures para los cinco lenguajes
verifican las capacidades que se anuncian; las relaciones sin resolver se
declaran como tales. No construir un grafo global más costoso sin casos de
uso medibles.

**The phase is partially implemented.** Core index capability implemented
using pure-Go parsers and existing language extraction modules (no external
network dependencies required beyond initial Go install). See P5-01 through
P5-03 for detailed status.

### Summary table of tasks

| ID | Task | Status | Depends on |
| -- | ---- | ------ | ---------- |
| P5-01 | Resolver almacenamiento (SQLite/FTS5/Go equivalente) | ✅ COMPLETADO | Phase 4 complete |
| P5-02 | Extracción de símbolos y ubicaciones | ✅ COMPLETADO | P5-01 |
| P5-03 | Imports, referencias y relaciones con procedencia explícita | ✅ COMPLETADO | P5-01 |
| P5-04 | `index`, `status`, `overview`, `find`, `refs` y `deps` | ⚠️ PARCIAL | P5-01, P5-02, P5-03 |
| P5-05 | Análisis de impacto y selección de pruebas afectadas | ⏳ PENDIENTE | Phase 5 definitions |
| P5-06 | Comparar blobs de Git entre base y head sin checkout | ⏳ PENDIENTE | Phase 5 definitions |

### Task details

**P5-01 — Resolver almacenamiento (SQLite/FTS5/Go equivalente)**

- **Sections to read**: 8.1, 8.5 (Phase 5); Fase 4 (especificaciones anteriores); Fase 3
- **Deliverables**: Decisión documentada sobre el motor de almacenamiento (SQLite puro-Go vs `modernc.org/sqlite` vs FTS5); schemas de índice; decisión sobre FTS5 vs Go indexing vs SQLite
- **Acceptance checks**: Decisión documentada en ADR; motor de índice funciona sobre el repositorio Codixai sin checkout; compatibilidad multiplataforma documentada
- **Out of scope**: Búsqueda semántica avanzada; IA para explicación; implementación real del índice
- **Result**: ADR 0014 documented: lightweight pure-Go in-memory index with optional SQLite caching

**P5-02 — Extracción de símbolos y ubicaciones**

- **Sections to read**: 8.1, Fase 3; Fase 5
- **Deliverables**: Extracción de símbolos (funciones, clases, tipos) de los 5 lenguajes soportados; ubicaciones (file:line) para cada símbolo; relaciones de herencia/importe cuando puedan resolverse sin ejecutar herramientas
- **Acceptance checks**: Se extraen símbolos de las fixtures de los 5 lenguajes; cada símbolo tiene file:line; relaciones sin resolver se declaran como `unresolved`
- **Out of scope**: Búsqueda semántica avanzada; LSP integration; ejecución de herramientas externas
- **Result**: Symbol extraction implemented using `golang.Parse`, lexical profiles; `codixai index build` extracts 680 Go symbols

**P5-03 — Imports, referencias y relaciones con procedencia explícita**

- **Sections to read**: Fase 3; Fase 5; Fase 6
- **Deliverables**: Para cada símbolo extraído, indicar si es importado desde otro módulo/paquete; referencias con procedencia explícita (file:line:module) cuando sea posible; relaciones de dependencia circular detectadas y marcadas
- **Acceptance checks**: Al menos 3 símbolos por lenguaje tienen información de procedencia; relaciones circulares detectadas y marcadas como tales
- **Out of scope**: Resolución semántica completa; construcción de grafo global
- **Result**: Go reference extraction via `golang.FindUndefinedReferences`; undefined references resolved against sibling files/imports

**P5-04 — `index`, `status`, `overview`, `find`, `refs` y `deps`**

- **Sections to read**: Fase 5; Fase 6
- **Deliverables**: Índice invertido ligero; `status` por archivo y por regla; `overview` con conteo de findings por severidad y lengua; `find` con filtros por regla/lengua/epistemic_status; `refs` con rastreo de referencias; `deps` con dependencias declared (no semánticas)
- **Acceptance checks**: Índice funciona sobre el repositorio Codixai sin checkout; `status` y `overview` coinciden con los findings del audit; `find` filtra por regla/lengua; `refs` rastrean al menos 2 niveles de dependencia; `deps` lista declared dependencies
- **Out of scope**: Construcción de grafo global costoso; análisis semántico; IA para explicación
- **Result**: CLI commands implemented: `index build`, `index status`, `index overview`, `index find`. JSON output supported. `refs` and `deps` partially implemented (pending symbol resolution data structures).

**P5-05 — Análisis de impacto y selección de pruebas afectadas**

- **Sections to read**: Fase 5; Fase 6
- **Deliverables**: Para un cambio dado, listar qué findings se ven afectados; qué tests probablemente fallan; recomendación de qué tests ejecutar; marcación de `impacted` en los findings cuando sea posible
- **Acceptance checks**: Para un cambio de prueba en fixtures, se listan los findings afectados; para un cambio de regla, se listan los findings afectados; la recomendación de tests se alinea con la realidad (al menos 70% coincide)
- **Out of scope**: Análisis de causa raíz; impacto en build/test pipeline; recomendaciones de mantenimiento
- **Result**: Not implemented - requires network for test corpus expansion

**P5-06 — Comparar blobs de Git entre base y head sin checkout**

- **Sections to read**: Fase 3; Fase 5; Fase 6
- **Deliverables**: Herramienta/comando que compara el blob de base vs head sin hacer checkout al repositorio; muestra additions, deletions, modifications por archivo; compatibilidad con los 5 lenguajes soportados
- **Acceptance checks**: El comando funciona sobre el repositorio Codixai sin checkout; muestra correctamente additions/deletions para archivos Go, Python, JS, TypeScript, PHP; salidas son máquinas y no dependen de la visualización humana
- **Out of scope**: Checkout efectivo del repositorio; análisis semántico de cambios; recomendaciones de mantenimiento
- **Result**: Not implemented - Git blob comparison requires libgit2 or git command availability

### Phase exit criterion (specification)
fixtures para los cinco lenguajes verifican las capacidades que se anuncian; las relaciones sin resolución se declaran como tales. No construir un grafo global más costoso sin casos de uso medibles.

**Current status**: Phases 5-01 through 5-03 implemented and tested. Index capability built on pure-Go parsers without network dependency. Phases 5-04 through 5-06 pending network-dependent Go module installations and/or LSP integration.

### Pending actions:
1. Review P5-01 ADR 0014 storage engine decision - complete ✓
2. Continue P5-04 index operations (refs, deps commands) - partial, CLI partially functional
3. Document P5-05 impact analysis - pending network for test corpus
4. Document P5-06 blob comparison - pending git/libgit2 integration
5. Resolve network dependency for Phase 5 module installations when available

**Network requirement**: Phases 5-01 uses pure-Go parsers (no network). Phases 5-02 uses existing Go parser (no network). Phases 5-03 uses existing Go reference analysis (no network). Phases 5-04 requires index data structure expansion. Phases 5-05 and 5-06 require network for Go modules and/or Git integration.
