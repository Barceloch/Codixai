# Codixai — Especificación técnica del proyecto

> **Estado:** especificación fundacional  
> **Versión:** 0.3.0  
> **Nombre del proyecto:** Codixai  
> **Nombre del ejecutable:** `codixai`  
> **Idioma principal de la documentación:** inglés técnico, con documentación inicial en español si ayuda a la comunidad objetivo  
> **Licencia propuesta:** Apache-2.0, pendiente de confirmar antes de publicar  
> **Objetivo:** construir una herramienta local-first para comprender, explorar y auditar repositorios de software, especialmente código creado o modificado con asistencia de IA.

---

## 1. Visión

Codixai es una herramienta de inteligencia de código cuyo primer punto de entrada será una CLI multiplataforma escrita en Go. Debe ayudar a los desarrolladores a:

- encontrar rápidamente archivos, símbolos, referencias y puntos de entrada;
- comprender cómo se conectan los componentes y cómo fluye una operación por el sistema;
- definir y comprobar convenciones y límites arquitectónicos;
- revisar cambios propuestos por agentes de IA;
- obtener resultados verificables, enlazados al código real;
- integrar estas capacidades en agentes como OpenCode y Gentle-Shell;
- evolucionar posteriormente hacia una aplicación web con mapas, grafos y diagramas interactivos.

**Principio rector:** Codixai no es un chatbot que adivina cómo funciona un repositorio ni un simple generador de grafos. Es un verificador local-first del trabajo realizado por agentes de IA: contrasta diffs y afirmaciones con evidencia reproducible, señala riesgos y deja claro qué pudo comprobar y qué sigue sin resolver. El análisis estructural es una capacidad habilitadora que se amplía según las necesidades verificables del producto.

### 1.1 Diferenciación del producto

Codixai no debe convertirse en otro explorador de archivos, generador de mapas de dependencias ni wrapper de herramientas de análisis estático. Su propuesta de valor es **inteligencia verificable del código y supervisión de cambios realizados por agentes de IA**.

Los pilares de diferenciación son:

1. **Mapa de evidencia:** los hallazgos deben apuntar a archivos, líneas, símbolos, relaciones y reglas que puedan verificarse. Cada resultado debe distinguir hechos observados, inferencias y preguntas pendientes.
2. **Análisis de impacto de cambios:** además de resumir un diff, identificar símbolos y módulos afectados mediante relaciones conocidas, explicar por qué se consideran afectados y sugerir verificaciones o pruebas relevantes. No afirmar impactos que el análisis no pueda justificar.
3. **Contexto compacto para agentes:** ofrecer resultados estructurados, limitados y pertinentes que permitan a agentes como OpenCode y Gentle-Shell consultar el repositorio sin leerlo indiscriminadamente. El contexto debe incluir referencias estables a la evidencia.
4. **Verificación posterior del trabajo de agentes:** comparar el estado real del repositorio, el diff y las verificaciones ejecutadas con las afirmaciones o criterios de aceptación proporcionados. Codixai no debe declarar que un cambio es correcto solo porque compila o porque un agente lo afirme.
5. **Memoria estructural versionada:** permitir comparar índices entre revisiones para explicar cómo cambió la estructura conocida. Los resultados deben identificar la revisión/base usada y la frescura del índice.
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
4. Verificar cambios contra una base Git y, cuando se proporcione, contrastarlos con afirmaciones explícitas del agente.
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
- Servidor local/API para clientes externos y una futura web.
- Visualización de grafos de dependencias, mapas de módulos y flujos de ejecución.
- Resolución semántica ampliada mediante LSP y herramientas propias de cada lenguaje.
- Explicaciones opcionales con proveedores de IA locales o remotos.
- Integración CI y políticas compartidas por equipos.

### 2.3 No objetivos del MVP

