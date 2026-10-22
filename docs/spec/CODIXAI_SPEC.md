# Codixai — Especificación técnica del proyecto

> **Estado:** especificación fundacional  
> **Versión:** 0.6.0  
> **Nombre del proyecto:** Codixai  
> **Nombre del ejecutable:** `codixai`  
> **Idioma principal de la documentación:** inglés técnico, con documentación inicial en español si ayuda a la comunidad objetivo  
> **Licencia:** MIT, confirmada por el mantenedor el 2026-10-20  
> **Objetivo:** construir una herramienta local-first para comprender, explorar y auditar repositorios de software, especialmente código creado o modificado con asistencia de IA.

---

## 0. Cómo usar esta especificación

Esta especificación está escrita para personas y para agentes de IA que implementan Codixai, en particular mediante OpenCode (ver sección 17).

**Lenguaje normativo.** "Debe" y "no debe" son obligatorios. "Debería" es una recomendación que solo se omite con una justificación documentada. "Puede" es opcional. Los bloques marcados como *ilustrativos* (ejemplos JSON o YAML, IDs de reglas de ejemplo) no son normativos: muestran la intención, no el contrato final.

**Precedencia ante contradicciones.** (1) Prevalecen los principios de las secciones 1.2 y 4. (2) Después, el texto de la fase en curso en la sección 14. (3) Después, los ejemplos. Un agente que detecte una contradicción no debe resolverla en silencio: debe detenerse y preguntar (sección 17.4).

**Alcance de cada trabajo.** Esta especificación define el producto. Lo que se implementa en cada momento lo fija la tarea activa en `docs/tasks/`. Todo lo que no figure en la tarea activa queda fuera de alcance, aunque aparezca aquí como objetivo posterior.

**Glosario.**

| Término | Significado |
|---|---|
| Hallazgo (`Finding`) | Resultado de una regla sobre un cambio o repositorio, con ubicación, evidencia y estado epistemológico. |
| Evidencia (`Evidence`) | Referencia verificable (archivo y rango, hunk del diff, regla aplicada, informe aportado) que sustenta un hallazgo o el resultado de un claim. |
| `epistemic_status` | Estado de un **hallazgo**: `observed` (hecho comprobado directamente en el código o diff), `inferred` (deducido por heurística o análisis parcial) o `unknown` (no se pudo determinar). |
| `resolution` | Estado de una **relación** entre símbolos o archivos: `confirmed`, `inferred` o `unresolved`. No se usa en hallazgos. |
| Claim | Afirmación explícita de un agente o persona sobre un cambio (por ejemplo, "no se tocó el módulo X"). Su resultado es `verified`, `contradicted` o `unresolved`. |
| Regla (`Rule`) | Comprobación determinista con ID estable `CDX-<ÁREA>-<NNN>`, versión y matriz de soporte por lenguaje. |
| Nivel de soporte | Por regla y lenguaje: `none`, `syntactic` (equivale al Nivel 1 de la sección 5) o `resolved` (Nivel 2 o superior). |
| Base, head, worktree | Orígenes de un diff: commits entre una base y `HEAD`, área de preparación (`--staged`) o cambios sin commit (`--worktree`). |
| Fixture | Caso de prueba versionado con `base/`, `head/` y resultado esperado (sección 15). |
| ADR | Registro de decisión arquitectónica en `docs/decisions/`. |

---

## 1. Visión

Codixai es una herramienta de inteligencia de código cuyo primer punto de entrada será una CLI multiplataforma escrita en Go. Debe ayudar a los desarrolladores a:

- encontrar rápidamente archivos, símbolos, referencias y puntos de entrada;
- comprender cómo se conectan los componentes y cómo fluye una operación por el sistema;
- definir y comprobar convenciones y límites arquitectónicos;
- revisar cambios propuestos por agentes de IA;
- obtener resultados verificables, enlazados al código real;
- integrar estas capacidades en agentes como OpenCode y Gentle-Shell;
- evolucionar posteriormente hacia una interfaz gráfica local, integrada en el mismo código y lanzable con `codixai ui`, con mapas, grafos y diagramas interactivos.

**Principio rector:** Codixai no es un chatbot que adivina cómo funciona un repositorio ni un simple generador de grafos. Es un verificador local-first del trabajo realizado por agentes de IA: contrasta diffs y afirmaciones con evidencia reproducible, señala riesgos y deja claro qué pudo comprobar y qué sigue sin resolver. El análisis estructural es una capacidad habilitadora que se amplía según las necesidades verificables del producto.

### 1.1 Diferenciación del producto

Codixai no debe convertirse en otro explorador de archivos, generador de mapas de dependencias ni wrapper de herramientas de análisis estático. Su propuesta de valor es **inteligencia verificable del código y supervisión de cambios realizados por agentes de IA**.

Los pilares de diferenciación son:

1. **Mapa de evidencia:** los hallazgos deben apuntar a archivos, líneas, símbolos, relaciones y reglas que puedan verificarse. Cada resultado debe distinguir hechos observados, inferencias y preguntas pendientes.
2. **Análisis de impacto de cambios:** además de resumir un diff, identificar símbolos y módulos afectados mediante relaciones conocidas, explicar por qué se consideran afectados y sugerir verificaciones o pruebas relevantes. No afirmar impactos que el análisis no pueda justificar.
3. **Contexto compacto para agentes:** ofrecer resultados estructurados, limitados y pertinentes que permitan a agentes como OpenCode y Gentle-Shell consultar el repositorio sin leerlo indiscriminadamente. El contexto debe incluir referencias estables a la evidencia.
4. **Verificación posterior del trabajo de agentes:** comparar el estado real del repositorio, el diff y los informes de verificación aportados (por ejemplo, resultados de pruebas generados fuera de Codixai y vinculados a una revisión concreta) con las afirmaciones o criterios de aceptación proporcionados. Codixai no ejecuta pruebas ni código del repositorio y no debe declarar que un cambio es correcto solo porque compila, porque un informe lo diga o porque un agente lo afirme.
5. **Memoria estructural versionada:** permitir comparar índices entre revisiones para explicar cómo cambió la estructura conocida. Los resultados deben identificar la revisión/base usada y la frescura del índice. Es una capacidad posterior (Fase 5) y no bloquea el primer hito: mientras tanto, el análisis de diff compara blobs de Git sin indexar el repositorio completo.
6. **Local-first y privacidad por defecto:** el análisis básico funciona sin cuenta, servicio remoto ni modelo de IA. El código no se envía a servicios externos por defecto.

La ventaja competitiva debe surgir de la combinación de evidencia, impacto, contexto de agentes y verificación; no de contar comandos, lenguajes soportados o efectos visuales. La especificación y el README deben comunicar esta propuesta con ejemplos concretos.

### 1.2 Principios de confianza

- Cada hallazgo debe incluir nivel de confianza o clasificación apropiada cuando sea significativo, junto con la evidencia que lo sustenta.
- Diferenciar explícitamente `observed`, `inferred` y `unknown` (o equivalentes definidos en el esquema de salida).
- Mostrar la revisión del repositorio, base del diff, versión del analizador y nivel de soporte del lenguaje cuando aplique.
- La ausencia de hallazgos no equivale a demostrar que el código es seguro o correcto.
- Nunca inventar referencias, relaciones, pruebas ejecutadas ni porcentajes de cobertura del análisis.

## 2. Objetivos y no objetivos

### 2.1 Objetivos del MVP

1. Ser instalable, ejecutable y útil localmente desde la primera versión pública.
2. Analizar repositorios con código Go, Python, PHP, JavaScript y TypeScript.
3. Indexar estructura sintáctica y metadatos sin ejecutar el código del proyecto.
4. Verificar cambios contra una base Git (commits, área de preparación y árbol de trabajo) y, cuando se proporcione, contrastarlos con afirmaciones explícitas del agente.
5. Ejecutar reglas deterministas para detectar señales como imports inexistentes, tests debilitados, supresiones nuevas, errores ignorados y placeholders.
6. Proporcionar evidencia, limitaciones y clasificación epistemológica en cada hallazgo.
7. Exportar JSON estable y preparar SARIF, baseline y uso en CI.
8. Mantener los cinco lenguajes objetivo desde el inicio: Go, Python, PHP, JavaScript y TypeScript, con capacidades explícitas por regla y lenguaje.
9. Ofrecer exploración estructural (símbolos, referencias y dependencias) según la necesite el verificador.
10. Ofrecer documentación, ejemplos, fixtures y pruebas para que la comunidad pueda contribuir.
11. Mantener las capas internas independientes de la CLI, MCP y cualquier interfaz futura.

### 2.2 Objetivos posteriores

- Integración MCP sobre los mismos casos de uso de dominio, después de estabilizar sus contratos.
- Integración de primera clase con OpenCode y Gentle-Shell mediante mecanismos públicos verificados.
- Salida SARIF, baseline de hallazgos y flujos CI.
- Selección probable de pruebas según el impacto de cambios.
- Interfaz gráfica local integrada en el mismo código (Wails, Svelte y daisyUI), lanzable con `codixai ui` (sección 8).
- Servidor local/API para clientes externos, solo si un caso de uso lo justifica.
- Visualización de grafos de dependencias, mapas de módulos y flujos de ejecución.
- Resolución semántica ampliada mediante LSP y herramientas propias de cada lenguaje.
- Explicaciones opcionales con proveedores de IA locales o remotos.
- Integración CI y políticas compartidas por equipos.

