# Mailpit Enhancement Design: Postmark API Emulation & MCP Server

## Executive Summary

This design document outlines the implementation of two new features for Mailpit:
1. **Postmark API Emulation**: Allow Mailpit to act as a drop-in replacement for Postmark during testing
2. **MCP Server Integration**: Enable external tools to read and analyze messages via the Model Context Protocol

## 1. Postmark API Emulation Endpoint

### Overview
Implement Postmark-compatible API endpoints to allow applications configured for Postmark to seamlessly send emails to Mailpit during development and testing.

### Design Architecture

```
┌─────────────────┐         ┌──────────────────────┐
│   Application   │ ──POST──▶│  Mailpit Postmark   │
│ (Postmark SDK)  │         │    API Emulator     │
└─────────────────┘         └──────────────────────┘
                                      │
                                      ▼
                            ┌──────────────────────┐
                            │  Mailpit Storage     │
                            │    (SQLite)          │
                            └──────────────────────┘
```

### Implementation Details

#### File Structure
```
server/
├── postmark/
│   ├── postmark.go        # Main Postmark API handler
│   ├── structs.go         # Postmark request/response structs
│   ├── converter.go       # Convert Postmark format to Mailpit format
│   └── postmark_test.go   # Unit tests
```

#### API Endpoints

##### 1. Single Email Send
- **Path**: `/postmark/email`
- **Method**: POST
- **Authentication**: Header `X-Postmark-Server-Token` (configurable)

**Request Structure**:
```go
type PostmarkEmailRequest struct {
    From          string                 `json:"From"`
    To            string                 `json:"To"`
    Cc            string                 `json:"Cc,omitempty"`
    Bcc           string                 `json:"Bcc,omitempty"`
    Subject       string                 `json:"Subject"`
    Tag           string                 `json:"Tag,omitempty"`
    HtmlBody      string                 `json:"HtmlBody,omitempty"`
    TextBody      string                 `json:"TextBody,omitempty"`
    ReplyTo       string                 `json:"ReplyTo,omitempty"`
    Headers       []PostmarkHeader       `json:"Headers,omitempty"`
    Attachments   []PostmarkAttachment   `json:"Attachments,omitempty"`
    MessageStream string                 `json:"MessageStream,omitempty"`
    Metadata      map[string]string      `json:"Metadata,omitempty"`
}

type PostmarkHeader struct {
    Name  string `json:"Name"`
    Value string `json:"Value"`
}

type PostmarkAttachment struct {
    Name        string `json:"Name"`
    Content     string `json:"Content"`     // Base64 encoded
    ContentType string `json:"ContentType"`
    ContentID   string `json:"ContentID,omitempty"`
}
```

**Response Structure**:
```go
type PostmarkEmailResponse struct {
    To          string    `json:"To"`
    SubmittedAt time.Time `json:"SubmittedAt"`
    MessageID   string    `json:"MessageID"`
    ErrorCode   int       `json:"ErrorCode"`
    Message     string    `json:"Message"`
}
```

##### 2. Batch Email Send
- **Path**: `/postmark/email/batch`
- **Method**: POST
- **Authentication**: Same as single email

**Request Structure**:
```go
type PostmarkBatchRequest []PostmarkEmailRequest // Max 500 messages
```

**Response Structure**:
```go
type PostmarkBatchResponse []PostmarkEmailResponse
```

#### Core Handler Implementation

```go
// server/postmark/postmark.go
package postmark

import (
    "encoding/json"
    "net/http"
    "github.com/axllent/mailpit/internal/storage"
    "github.com/axllent/mailpit/internal/smtpd"
)

func SendEmailHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Validate authentication token
    token := r.Header.Get("X-Postmark-Server-Token")
    if !validateToken(token) {
        sendErrorResponse(w, 401, "Invalid token")
        return
    }
    
    // 2. Parse request body
    var req PostmarkEmailRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendErrorResponse(w, 422, "Invalid JSON")
        return
    }
    
    // 3. Convert to Mailpit message format
    message := convertToMailpitMessage(req)
    
    // 4. Store in database
    id, err := storage.Store(message)
    if err != nil {
        sendErrorResponse(w, 500, "Storage error")
        return
    }
    
    // 5. Send success response
    resp := PostmarkEmailResponse{
        To:          req.To,
        SubmittedAt: time.Now(),
        MessageID:   id,
        ErrorCode:   0,
        Message:     "OK",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

#### Configuration

Add to `config/config.go`:
```go
var (
    // Enable Postmark API emulation
    EnablePostmarkAPI bool
    
    // Postmark API token for authentication
    PostmarkAPIToken string
    
    // Accept any Postmark token (for testing)
    PostmarkAcceptAnyToken bool
)
```

Command-line flags:
```
--postmark-api           Enable Postmark API emulation
--postmark-token         Set Postmark API authentication token
--postmark-accept-any    Accept any authentication token
```

#### Router Integration

Update `server/server.go`:
```go
// In apiRoutes() function
if config.EnablePostmarkAPI {
    r.HandleFunc("/postmark/email", postmarkAuthMiddleware(postmark.SendEmailHandler)).Methods("POST")
    r.HandleFunc("/postmark/email/batch", postmarkAuthMiddleware(postmark.SendBatchHandler)).Methods("POST")
}
```

---

## 2. MCP Server Endpoint

### Overview
Implement a Model Context Protocol (MCP) server that exposes Mailpit's message data to AI assistants and automation tools for debugging and analysis.

### Design Architecture

```
┌──────────────────┐         ┌──────────────────────┐
│   MCP Client     │ ◀──RPC──▶│   Mailpit MCP       │
│ (Claude, etc.)   │         │      Server          │
└──────────────────┘         └──────────────────────┘
                                      │
                                      ▼
                            ┌──────────────────────┐
                            │  Mailpit Storage     │
                            │    (SQLite)          │
                            └──────────────────────┘
