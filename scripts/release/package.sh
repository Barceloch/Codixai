#!/usr/bin/env bash
#
# Package one release artifact of Codixai.
#
# The script stages the binary with the licence, a VERSION file and the
# installation instructions of the platform, and writes either a tar.gz
# (linux) or a zip (windows). The same script runs in the release workflow
# and on a developer machine: the packaged layout never depends on the
# machine that produced it.
#
# Usage:
#   scripts/release/package.sh \
#     --version 0.1.0-rc.2 --os linux --arch amd64 \
#     --variant cli --binary /tmp/codixai --outdir dist/release
#
# The archive is written to <outdir> and its SHA-256 is printed on the last
# line of the standard output.

set -euo pipefail

version=""
os=""
arch=""
variant=""
binary=""
outdir=""

usage() {
  sed -n '4,20p' "$0" | sed 's/^# \{0,1\}//'
}

fail() {
  printf 'package: %s\n' "$1" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) version="${2-}"; shift 2 ;;
    --os) os="${2-}"; shift 2 ;;
    --arch) arch="${2-}"; shift 2 ;;
    --variant) variant="${2-}"; shift 2 ;;
    --binary) binary="${2-}"; shift 2 ;;
    --outdir) outdir="${2-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) fail "unknown argument $1: run scripts/release/package.sh --help" ;;
  esac
done

[ -n "$version" ] || fail "--version is required, for example --version 0.1.0-rc.2"
[ -n "$os" ] || fail "--os is required: linux or windows"
[ -n "$arch" ] || fail "--arch is required, for example --arch amd64"
[ -n "$variant" ] || fail "--variant is required: cli or ui"
[ -n "$binary" ] || fail "--binary is required: the path of the compiled binary"
[ -n "$outdir" ] || fail "--outdir is required: the directory of the archives"

case "$version" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) fail "--version $version is not a semantic version such as 0.1.0-rc.2" ;;
esac
case "$os" in
  linux|windows) ;;
  *) fail "--os $os is not supported: choose linux or windows" ;;
esac
case "$arch" in
  amd64|arm64) ;;
  *) fail "--arch $arch is not supported: choose amd64 or arm64" ;;
esac
case "$variant" in
  cli|ui) ;;
  *) fail "--variant $variant is not supported: choose cli or ui" ;;
esac
[ -f "$binary" ] || fail "--binary $binary does not exist: build the binary before packaging it"

if [ "$os" = "windows" ]; then
  executable="codixai.exe"
  archive_extension="zip"
else
  executable="codixai"
  archive_extension="tar.gz"
fi

name="codixai_${version}_${os}_${arch}_${variant}"
staging="${outdir}/${name}"
archive="${outdir}/${name}.${archive_extension}"

rm -rf "$staging"
mkdir -p "$staging"

install -m 0755 "$binary" "${staging}/${executable}"
install -m 0644 LICENSE "${staging}/LICENSE"

cat > "${staging}/VERSION" <<EOF
version: ${version}
os: ${os}
arch: ${arch}
variant: ${variant}
repository: https://github.com/Barceloch/Codixai
EOF

cat > "${staging}/README.md" <<EOF
# Codixai ${version} — ${os}/${arch}, ${variant} variant

Codixai verifies the changes of a Git repository against a deterministic rule
catalog. It reads the repository, it never executes its code, it never invokes
a package manager and it never accesses the network.