### 2.3 No objetivos del MVP

- No construir todavía la interfaz gráfica (sección 8, Fase 7); sí mantener desde el inicio la arquitectura que permita añadirla sin reescribir el análisis.
- No modificar ni refactorizar automáticamente el repositorio analizado.
- No afirmar que el análisis estático reconstruye todos los flujos dinámicos.
- No exigir una API, cuenta remota, base de datos de servidor o modelo de IA.
- No enviar código a servicios externos de manera predeterminada.
- No soportar inicialmente todos los frameworks ni todas las construcciones dinámicas de los cinco lenguajes.
- No introducir microservicios, un grafo distribuido ni una base vectorial sin una necesidad validada.

## 3. Usuarios y casos de uso principales

### 3.1 Desarrollador que hereda un repositorio

Quiere identificar lenguajes, estructura, puntos de entrada, símbolos y dependencias antes de cambiar código.

### 3.2 Desarrollador que usa agentes de IA

Quiere comprobar qué cambió un agente, si incumplió reglas arquitectónicas y qué partes del sistema podrían verse afectadas.

### 3.3 Agente de programación

Quiere consultar símbolos, dependencias, convenciones y hallazgos mediante comandos reproducibles y JSON legible por máquina.

### 3.4 Contribuidor comunitario

Quiere añadir un lenguaje, regla, adaptador, documentación o prueba sin necesitar acceso a servicios privados ni a una infraestructura central.

## 4. Principios de diseño no negociables

1. **Local-first:** indexación y análisis básicos funcionan sin conexión a internet.
2. **Sin ejecución de código analizado:** indexar o auditar no ejecuta scripts, hooks, tests ni aplicaciones del repositorio.
3. **Evidencias:** cada hallazgo debe incluir la regla aplicada y referencias al archivo y rango de líneas cuando estén disponibles.
4. **Incertidumbre explícita:** distinguir relaciones confirmadas, inferidas y desconocidas.
5. **Determinismo primero:** las reglas automatizables no dependen de un LLM.
6. **IA opcional:** cualquier función de IA se añade mediante adaptadores y declara qué contexto puede enviar.
7. **Solo lectura por defecto:** los comandos de inspección no alteran los archivos fuente.
8. **Reproducibilidad:** la configuración, versiones de reglas y formato de salida deben ser identificables.
9. **Incrementalidad:** los cambios deben invalidar solo los elementos y relaciones afectados cuando sea viable.
10. **Extensibilidad sin complejidad prematura:** los contratos deben permitir nuevas interfaces, sin implementar ahora sistemas que todavía no hacen falta.
11. **Privacidad por defecto:** secretos y archivos excluidos no deben entrar en prompts, informes ni registros.
12. **Comunidad primero:** instrucciones de contribución, pruebas reproducibles, decisiones arquitectónicas documentadas y tareas iniciales claras.

## 5. Lenguajes y alcance del análisis

Codixai debe reconocer desde el inicio:

- Go
- Python
- PHP
- JavaScript
- TypeScript

El soporte se divide en niveles, y la CLI debe comunicar qué nivel se ha completado para cada lenguaje.

### Nivel 1 — estructura sintáctica

- detectar lenguaje y archivos;
- extraer símbolos disponibles: módulos, clases, interfaces, funciones y métodos;
- conservar rangos de líneas y columnas;
- extraer imports y declaraciones estructurales;
- informar errores de parseo sin detener la indexación del resto del repositorio.

### Nivel 2 — relaciones estáticas básicas

- resolver imports locales cuando sea posible;
- construir relaciones de dependencia;
- identificar referencias y llamadas solo cuando el análisis pueda justificarlas;
- clasificar las relaciones por procedencia y confianza.

### Nivel 3 — análisis semántico específico

Se incorpora por adaptador y fase. Go puede usar `go/parser`, `go/ast`, `go/types` y `go/packages`; los otros lenguajes pueden usar Tree-sitter, LSP, analizadores de ecosistema y herramientas externas. No exigir el mismo nivel semántico a todos los lenguajes.

La documentación y la salida deben indicar las limitaciones. La ausencia de una relación en el índice no demuestra que no exista en ejecución.

## 6. Stack técnico inicial

### 6.1 Núcleo

- **Go:** lenguaje principal.
- **Cobra:** CLI y subcomandos.
- **Lip Gloss (Charm):** estilos, colores, bordes, alineación y jerarquía visual consistente en terminal.
- **Glamour (Charm):** renderizado de Markdown y explicaciones en terminal.
- **Bubbles (Charm):** componentes de progreso y spinner cuando aporten valor; incorporar solo los componentes necesarios.
- **Chroma:** candidato para resaltado de sintaxis de fragmentos de código; validar compatibilidad, mantenimiento y licencias antes de fijar versión.
- **Bubble Tea (Charm):** reservado para una futura interfaz de terminal interactiva (TUI); no es requisito para el MVP inicial.
- **Wails, Svelte, Tailwind CSS y daisyUI (Fase 7):** interfaz gráfica local opcional en el mismo código. Wails aporta la ventana nativa y el enlace entre Go y el frontend; Svelte, Tailwind y daisyUI se compilan en tiempo de build y se incrustan como activos estáticos. Ni Node.js ni estas dependencias son necesarios para compilar o ejecutar la CLI (sección 8).
- **SQLite:** índice local persistente. No es necesario para `audit --diff` en las Fases 1 a 4; su decisión puede diferirse hasta antes de la Fase 5.
- **FTS5:** búsqueda textual, si está disponible en el driver SQLite elegido.
- **Tree-sitter:** parsing multilenguaje con gramáticas mantenidas y licencias revisadas. Está en la ruta crítica de la Fase 1 para Python, PHP, JavaScript y TypeScript (Go usa su toolchain estándar), por lo que su estrategia de integración (CGO o WASM) se decide en la Fase 0.
- **Bibliotecas estándar de Go:** concurrencia, sistema de archivos, JSON, contexto y pruebas.
- **Git CLI:** detección de ramas, estado y diffs; sin ejecutar hooks del repositorio.

Dependencias concretas, versiones y licencias se verifican durante la fase de implementación y se fijan en el lock/sum correspondiente. Evitar dependencias innecesarias.

### 6.2 Analizadores complementarios

- `go/parser`, `go/ast`, `go/types`, `go/packages` para análisis semántico de Go.
- Tree-sitter para el análisis estructural común.
- ast-grep y Semgrep como integraciones opcionales para reglas adicionales.
- LSP como capacidad posterior para referencias semánticas de los lenguajes compatibles.

Las herramientas externas deben detectarse como opcionales, informar su disponibilidad y fallar con mensajes útiles. El MVP básico no debe requerir que estén instaladas.

### 6.3 Almacenamiento

Usar SQLite como índice interno de Codixai, no como base de datos de las aplicaciones analizadas.

Persistir al menos:

- metadatos del repositorio e índice;
- archivos y hashes;
- símbolos y ubicaciones;
- relaciones y procedencia;
- diagnósticos de análisis;
- reglas y resultados de auditoría;
- versión del esquema y versión de Codixai que creó el índice.

No almacenar copias completas de los archivos si no es necesario. El índice debe poder reconstruirse a partir del repositorio.

Elegir el driver SQLite tras verificar compatibilidad multiplataforma, CGO y FTS5. Probar específicamente en Linux; no afirmar portabilidad hasta validar Windows y macOS.

El baseline de hallazgos (Fase 3) es un archivo versionado que vive en el repositorio analizado y puede compartirse en CI; no forma parte del índice SQLite, que es una caché local reconstruible.

## 7. Arquitectura interna

Separar el dominio y los casos de uso de la CLI y de los proveedores concretos.

```text
cmd/codixai/
internal/
  domain/
  application/
    index/
    search/
    explore/
    trace/
    audit/
    report/
  analysis/
    parser/
    extractor/
    resolver/
    graph/
  languages/
    registry/
    go/
    python/
    php/
    javascript/
    typescript/
  policies/
  retrieval/
  storage/
  integrations/
    git/
    astgrep/
    semgrep/
    lsp/
  ai/
  cli/
  ui/                   # adaptador Wails; solo con la etiqueta de build `ui` (Fase 7)
ui/
  frontend/             # Svelte + Vite + Tailwind + daisyUI (Fase 7)
api/
  openapi/              # solo si se decide una API HTTP externa (Fase 7.4); no es necesaria para la UI
testdata/
  go/
  python/
  php/
  javascript/
  typescript/
tests/
  integration/
  acceptance/
docs/
  architecture/
  decisions/
  guides/
```

Esta estructura es una guía modular, no una orden de crear paquetes vacíos. Crear un paquete solo cuando tenga responsabilidad real.

### 7.1 Dirección de dependencias

- `domain` no importa CLI, almacenamiento, frameworks, proveedores de IA ni herramientas externas.
- `application` implementa casos de uso mediante contratos e interfaces.
- `analysis` y `languages` producen información estructurada.
- `storage` implementa persistencia mediante los contratos requeridos.
- `cli` convierte argumentos en llamadas a casos de uso y representa los resultados.
- La UI (Wails), la CLI, el servidor MCP y cualquier API futura llaman a los mismos casos de uso de `application`, sin reimplementar el análisis. `domain` y `application` no importan CLI, Wails ni código de UI.
- Los adaptadores de IA nunca son requisito para buscar, indexar o auditar reglas deterministas.