```

### Implementation Details

#### File Structure
```
server/
├── mcp/
│   ├── server.go          # MCP server implementation
│   ├── tools.go           # MCP tool definitions
│   ├── transport.go       # Transport layer (stdio/HTTP)
│   └── mcp_test.go        # Unit tests
```

#### MCP Tools

##### 1. List Messages Tool
```go
type ListMessagesInput struct {
    Limit  int    `json:"limit" jsonschema:"maximum number of messages to return"`
    Search string `json:"search,omitempty" jsonschema:"search query"`
    Tag    string `json:"tag,omitempty" jsonschema:"filter by tag"`
}

type ListMessagesOutput struct {
    Messages []MessageSummary `json:"messages"`
    Total    int              `json:"total"`
}

type MessageSummary struct {
    ID      string    `json:"id"`
    From    string    `json:"from"`
    To      []string  `json:"to"`
    Subject string    `json:"subject"`
    Date    time.Time `json:"date"`
    Tags    []string  `json:"tags"`
}
```

##### 2. Get Message Tool
```go
type GetMessageInput struct {
    ID         string `json:"id" jsonschema:"required,message ID"`
    IncludeRaw bool   `json:"includeRaw,omitempty" jsonschema:"include raw message"`
}

type GetMessageOutput struct {
    ID          string            `json:"id"`
    From        string            `json:"from"`
    To          []string          `json:"to"`
    Subject     string            `json:"subject"`
    Date        time.Time         `json:"date"`
    HTMLBody    string            `json:"htmlBody,omitempty"`
    TextBody    string            `json:"textBody,omitempty"`
    Headers     map[string]string `json:"headers"`
    Attachments []AttachmentInfo  `json:"attachments"`
    Raw         string            `json:"raw,omitempty"`
}
```

##### 3. Search Messages Tool
```go
type SearchMessagesInput struct {
    Query     string    `json:"query" jsonschema:"required,search query"`
    DateFrom  time.Time `json:"dateFrom,omitempty"`
    DateTo    time.Time `json:"dateTo,omitempty"`
    Limit     int       `json:"limit,omitempty"`
}

type SearchMessagesOutput struct {
    Results []MessageSummary `json:"results"`
    Total   int              `json:"total"`
}
```

##### 4. Analyze Message Tool
```go
type AnalyzeMessageInput struct {
    ID string `json:"id" jsonschema:"required,message ID"`
}

type AnalyzeMessageOutput struct {
    ID           string           `json:"id"`
    HTMLCheck    HTMLCheckResult  `json:"htmlCheck,omitempty"`
    LinkCheck    LinkCheckResult  `json:"linkCheck,omitempty"`
    SpamScore    float64          `json:"spamScore,omitempty"`
    Deliverability string         `json:"deliverability"`
}
```

#### MCP Server Implementation

```go
// server/mcp/server.go
package mcpserver

import (
    "context"
    "github.com/modelcontextprotocol/go-sdk/pkg/mcp"
    "github.com/axllent/mailpit/internal/storage"
)

func InitMCPServer() *mcp.Server {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "mailpit-mcp",
        Version: "v1.0.0",
    }, nil)
    
    // Register tools
    registerListMessagesTool(server)
    registerGetMessageTool(server)
    registerSearchMessagesTool(server)
    registerAnalyzeMessageTool(server)
    
    return server
}

func registerListMessagesTool(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "list_messages",
        Description: "List recent messages in Mailpit",
    }, ListMessages)
}

