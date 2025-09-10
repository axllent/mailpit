package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/gorilla/websocket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	// server is the MCP server instance
	server *mcp.Server

	// upgrader for WebSocket connections
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin for development
			// In production, you might want to restrict this
			return true
		},
	}
)

// Start starts the MCP server based on configuration
func Start() {
	if !config.EnableMCPServer {
		return
	}

	// Initialize the server
	InitMCPServer()

	switch config.MCPTransport {
	case "stdio":
		RunMCPStdio()
	case "websocket", "http":
		// For WebSocket, we need to set up the HTTP server
		// This would typically be integrated with the main HTTP server
		logger.Log().Info("[mcp] WebSocket transport will be available via HTTP server")
	default:
		logger.Log().Errorf("[mcp] unknown transport: %s", config.MCPTransport)
	}
}

// InitMCPServer initializes the MCP server with all tools
func InitMCPServer() *mcp.Server {
	server = mcp.NewServer("mailpit-mcp", config.Version, nil)

	// Register tools
	registerListMessagesTool(server)
	registerGetMessageTool(server)
	registerSearchMessagesTool(server)
	registerAnalyzeMessageTool(server)

	logger.Log().Info("[mcp] server initialized with 4 tools")

	return server
}

// registerListMessagesTool registers the list_messages tool
func registerListMessagesTool(s *mcp.Server) {
	s.AddTools(mcp.NewServerTool(
		"list_messages",
		"List recent messages in Mailpit with optional filtering by tag or search query",
		ListMessages,
	))
}

// registerGetMessageTool registers the get_message tool
func registerGetMessageTool(s *mcp.Server) {
	s.AddTools(mcp.NewServerTool(
		"get_message",
		"Get full details of a specific message by ID, optionally including raw message content",
		GetMessage,
	))
}

// registerSearchMessagesTool registers the search_messages tool
func registerSearchMessagesTool(s *mcp.Server) {
	s.AddTools(mcp.NewServerTool(
		"search_messages",
		"Search messages using Mailpit's search syntax with optional date filters",
		SearchMessages,
	))
}

// registerAnalyzeMessageTool registers the analyze_message tool
func registerAnalyzeMessageTool(s *mcp.Server) {
	s.AddTools(mcp.NewServerTool(
		"analyze_message",
		"Analyze a message for HTML compatibility, link validity, and spam score",
		AnalyzeMessage,
	))
}

// RunMCPStdio runs the MCP server over stdio transport
func RunMCPStdio() {
	if !config.EnableMCPServer {
		return
	}

	logger.Log().Info("[mcp] starting server on stdio transport")

	server := InitMCPServer()
	
	// Create stdio transport
	trans := mcp.NewStdioTransport()

	// Run server
	if err := server.Run(context.Background(), trans); err != nil {
		logger.Log().Errorf("[mcp] stdio server error: %v", err)
	}
}

// RunMCPHTTP runs the MCP server over HTTP/WebSocket transport
func RunMCPHTTP(addr string) {
	if !config.EnableMCPServer {
		return
	}

	logger.Log().Infof("[mcp] starting server on HTTP transport at %s", addr)

	// Initialize the global server
	InitMCPServer()

	// Set up HTTP handlers
	http.HandleFunc("/mcp", handleMCPWebSocket)
	http.HandleFunc("/mcp/rpc", handleMCPHTTP)
	http.HandleFunc("/mcp/health", handleHealth)

	// Start HTTP server
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Log().Fatalf("[mcp] HTTP server error: %v", err)
	}
}

// handleMCPWebSocket handles WebSocket connections for MCP
func handleMCPWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check authentication if configured
	if config.MCPAuthToken != "" {
		token := r.Header.Get("Authorization")
		expectedToken := fmt.Sprintf("Bearer %s", config.MCPAuthToken)
		if token != expectedToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log().Errorf("[mcp] WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	logger.Log().Debugf("[mcp] WebSocket connection from %s", r.RemoteAddr)

	// Create WebSocket transport wrapper
	trans := &wsTransport{conn: conn}

	// Run server with this connection
	if err := server.Run(context.Background(), trans); err != nil {
		logger.Log().Errorf("[mcp] WebSocket server error: %v", err)
	}
}

// handleMCPHTTP handles HTTP JSON-RPC requests for MCP
func handleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	// Check authentication if configured
	if config.MCPAuthToken != "" {
		token := r.Header.Get("Authorization")
		expectedToken := fmt.Sprintf("Bearer %s", config.MCPAuthToken)
		if token != expectedToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON-RPC request
	var req json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Process through MCP server
	// Note: This is a simplified implementation
	// A full implementation would need to handle JSON-RPC properly
	logger.Log().Debugf("[mcp] HTTP request from %s", r.RemoteAddr)

	// For now, return a not implemented response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32601,
			"message": "HTTP JSON-RPC not fully implemented",
		},
		"id": nil,
	})
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": config.Version,
		"tools": []string{
			"list_messages",
			"get_message",
			"search_messages",
			"analyze_message",
		},
	})
}

// wsTransport is a WebSocket transport wrapper for MCP
type wsTransport struct {
	conn *websocket.Conn
}

// Connect implements the Transport interface
func (t *wsTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	// Return a WebSocket connection wrapper with unique ID
	return &wsConnection{
		conn: t.conn,
		id:   fmt.Sprintf("ws-%d", time.Now().UnixNano()),
	}, nil
}

// wsConnection wraps a WebSocket connection as an MCP Connection
type wsConnection struct {
	conn *websocket.Conn
	id   string
}

// Read reads a JSON-RPC message
func (c *wsConnection) Read(ctx context.Context) (mcp.JSONRPCMessage, error) {
	var msg mcp.JSONRPCMessage
	err := c.conn.ReadJSON(&msg)
	return msg, err
}

// Write writes a JSON-RPC message
func (c *wsConnection) Write(ctx context.Context, msg mcp.JSONRPCMessage) error {
	return c.conn.WriteJSON(msg)
}

// Close closes the connection
func (c *wsConnection) Close() error {
	return c.conn.Close()
}

// SessionID returns the session ID
func (c *wsConnection) SessionID() string {
	return c.id
}