### 7.2 Modelo común

Definir modelos de dominio estables, con identificadores y ubicaciones explícitas:

- `Repository`
- `File`
- `Symbol`
- `Relation`
- `Diagnostic`
- `Rule`
- `Finding`
- `AnalysisRun`

Cada `Relation` debe registrar su tipo, origen, destino cuando se conozca, evidencia y método de resolución. La confianza no debe fingir una precisión matemática si solo es una heurística; preferir estados categóricos como `confirmed`, `inferred` y `unresolved`.

Los hallazgos deben contener identificador de regla, severidad, explicación, archivo/rango, evidencia, estado epistemológico (`epistemic_status`: `observed`, `inferred` o `unknown`) y, opcionalmente, recomendación.

Ejemplo ilustrativo (el esquema definitivo se fija en la Fase 0 y se versiona con `schema_version`):

```json
{
  "schema_version": "0.1",
  "rule_id": "CDX-TST-001",
  "rule_version": "1",
  "severity": "warning",
  "epistemic_status": "inferred",
  "language": "go",
  "message": "Se eliminó una aserción de un test existente.",
  "location": { "path": "internal/payment/service_test.go", "start_line": 41, "end_line": 41 },
  "revision": { "base": "<base>", "head": "<head>" },
  "evidence": [ { "kind": "diff_hunk", "ref": "hunk:3" } ],
  "support": { "level": "syntactic", "limitations": ["no se comprobó si otra aserción cubre el mismo comportamiento"] },
  "recommendation": "Confirmar que la cobertura equivalente sigue existiendo."
}
```

## 8. Interfaz gráfica local (Wails, Svelte y daisyUI)

Codixai incluirá una interfaz gráfica local opcional, en el mismo código y, en su variante con UI, en el mismo binario que la CLI. Se lanza con `codixai ui` (alias: `codixai --ui`). La UI no se construye en el MVP (Fase 7), pero la arquitectura debe permitirla desde la Fase 0: ese es el objetivo de esta sección.

### 8.1 Qué es y qué no es Wails

Wails construye aplicaciones de escritorio con backend en Go y frontend web renderizado en el webview del sistema (WebView2 en Windows, WebKit en macOS y Linux), sin incluir un navegador completo. **No es un servidor web:** `codixai ui` abre una ventana nativa y no escucha en ningún puerto. Consecuencias que deben asumirse:

- el modo de escritorio depende de bibliotecas nativas del sistema y, según la plataforma, de CGO (confirmar por plataforma en el ADR 7.0);
- no funciona sin entorno gráfico (SSH, contenedores, CI), por lo que la CLI sigue siendo la interfaz principal para esos entornos;
- la compilación cruzada del modo de escritorio es limitada: se compila en CI por plataforma (verificar en el ADR 7.0);
- Wails tiene dos versiones mayores con APIs incompatibles. En la documentación consultada, v2 es la versión estable y v3 está en beta, con servicios enlazados tipados y un modo servidor (sin ventanas nativas ni CGO) que sirve el frontend por HTTP. La elección es la decisión pendiente 11.

### 8.2 Variantes de build

| Variante | Compilación | Requisitos | Uso |
|---|---|---|---|
| Solo CLI (predeterminada) | `go build ./cmd/codixai` | Toolchain de Go y lo que exija el parsing (ADR de la Fase 0). Sin Node.js, sin Wails, sin webview. | Terminales, CI, contenedores, `go install` |
| Con UI | Compilar el frontend y luego `go build` con la etiqueta `ui` (o la herramienta de Wails; se fija en el ADR 7.0) | Node.js solo en build; bibliotecas nativas del webview en ejecución | Uso de escritorio |
| Servidor local (opcional) | Etiquetas definidas en el ADR 7.0 | Sin ventana nativa | Acceso desde navegador en loopback |

La variante con UI contiene toda la CLI. En la variante solo-CLI, `codixai ui` termina con un error de uso claro que explica cómo obtener la variante con UI. El código de UI vive tras etiquetas de build para que la variante solo-CLI no importe Wails ni requiera los activos compilados del frontend. Como `go install` no puede compilar el frontend, produce siempre la variante solo-CLI.

### 8.3 Arquitectura

La dirección de dependencias es `ui → application → domain`, nunca al revés.

```text
internal/ui/        # adaptador Wails (etiqueta ui): arranque, ventana y servicios enlazados
ui/frontend/        # Svelte + Vite + Tailwind + daisyUI; se compila a ui/frontend/dist
ui/embed.go         # incrusta los activos compilados (etiqueta ui)
```

- **Servicios enlazados finos:** traducen entre la UI y los casos de uso. No contienen reglas ni lógica de análisis.
- **Un único contrato:** los servicios devuelven los mismos DTO versionados (`schema_version`) que `--format json`. Los tipos de TypeScript se generan desde Go y no se duplican a mano. Una prueba automatizada comprueba que, para el mismo cambio, la UI y la CLI muestran los mismos resultados.
- **Operaciones largas como trabajos:** iniciar una operación devuelve un `job_id`; el progreso y el resultado llegan por eventos; son cancelables mediante `context` y no bloquean la ventana. El progreso representa fases reales, como en la CLI.
- **Límites, filtros y paginación** en toda consulta. No devolver grafos ilimitados por defecto.
- **Estado:** la UI no guarda resultados de análisis propios; solo preferencias (tema, idioma, repositorios recientes) en el directorio de configuración del usuario. Sin telemetría.
- **Solo lectura:** la UI nunca modifica el repositorio analizado; solo escribe exportaciones en rutas que la persona elige.

Operaciones conceptuales que la UI debe poder exponer, según la fase en que existan sus casos de uso: resumen del repositorio, auditoría de diffs, hallazgos y evidencias, verificación de claims, búsqueda de símbolos, dependencias entrantes y salientes, subgrafos acotados, flujos con evidencias y exportación de informes.

### 8.4 Frontend: Svelte, Tailwind y daisyUI

- **SPA estática** con Svelte, Vite y TypeScript, sin renderizado en servidor. Fijar versiones exactas al comenzar la Fase 7 y consultar la documentación vigente de Svelte, Tailwind y daisyUI, porque su configuración cambia entre versiones mayores; no asumirla de memoria.
- **daisyUI** como plugin de Tailwind en tiempo de build. Tema claro y oscuro según el sistema, con alternancia manual, y colores semánticos para severidad.
- **Accesibilidad:** la información no se transmite solo con color (severidad y estado epistemológico llevan texto e icono); navegación completa por teclado con foco visible; roles y etiquetas ARIA en tablas, diálogos y pestañas; contraste suficiente; respeto de `prefers-reduced-motion`. Es el equivalente de `--no-color` y `--plain` en la CLI.
- **Textos** en un catálogo de mensajes desde el primer componente, preparados para español e inglés (decisión pendiente 14).
- **Sin recursos remotos:** fuentes, iconos y estilos incluidos localmente; sin CDN.
- **Dependencias npm:** mínimas, con lockfile, licencias revisadas e instalación con `npm ci` en CI. Node.js solo se necesita en build.
- **Pruebas:** componentes y flujos principales, comprobaciones básicas de accesibilidad y contrato con los DTO de Go.

### 8.5 Seguridad de la interfaz

- **Activos incrustados y sin red:** sin peticiones externas, CSP restrictiva, sin `eval` ni scripts en línea y sin navegación a orígenes externos. Los enlaces externos se abren en el navegador del sistema desde Go, con una lista de destinos permitidos. Las herramientas de desarrollo se deshabilitan en los releases (verificar la opción en la versión de Wails elegida).
- **Contenido del repositorio no confiable:** fragmentos de código, rutas, mensajes de commit y textos de reglas se muestran siempre como texto. Está prohibido `{@html}` y construir HTML con ellos; el resaltado de sintaxis se genera a partir de tokens escapados. Es el mismo principio que protege a la CLI y a los agentes de la inyección de instrucciones.
- **Rutas acotadas:** la UI nunca lee archivos arbitrarios. Todo acceso pasa por servicios de Go restringidos a la raíz seleccionada, con canonicalización y rechazo de escapes por enlaces simbólicos, igual que la CLI. Los archivos fuera de la raíz (claims, informes) solo se abren mediante un diálogo nativo elegido por la persona.
- **Modo servidor (si se adopta):** loopback por defecto y puerto aleatorio, token por sesión, validación de `Host` y `Origin` (contra peticiones de otros sitios y DNS rebinding), sin CORS abierto, solo lectura y exposición a otras interfaces únicamente con una opción explícita y una advertencia.
- **Sin telemetría ni envío de código.**

### 8.6 Si la UI no puede abrirse

`codixai ui` debe fallar con un mensaje accionable (sin entorno gráfico, webview no disponible, variante sin UI) e indicar alternativas (`--format json`, informe SARIF, modo servidor si existe). Nunca debe quedarse colgado.

### 8.7 Requisitos que se cumplen desde la Fase 0

1. `application` no depende de CLI ni de UI, y una prueba de arquitectura lo verifica.
2. Los resultados son DTO versionados y serializables.
3. Las operaciones largas reciben `context` y reportan progreso.
4. Todo acceso a archivos está acotado a la raíz seleccionada.
5. El código específico de UI queda tras etiquetas de build.

## 9. Interfaz CLI inicial