func ListMessages(ctx context.Context, req *mcp.CallToolRequest, input ListMessagesInput) (*mcp.CallToolResult, ListMessagesOutput, error) {
    // Query storage
    messages, total, err := storage.List(input.Limit, 0, input.Search)
    if err != nil {
        return nil, ListMessagesOutput{}, err
    }
    
    // Convert to output format
    output := ListMessagesOutput{
        Messages: convertToSummaries(messages),
        Total:    total,
    }
    
    return nil, output, nil
}
```

#### Transport Options

##### Option 1: stdio Transport (for local tools)
```go
func RunMCPStdio() {
    server := InitMCPServer()
    server.Run(context.Background(), &mcp.StdioTransport{})
}
```

##### Option 2: HTTP Transport (for remote access)
```go
func RunMCPHTTP(addr string) {
    server := InitMCPServer()
    
    http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
        // Handle MCP over HTTP/WebSocket
        server.ServeHTTP(w, r)
    })
    
    http.ListenAndServe(addr, nil)
}
```

#### Configuration

Add to `config/config.go`:
```go
var (
    // Enable MCP server
    EnableMCPServer bool
    
    // MCP server transport type
    MCPTransport string // "stdio" or "http"
    
    // MCP HTTP server address
    MCPHTTPAddr string
    
    // MCP authentication token
    MCPAuthToken string
)
```

Command-line flags:
```
--mcp-server             Enable MCP server
--mcp-transport          MCP transport type (stdio|http) [default: stdio]
--mcp-http-addr          MCP HTTP server address [default: :8026]
--mcp-auth-token         MCP authentication token
```

#### Integration with Main Application

Update `main.go`:
```go
func main() {
    // ... existing initialization ...
    
    if config.EnableMCPServer {
        if config.MCPTransport == "stdio" {
            go mcpserver.RunMCPStdio()
        } else if config.MCPTransport == "http" {
            go mcpserver.RunMCPHTTP(config.MCPHTTPAddr)
        }
    }
    
    // ... rest of application ...
}
```

---

## Testing Strategy

### Postmark API Testing
1. **Unit Tests**: Test request/response conversion, authentication
2. **Integration Tests**: Test full email flow through Postmark API
3. **Compatibility Tests**: Verify against official Postmark SDK

### MCP Server Testing
1. **Unit Tests**: Test each MCP tool independently
2. **Integration Tests**: Test with MCP client SDK
3. **Performance Tests**: Verify handling of large message volumes

---

## Security Considerations

### Postmark API
- Token-based authentication
- Rate limiting to prevent abuse
- Input validation and sanitization
- Maximum payload size enforcement

### MCP Server
- Authentication for HTTP transport
- Read-only access (no message modification)
- Query result limits to prevent resource exhaustion
- Sanitized output to prevent data leakage

---

## Migration Path

### Phase 1: Implementation
1. Implement Postmark API emulation
2. Implement MCP server with basic tools
3. Add comprehensive testing

### Phase 2: Enhancement
1. Add more Postmark API endpoints (webhooks, stats)
2. Add advanced MCP tools (bulk operations, analytics)
3. Performance optimization

### Phase 3: Documentation
1. Update API documentation
2. Create usage examples
3. Add configuration guides

---

## Dependencies

### New Go Dependencies
```go
// For MCP server
github.com/modelcontextprotocol/go-sdk v0.1.0

// Existing dependencies used
github.com/gorilla/mux        // Routing
github.com/jhillyerd/enmime   // Email parsing
```

---

## Configuration Examples

### Basic Setup
```bash
# Enable both features
mailpit --postmark-api --postmark-accept-any \
        --mcp-server --mcp-transport stdio
```

### Production Setup
```bash
# With authentication
mailpit --postmark-api --postmark-token="test-token-12345" \
        --mcp-server --mcp-transport http \
        --mcp-http-addr=":8026" \
        --mcp-auth-token="mcp-secret-token"
```

### Docker Compose
```yaml
services:
  mailpit:
    image: axllent/mailpit
    ports:
      - "1025:1025"  # SMTP
      - "8025:8025"  # Web UI
      - "8026:8026"  # MCP HTTP
    environment:
      MP_POSTMARK_API: "true"
      MP_POSTMARK_ACCEPT_ANY: "true"
      MP_MCP_SERVER: "true"
      MP_MCP_TRANSPORT: "http"
```

---

## Success Metrics

1. **Postmark API**: 100% compatibility with basic Postmark SDK operations
2. **MCP Server**: Successful integration with Claude and other MCP clients
3. **Performance**: No degradation in existing Mailpit performance
4. **Testing**: >80% code coverage for new features