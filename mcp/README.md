# Mailpit MCP Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [Mailpit](https://mailpit.axe.dev/), enabling AI assistants to interact with email testing workflows.

## Overview

This MCP server provides AI-powered tools and agents (like Claude, Cursor, or VS Code with Copilot) the ability to:

- Browse, search, and manage emails in Mailpit
- Analyze email content, headers, and attachments
- Validate emails for HTML compatibility, broken links, and spam scores
- Send test emails and manage email tags
- Access pre-built prompts for common email testing workflows

## Installation

### Docker

```bash
docker run -d \
  --name mailpit-mcp \
  -e MAILPIT_URL=http://mailpit:8025 \
  -p 3000:3000 \
  amirhmoradi/mailpit-mcp:latest
```

### Docker Compose

```yaml
services:
  mailpit:
    image: axllent/mailpit:latest
    ports:
      - "8025:8025"
      - "1025:1025"

  mailpit-mcp:
    image: amirhmoradi/mailpit-mcp:latest
    environment:
      MAILPIT_URL: http://mailpit:8025
      MCP_TRANSPORT: http
    ports:
      - "3000:3000"
    depends_on:
      - mailpit
```

### Build from Source

```bash
cd mcp
go build -o mailpit-mcp-server ./cmd/mailpit-mcp-server
```

## Configuration

Configure the MCP server using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MAILPIT_URL` | Mailpit server URL | `http://localhost:8025` |
| `MAILPIT_AUTH_USER` | Basic auth username | (none) |
| `MAILPIT_AUTH_PASS` | Basic auth password | (none) |
| `MAILPIT_TIMEOUT` | Request timeout | `30s` |
| `MCP_TRANSPORT` | Transport mode: `stdio` or `http` | `stdio` |
| `MCP_HTTP_HOST` | HTTP server host | `127.0.0.1` |
| `MCP_HTTP_PORT` | HTTP server port | `3000` |
| `MCP_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |

## Usage with AI Tools

### Claude Desktop

Add to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "mailpit": {
      "command": "/path/to/mailpit-mcp-server",
      "env": {
        "MAILPIT_URL": "http://localhost:8025"
      }
    }
  }
}
```

### Cursor / VS Code

For HTTP transport (recommended for IDE integration):

```json
{
  "mcp.servers": {
    "mailpit": {
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

## Available Tools

### Message Management

| Tool | Description |
|------|-------------|
| `list_messages` | List messages with pagination |
| `search_messages` | Search messages using query syntax |
| `get_message` | Get full message details |
| `get_message_headers` | Get message headers |
| `get_message_source` | Get raw email source (RFC 2822) |
| `delete_messages` | Delete messages by ID |
| `delete_search` | Delete messages matching a search |
| `set_read_status` | Mark messages as read/unread |

### Content Tools

| Tool | Description |
|------|-------------|
| `get_message_html` | Get rendered HTML content |
| `get_message_text` | Get plain text content |
| `get_attachment` | Download an attachment |

### Validation Tools

| Tool | Description |
|------|-------------|
| `check_html` | Check HTML email client compatibility |
| `check_links` | Validate links in an email |
| `check_spam` | Get SpamAssassin analysis |

### Tag Management

| Tool | Description |
|------|-------------|
| `list_tags` | List all tags |
| `set_tags` | Set tags on messages |
| `rename_tag` | Rename a tag |
| `delete_tag` | Delete a tag |

### Testing Tools

| Tool | Description |
|------|-------------|
| `send_message` | Send a test email |
| `release_message` | Relay a message to external SMTP |
| `get_chaos` | Get chaos testing configuration |
| `set_chaos` | Configure chaos testing triggers |

### System Tools

| Tool | Description |
|------|-------------|
| `get_info` | Get server information and stats |
| `get_webui_config` | Get Mailpit configuration |

## Available Resources

| Resource URI | Description |
|--------------|-------------|
| `mailpit://info` | Server information (JSON) |
| `mailpit://messages/latest` | Latest message summary |
| `mailpit://tags` | All current tags (JSON) |
| `mailpit://config` | Mailpit configuration (JSON) |

## Available Prompts

| Prompt | Description |
|--------|-------------|
| `analyze_latest_email` | Comprehensive analysis of the most recent email |
| `debug_email_delivery` | Debug delivery issues for a message |
| `check_email_quality` | Full quality check (HTML, links, spam) |
| `search_emails` | Help construct search queries |
| `compose_test_email` | Create and send test emails |
| `analyze_email_headers` | Deep header analysis |
| `compare_emails` | Compare two emails (A/B testing) |
| `summarize_inbox` | Summarize inbox state |

## Search Syntax

The search tool supports Mailpit's query syntax:

- `from:email@example.com` - Sender address
- `to:email@example.com` - Recipient address
- `subject:keyword` - Subject contains text
- `message-id:id` - Specific Message-ID
- `tag:tagname` - Has specific tag
- `is:read` / `is:unread` - Read status
- `has:attachment` - Has attachments
- `before:YYYY-MM-DD` - Before date
- `after:YYYY-MM-DD` - After date

Multiple terms are combined with AND logic.

## Development

### Prerequisites

- Go 1.24+
- Running Mailpit instance

### Build

```bash
cd mcp
go build ./...
```

### Test

```bash
cd mcp
go test ./...
```

### Run Locally

```bash
# STDIO mode (for Claude Desktop)
MAILPIT_URL=http://localhost:8025 ./mailpit-mcp-server

# HTTP mode (for IDE integration)
MAILPIT_URL=http://localhost:8025 MCP_TRANSPORT=http ./mailpit-mcp-server
```

## License

This project is part of Mailpit and is licensed under the MIT License.