Los siguientes comandos son el contrato evolutivo del producto. Priorizar primero la verificación de diffs y claims; implementar cada comando solo cuando exista una capacidad real detrás. El índice general no debe bloquear la primera utilidad pública.

```bash
codixai version
codixai init
codixai index
codixai status
codixai overview
codixai find "PaymentService"
codixai refs "PaymentService.Create"
codixai deps ./internal/payment
codixai audit
codixai audit --diff --base main
codixai audit --diff --staged
codixai audit --diff --worktree
codixai verify --base main --claims claims.json
codixai baseline create
codixai report --format json
codixai report --format sarif
codixai ui --root .
```

### Experiencia visual de terminal

La CLI es una interfaz de producto, no una salida de depuración. Desde los primeros comandos debe tener una presentación consistente, accesible y legible, sin obligar al usuario a usar una TUI interactiva.

**Prioridad de implementación:** en la Fase 1 basta el mínimo (estilos centralizados, `--no-color`, `NO_COLOR`, `--plain`, JSON limpio y errores accionables). Glamour, Chroma y Bubbles se incorporan cuando aporten valor medible y no bloquean el primer hito.

- Definir una paleta y un sistema de estilos centralizados, con identidad tecnológica sobria: fondo no controlado por la herramienta, texto principal de alto contraste y acentos cian/violeta/verde para categorías semánticas. No codificar colores en cada comando.
- Usar Lip Gloss para estilos y disposición; Glamour para Markdown; evaluar Chroma para fragmentos de código y Bubbles para barras/spinners. Verificar APIs, versiones, licencias y compatibilidad multiplataforma antes de fijar dependencias.
- Mostrar tablas, árboles, etiquetas de estado, iconos o símbolos con alternativas textuales y mensajes de error accionables. No sacrificar legibilidad por decoración.
- `index` debe mostrar fases reales del trabajo (por ejemplo: descubrir archivos, analizar sintaxis, extraer símbolos, guardar índice). Usar barra porcentual solo si el total y el avance son medibles; en caso contrario, usar spinner y fase actual. Nunca simular progreso.
- El progreso debe finalizar limpiamente, respetar cancelación/interrupción y no dejar líneas de terminal corruptas.
- Resaltar sintaxis solo cuando el formato de salida sea terminal interactiva y el contenido sea código. El resaltado es una capa de presentación, nunca parte del resultado de análisis.
- Centralizar la presentación en un paquete interno (por ejemplo, `internal/cli/ui`) separado de los casos de uso y del modelo de dominio. Los resultados de análisis deben poder renderizarse como texto, salida enriquecida o JSON sin duplicar lógica.
- Añadir pruebas para renderizado sin color, salida redirigida, terminal estrecha y errores. Las pruebas no deben depender de capturas frágiles de secuencias ANSI exactas cuando pueda verificarse el contenido semántico.

### Comportamiento

- La ruta del proyecto es opcional cuando se ejecuta desde dentro del repositorio; aceptar `--root` para especificarla.
- Proporcionar `--format text|json` donde corresponda.
- Mantener JSON limpio en stdout; diagnósticos y progreso van a stderr.
- Ofrecer `--no-color`, respetar `NO_COLOR` y detectar terminales sin soporte de color.
- Soportar `--plain` para una salida simple sin adornos, animaciones ni barras; mantener mensajes y estructura legibles.
- No mezclar progreso ni diagnósticos con JSON en stdout: progreso y diagnósticos van a stderr; stdout contiene únicamente el documento solicitado.
- Al redirigir stdout o ejecutarse en CI/entornos no interactivos, desactivar automáticamente animaciones, secuencias ANSI y progreso dinámico, salvo que se solicite explícitamente un formato compatible.
- La presentación enriquecida no debe ser necesaria para comprender el resultado: estados, severidades y errores deben tener una representación textual clara.
- Los comandos de consulta no deben modificar fuentes.
- Errores claros, códigos de salida documentados y mensajes que expliquen cómo recuperarse.
- `init` crea una configuración mínima y no sobrescribe archivos sin consentimiento.
- `index` es idempotente y puede reanudarse después de una interrupción.
- `audit --diff` detecta cambios respecto a una base definida; debe permitir especificar la base explícitamente y no asumir silenciosamente que cualquier rama equivale a la rama principal.
- `audit --diff` admite tres orígenes de cambios: `--base <ref>` (commits entre la base y `HEAD`), `--staged` (área de preparación) y `--worktree` (cambios sin commit). Los agentes suelen trabajar sin confirmar sus cambios, por lo que los tres modos deben producir resultados consistentes y declarar en la salida qué origen se analizó.
- `codixai ui` (alias `codixai --ui`) abre la interfaz gráfica de la sección 8. Solo existe en la variante con UI; en la variante solo-CLI termina con un error de uso que explica cómo obtenerla.

Los comandos `refs`, `deps` y `audit --diff` solo pueden prometer las relaciones y capacidades realmente implementadas. Si un lenguaje todavía no tiene resolución semántica, informar ese límite.

### Códigos de salida propuestos

- `0`: operación correcta, sin infracciones que deban fallar el comando.
- `1`: error de ejecución o configuración.
- `2`: argumentos inválidos.
- `3`: auditoría o verificación completada con hallazgos que superan el umbral de fallo configurado, o con claims `contradicted`. Los claims `unresolved` solo provocan este código si la política lo configura así.

Documentar esta convención y mantenerla estable. El formato JSON debe incluir `schema_version` para permitir evolución compatible.

## 10. Configuración del repositorio analizado

Archivo sugerido: `.codixai.yml`.

Ejemplo inicial:

```yaml
version: 1

project:
  name: my-project

analysis:
  languages:
    - go
    - python
    - php
    - javascript
    - typescript

  exclude:
    - vendor/**
    - node_modules/**
    - dist/**
    - build/**
    - .git/**
    - coverage/**

audit:
  fail_on: error
  baseline: .codixai/baseline.json   # archivo versionado en el repositorio (Fase 3)
  rules:
    CDX-IMP-001:                      # ID ilustrativo
      enabled: true
      severity: error
  suppressions: []                    # explícitas, justificadas y acotadas (ver sección 11)

output:
  format: text
```

La configuración real debe validarse, documentar valores por defecto y permitir exclusiones adicionales. Respetar `.gitignore` como comportamiento predeterminado, con opción explícita para cambiarlo. Los archivos de dependencias y generados deben excluirse del análisis profundo por defecto, pero la configuración debe permitir incluirlos cuando sea útil.

No indexar secretos ni archivos sensibles. Añadir límites configurables para tamaño de archivo, cantidad de archivos, profundidad de recorrido y recursos usados.

## 11. Motor de reglas

Las reglas se dividen en tres categorías:

1. **Deterministas:** imports prohibidos, límites entre capas, patrones sintácticos y convenciones comprobables.
2. **Basadas en análisis:** relaciones duplicadas probables, contratos inconsistentes o posibles faltas de pruebas, con evidencia y advertencias.
3. **Asistidas por IA:** explicaciones y recomendaciones opcionales, nunca presentadas como verificaciones deterministas.

Cada regla debe tener ID estable, descripción, severidad, alcance por lenguaje, implementación/versionado, evidencia requerida y pruebas positivas y negativas.

**Convención de IDs:** `CDX-<ÁREA>-<NNN>` (por ejemplo, `CDX-IMP-001`). Un ID nunca se reutiliza con otra semántica; los cambios de comportamiento incrementan `rule_version`.

**Matriz de soporte por regla y lenguaje:** cada regla publica, para cada uno de los cinco lenguajes, su nivel de soporte (`none`, `syntactic`, `resolved`), el método usado (parser propio, Tree-sitter, toolchain del lenguaje, herramienta externa opcional) y sus limitaciones conocidas. La matriz se mantiene junto a las fixtures y se valida en CI: una regla no puede declarar soporte en un lenguaje sin fixtures positivas y negativas.

**Reglas candidatas posteriores (no comprometidas, sujetas a la evaluación de la Fase 1):**

- dependencias nuevas añadidas a manifiestos o lockfiles sin que el claim del agente las declare;
- cambios fuera del alcance declarado de la tarea (vía claim `scope`, Fase 2);
- aserciones relajadas o dobles de prueba que ocultan el comportamiento real;
- reimplementación probable de utilidades que ya existen en el repositorio (Fase 6);
- cambios de API pública sin actualizar consumidores, pruebas o documentación (Fase 6).

La primera versión debe implementar pocas reglas útiles y verificables antes de crear un lenguaje genérico complejo para reglas personalizadas.

Las excepciones deben ser explícitas, justificadas y acotadas a una regla o ubicación. No suprimir globalmente los hallazgos para ocultar falsos positivos.

## 12. Integración con agentes

La integración con agentes es un objetivo principal, no una ocurrencia posterior. Sin embargo, no asumir protocolos o sistemas de plugins sin comprobar sus interfaces actuales.

### 12.1 Base común

- Operaciones de dominio estables y reutilizables, sin acoplamiento a la CLI ni a un protocolo.
- Comandos deterministas.
- Salida JSON con esquema versionado.
- `--help` útil para que el agente descubra capacidades.
- Códigos de salida estables.
- Modo de solo lectura.
- Límites de tamaño de salida y tiempo.
- Errores en formato legible por máquina cuando se solicita JSON.

### 12.2 MCP, OpenCode y Gentle-Shell

