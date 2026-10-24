//go:build ui && bindings

package ui

import (
	"github.com/wailsapp/wails/v2"
)

// runWindow generates the TypeScript bindings of the bound struct.
//
// Wails v2 compiles its application layer with the bindings tag as a
// generator: wails.Run walks the Bind list and writes wailsjs/ to the
// wailsjsdir of wails.json. The command that uses this path is
// "go run -tags \"ui bindings\" ./cmd/codixai ui --root .", which never
// opens a window.
//
// The file exists because the entry point of Codixai is cmd/codixai and the
// Wails CLI builds the main package at the root of the project, which does
// not exist here (see P7-04).
func runWindow(app *App, title string) error {
	return wails.Run(windowOptions(app, title))
}
