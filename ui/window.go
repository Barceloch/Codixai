//go:build ui && (production || dev)

package ui

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// runWindow opens the Wails v2 window of the variant.
//
// The static frontend of the assets directory is served by the asset
// server, which also serves the runtime scripts (/wails/ipc.js and
// /wails/runtime.js) that the page loads. The App pointer is bound so the
// exported methods reach the page as window.go.<package>.App.
//
// The file is compiled only with the Wails build tags. Wails v2 gates its
// whole application layer behind production or dev, so the window does not
// exist in a build with the ui tag alone.
func runWindow(app *App, title string) error {
	return wails.Run(&options.App{
		Title:  title,
		Width:  1100,
		Height: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	})
}