MCP es una interfaz de integración planificada, pero no debe contener lógica de análisis propia. Exponer los mismos casos de uso tipados del núcleo y limitar las herramientas a operaciones de solo lectura. Antes de comprometer un diseño concreto, revisar y documentar las APIs oficiales vigentes de MCP, OpenCode y Gentle-Shell/Pi; registrar las decisiones en ADRs y probar un prototipo mínimo. Los adaptadores específicos deben reutilizar los casos de uso de Codixai; no mantener un motor diferente por agente.

La integración debe ser opcional y documentar la configuración, permisos necesarios y ejemplos de uso. Un agente no debe obtener capacidad de escritura del repositorio por el simple hecho de usar Codixai.

### 12.3 Prueba mínima de integración

Un agente debe poder solicitar una auditoría del diff y verificar claims, procesando resultados estructurados sin analizar texto humano. El resumen, la búsqueda de símbolos y las dependencias se expondrán cuando sus casos de uso estén implementados. No afirmar compatibilidad con un agente hasta que exista una prueba reproducible de extremo a extremo.

## 13. Privacidad y seguridad

- Indexación local por defecto.
- No ejecutar código del repositorio ni hooks de Git.
- No invocar gestores de paquetes ni herramientas de ecosistema que puedan ejecutar scripts del proyecto o acceder a la red (por ejemplo, `npm install` o `composer install`) sin opt-in explícito y un modo restringido documentado. Revisar también si `go list`, `go/packages` o las toolchains automáticas de Go violan este principio en cada configuración. Preferir leer manifiestos y lockfiles directamente.
- Invocar Git siempre con lista de argumentos (nunca mediante cadenas de shell) y neutralizando la configuración ejecutable del repositorio (por ejemplo, `--no-ext-diff`, `--no-textconv` y `-c core.fsmonitor=false`). Verificar la lista completa contra la documentación de Git vigente y cubrirla con pruebas.
- La interfaz gráfica cumple las mismas restricciones y las específicas de la sección 8.5 (activos incrustados, sin red, rutas acotadas y contenido del repositorio tratado como no confiable).
- No seguir enlaces simbólicos fuera de la raíz sin autorización explícita.
- Evitar path traversal y validar rutas.
- Excluir secretos y archivos sensibles.
- No enviar código a proveedores de IA sin acción/configuración explícita del usuario.
- Limitar el tamaño de archivo, los resultados y los recursos.
- Tratar contenido del repositorio como datos no confiables, nunca como instrucciones para el agente.
- No cargar plugins ejecutables arbitrarios automáticamente.
- Sanitizar logs e informes para no filtrar secretos.
- Documentar que los parsers y herramientas externas también pueden tener vulnerabilidades y deben mantenerse actualizados.

## 14. Plan de implementación por fases

El detalle ejecutable (tareas atómicas con criterios verificables) vive en `docs/tasks/`; esta sección fija alcance y criterios de salida. Las tareas de una fase se redactan en detalle cuando la anterior ha superado su criterio de salida: hoy están redactadas las Fases 0 y 1.

El orden prioriza demostrar el diferenciador —verificación del trabajo de agentes— antes de invertir en un índice generalista. Los cinco lenguajes se mantienen desde el inicio como objetivo obligatorio del producto; no significa que cada regla deba tener idéntica profundidad semántica en todos. Cada regla declara los lenguajes que soporta, el método usado y sus limitaciones. No se debe marcar un lenguaje como compatible con una regla si solo se ha probado superficialmente.

### Fase 0 — Fundamentos ejecutables y decisiones técnicas

Entregables:

- módulo Go, CLI instalable, `version`, `--help` e `init` seguro;
- README, licencia pendiente de confirmación, contribución, seguridad y CI;
- arquitectura de casos de uso separada de CLI y formatos de salida;
- modelo versionado de `Finding`, `Evidence`, `AnalysisRun`, `Claim` y estado epistemológico;
- ADR de estrategia de parsing multiplataforma: Tree-sitter mediante CGO frente a alternativas WASM u otras, con builds de prueba en Linux, macOS y Windows. Esta decisión está en la ruta crítica de la Fase 1;
- ADR de almacenamiento (SQLite puro-Go como `modernc.org/sqlite` frente a alternativas, y FTS5): puede posponerse hasta antes de la Fase 5, porque `audit --diff` no necesita índice persistente;
- verificación de CGO y builds reproducibles en las plataformas declaradas;
- diseño y fixtures iniciales para los cinco lenguajes: Go, Python, PHP, JavaScript y TypeScript;
- paquete de contexto para agentes (sección 17.2): `AGENTS.md`, `opencode.json`, convenciones y tareas, probado con al menos una tarea real.

Criterio de salida: clon limpio compilable, pruebas sin servicios externos y decisiones de dependencias documentadas; no se considera suficiente crear directorios vacíos.

### Fase 1 — Spike del verificador de diffs (primer hito de valor)

Implementar un `codixai audit --diff` de solo lectura (con `--base <ref>`, `--staged` o `--worktree`) que analice el diff sin hacer checkout ni ejecutar código del repositorio. Crear una colección versionada de diffs reales o reproducibles producidos por agentes, con casos positivos y negativos.

La fase se entrega por reglas, no en bloque: cada combinación regla/lenguaje se publica cuando alcanza su criterio de calidad, sin esperar a completar todas las combinaciones posibles.

Reglas iniciales candidatas, seleccionadas por evidencia y falsos positivos:

1a. (`CDX-IMP-001`) imports o dependencias que no están declarados en los manifiestos o lockfiles ni pertenecen a la biblioteca estándar o integrados del lenguaje (con listas versionadas por lenguaje y versión declarada). Se resuelve leyendo manifiestos directamente, sin red ni ejecución;
1b. (`CDX-REF-001`) símbolos locales referenciados que no existen: solo donde pueda resolverse con fiabilidad y sin ejecutar herramientas (por ejemplo, Go mediante su toolchain, respetando la restricción de seguridad de la sección 13). En el resto de lenguajes, el resultado es `unresolved` hasta que exista resolución en la Fase 5;
2. (`CDX-TST-001`) tests borrados, deshabilitados o debilitados en el diff;
3. (`CDX-SUP-001`) supresiones nuevas de analizadores (`//nolint`, `# noqa`, `@ts-ignore`, `eslint-disable`, entre otras variantes específicas de lenguaje);
4. (`CDX-ERR-001`) patrones explícitos de errores ignorados o manejadores vacíos;
5. (`CDX-PLH-001`) stubs/placeholders obvios introducidos por el cambio.

Cada regla debe tener implementación por lenguaje, fixtures positivas/negativas, evidencia exacta y estado epistemológico (`observed`, `inferred` o `unknown`). Evitar reglas regex genéricas cuando una API de parser fiable esté disponible. No llamar “bug confirmado” a una señal heurística.

Criterio de salida: los cinco lenguajes se reconocen y tienen fixtures de aceptación desde este hito; cada regla se activa únicamente en lenguajes donde tenga cobertura probada. Publicar una evaluación inicial de precisión, falsos positivos, limitaciones y casos no soportados. El spike sirve para decidir qué reglas conservar, no para prometer una fecha de producto completo.

**Puerta de decisión al cierre de la Fase 1:** antes de empezar la Fase 2, decidir con datos qué reglas se conservan, cuáles se rebajan a `inferred` y cuáles se descartan. Para ello, antes de empezar deben definirse los umbrales de precisión y de falsos positivos por regla y lenguaje (decisión pendiente 7), revisarse diffs reales de agentes ajenos a quien mantiene el proyecto y recogerse opiniones de personas usuarias externas. Si ninguna regla muestra utilidad demostrable, revisar la propuesta de valor antes de ampliar el alcance.

### Fase 2 — Claims, evidencias y contrato de verificación

- `codixai verify --base <ref> --claims <file>`;
- esquema versionado para afirmaciones del agente, con resultados `verified`, `contradicted` y `unresolved`;
- contrastar claims con el diff y los resultados de las reglas, sin inferir intención no declarada;
- referencias a commit/base, archivo y rangos de línea, regla, evidencia y limitaciones;
- salida JSON estable, códigos de salida documentados y stderr separado de stdout;
- soporte de `--format json` sin adornos mezclados en stdout.

**Tipos iniciales de claim (ilustrativos, sujetos a ADR):**

- `files_unchanged`: no se tocaron rutas o patrones indicados;
- `scope`: los cambios se limitan a las rutas declaradas;
- `tests_added` / `tests_modified`: el diff añade o modifica pruebas en las rutas o símbolos indicados;
- `dependency_unchanged` / `dependency_added`: manifiestos y lockfiles sin cambios, o con las dependencias declaradas;
- `api_unchanged`: firmas públicas sin cambios, solo donde exista soporte;
- `tests_passed`: se contrasta únicamente con un informe aportado (ver abajo);
- `behavior_preserved`: no verificable por análisis estático; queda `unresolved` salvo que se aporte evidencia concreta.

**Informes de ejecución como evidencia aportada:** Codixai no ejecuta pruebas (principio 2 de la sección 4). Puede leer informes generados por el agente o por CI (por ejemplo, JUnit XML o `go test -json`; formatos definitivos por decidir) y vincularlos a la revisión analizada. El resultado se registra con procedencia externa y nunca como observado directamente por Codixai. Un informe que no pueda vincularse a la revisión actual deja el claim `unresolved`.

