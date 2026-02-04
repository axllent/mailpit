// Package main provides the entry point for the Mailpit MCP server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/axllent/mailpit/mcp/internal/server"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Load configuration from environment
	cfg := server.DefaultConfig()
	cfg.LoadFromEnv()

	// Create Mailpit API client
	mailpitClient := cfg.NewMailpitClient()

	// Create MCP server
	mcpServer := server.New(mailpitClient)

	// Run with the appropriate transport
	switch cfg.Transport {
	case "stdio":
		runSTDIO(mcpServer)
	case "http":
		runHTTP(mcpServer, cfg)
	default:
		log.Fatalf("Unknown transport: %s (use 'stdio' or 'http')", cfg.Transport)
	}
}

// runSTDIO runs the MCP server over standard input/output.
func runSTDIO(s *mcp.Server) {
	// Log to stderr since stdout is used for MCP communication
	log.SetOutput(os.Stderr)
	log.Printf("Starting Mailpit MCP server (STDIO transport, version %s)", server.Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	if err := s.Run(ctx, mcp.NewStdioTransport()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runHTTP runs the MCP server over HTTP with SSE.
func runHTTP(s *mcp.Server, cfg *server.Config) {
	log.Printf("Starting Mailpit MCP server (HTTP transport, version %s)", server.Version)
	log.Printf("Listening on %s:%d", cfg.HTTPHost, cfg.HTTPPort)
	log.Printf("Connecting to Mailpit at %s", cfg.MailpitURL)

	// Create SSE handler
	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s
	})

	// Set up routes
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down HTTP server...")
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		cancel()
	}()

	// Start server
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
