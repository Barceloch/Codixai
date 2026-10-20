// Command codixai verifies the changes of a repository against the rule
// catalog of Codixai.
//
// The binary is the composition root of the tool: it wires the use cases of
// the application layer with their providers and it delegates the
// presentation to the internal cli package.
//
// The version of the tool is injected at build time with the linker flag
// "-X main.version=<version>". It stays "dev" when the binary is built
// without it.
package main

import (
	"context"
	"os"

	"github.com/barceloch/codixai/internal/cli"
)

// version is the version of the tool.
//
// It is injected at build time:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/codixai
var version = "dev"

func main() {
	// The version of the binary is the version of the CLI.
	cli.Version = version
	os.Exit(cli.Execute(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