Ejemplo ilustrativo de `claims.json`:

```json
{
  "schema_version": "0.1",
  "base": "main",
  "claims": [
    { "id": "c1", "type": "tests_added", "paths": ["internal/payment/**"] },
    { "id": "c2", "type": "files_unchanged", "paths": ["internal/billing/**"] },
    { "id": "c3", "type": "tests_passed", "report": "reports/go-test.json" }
  ]
}
```

Criterio de salida: claims de prueba detectables en fixtures quedan verificados, contradichos o sin resolver de forma reproducible; ningún estado se infiere sin evidencia suficiente.

### Fase 3 — Adopción en CI: baseline y SARIF

- exportación SARIF con reglas e ubicaciones consistentes;
- baseline versionado para que CI pueda fallar por hallazgos nuevos sin convertir deuda previa en ruido constante;
- huella estable de cada hallazgo (regla, ruta y contexto normalizado, no solo el número de línea) para que el baseline sobreviva a desplazamientos de código;
- severidades configurables, supresiones explícitas y auditables;
- códigos de salida diferenciados para error de ejecución y hallazgos que incumplen la política;
- pruebas de salida en CI y documentación de integración.

Criterio de salida: un proyecto de ejemplo puede ejecutar Codixai en CI, importar SARIF en una plataforma compatible y detectar únicamente hallazgos nuevos según el baseline configurado.

### Fase 4 — Primeras integraciones de agentes y MCP

- verificar la documentación y APIs públicas vigentes de MCP, OpenCode y Gentle-Shell/Pi;
- prototipo de servidor MCP local, con operaciones de solo lectura y esquemas tipados;
- adaptadores o instrucciones de integración solo cuando el punto de extensión se haya confirmado;
- límites de tiempo/tamaño, errores estructurados y pruebas de extremo a extremo;
- evaluar el uso de `audit --diff --staged` o `--worktree` mediante hooks del agente o `pre-commit`, siempre como opt-in explícito: Codixai no instala hooks automáticamente;
- no ejecutar acciones del agente ni modificar el repositorio por petición implícita.

Criterio de salida: al menos una integración documentada y probada invoca el mismo motor de verificación que la CLI. MCP y los adaptadores no duplican reglas ni lógica de dominio.

### Fase 5 — Análisis estructural e impacto bajo demanda

Ampliar el índice solo para mejorar reglas o casos de uso demostrados:

- ADR de almacenamiento resuelto antes de empezar (SQLite, FTS5 y compatibilidad multiplataforma);

- extracción de símbolos y ubicaciones;
- imports, referencias y relaciones con procedencia explícita;
- `index`, `status`, `overview`, `find`, `refs` y `deps`;
- análisis de impacto y selección probable de pruebas afectadas;
- comparar blobs de Git entre base y head sin checkout, cuando sea suficiente.

Criterio de salida: fixtures para los cinco lenguajes verifican las capacidades que se anuncian; las relaciones sin resolver se declaran como tales. No construir un grafo global más costoso sin casos de uso medibles.

### Fase 6 — Reglas ampliadas y análisis semántico

- detectar cambios de API pública y revisar consumidores;
- similitud de código para señalar posibles reimplementaciones de utilidades existentes;
- resolutores específicos por lenguaje, herramientas de ecosistema y LSP donde aporten valor;
- análisis de arquitectura y convenciones configurables;
- IA opcional para explicación, siempre separada de resultados deterministas y basada en evidencia.

Criterio de salida: cada capacidad nueva tiene fixtures, métricas, límites y justificación de valor frente al coste de mantenimiento.

### Fase 7 — Interfaz gráfica local (Wails, Svelte y daisyUI)

No comenzar esta fase hasta que CLI, formato de hallazgos y contratos de verificación hayan demostrado estabilidad. La excepción es el spike 7.0, que puede adelantarse tras la Fase 3 porque el riesgo principal está en el empaquetado y la distribución, no en las pantallas.

**7.0 Spike de empaquetado y ADR**

- ADR de la versión de Wails (decisión pendiente 11) con un prototipo mínimo enlazado a un caso de uso real de solo lectura;
- compilación con la etiqueta `ui` en Linux, macOS y Windows desde CI, con los requisitos de sistema documentados;
- comprobar que la variante sin etiqueta sigue compilando sin Node.js, sin Wails y sin dependencias de webview;
- medir tamaño y tiempo de arranque de ambas variantes;
- ADR sobre el modo servidor local (decisión pendiente 12).

**7.1 UI de auditoría (primer MVP visual)**

- seleccionar repositorio (diálogo nativo o `--root`);
- ejecutar `audit --diff` con `--base`, `--staged` y `--worktree` como trabajos cancelables y con progreso real;
- lista de hallazgos con filtro y orden por regla, severidad, estado epistemológico, lenguaje y ruta;
- detalle del hallazgo: evidencia, hunk del diff, nivel de soporte de la regla, limitaciones y recomendación;
- exportar JSON y SARIF mediante un diálogo de guardado;
- tema claro y oscuro, accesibilidad y textos preparados para traducción.

**7.2 UI de verificación (requiere la Fase 2):** carga de claims e informes aportados y presentación de resultados `verified`, `contradicted` y `unresolved` con su evidencia.

**7.3 Visualización estructural (requiere las Fases 5 y 6):** grafos acotados de dependencias, impacto de un cambio y mapas de módulos, siempre con límites, filtros y paginación, y con las relaciones `unresolved` visualmente distinguibles.

**7.4 API local para clientes externos:** solo si un caso de uso la justifica, con las mismas reglas de seguridad de la sección 8.5.

Criterio de salida: las pantallas llaman únicamente a casos de uso de `application` (sin lógica de análisis en el frontend); la salida mostrada coincide con la de `--format json` para el mismo cambio, comprobado de forma automatizada; ambas variantes se compilan y prueban en CI; y se documentan las plataformas verificadas y los requisitos del sistema.

### Criterio transversal de avance

Cada fase debe responder: qué problema real resuelve, qué evidencia demuestra que funciona, qué falsos positivos se observaron, qué lenguajes y casos cubre, y qué queda fuera. Si no se puede medir utilidad o fiabilidad, no se debe ampliar el alcance por inercia.

## 15. Pruebas y criterios de aceptación

Mantener fixtures pequeñas, legibles y versionadas para los cinco lenguajes desde el inicio, incluyendo errores deliberados, casos límite y ejemplos de sintaxis dinámica. Cada regla debe declarar una matriz explícita de soporte por lenguaje; el objetivo de cinco lenguajes no autoriza a presentar como soportada una regla que no tenga pruebas.

Criterios mínimos propuestos:

- compilación y pruebas desde un clon limpio;
- cobertura de los comandos implementados;
- ubicaciones correctas para todos los símbolos de las fixtures de aceptación;
- detección de infracciones sembradas para cada regla/lenguaje declarado como soportado;
- pruebas negativas y un informe de falsos positivos;
- verificación de claims con estados `verified`, `contradicted` y `unresolved`;
- `tests_passed` sin informe vinculado a la revisión actual queda `unresolved`;
- `--base`, `--staged` y `--worktree` producen resultados consistentes sobre el mismo cambio;
- la matriz de soporte regla × lenguaje se valida en CI contra las fixtures existentes;
- exportación JSON estable y, cuando se implemente, SARIF validado;
- baseline que diferencia hallazgos nuevos de los existentes;
- indexación idempotente e incremental;
- recuperación segura de bases de índice dañadas o incompatibles;
- salida JSON validada contra su esquema;
- auditoría sin modificar fuentes;
- comprobaciones de exclusión de secretos y rutas fuera del proyecto;
- CI reproducible sin API keys ni servicios privados.

Los porcentajes de precisión y los límites de rendimiento se fijarán tras obtener una línea base en proyectos representativos. No declarar métricas de rendimiento sin mediciones reproducibles.

## 16. Distribución y contribución comunitaria

Codixai debe poder compartirse desde las primeras fases, aunque se encuentre en versión experimental.

### Archivos públicos requeridos

- `README.md`: propósito, estado, instalación y ejemplo funcional.
- `LICENSE`: licencia aprobada por el mantenedor.
- `CONTRIBUTING.md`: entorno, estilo, pruebas y proceso de pull request.
- `CODE_OF_CONDUCT.md`.
- `SECURITY.md`: reporte responsable de vulnerabilidades.
- `CHANGELOG.md`.
- `docs/architecture/`: arquitectura y límites conocidos.
- `docs/decisions/`: ADRs para decisiones importantes.
- plantillas de issues y pull requests.
- CI de pruebas y análisis estático.

### Contribuciones

- Las nuevas capacidades de lenguaje deben aportar fixtures y pruebas.
- Las nuevas reglas deben incluir casos positivos y negativos.
- Los cambios al formato JSON requieren compatibilidad o incremento de versión del esquema.
- Las dependencias nuevas deben justificar su necesidad, licencia y mantenimiento.
- Los cambios arquitectónicos significativos deben documentarse mediante ADR.
- Los bugs deben incluir pasos reproducibles y, cuando sea posible, un fixture mínimo.
- No aceptar contribuciones que añadan telemetría, envío de código o ejecución remota sin revisión explícita.

### Publicación

Publicar releases con etiquetas Git, changelog y binarios reproducibles para las plataformas que se hayan validado. Ofrecer instalación desde el código fuente y documentar qué plataformas están probadas.

