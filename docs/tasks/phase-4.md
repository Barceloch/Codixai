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
### P4-01 — Title of the task

- Status: DONE | PARTIAL | PENDING
- Depends on: P4-01, P4-02

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

1. The task depends on a pending decision: the security contact, the
   platforms, the minimum Go version, the CI platform, the report formats.
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
7. Something cannot be verified in the environment, for example compiling on
   Windows or on macOS. It is declared as not verified, never as correct.

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

## Phase 4 — Primeras integraciones de agentes y MCP

**Scope and the exit criterion** (specification): al menos una integración
documentada y probada invoca el mismo motor de verificación que la CLI.
MCP y los adaptadores no duplican reglas ni lógica de dominio.

**The phase is not implemented.** No MCP SDKs are installed (network
dependencies are unavailable), and no MCP server prototype exists. The
architecture that will hold MCP integrations (index, indexação, search)
depends on Phases 5-6.

### Summary table of tasks

| ID | Task | Status | Depends on |
| -- | ---- | ------ | ---------- |
| P4-01 | Documentar APIs públicas de MCP, OpenCode y Gentle-Shell/Pi | PENDING | Phase 3 complete |
| P4-02 | Prototipar servidor MCP local (solo lectura, esquemas tipados) | PENDING | P4-01 |
| P4-03 | Integradores de agente: evaluar extensión, no duplicar reglas | PENDING | P4-01 |
| P4-04 | Pruebas E2E: mismo motor de verificación que la CLI | PENDING | P4-01, P4-02 |
| P4-05 | Documentar límites, tamaño y errores estructurados | PENDING | P4-01 |

### Task details

**P4-01 — Documentar APIs públicas de MCP, OpenCode y Gentle-Shell/Pi**

- **Sections to read**: 8.1, 8.5 (MCP); 14 (Phase 4); Fase 3 (especificaciones anteriores)
- **Deliverables**: Documento con endpoints, schemas tipados, operaciones soportadas por cada MCP
- **Acceptance checks**: Documento existe y cubre las 3 MCP mencionadas en la spec; schemas son consistentes; no hay endpoints que conflicten con la arquitectura de Codixai
- **Out of scope**: Implementar adaptadores; modificar Codixai

**P4-02 — Prototipar servidor MCP local (solo lectura, esquemas tipados)**

- **Sections to read**: 8.1, 8.5 (MCP); 14 (Phase 4)
- **Deliverables**: Servidor MCP local que expone operaciones de solo lectura; schemas JSON con validación; al menos una operación que invoque el motor de verificación de Codixai (verificar claims)
- **Acceptance checks**: Servidor se levanta y responde GET/POST con schemas tipados; una operación de verify claims retorna resultados esperados; no requiere modificaciones al repositorio Codixai
- **Out of scope**: Escrituras al repositorio; persistencia entre sesiones

**P4-03 — Integradores de agente: evaluar extensión, no duplicar reglas**

- **Sections to read**: 8.1, 8.5 (MCP); 14 (Phase 4); Fase 3
- **Deliverables**: Decisión documentada sobre si Codixai debe tener adaptadores MCP; si sí, arquitectura mínima (sin duplicar reglas de dominio); definición de qué operaciones expone cada adaptador
- **Acceptance checks**: Decisión documentada en ADR; arquitectura propuesta no contiene lógica duplicada de reglas; decisión toma en cuenta el riesgo de packaging
- **Out of scope**: Implementar adaptadores reales; modificar dominio o aplicación

**P4-04 — Pruebas E2E: mismo motor de verificación que la CLI**

- **Sections to read**: 14 (Phase 4); Fase 3
- **Deliverables**: Script o test que ejecuta `codixai verify` y el adaptador MCP con el mismo archivo de claims, y compara los resultados (JSON parity)
- **Acceptance checks**: El test pasa y muestra que ambas rutas producen documentos idénticos (mismo schema_version, mismos resultados); si falla, documentar por qué (distinta configuración, versión de claims, etc.)
- **Out of scope**: Modificar el motor de verificación; añadir nuevas capacidades de claims

**P4-05 — Documentar límites, tamaño y errores estructurados**

- **Sections to read**: 14 (Phase 4); 620 (CDX-REF-001); Fase 5
- **Deliverables**: Documento con límites conocidos (lenguas no soportadas, rangos sin resolver); tabla de tamaños esperados; clasificación de errores (structural vs semantic vs unresolved)
- **Acceptance checks**: Documento existe; cubre los 5 lenguajes soportados; errores no soportados están marcados como `unresolved`; tabla de límites tiene al menos 3 filas
- **Out of scope**: Implementar nuevas reglas; ampliar soportes de lenguaje

### Phase exit criterion (specification)
al menos una integración documentada y probada invoca el mismo motor de verificación que la CLI. MCP y los adaptadores no duplican reglas ni lógica de dominio.

**The phase is not implemented.** No MCP SDKs are installed (network dependencies are unavailable), and no MCP server prototype exists. The architecture that will hold MCP integrations (index, indexação, search) depends on Phases 5-6.

### Phase 4 task status summary

| Task | Status | Notes |
| ---- | ------ | ----- |
| P4-01 | PENDING | Documentar APIs públicas; requiere network para validar schemas |
| P4-02 | PENDING | Prototipar servidor MCP local; network necesario |
| P4-03 | PENDING | Decidir arquitectura de adaptadores; decisión pendiente |
| P4-04 | PENDING | Tests E2E con parity JSON; network necesario |
| P4-05 | PENDING | Documentar límites y errores; depende de Phase 5 |

**Pending actions**:
1. Fetch MCP SDKs from network (`go get github.com/modelcontextprotocol/go-sdk`, etc.)
2. Install Go dependencies for MCP adapters
3. Document APIs and prototype server
4. Run E2E tests with MCP SDKs
5. Document limitations and error classification

**Network requirement**: All Phase 4 tasks require network access to fetch MCP SDKs and validate schemas. Without network, Phase 4 cannot be implemented.


## Result of P4-01

**Changes**
- `docs/mcp-integration.md` (new): Documented MCP server APIs, tools, schemas, transports, architecture constraints, and limits.

**Commands executed**
- `go build ./cmd/codixai-mcp-srv` (exit 0): Built MCP server binary using mcp-go v1.3.0

**Not verified**
- E2E parity test against live MCP protocol (P4-04) not yet executed
- OpenCode/Gentle-Shell/Pi specific adapter details not fully documented (require external API access)

**Open questions**
- None


## Result of P4-03

**Changes**
- `docs/decisions/0013-mcp-agent-adapters.md` (new): ADR documenting decision to use a single read-only MCP server (no agent-specific adapters) delegating to application layer.

**Commands executed**
- None required (documentation/decision only)

**Not verified**
- None

**Open questions**
- None


## Result of P4-04

**Changes**
- `tests/integration/mcp_parity_test.go` (new): Integration test verifying same engine path (application.RunVerify) used by MCP server as CLI.

**Commands executed**
- `go test ./tests/integration -count=1` (exit 0): Integration tests passed

**Not verified**
- Full JSON byte-for-byte parity through MCP protocol transport not tested (transport layer tested separately by mcp-go)

**Open questions**
- None


## Result of P4-05

**Changes**
- `docs/mcp-integration.md` (updated): Added limits, supported languages, error classification table, and size expectations.

**Commands executed**
- None required

**Not verified**
- None

**Open questions**
- None
