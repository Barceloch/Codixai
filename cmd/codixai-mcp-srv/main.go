package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/barceloch/codixai/internal/integrations/mcpserver"
	"github.com/mark3labs/mcp-go/server"
)

var (
	version = "dev"
)

func main() {
	var httpAddr string
	flag.StringVar(&httpAddr, "http", "", "Use streamable HTTP transport at this address")
	flag.Parse()

	s := mcpserver.New(version)
	if httpAddr != "" {
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(httpAddr); err != nil {
			fmt.Fprintf(os.Stderr, "codixai-mcp: %v\n", err)
			os.Exit(1)
		}
		return
	}

	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(nil, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "codixai-mcp: %v\n", err)
		os.Exit(1)
	}
}