**Variantes (desde la Fase 7):** la variante solo-CLI (la predeterminada de `go build` y `go install`, sin Node.js ni webview) y la variante con UI (CLI más `codixai ui`). La segunda se compila en CI por plataforma, porque el frontend se incrusta como activos y el modo de escritorio depende de bibliotecas nativas. Ambas salen del mismo código. **Política de distribución (decisión 13, resuelta el 2026-10-22): el artefacto publicado es un binario único que contiene la CLI y la interfaz gráfica.** La variante solo-CLI no se publica como artefacto de release: se mantiene para `go install`, para las tuberías y para los entornos sin sistema gráfico. Cada release indica los requisitos de sistema del binario publicado (por ejemplo, el runtime WebView2 en Windows y las bibliotecas WebKitGTK en Linux; confirmar la lista en el ADR 7.0). La decisión está en el ADR 0010 del repositorio.

La licencia del proyecto es MIT, confirmada por el mantenedor el 2026-10-20 y registrada en el archivo `LICENSE` del repositorio. La decisión y su historial están en el ADR 0004 del repositorio.

## 17. Implementación con agentes de IA (OpenCode)

Codixai se desarrollará con agentes y el cliente objetivo es OpenCode. El modelo de IA concreto no está fijado: las instrucciones deben funcionar con cualquier modelo competente, sin depender de una ventana de contexto grande ni de capacidades específicas. El propio proceso debe demostrar las propiedades que la herramienta promueve.

### 17.1 Protocolo por tarea

1. Leer `AGENTS.md`, la tarea activa en `docs/tasks/` y solo las secciones de esta especificación que la tarea indica.
2. Inspeccionar el código existente y las pruebas antes de proponer cambios.
3. Confirmar el alcance y los criterios de aceptación de la tarea; si algo es ambiguo, aplicar la sección 17.4.
4. Implementar el cambio más pequeño que satisfaga la tarea.
5. Ejecutar las comprobaciones de aceptación de la tarea y las comprobaciones generales (formato, `go vet`, pruebas).
6. Revisar el propio diff: dependencias, alcance, convenciones y secretos.
7. Informar con el formato de resultado de `docs/tasks/README.md`: cambios, comandos ejecutados con su resultado real, lo no verificado y las preguntas pendientes.

El agente no debe afirmar que una prueba pasó si no la ejecutó, ni inventar comandos, APIs, métricas, diffs reales o funcionalidades de herramientas externas. No debe debilitar, desactivar ni borrar pruebas para que pasen. La especificación es el contrato; las decisiones que la modifiquen deben actualizarla y quedar documentadas en un ADR.

### 17.2 Paquete de contexto para OpenCode

OpenCode lee `AGENTS.md` desde la raíz del proyecto y permite añadir archivos de instrucciones mediante el campo `instructions` de `opencode.json`. Los archivos de `instructions` se combinan con `AGENTS.md` y se cargan en cada sesión, por lo que deben ser breves. Esta especificación completa es demasiado grande para cargarse siempre: se lee por secciones según la tarea.

| Archivo | Función | ¿Se carga siempre? |
|---|---|---|
| `AGENTS.md` | Invariantes, flujo de trabajo, condiciones de parada y comandos. Máximo aproximado de 100 líneas. | Sí |
| `opencode.json` | Referencia a `instructions` y permisos de ejecución. | Sí |
| `docs/guides/implementation-conventions.md` | Convenciones concretas de Go, salida, pruebas y Git. | Sí (vía `instructions`) |
| `docs/tasks/README.md` | Formato de tarea, informe de resultado y reglas de parada. | Sí (vía `instructions`) |
| `docs/tasks/phase-0.md`, `phase-1.md` | Tareas atómicas con criterios verificables. | No: se lee la tarea activa |
| `docs/spec/CODIXAI_SPEC.md` | Esta especificación. | No: por secciones |
| `.opencode/agents/codixai-reviewer.md` | Subagente de revisión de solo lectura. | Bajo demanda |
| `.opencode/commands/task.md`, `review.md` | Comandos `/task` y `/review`. | Bajo demanda |

### 17.3 Mapa de lectura por tipo de tarea

| Tipo de tarea | Secciones a leer |
|---|---|
| Modelo de dominio y esquema JSON | 0 (glosario), 1.2, 4, 7.2, 9 (comportamiento y códigos de salida) |
| CLI y presentación | 4, 9, 10 |
| Git y diffs | 4, 9, 13, 14 (Fase 1) |
| Reglas | 0, 1.2, 5, 11, 14 (Fase 1), 15 |
| Parsing y lenguajes | 5, 6, 7, 13 |
| Claims y `verify` | 1.1 (pilar 4), 14 (Fase 2), 15 |
| Documentación y comunidad | 16, 18 |
| ADRs y dependencias | 6, 14 (Fase 0), decisiones pendientes |

### 17.4 Condiciones de parada: cuándo preguntar

El agente debe detenerse y preguntar a la persona responsable, sin elegir por su cuenta, cuando:

1. La tarea depende de una decisión pendiente (URL del repositorio, licencia, contacto de seguridad, plataformas, versión mínima de Go, plataforma de CI, formatos de informes).
2. La especificación o la tarea es ambigua o se contradice.
3. Hace falta una dependencia no prevista, acceso a la red o ejecutar código del repositorio analizado.
4. El cambio altera un formato de salida o un contrato ya fijado (`schema_version`).
5. Una prueba falla sin causa clara tras dos intentos razonablemente distintos. Debe informar del fallo, no relajar la prueba.
6. Marcar el soporte de una regla en un lenguaje exigiría saltarse las fixtures.
7. Algo no puede verificarse en el entorno (por ejemplo, compilar en Windows o macOS): debe declararlo como no verificado, nunca como correcto.
8. Una tarea de UI exige elegir la versión de Wails, añadir una dependencia npm o cambiar la política de distribución (decisiones 11 a 13).

Valores provisionales que el agente puede usar sin preguntar: la versión de Go instalada (registrándola en un ADR), identificadores y comentarios de código en inglés técnico y mensajes de commit en inglés. Todo lo demás de la lista anterior requiere confirmación.

### 17.5 Permisos recomendados

Permitir sin pedir confirmación: compilar, `go vet`, formatear, ejecutar pruebas y leer el estado de Git. Pedir confirmación: `git commit`, `go get`, `go mod`, `go list`, cualquier acceso a la red y comandos no listados. Denegar: `git push`, operaciones destructivas (`git reset --hard`, borrado recursivo) y modificar configuración global. El detalle está en `opencode.json`.

### 17.6 Supuestos sobre OpenCode que deben verificarse

La documentación de OpenCode muestra más de una generación del formato de configuración. En las páginas consultadas aparecen, por un lado, `agent`, `permission` y `.opencode/agents/` o `.opencode/command/`, y por otro `agents`, `permissions` (como lista ordenada de reglas) y `.opencode/commands/`. Antes de la primera tarea, la persona responsable debe comprobar con la versión instalada (`opencode --version` y la documentación de `opencode.ai/docs`: *rules*, *agents*, *commands*, *permissions*) que:

- `AGENTS.md` en la raíz se carga automáticamente;
- el campo `instructions` de `opencode.json` acepta rutas relativas;
- los agentes en `.opencode/agents/*.md` y los comandos en `.opencode/commands/*.md` se descubren;
- la sintaxis de `permission` del paquete es válida en esa versión.

Si algo difiere, ajustar el paquete (no la especificación) y registrar la decisión en un ADR. No afirmar compatibilidad con OpenCode hasta completar una tarea real de extremo a extremo.

### 17.7 Calidad de una tarea

Una tarea está lista para ejecutarse (*Definition of Ready*) cuando tiene objetivo, dependencias, lista de secciones a leer, entregables, comprobaciones de aceptación ejecutables y fuera de alcance explícito. Está terminada (*Definition of Done*) cuando las comprobaciones se ejecutaron y su resultado real se informó, no se añadió nada fuera de alcance, las decisiones nuevas tienen ADR y el diff fue revisado. Una tarea debería corresponder a un cambio revisable por una persona en una sola sesión.

### 17.8 Autoverificación con Codixai

Cuando exista `audit --diff --worktree`, cada tarea incluirá ejecutarlo sobre el propio cambio del agente. Los hallazgos que el agente considere falsos positivos se informan a la persona responsable con su evidencia; no se suprimen silenciosamente. Este uso alimenta el corpus de la Fase 1 siempre que la persona responsable lo apruebe.

### 17.9 Reglas para tareas de la interfaz gráfica

- Antes de escribir código de Wails, detectar la versión desde `go.mod` (`github.com/wailsapp/wails/v2` o `.../v3`) y la herramienta de línea de comandos correspondiente. Las APIs de v2 y v3 son incompatibles: no mezclarlas ni adivinar; ante la duda, preguntar.
- `domain` y `application` no importan Wails ni nada de `ui/`. El adaptador solo traduce.
- El frontend usa tipos generados desde Go; no se duplican a mano.
- Está prohibido `{@html}` y cualquier inserción de HTML a partir de contenido del repositorio; activar la regla de lint correspondiente (por ejemplo, `svelte/no-at-html-tags` de eslint-plugin-svelte).
- Instalar dependencias de frontend con `npm ci`, idealmente con `--ignore-scripts`. No ejecutar `npm install` si modifica el lockfile sin pedir confirmación.
- El agente no debe afirmar que una pantalla "se ve bien" sin evidencia: se verifica con pruebas automatizadas y, cuando haga falta juicio visual, lo revisa la persona responsable.