- No construir todavía una aplicación web.
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
- **SQLite:** índice local persistente.
- **FTS5:** búsqueda textual, si está disponible en el driver SQLite elegido.
- **Tree-sitter:** parsing multilenguaje con gramáticas mantenidas y licencias revisadas.
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
api/
  openapi/              # contrato futuro; no requiere API implementada en el MVP
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
- La futura web llamará a los mismos casos de uso a través de una API, sin reimplementar el análisis.
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

Los hallazgos deben contener identificador de regla, severidad, explicación, archivo/rango, evidencia, estado de confianza y, opcionalmente, recomendación.

## 8. Contratos para la futura interfaz web

Aunque no se implemente una web en el MVP, los casos de uso deben devolver estructuras que puedan serializarse.

La futura API deberá poder exponer operaciones conceptuales como:

- obtener resumen del repositorio;
- buscar símbolos;
- obtener un símbolo y sus ubicaciones;
- consultar dependencias entrantes y salientes;
- recuperar subgrafos acotados;
- consultar flujos y sus evidencias;
- ejecutar o consultar auditorías.

No devolver grafos ilimitados por defecto: las consultas deben admitir límites, filtros y paginación.

No implementar aún servidor HTTP, autenticación web, frontend, WebSockets ni contratos OpenAPI completos hasta validar los casos de uso. Mantener la carpeta `api/openapi` solo cuando exista una especificación concreta que mantener; no crear documentación ficticia.

Cuando se añada servidor local, protegerlo contra accesos de otros sitios, acceso arbitrario a rutas, peticiones no autorizadas y exposición accidental a la red. Escuchar en loopback por defecto.

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
codixai verify --base main --claims claims.json
codixai baseline create
codixai report --format json
codixai report --format sarif
```

### Experiencia visual de terminal

La CLI es una interfaz de producto, no una salida de depuración. Desde los primeros comandos debe tener una presentación consistente, accesible y legible, sin obligar al usuario a usar una TUI interactiva.

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

Los comandos `refs`, `deps` y `audit --diff` solo pueden prometer las relaciones y capacidades realmente implementadas. Si un lenguaje todavía no tiene resolución semántica, informar ese límite.

### Códigos de salida propuestos

- `0`: operación correcta, sin infracciones que deban fallar el comando.
- `1`: error de ejecución o configuración.
- `2`: argumentos inválidos.
- `3`: auditoría completada con hallazgos que superan el umbral de fallo configurado.

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

El orden prioriza demostrar el diferenciador —verificación del trabajo de agentes— antes de invertir en un índice generalista. Los cinco lenguajes se mantienen desde el inicio como objetivo obligatorio del producto; no significa que cada regla deba tener idéntica profundidad semántica en todos. Cada regla declara los lenguajes que soporta, el método usado y sus limitaciones. No se debe marcar un lenguaje como compatible con una regla si solo se ha probado superficialmente.

### Fase 0 — Fundamentos ejecutables y decisiones técnicas

Entregables:

- módulo Go, CLI instalable, `version`, `--help` e `init` seguro;
- README, licencia pendiente de confirmación, contribución, seguridad y CI;
- arquitectura de casos de uso separada de CLI y formatos de salida;
- modelo versionado de `Finding`, `Evidence`, `AnalysisRun`, `Claim` y estado epistemológico;
- ADR comparando SQLite puro-Go (`modernc.org/sqlite`) y alternativas, FTS5 y estrategia de parsing multiplataforma;
- verificación de CGO y builds reproducibles en las plataformas declaradas;
- diseño y fixtures iniciales para los cinco lenguajes: Go, Python, PHP, JavaScript y TypeScript.

Criterio de salida: clon limpio compilable, pruebas sin servicios externos y decisiones de dependencias documentadas; no se considera suficiente crear directorios vacíos.

### Fase 1 — Spike del verificador de diffs (primer hito de valor)

Implementar un `codixai audit --diff --base <ref>` de solo lectura que analice el diff sin hacer checkout ni ejecutar código del repositorio. Crear una colección versionada de diffs reales o reproducibles producidos por agentes, con casos positivos y negativos.

Reglas iniciales candidatas, seleccionadas por evidencia y falsos positivos:

1. imports/dependencias o símbolos referenciados que no se pueden resolver;
2. tests borrados, deshabilitados o debilitados en el diff;
3. supresiones nuevas de analizadores (`//nolint`, `# noqa`, `@ts-ignore`, `eslint-disable`, entre otras variantes específicas de lenguaje);
4. patrones explícitos de errores ignorados o manejadores vacíos;
5. stubs/placeholders obvios introducidos por el cambio.

