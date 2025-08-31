package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	handlers "go-call-tracer/internal/server"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// CLI flags
	mode := flag.String("mode", "stdio", "Transport mode: stdio or sse")
	addr := flag.String("addr", ":8080", "HTTP listen address for SSE")
	path := flag.String("path", "/mcp/sse", "HTTP path for SSE connections")
	flag.Parse()

	// Create a new MCP server
	s := server.NewMCPServer(
		"Go Code Tracer 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register all tools and their corresponding handlers
	handlers.RegisterTools(s)

	switch *mode {
	case "stdio":
		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	case "sse":
		// Start an HTTP server and mount the MCP SSE handler at the given path.
		// The mcp-go library exposes SSE serving helpers; try ServeSSE first.
		// If unavailable, fall back to exposing the handler via http.
		// Preferred: server.ServeSSE(s, *addr)
		// Fallback: use server.SSEHandler(s) if provided by the library.

		// Create an SSE server and mount its handlers at the provided path.
		sseServer := server.NewSSEServer(s)

		// Compute message endpoint path from SSE path. If user supplied
		// a path like "/mcp/sse" we'll register message handler at
		// "/mcp/message". Otherwise we append "/message".
		ssePath := *path
		messagePath := strings.Replace(ssePath, "/sse", "/message", 1)
		if messagePath == ssePath {
			messagePath = strings.TrimRight(ssePath, "/") + "/message"
		}

		http.Handle(ssePath, sseServer.SSEHandler())
		http.Handle(messagePath, sseServer.MessageHandler())

		log.Printf("Starting SSE server on %s (SSE: %s, Message: %s)", *addr, ssePath, messagePath)
		if err := http.ListenAndServe(*addr, nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}
