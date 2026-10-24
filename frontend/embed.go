// Package frontend embeds the compiled Svelte frontend of the ui variant.
//
// The package holds no logic: it only exposes the files that Vite writes to
// frontend/dist, so the Wails asset server can serve them from the binary.
// The embed lives in its own package because go:embed cannot reach a parent
// directory, and the ui package lives in ui/ while the frontend source lives
// in frontend/.
//
// The default build of the tool never imports this package: the CLI only
// variant does not need Node.js, Wails or the compiled frontend.
package frontend

import "embed"

// Assets holds the compiled frontend.
//
// The dist directory is a build artifact. The repository keeps a
// dist/.gitkeep so that the package compiles before the first
// "pnpm run build"; the build itself writes index.html and the bundled
// assets there.
//
//go:embed all:dist
var Assets embed.FS