Cada regla debe tener implementación por lenguaje, fixtures positivas/negativas, evidencia exacta y estados `confirmed`, `inferred` o `unresolved`. Evitar reglas regex genéricas cuando una API de parser fiable esté disponible. No llamar “bug confirmado” a una señal heurística.

Criterio de salida: los cinco lenguajes se reconocen y tienen fixtures de aceptación desde este hito; cada regla se activa únicamente en lenguajes donde tenga cobertura probada. Publicar una evaluación inicial de precisión, falsos positivos, limitaciones y casos no soportados. El spike sirve para decidir qué reglas conservar, no para prometer una fecha de producto completo.

### Fase 2 — Claims, evidencias y contrato de verificación

- `codixai verify --base <ref> --claims <file>`;
- esquema versionado para afirmaciones del agente, con resultados `verified`, `contradicted` y `unresolved`;
- contrastar claims con el diff y los resultados de las reglas, sin inferir intención no declarada;
- referencias a commit/base, archivo y rangos de línea, regla, evidencia y limitaciones;
- salida JSON estable, códigos de salida documentados y stderr separado de stdout;
- soporte de `--format json` sin adornos mezclados en stdout.

Criterio de salida: claims de prueba detectables en fixtures quedan verificados, contradichos o sin resolver de forma reproducible; ningún estado se infiere sin evidencia suficiente.

### Fase 3 — Adopción en CI: baseline y SARIF

- exportación SARIF con reglas e ubicaciones consistentes;
- baseline versionado para que CI pueda fallar por hallazgos nuevos sin convertir deuda previa en ruido constante;
- severidades configurables, supresiones explícitas y auditables;
- códigos de salida diferenciados para error de ejecución y hallazgos que incumplen la política;
- pruebas de salida en CI y documentación de integración.

Criterio de salida: un proyecto de ejemplo puede ejecutar Codixai en CI, importar SARIF en una plataforma compatible y detectar únicamente hallazgos nuevos según el baseline configurado.

### Fase 4 — Primeras integraciones de agentes y MCP

- verificar la documentación y APIs públicas vigentes de MCP, OpenCode y Gentle-Shell/Pi;
- prototipo de servidor MCP local, con operaciones de solo lectura y esquemas tipados;
- adaptadores o instrucciones de integración solo cuando el punto de extensión se haya confirmado;
- límites de tiempo/tamaño, errores estructurados y pruebas de extremo a extremo;
- no ejecutar acciones del agente ni modificar el repositorio por petición implícita.

Criterio de salida: al menos una integración documentada y probada invoca el mismo motor de verificación que la CLI. MCP y los adaptadores no duplican reglas ni lógica de dominio.

### Fase 5 — Análisis estructural e impacto bajo demanda

Ampliar el índice solo para mejorar reglas o casos de uso demostrados:

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

### Fase 7 — API y aplicación visual futura

- API local sobre los casos de uso existentes;
- grafos y diagramas interactivos para explicar cambios e impacto;
- frontend independiente y controles de acceso;
- pruebas de API, seguridad y visualización.

No comenzar esta fase hasta que CLI, formato de hallazgos y contratos de verificación hayan demostrado estabilidad.

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

La licencia Apache-2.0 es una propuesta adecuada para facilitar contribuciones y adopción empresarial, pero debe confirmarse antes de distribuir el repositorio.

## 17. Flujo de desarrollo con agentes de IA

Codixai se desarrollará con ayuda de agentes, por lo que el propio proceso debe demostrar las propiedades que la herramienta pretende promover.

