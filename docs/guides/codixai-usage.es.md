# Guía de uso de Codixai

Esta guía explica cómo instalar y usar Codixai **sin instalar Go**: con los
binarios precompilados de la página de releases, desde la línea de comandos y
desde la ventana de escritorio.

- Versión documentada: **0.1.0-rc.2**
- Repositorio: <https://github.com/Barceloch/Codixai>
- Descargas: <https://github.com/Barceloch/Codixai/releases>
- Contrato del producto (especificación, en español):
  [`docs/spec/CODIXAI_SPEC.md`](../spec/CODIXAI_SPEC.md)

## Índice

1. [Qué es Codixai](#1-qué-es-codixai)
2. [Descarga e instalación](#2-descarga-e-instalación)
3. [Primeros pasos con la CLI](#3-primeros-pasos-con-la-cli)
4. [Referencia de la CLI](#4-referencia-de-la-cli)
5. [Uso de la interfaz gráfica](#5-uso-de-la-interfaz-gráfica)
6. [Configuración del repositorio](#6-configuración-del-repositorio)
7. [Verificar la descarga](#7-verificar-la-descarga)
8. [Problemas frecuentes](#8-problemas-frecuentes)
9. [Limitaciones conocidas](#9-limitaciones-conocidas)
10. [Estado de verificación de los binarios](#10-estado-de-verificación-de-los-binarios)

---

## 1. Qué es Codixai

Codixai lee los cambios de un repositorio Git, aplica un catálogo de reglas
determinista y escribe un documento versionado con lo que observó, lo que
infirió y lo que no pudo comprobar.

Lo que **no** hace, por diseño:

- No ejecuta el código del repositorio analizado (ni tests, ni builds, ni
  scripts, ni gestores de paquetes).
- No accede a la red.
- No modifica el repositorio que analiza.

La única excepción es `git`, que se invoca con subcomandos de solo lectura y
con el endurecimiento descrito en
[`docs/decisions/0006-git-invocation-hardening.md`](../decisions/0006-git-invocation-hardening.md).

El estado epistémico de un hallazgo es parte del contrato: `observed` significa
comprobado contra el código o el diff, `inferred` significa que viene de una
heurística y `unknown` significa que la herramienta no pudo decidir. Una
heurística nunca se presenta como un bug confirmado.

## 2. Descarga e instalación

Cada release publica cuatro paquetes. Elige el que corresponda a tu sistema:

| Sistema | Variante | Archivo |
| ------- | -------- | ------- |
| Linux x86_64 | Solo CLI | `codixai_0.1.0-rc.2_linux_amd64_cli.tar.gz` |
| Linux x86_64 | CLI + ventana | `codixai_0.1.0-rc.2_linux_amd64_ui.tar.gz` |
| Windows x86_64 | Solo CLI | `codixai_0.1.0-rc.2_windows_amd64_cli.zip` |
| Windows x86_64 | CLI + ventana | `codixai_0.1.0-rc.2_windows_amd64_ui.zip` |

La variante `ui` contiene la misma línea de comandos **más** la ventana: es un
único binario. La variante `cli` existe para servidores, contenedores y
sistemas sin sesión gráfica.

Cada paquete incluye el binario, `README.md`, `README.es.md`, `VERSION` y
`LICENSE`. El archivo `SHA256SUMS` del release cubre los cuatro paquetes.

### Requisitos

**CLI en Linux.** Ninguno más allá del sistema operativo: el binario es
estático (`CGO_ENABLED=0`), así que no depende de la versión de la biblioteca C
de la distribución.

**CLI en Windows.** Windows 10 o posterior. No necesita Go ni el runtime de
WebView2.

**UI en Linux.** GTK 3 y WebKitGTK 4.1 en tiempo de ejecución:

```bash
# Ubuntu 24.04, Debian 13 y derivados
sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0

# Fedora
sudo dnf install gtk3 webkit2gtk4.1

# Arch y Manjaro
sudo pacman -S gtk3 webkit2gtk-4.1
```

Además, glibc 2.39 o posterior: el binario se compila en Ubuntu 24.04 porque
la ventana enlaza contra las bibliotecas del sistema. Un binario compilado en
una distribución de liberación continua (Arch, Manjaro) no arranca en Debian
ni en Ubuntu, y por eso no se publica uno.

**UI en Windows.** El runtime Microsoft Edge WebView2. Windows 11 lo incluye;
en Windows 10 puede instalarlo el actualizador de Edge o descargarse de
<https://developer.microsoft.com/microsoft-edge/webview2/>.

### Instalar en Linux

```bash
tar -xzf codixai_0.1.0-rc.2_linux_amd64_cli.tar.gz
cd codixai_0.1.0-rc.2_linux_amd64_cli
chmod +x codixai
./codixai --help
./codixai version
```

Para dejarlo en el `PATH`:

```bash
mkdir -p ~/.local/bin
install -m 0755 codixai ~/.local/bin/codixai
codixai version
```

### Instalar en Windows

Descomprime el `.zip` y ejecuta desde PowerShell:

```powershell
.\codixai.exe --help
.\codixai.exe version
```

Windows puede pedir confirmación de SmartScreen la primera vez: el binario no
está firmado.

## 3. Primeros pasos con la CLI

Todos los comandos aceptan `--root` para analizar un repositorio distinto del
directorio actual:

```bash
# Los cambios sin confirmar del repositorio actual
codixai audit --diff --worktree

# El rango entre una base y HEAD
codixai audit --diff --base main

# El área de preparación (staging)
codixai audit --diff --staged

# El mismo documento, en JSON, para una herramienta
codixai audit --diff --worktree --format json

# SARIF 2.1.0, para una plataforma de revisión
codixai report --format sarif --base main

# Otro repositorio
codixai --root ../otro-repositorio audit --diff --worktree
```

`audit`, `report` y `baseline create` exigen **exactamente un origen**:
`--base`, `--staged` o `--worktree`.

### Códigos de salida

| Código | Significado |
| ------ | ----------- |
| `0` | La operación terminó y ningún hallazgo alcanza el umbral. |
| `1` | Error de ejecución o de configuración. |
| `2` | Argumentos inválidos. |
| `3` | La auditoría terminó y los hallazgos alcanzan el umbral `fail_on`. |

La convención es estable: un comando nuevo no inventa un código.

### El contrato de la salida

- La **salida estándar** contiene exactamente un documento. Sin banners, sin
  color, sin progreso.
- Los **diagnósticos**, los avisos y los errores van a la **salida de error**,
  con el prefijo `codixai:`.
- Todo documento JSON lleva `schema_version` (`0.1`) y `tool_version`.
- El documento es determinista: el mismo cambio produce el mismo documento,
  salvo `generated_at`.

## 4. Referencia de la CLI

```text
codixai <comando> [flags]

comandos: audit, report, baseline, verify, init, index, impact, ui, version, help
```

### Flags globales

| Flag | Qué hace |
| ---- | -------- |
| `--root <ruta>` | Raíz del repositorio analizado. Por defecto, el directorio actual. |
| `--format <valor>` | `text` o `json`; `sarif` en `report`. |
| `--no-color` | Desactiva el color. |
| `--plain` | Sin ninguna decoración. |
| `--version` | Escribe la versión y termina. |
| `-h`, `--help` | Ayuda del comando. |

### `audit`

| Flag | Qué hace |
| ---- | -------- |
| `--diff` | Audita los cambios. Obligatorio en esta versión. |
| `--base <ref>` | Compara con una revisión (`main`, `origin/main`, un hash). |
| `--staged` | Audita el área de preparación. |
| `--worktree` | Audita los cambios sin confirmar. |
| `--rules` | Escribe el catálogo de reglas y termina. |

```bash
codixai audit --rules
codixai audit --diff --worktree --plain
codixai audit --diff --base origin/main --format json > audit.json
```

### `report`

Escribe el documento para máquinas del mismo análisis. `--format sarif` es el
formato por defecto del comando; `--format json` escribe el mismo documento,
byte por byte, que `audit --format json`.

```bash
codixai report --format sarif --base main > codixai.sarif
```

### `verify`

Comprueba las afirmaciones de un agente contra el repositorio. Necesita
`--base` y `--claims`.

```bash
codixai verify --base main --claims claims.json
codixai verify --base main --claims claims.json --format json
```

El archivo de afirmaciones es un documento JSON versionado:

```json
{
  "schema_version": "0.1",
  "base": "main",
  "claims": [
    { "id": "c1", "type": "tests_added", "paths": ["internal/payment/**"] },
    { "id": "c2", "type": "files_unchanged", "paths": ["internal/billing/**"] },
    { "id": "c3", "type": "tests_passed", "report": "reports/go-test.xml" }
  ]
}
```

Cada afirmación termina en uno de tres estados: `verified`, `contradicted` o
`unresolved`. **`unresolved` no es un fallo**: significa que la evidencia
disponible no alcanzó para decidir, nunca que la afirmación sea cierta.

Si el archivo declara `base`, tiene que coincidir con la base de la
comparación; si no, las afirmaciones se referirían a otra revisión.

### `baseline`

Gestiona la línea base de hallazgos del repositorio: un archivo versionado que
registra los hallazgos que el repositorio ya tiene, para que una tubería falle
solo con los nuevos.

```bash
codixai baseline create --base main
codixai baseline create --base main --out .codixai/baseline.json --force
```

La identidad de un hallazgo es su *fingerprint* (regla, ruta y contexto
normalizado), nunca el número de línea: la línea base sobrevive al movimiento
del código. El archivo se confirma con el repositorio y se revisa como el
código.

### `init`

Crea `.codixai.yml` con los valores por defecto documentados. Nunca sobrescribe
un archivo existente; `--force` lo reemplaza de forma explícita.

```bash
codixai init
```

### `index` e `impact`

`index` construye y consulta el índice estructural del repositorio; `impact`
analiza el impacto de un cambio sobre los hallazgos y los tests.

```bash
codixai index build
codixai index status
codixai index overview
codixai index find <símbolo>
codixai index refs <símbolo>
codixai index deps
codixai index diff --base main

codixai impact --base main
```

El índice se guarda en el directorio local del proyecto y no se publica.

### `ui`

Abre la ventana. Solo existe en la variante `ui`: en el paquete `cli` responde
con un error de uso (código `2`) que explica cómo obtener la variante con
interfaz.

```bash
codixai ui
codixai ui --root /ruta/al/repositorio
```

## 5. Uso de la interfaz gráfica

1. Ejecuta `codixai ui` (o `codixai ui --root <ruta>` para abrir un repositorio
   concreto).
2. En la pestaña **Auditoría**, elige la raíz del repositorio con **Elegir…** o
   escríbela.
3. Elige el origen: los cambios sin confirmar (worktree), el área de
   preparación (staged) o una revisión base.
4. Pulsa **Ejecutar auditoría**. La ventana muestra las fases reales del caso
   de uso; **Cancelar** detiene la ejecución de forma cooperativa.
5. Cuando termina, la ventana muestra el resumen, la lista de hallazgos con sus
   filtros y, al seleccionar una fila, el detalle: severidad, estado
   epistémico, evidencia, soporte, limitaciones, recomendación y la revisión.
6. **Exportar** escribe el mismo documento que la CLI: JSON con la misma
   codificación y, en la auditoría, también SARIF. La ruta se elige en el
   diálogo nativo y no puede quedar dentro del repositorio analizado.

La pestaña **Verificación** hace lo mismo con un archivo de afirmaciones:
eligiendo la raíz, la revisión base y el archivo `.json` de claims. Muestra una
fila por afirmación con su estado (`verified`, `contradicted`, `unresolved`), el
detalle de la evidencia y sus limitaciones. La exportación de la verificación es
**solo JSON** en esta versión: el SARIF de una verificación todavía no está
implementado.

La ventana nunca modifica el repositorio, nunca ejecuta su código y nunca
accede a la red. Si abres la interfaz en un navegador sin la capa nativa,
verás un aviso explicando que hace falta la ventana de Codixai.

## 6. Configuración del repositorio

`codixai init` crea `.codixai.yml` en la raíz del repositorio. Todas las claves
son opcionales y los valores del archivo son los predeterminados documentados:

```yaml
version: 1

project:
  name: mi-proyecto

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
  respect_gitignore: true

audit:
  fail_on: error          # error | warning | none
  baseline: .codixai/baseline.json
  rules:
  suppressions: []

output:
  format: text
```

Los errores del cargador nombran el archivo y la línea: un valor inválido no se
ignora en silencio.

## 7. Verificar la descarga

Cada release publica `SHA256SUMS` con las cuatro sumas. En Linux:

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

En Windows PowerShell:

```powershell
$expected = (Select-String -Path SHA256SUMS -Pattern 'windows_amd64_cli').Line.Split()[0]
(Get-FileHash .\codixai_0.1.0-rc.2_windows_amd64_cli.zip -Algorithm SHA256).Hash.ToLower() -eq $expected
```

Una suma que no coincide significa que el archivo no es el que se publicó: no
lo ejecutes.

## 8. Problemas frecuentes

**`codixai ui` responde que no está disponible en esta compilación.**
Estás usando el paquete `cli`. Descarga el paquete `ui` de la misma versión.

**La ventana no arranca en Linux y menciona WebKitGTK.**
Falta la biblioteca en tiempo de ejecución. Instala `libwebkit2gtk-4.1-0` (o el
nombre equivalente de tu distribución) y vuelve a intentarlo.

**La ventana no arranca y menciona `GLIBC_2.39` (o una versión superior).**
Tu distribución es anterior a Ubuntu 24.04 o Debian 13. Usa el paquete `cli`,
que es estático, o compila desde el código en tu sistema.

**La ventana no arranca en Windows y menciona WebView2.**
Instala el runtime Microsoft Edge WebView2.

**El JSON sale vacío o el comando falla con código 1.**
La salida estándar solo lleva el documento; el motivo está en la salida de
error. Ejecuta el mismo comando sin redirección para verlo.

**`audit` falla con código 3.**
No es un error de ejecución: hay hallazgos que alcanzan el umbral `fail_on`.
Ajusta `audit.fail_on` en `.codixai.yml` o usa una línea base.

## 9. Limitaciones conocidas

- **Solo Linux y Windows sobre x86_64** en esta entrega. No hay binarios ARM ni
  macOS.
- **La UI de Linux depende de las bibliotecas del sistema.** Se compila en
  Ubuntu 24.04; no se publica un binario universal ni un AppImage.
- **La UI de Windows se compila pero no se ejecuta en la tubería de release**:
  el runner no tiene escritorio interactivo. Ver la sección siguiente.
- **La verificación exporta solo JSON** desde la ventana.
- **El binario no está firmado.** Windows SmartScreen puede avisar.
- **El servidor MCP** (`codixai-mcp-srv`) no forma parte de esta entrega.
- **Un resultado vacío no prueba que el cambio sea correcto.** La ausencia de
  hallazgos no es una garantía.

## 10. Estado de verificación de los binarios

Esta tabla distingue lo que se compiló de lo que se ejecutó de verdad. Es parte
del contrato de honestidad del proyecto: una compilación correcta no es una
prueba de ejecución.

| Paquete | Compilado | Ejecutado en la tubería |
| ------- | --------- | ----------------------- |
| CLI Linux | Sí | Sí: `--help`, `version`, `audit`, `report` |
| UI Linux | Sí | Sí: la ventana abre y cierra bajo Xvfb |
| CLI Windows | Sí | Sí: `--help`, `version`, `ui` (error de uso esperado) |
| UI Windows | Sí | **No**: el runner no tiene escritorio interactivo |

Los detalles de cada ejecución local y de cada verificación quedan en el
informe del release y en
[`odd/tasks/release-v0.1.0-rc.2.md`](../../odd/tasks/release-v0.1.0-rc.2.md).

## Más documentación

- [`README.md`](../../README.md) — presentación del proyecto y compilación desde
  el código (en inglés).
- [`CHANGELOG.md`](../../CHANGELOG.md) — qué cambió en cada versión.
- [`docs/decisions/`](../decisions/) — las decisiones de arquitectura.
- [`docs/tasks/`](../tasks/) — el trabajo ejecutable por fases.
