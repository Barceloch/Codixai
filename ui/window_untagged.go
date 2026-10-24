//go:build ui && !production && !dev && !bindings

package ui

import "fmt"

// runWindow explains that the window needs the build tags of Wails v2.
//
// Wails v2 compiles its application layer only with the production or dev
// build tag: without them, wails.Run returns "Wails applications will not
// build without the correct build tags", which does not tell the person
// what to type. The file replaces that message with the command that
// produces a working window.
//
// The webkit2_41 tag is not part of the message on purpose: it selects the
// WebKitGTK version of the target system, so it belongs to the build
// instructions of the platform and not to this error.
func runWindow(_ *App, _ string) error {
	return fmt.Errorf("this binary was built with the ui tag but without the build tags of Wails v2: " +
		"build the window with 'go build -tags \"ui production\" ./cmd/codixai', " +
		"or install a release binary, which is built with them")
}