Antes de cada tarea:

1. Leer esta especificación y las decisiones arquitectónicas relevantes.
2. Inspeccionar el código existente y las pruebas antes de proponer cambios.
3. Definir el alcance y los criterios de aceptación.
4. Implementar el cambio más pequeño que satisfaga el objetivo.
5. Ejecutar pruebas y análisis pertinentes.
6. Revisar el diff, dependencias y convenciones.
7. Resumir resultados, limitaciones y tareas pendientes.

El agente no debe afirmar que una prueba pasó si no la ejecutó, ni inventar comandos, APIs o funcionalidades de herramientas externas.

La especificación es el contrato inicial; las decisiones que la modifiquen deben actualizarla y quedar documentadas.

## 18. Definition of Done del primer hito

El primer hito público debe demostrar el diferenciador mediante auditoría de diffs; la presentación visual es importante, pero no sustituye la utilidad verificable. El hito está completo cuando:

- [ ] El proyecto compila con una versión de Go documentada.
- [ ] `codixai version` y `codixai --help` funcionan.
- [ ] `codixai init` crea una configuración mínima sin sobrescribir archivos.
- [ ] La CLI usa un sistema de estilos centralizado y la salida interactiva distingue visualmente títulos, estados, severidades, rutas y fragmentos de código.
- [ ] `--no-color`, `NO_COLOR` y `--plain` funcionan; la salida redirigida no contiene secuencias ANSI ni animaciones.
- [ ] El progreso de cualquier operación representa fases reales y usa porcentaje solo cuando puede calcularse con honestidad.
- [ ] `audit --diff` analiza cambios en modo de solo lectura y no requiere indexar el repositorio entero.
- [ ] Los cinco lenguajes (Go, Python, PHP, JavaScript y TypeScript) tienen fixtures de aceptación desde el inicio; cada regla informa en cuáles está realmente soportada.
- [ ] Las reglas iniciales incluyen casos positivos y negativos y generan evidencias verificables.
- [ ] Los resultados distinguen hechos observados, inferencias y aspectos sin resolver.
- [ ] Existe un corpus reproducible de diffs y un resumen inicial de falsos positivos y limitaciones.
- [ ] La salida JSON es válida y no contiene banners, colores, progreso ni mensajes mezclados en stdout.
- [ ] Los primeros comandos muestran la propuesta de valor de Codixai con evidencia y limitaciones explícitas, sin afirmar capacidades no implementadas.
- [ ] Existe una prueba automatizada del recorrido básico.
- [ ] El README permite clonar, compilar, probar y ejecutar el programa.
- [ ] La licencia, el estado experimental y las limitaciones están documentados.
- [ ] CI ejecuta las pruebas sin credenciales ni servicios externos.
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
2. Confirmación de licencia Apache-2.0.
3. Plataformas que se soportarán y probarán en la primera release.
4. Versión mínima de Go y política de actualización.
5. Driver SQLite y estrategia Tree-sitter/CGO/WASM, tras un ADR con builds de prueba.
6. Formato definitivo de configuración, claims y reglas.
7. Umbrales y criterios para medir falsos positivos en el corpus inicial.
8. Proveedores de IA que se soportarán, si se incorporan.
9. Mecanismo exacto de integración de MCP, OpenCode y Gentle-Shell tras verificar sus APIs actuales.

Estas decisiones no deben bloquear la creación de la base ejecutable, salvo aquellas necesarias para publicar un paquete con nombre o licencia definitivos.

**Resultado esperado:** Codixai comienza como una CLI real, instalable, comprobable y visualmente cuidada cuyo primer valor es verificar diffs y afirmaciones de agentes con evidencia, reglas evaluadas y limitaciones explícitas. Los cinco lenguajes permanecen en el alcance desde el inicio, con cobertura transparente por regla. El índice estructural, MCP, las integraciones de agentes y la web se incorporan en el orden que justifique la utilidad demostrada, sin duplicar el motor de análisis.
