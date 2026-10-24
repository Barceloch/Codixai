//go:build ui && (production || dev) && !bindings

package ui

import (
	"github.com/wailsapp/wails/v2"
)

// runWindow opens the Wails v2 window of the variant.
//
// The static frontend of frontend/dist is served by the asset server, which
// also serves the runtime scripts (/wails/ipc.js and /wails/runtime.js)
// that the Svelte application imports through the generated wrappers.
//
// The file is compiled only with the Wails build tags. Wails v2 gates its
// whole application layer behind production or dev, so the window does not
// exist in a build with the ui tag alone.
func runWindow(app *App, title string) error {
	return wails.Run(windowOptions(app, title))
}