This package is built from the source of the tag \`v${version}\` by the release
workflow of https://github.com/Barceloch/Codixai.

A Spanish guide is included as README.es.md and in the repository at
https://github.com/Barceloch/Codixai/blob/feat/phase-0-1-verifier/docs/guides/codixai-usage.es.md

## Contents

| File | What it is |
| ---- | ---------- |
| \`${executable}\` | The binary: the command line and, in the \`ui\` variant, the desktop window. |
| \`VERSION\` | The version, the platform and the variant of this package. |
| \`LICENSE\` | The licence of the project. |
| \`README.md\` | This file. |
| \`README.es.md\` | The same instructions in Spanish. |
EOF

if [ "$variant" = "cli" ]; then
  cat >> "${staging}/README.md" <<EOF

## Requirements

- ${os} on ${arch}.
EOF
  if [ "$os" = "linux" ]; then
    cat >> "${staging}/README.md" <<'EOF'
- A Linux kernel on x86_64. The binary is static (`CGO_ENABLED=0`), so it does
  not need Go and it does not depend on the version of the C library of the
  distribution.
EOF
  else
    cat >> "${staging}/README.md" <<'EOF'
- Windows 10 or later on x86_64. It does not need Go and it does not need the
  WebView2 runtime: this package has no graphical interface.
EOF
  fi
  cat >> "${staging}/README.md" <<EOF

## Install and run

The \`cli\` variant has no window: \`codixai ui\` answers with an error that
explains how to obtain the variant with the interface.

EOF
  if [ "$os" = "linux" ]; then
    cat >> "${staging}/README.md" <<EOF
\`\`\`bash
mkdir -p ~/.local/bin
install -m 0755 ${executable} ~/.local/bin/${executable}
${executable} --help
${executable} version
\`\`\`
EOF
  else
    cat >> "${staging}/README.md" <<EOF
\`\`\`powershell
.\\${executable} --help
.\\${executable} version
\`\`\`
EOF
  fi
  cat >> "${staging}/README.md" <<'EOF'

## First commands

```bash
codixai audit --diff --worktree        # audit the uncommitted changes
codixai audit --diff --base main       # audit the range main..HEAD
codixai audit --diff --worktree --format json
codixai report --format sarif --base main
codixai verify --base main --claims claims.json
codixai baseline create --base main
codixai init
```

Exit codes: `0` success, `1` execution error, `2` invalid arguments, `3`
findings that reach the `fail_on` threshold.

## Verify the download

```bash
sha256sum -c SHA256SUMS --ignore-missing
```
EOF
else
  cat >> "${staging}/README.md" <<EOF

## Requirements

- ${os} on ${arch}.
EOF
  if [ "$os" = "linux" ]; then
    cat >> "${staging}/README.md" <<'EOF'
- GTK 3 and WebKitGTK 4.1 at run time:

  ```bash
  # Ubuntu 24.04, Debian 13 and derivatives
  sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
  # Fedora
  sudo dnf install gtk3 webkit2gtk4.1
  # Arch and Manjaro
  sudo pacman -S gtk3 webkit2gtk-4.1
  ```

- glibc 2.39 or later. The binary is built in Ubuntu 24.04 because the window
  links against the system libraries: a binary built on a rolling release such
  as Arch does not start on Debian and Ubuntu.
EOF
  else
    cat >> "${staging}/README.md" <<'EOF'
- The Microsoft Edge WebView2 runtime. Windows 11 ships it; on Windows 10 it
  may be installed by the Edge updater or from
  https://developer.microsoft.com/microsoft-edge/webview2/
EOF
  fi
  cat >> "${staging}/README.md" <<EOF

## Install and run

The \`${variant}\` variant is one binary: it holds the command line and the
window. \`codixai ui\` opens the window.

EOF
  if [ "$os" = "linux" ]; then
    cat >> "${staging}/README.md" <<EOF
\`\`\`bash
chmod +x ${executable}
${executable} --help
${executable} ui --root /path/to/a/git/repository
\`\`\`
EOF
  else
    cat >> "${staging}/README.md" <<EOF
\`\`\`powershell
.\\${executable} --help
.\\${executable} ui --root C:\\path\\to\\a\\git\\repository
\`\`\`
EOF
  fi
  cat >> "${staging}/README.md" <<'EOF'

The window audits a repository with the same use case as `codixai audit`: it
never modifies the repository, it never executes its code and it never
accesses the network.

## Known limitations

- The window is not a replacement for the command line in a headless
  environment: it needs a running graphical session.
- The verification screen exports JSON only; the SARIF export of a
  verification is not implemented in this version.
- The binary is not signed. Windows SmartScreen may warn on the first run.

## Verify the download

```bash
sha256sum -c SHA256SUMS --ignore-missing
```
EOF
fi

cat > "${staging}/README.es.md" <<EOF
# Codixai ${version} — ${os}/${arch}, variante ${variant}

Codixai verifica los cambios de un repositorio Git contra un catálogo de
reglas determinista. Lee el repositorio, nunca ejecuta su código, nunca
invoca un gestor de paquetes y nunca accede a la red.

Este paquete se compila desde el código de la etiqueta \`v${version}\` con el
flujo de publicación de https://github.com/Barceloch/Codixai.

La guía completa en español está en el repositorio:
https://github.com/Barceloch/Codixai/blob/feat/phase-0-1-verifier/docs/guides/codixai-usage.es.md

## Contenido

| Archivo | Qué es |
| ------- | ------ |
| \`${executable}\` | El binario: la línea de comandos y, en la variante \`ui\`, la ventana de escritorio. |
| \`VERSION\` | La versión, la plataforma y la variante de este paquete. |
| \`LICENSE\` | La licencia del proyecto. |
| \`README.md\` | Las mismas instrucciones en inglés. |
| \`README.es.md\` | Este archivo. |
EOF

if [ "$variant" = "cli" ]; then
  cat >> "${staging}/README.es.md" <<'EOF'

## Requisitos

- Linux o Windows sobre x86_64, según el paquete.
- No hace falta Go. En Linux el binario es estático (`CGO_ENABLED=0`), así que
  no depende de la versión de la biblioteca C de la distribución.
- Esta variante no tiene ventana: `codixai ui` responde con un error que
  explica cómo obtener la variante con interfaz.

## Instalar y ejecutar

Linux:

```bash
mkdir -p ~/.local/bin
install -m 0755 codixai ~/.local/bin/codixai
codixai --help
codixai version
```

Windows PowerShell:

```powershell
.\codixai.exe --help
.\codixai.exe version
```

## Primeros comandos

```bash
codixai audit --diff --worktree        # audita los cambios sin confirmar
codixai audit --diff --base main       # audita el rango main..HEAD
codixai audit --diff --worktree --format json
codixai report --format sarif --base main
codixai verify --base main --claims claims.json
codixai baseline create --base main
codixai init
```

Códigos de salida: `0` éxito, `1` error de ejecución, `2` argumentos
inválidos, `3` hallazgos que alcanzan el umbral `fail_on`.

## Verificar la descarga

```bash
sha256sum -c SHA256SUMS --ignore-missing
```
EOF
else
  cat >> "${staging}/README.es.md" <<EOF

## Requisitos

- ${os} sobre ${arch}.
EOF
  if [ "$os" = "linux" ]; then
    cat >> "${staging}/README.es.md" <<'EOF'
- GTK 3 y WebKitGTK 4.1 en tiempo de ejecución:

  ```bash
  # Ubuntu 24.04, Debian 13 y derivados
  sudo apt install libgtk-3-0 libwebkit2gtk-4.1-0
  # Fedora
  sudo dnf install gtk3 webkit2gtk4.1
  # Arch y Manjaro
  sudo pacman -S gtk3 webkit2gtk-4.1
  ```

- glibc 2.39 o posterior. El binario se compila en Ubuntu 24.04 porque la
  ventana enlaza contra las bibliotecas del sistema: un binario compilado en
  una distribución de liberación continua como Arch no arranca en Debian ni
  en Ubuntu.
EOF
  else
    cat >> "${staging}/README.es.md" <<'EOF'
- El runtime Microsoft Edge WebView2. Windows 11 lo incluye; en Windows 10
  puede instalarlo el actualizador de Edge o descargarse de
  https://developer.microsoft.com/microsoft-edge/webview2/
EOF
  fi
  cat >> "${staging}/README.es.md" <<EOF

## Instalar y ejecutar

La variante \`${variant}\` es un solo binario: contiene la línea de comandos y
la ventana. \`codixai ui\` abre la ventana.

Linux:

\`\`\`bash
chmod +x ${executable}
${executable} --help
${executable} ui --root /ruta/a/un/repositorio/git
\`\`\`

Windows PowerShell:

\`\`\`powershell
.\\${executable} --help
.\\${executable} ui --root C:\\ruta\\a\\un\\repositorio\\git
\`\`\`

La ventana audita un repositorio con el mismo caso de uso que \`codixai
audit\`: nunca modifica el repositorio, nunca ejecuta su código y nunca accede
a la red.

## Limitaciones conocidas

- La ventana no sustituye a la línea de comandos en un entorno sin sesión
  gráfica: necesita un servidor gráfico en ejecución.
- La pantalla de verificación exporta solo JSON; la exportación SARIF de una
  verificación no está implementada en esta versión.
- El binario no está firmado. Windows SmartScreen puede avisar en la primera
  ejecución.

## Verificar la descarga

\`\`\`bash
sha256sum -c SHA256SUMS --ignore-missing
\`\`\`
EOF
fi

# Freeze the timestamps so that packaging the same bytes twice produces the
# same archive: the release must not depend on the clock of the machine.
find "$staging" -exec touch -t 198001010000.00 {} +

mkdir -p "$outdir"
rm -f "$archive"
if [ "$archive_extension" = "zip" ]; then
  command -v zip >/dev/null 2>&1 || fail "zip is required to package a windows artifact: install it and run the script again"
  ( cd "$outdir" && zip -qrX "${name}.zip" "${name}" )
else
  tar --sort=name --mtime='1980-01-01 00:00:00 UTC' --owner=0 --group=0 \
    --numeric-owner -czf "$archive" -C "$outdir" "$name"
fi

digest=$(sha256sum "$archive" | awk '{print $1}')
printf '%s  %s\n' "$digest" "$(basename "$archive")"
