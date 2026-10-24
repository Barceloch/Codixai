//go:build ui && (production || dev || bindings)

package ui

import (
	"github.com/barceloch/codixai/frontend"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// windowOptions returns the options of the Wails v2 window.
//
// The compiled frontend lives in frontend/dist and is embedded by the
// frontend package. The App pointer is bound so the exported methods reach
// the page as window.go.ui.App.
//
// The function is shared by the production, the development and the
// bindings builds, so the generated TypeScript bindings describe exactly
// the methods that the window exposes.
func windowOptions(app *App, title string) *options.App {
	return &options.App{
		Title:  title,
		Width:  1180,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: frontend.Assets,
		},
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	}
}
