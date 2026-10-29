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
### P6-01 — Title of the task

- Status: DONE | PARTIAL | PENDING
- Depends on: P6-01, P6-02

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

## Phase 6 — Reglas ampliadas y análisis semántico

**Scope and the exit criterion** (specification): cada capacidad nueva tiene
fixtures, métricas, límites y justificación de valor frente al coste de
mantenimiento.

**The phase is not implemented.** No resolutores semánticos están instalados.
Las capacidades nuevas (detectores de API pública, similitud de código,
resolutores específicos por lenguaje, LSP, análisis de arquitectura, IA
opcional) dependen de las integraciones MCP de la Fase 4 y los Go modules
de la Fase 5. Sin red, la Fase 6 no puede ser implementada.

### Summary table of tasks

| ID | Task | Status | Depends on |
| -- | ---- | ------ | ---------- |
| P6-01 | Detectar cambios de API pública y revisar consumidores | PENDING | Phase 5 complete |
| P6-02 | Similitud de código para señalar posibles reimplementaciones | PENDING | Phase 5 complete |
| P6-03 | Resolutores específicos por lenguaje, herramientas de ecosistema y LSP | PENDING | Phase 5 complete |
| P6-04 | Análisis de arquitectura y convenciones configurables | PENDING | Phase 5 complete |
| P6-05 | IA opcional para explicación, basada en evidencia | PENDING | Phase 5 complete |

### Task details

**P6-01 — Detectar cambios de API pública y revisar consumidores**

- **Sections to read**: Fase 5; Fase 6; 8.5 (MCP); 14 (Phase 6)
- **Deliverables**: Detector de cambios de API pública que señale consumidores afectados; reporte de breaking changes; lista de consumidores que deben actualizarse
- **Acceptance checks**: El detector funciona sobre los 5 lenguajes soportados; identifica al menos 2 tipos de breaking changes (remoción de función, cambio de firma); reporte incluye impacto en consumidores
- **Out of scope**: Actualización automática de consumidores; análisis de causa raíz; mantenimiento de dependencias

**P6-02 — Similitud de código para señalar posibles reimplementaciones**

- **Sections to read**: Fase 6; Fase 5; Fase 3
- **Deliverables**: Detector de similitud de código que señale posibles reimplementaciones de utilidades existentes; reporte de similitud (%) y archivos involucrados; umbral configurable
- **Acceptance checks**: El detector funciona sobre los 5 lenguajes soportados; encuentra al menos 2 pares de funciones con similitud > 70%; reporte incluye el código fuente de ejemplo
- **Out of scope**: Refactorización automática; eliminación de código duplicado; mantenimiento de dependencias

**P6-03 — Resolutores específicos por lenguaje, herramientas de ecosistema y LSP**

- **Sections to read**: Fase 5; Fase 6; Fase 3; integraciones MCP
- **Deliverables**: Resolutores por lenguaje (Go, Python, PHP, JavaScript, TypeScript) que utilicen herramientas del ecosistema (gofmt, pylint, phpab, eslint, tsc); integración LSP donde esté disponible; reportes de errores y advertencias
- **Acceptance checks**: Al menos un resolutor por lenguaje esté funcional; las herramientas del ecosistema estén disponibles; los reportes incluyan errores y advertencias conocidas
- **Out of scope**: Configuración completa de LSP; actualización de dependencias del ecosistema; mantenimiento continuo

**P6-04 — Análisis de arquitectura y convenciones configurables**

- **Sections to read**: Fase 5; Fase 6; Fase 3
- **Deliverables**: Conjunto de reglas de análisis de arquitectura (acoplamiento, cohesión, dependencias cíclicas); conjunto de convenciones configurables (estilo de nombre, convenciones de importación); reporte de violaciones con ubicación y gravedad
- **Acceptance checks**: Las reglas de arquitectura funcionan sobre el repositorio Codixai; las convenciones configurables tienen al menos 3 opciones; las violaciones incluyen ubicación y gravedad
- **Out of scope**: Análisis semántico profundo; recomendaciones de mantenimiento a largo plazo

**P6-05 — IA opcional para explicación, basada en evidencia**

- **Sections to read**: Fase 6; Fase 5; Fase 3
- **Deliverables**: Componente IA opcional que proporcione explicaciones basadas en evidencia para los findings; separación clara entre resultados deterministas y explicaciones IA; integración con el flujo de trabajo existente
- **Acceptance checks**: El componente IA está marcado como opcional y no afecta el flujo determinista; las explicaciones son breves (máx. 2 oraciones); las explicaciones se basan en evidencia observable del código
- **Out of scope**: IA generativa libre; explicaciones sin evidencia; reemplazo del flujo determinista

### Phase exit criterion (specification)
cada capacidad nueva tiene fixtures, métricas, límites y justificación de valor frente al coste de mantenimiento.

**The phase is not implemented.** No resolutores semánticos están instalados. Las capacidades nuevas dependen de las integraciones MCP de la Fase 4 y los Go modules de la Fase 5. Sin red, la Fase 6 no puede ser implementada.

### Phase 6 task status summary

| Task | Status | Notes |
| ---- | ------ | ----- |
| P6-01 | PENDING | Detectar cambios API; requiere network |
| P6-02 | PENDING | Similitud de código; requiere network |
| P6-03 | PENDING | Resolutores por lenguaje; requiere network |
| P6-04 | PENDING | Análisis de arquitectura; requiere network |
| P6-05 | PENDING | IA opcional; requiere network |

**Pending actions**:
1. Fetch MCP SDKs and Go modules from network (Phases 4-5)
2. Install dependencies for semantic analysis capabilities
3. Implement API change detectors for 5 languages
4. Implement code similarity detectors
5. Implement language-specific resolvers with ecosystem tools
6. Implement architecture analysis rules
7. Implement optional IA explanation component

**Network requirement**: All Phase 6 tasks require network access to fetch MCP SDKs, Go modules from Phases 4-5, and documentation. Without network, Phase 6 cannot be implemented.

---
