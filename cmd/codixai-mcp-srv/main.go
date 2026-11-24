package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		cancel()
	}()

	stdioServer := server.NewStdioServer(s)
	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "codixai-mcp: %v\n", err)
		os.Exit(1)
	}
}