## 18. Definition of Done del primer hito

El primer hito público debe demostrar el diferenciador mediante auditoría de diffs; la presentación visual es importante, pero no sustituye la utilidad verificable. Los criterios se agrupan por prioridad: los de valor verificable son bloqueantes; los de presentación y comunidad son necesarios para publicar, pero no deben retrasar la validación del verificador.

### 18.1 Valor verificable (bloqueante)

- [ ] `audit --diff` analiza cambios con `--base`, `--staged` y `--worktree` en modo de solo lectura y no requiere indexar el repositorio entero.
- [ ] Los cinco lenguajes (Go, Python, PHP, JavaScript y TypeScript) tienen fixtures de aceptación desde el inicio; cada regla informa en cuáles está realmente soportada mediante su matriz de soporte.
- [ ] Las reglas iniciales incluyen casos positivos y negativos y generan evidencias verificables.
- [ ] Los resultados distinguen hechos observados, inferencias y aspectos sin resolver.
- [ ] Existe un corpus reproducible de diffs y un resumen inicial de falsos positivos y limitaciones, con la decisión de qué reglas se conservan.
- [ ] Los primeros comandos muestran la propuesta de valor de Codixai con evidencia y limitaciones explícitas, sin afirmar capacidades no implementadas.

### 18.2 Contrato y robustez

- [ ] La salida JSON es válida, incluye `schema_version` y no contiene banners, colores, progreso ni mensajes mezclados en stdout.
- [ ] Los códigos de salida están documentados y probados.
- [ ] Existe una prueba automatizada del recorrido básico.
- [ ] CI ejecuta las pruebas sin credenciales ni servicios externos.
- [ ] El proyecto compila con una versión de Go documentada y la estrategia de parsing multiplataforma está decidida mediante ADR.

### 18.3 Presentación, uso y comunidad

- [ ] `codixai version`, `codixai --help` e `init` funcionan; `init` crea una configuración mínima sin sobrescribir archivos.
- [ ] La CLI usa un sistema de estilos centralizado y la salida interactiva distingue visualmente títulos, estados, severidades, rutas y fragmentos de código.
- [ ] `--no-color`, `NO_COLOR` y `--plain` funcionan; la salida redirigida no contiene secuencias ANSI ni animaciones.
- [ ] El progreso de cualquier operación representa fases reales y usa porcentaje solo cuando puede calcularse con honestidad.
- [ ] El README permite clonar, compilar, probar y ejecutar el programa.
- [ ] La licencia, el estado experimental y las limitaciones están documentados.
- [ ] Las instrucciones de contribución permiten que una persona externa comience a colaborar.

No considerar completado este hito solo porque el repositorio o los directorios hayan sido creados.

## 19. Primer comando de desarrollo

El primer paso de implementación debe ser pequeño y verificable:

```bash
go mod init <URL_REAL_DEL_REPOSITORIO>
go test ./...
go run ./cmd/codixai --help
```

`<URL_REAL_DEL_REPOSITORIO>` es un marcador que debe reemplazarse por la URL elegida; no inventar una URL ni asumir un nombre de usuario u organización.

Antes de fijar dependencias o comenzar la fase de análisis, el agente debe revisar sus versiones y licencias y verificar las APIs públicas actuales de las herramientas seleccionadas.

---

## Decisiones pendientes que requieren confirmación

1. URL definitiva del repositorio y organización propietaria.
2. Confirmación de licencia. **Resuelta el 2026-10-20: MIT.**
3. Plataformas que se soportarán y probarán en la primera release.
4. Versión mínima de Go y política de actualización.
5. Estrategia Tree-sitter/CGO/WASM (bloqueante para la Fase 1) y driver SQLite (diferible hasta antes de la Fase 5), cada uno tras un ADR con builds de prueba.
6. Formato definitivo de configuración, claims y reglas.
7. Umbrales y criterios para medir falsos positivos en el corpus inicial.
8. Proveedores de IA que se soportarán, si se incorporan.
9. Mecanismo exacto de integración de MCP, OpenCode y Gentle-Shell tras verificar sus APIs actuales.
10. Formatos de informes de pruebas aceptados como evidencia aportada (por ejemplo, JUnit XML, `go test -json`, TAP) y reglas de vinculación a una revisión.
11. Versión de Wails (v2 estable o v3 beta, que ofrece servicios tipados y modo servidor) y política de actualización; decidir con el spike 7.0 y registrar en un ADR. No mezclar APIs de ambas versiones.
12. Si se incluye un modo servidor local (navegador en loopback) además de la ventana de escritorio, y con qué controles de seguridad.
13. Política de distribución de la variante con UI (binario único CLI más UI frente a solo-CLI como predeterminada) y plataformas soportadas para la UI. **Resuelta el 2026-10-22: binario único CLI más UI** (ADR 0010 del repositorio). Las plataformas soportadas para la UI siguen pendientes.
14. Idiomas de la interfaz (español, inglés) y política de traducción.

Las decisiones 11 a 14 solo bloquean la Fase 7.

Estas decisiones no deben bloquear la creación de la base ejecutable, salvo aquellas necesarias para publicar un paquete con nombre o licencia definitivos y la estrategia de parsing (decisión 5), necesaria antes de implementar las reglas de la Fase 1.

**Resultado esperado:** Codixai comienza como una CLI real, instalable, comprobable y visualmente cuidada cuyo primer valor es verificar diffs y afirmaciones de agentes con evidencia, reglas evaluadas y limitaciones explícitas. Los cinco lenguajes permanecen en el alcance desde el inicio, con cobertura transparente por regla. El índice estructural, MCP, las integraciones de agentes y la web se incorporan en el orden que justifique la utilidad demostrada, sin duplicar el motor de análisis.

---

## Apéndice A — Registro de cambios

### Mantenimiento posterior a la versión 0.6.0

- 2026-10-22: el mantenedor decide que el artefacto publicado es un **binario único con la CLI y la interfaz gráfica** (decisión 13 resuelta, ADR 0010 del repositorio).
- 2026-10-20: el mantenedor confirma la licencia **MIT** y resuelve la decisión pendiente 2. El repositorio publica el archivo `LICENSE` con el texto íntegro de la licencia. La decisión está registrada en el ADR 0004 del repositorio, junto con la URL del repositorio y el camino del módulo.

### Versión 0.6.0 respecto a 0.5.0

- Se integra una interfaz gráfica local con Wails, Svelte y daisyUI, lanzable con `codixai ui` (alias `codixai --ui`), en el mismo código mediante etiquetas de build.
- Se reescribe la sección 8 (qué es Wails, variantes de build, arquitectura, frontend, seguridad y requisitos desde la Fase 0) y se redefine la Fase 7 con un spike de empaquetado adelantable.
- Se aclara que Wails abre una ventana nativa y no un servidor; el modo servidor es opcional y está sujeto a decisión.
- Se añaden las decisiones pendientes 11 a 14, la sección 17.9 y la variante de distribución con UI.
- Se crea el paquete de contexto para OpenCode (`AGENTS.md`, `opencode.json`, convenciones, tareas de las Fases 0 y 1, agente revisor y comandos), que la versión 0.5.0 mencionaba pero no incluía.

### Versión 0.5.0 respecto a 0.4.0

- Se añade la sección 0 (lenguaje normativo, precedencia, alcance por tarea y glosario).
- Se unifican los estados: `epistemic_status` (`observed`, `inferred`, `unknown`) para hallazgos y `resolution` (`confirmed`, `inferred`, `unresolved`) para relaciones. Antes la Fase 1 usaba los de relaciones para hallazgos.
- Se asignan IDs a las reglas de la Fase 1.
- La sección 17 se reescribe para OpenCode: paquete de contexto, mapa de lectura por tarea, condiciones de parada, permisos y supuestos a verificar.
- Se añade la restricción de invocar Git neutralizando configuración ejecutable del repositorio.
- Se redactan las tareas de las Fases 0 y 1 en `docs/tasks/`.

### Versión 0.4.0 respecto a 0.3.0

- Se separa el soporte de imports (1a, basado en manifiestos) del de símbolos locales (1b) en la Fase 1, para no prometer resolución semántica sin índice.
- `audit --diff` admite `--base`, `--staged` y `--worktree`, porque los agentes trabajan habitualmente con cambios sin confirmar.
- Se aclara que Codixai no ejecuta pruebas: `tests_passed` se contrasta solo con informes aportados y vinculados a la revisión (Fase 2), y se añade un catálogo inicial de tipos de claim.
- La estrategia Tree-sitter/CGO/WASM pasa a ser bloqueante para la Fase 1; la decisión de SQLite se difiere hasta antes de la Fase 5.
- Se añaden: puerta de decisión al cierre de la Fase 1, entrega por reglas, convención de IDs, matriz de soporte regla × lenguaje, huella estable para el baseline y ejemplos ilustrativos de hallazgo y de `claims.json`.
- Se añade una restricción de seguridad sobre herramientas de ecosistema que puedan ejecutar scripts o usar la red.
- El Definition of Done se reorganiza por prioridad (valor verificable, contrato y robustez, presentación y comunidad), sin eliminar criterios.
- Se mantienen los cinco lenguajes desde el inicio, tal como se decidió en 0.3.0.
