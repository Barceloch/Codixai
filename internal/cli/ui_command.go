//go:build !ui

package cli

import (
	"context"
)

// newUICommand builds the ui command of the CLI only variant.
//
// The window, its assets and the Wails dependency belong to the ui build
// variant: this file is the whole command in the default build, and it ends
// with a usage error that explains how to obtain the variant with the
// interface.
func newUICommand(env *Env) *Command {
	command := &Command{
		Use:   "ui",
		Short: "Open the graphical interface (not available in this build).",
		Long: "Open the graphical interface of Codixai.\n" +
			"\n" +
			"The command is not part of the CLI only build. The interface, its\n" +
			"assets and the Wails dependency are compiled only with the ui build\n" +
			"tag, so that the default build needs neither Node.js nor Wails nor\n" +
			"the libraries of the webview.\n" +
			"\n" +
			"Build the variant with the interface with\n" +
			"'go build -tags \"ui production\" ./cmd/codixai', or install a release\n" +
			"binary, which holds the interface together with every other command.",
		Example: "go build -tags \"ui production\" ./cmd/codixai\n" +
			"./codixai ui",
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the ui command does not accept arguments, got %q", args[0])
		}
		return Usagef("the ui command is not available in this build: it is the CLI only variant. " +
			"Build the variant with the interface with 'go build -tags \"ui production\" ./cmd/codixai', " +
			"or install a release binary, which includes the interface")
	}
	return command
}
