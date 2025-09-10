<h1 align="center">
  Mailpit - email testing for developers
</h1>

<div align="center">
    <a href="https://github.com/axllent/mailpit/actions/workflows/tests.yml"><img src="https://github.com/axllent/mailpit/actions/workflows/tests.yml/badge.svg" alt="CI Tests status"></a>
    <a href="https://github.com/axllent/mailpit/actions/workflows/release-build.yml"><img src="https://github.com/axllent/mailpit/actions/workflows/release-build.yml/badge.svg" alt="CI build status"></a>
    <a href="https://github.com/axllent/mailpit/actions/workflows/build-docker.yml"><img src="https://github.com/axllent/mailpit/actions/workflows/build-docker.yml/badge.svg" alt="CI Docker build status"></a>
    <a href="https://github.com/axllent/mailpit/actions/workflows/codeql-analysis.yml"><img src="https://github.com/axllent/mailpit/actions/workflows/codeql-analysis.yml/badge.svg" alt="Code quality"></a>
    <a href="https://goreportcard.com/report/github.com/axllent/mailpit"><img src="https://goreportcard.com/badge/github.com/axllent/mailpit" alt="Go Report Card"></a>
    <br>
    <a href="https://github.com/axllent/mailpit/releases/latest"><img src="https://img.shields.io/github/v/release/axllent/mailpit.svg" alt="Latest release"></a>
    <a href="https://hub.docker.com/r/axllent/mailpit"><img src="https://img.shields.io/docker/pulls/axllent/mailpit.svg" alt="Docker pulls"></a>
</div>
<br>
<p align="center">
  <a href="https://mailpit.axllent.org">Website</a>  •
  <a href="https://mailpit.axllent.org/docs/">Documentation</a>  •
  <a href="https://mailpit.axllent.org/docs/api-v1/">API</a>
</p>

<hr>

**Mailpit** is a small, fast, low memory, zero-dependency, multi-platform email testing tool & API for developers.

It acts as an SMTP server, provides a modern web interface to view & test captured emails, and includes an API for automated integration testing.

Mailpit was originally **inspired** by MailHog which is [no longer maintained](https://github.com/mailhog/MailHog/issues/442#issuecomment-1493415258) and hasn't seen active development or security updates for a few years now.

![Mailpit](https://raw.githubusercontent.com/axllent/mailpit/develop/server/ui-src/screenshot.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Postmark API Emulation](#postmark-api-emulation)
- [MCP Server for AI Assistants](#mcp-server-for-ai-assistants)
  - [MCP with Docker](#mcp-with-docker)

## Features

- Runs entirely from a single [static binary](https://mailpit.axllent.org/docs/install/) or multi-architecture [Docker images](https://mailpit.axllent.org/docs/install/docker/)
- Modern web UI with advanced [mail search](https://mailpit.axllent.org/docs/usage/search-filters/) to view emails (formatted HTML, highlighted HTML source, text, headers, raw source, and MIME attachments
including image thumbnails), including optional [HTTPS](https://mailpit.axllent.org/docs/configuration/http/) & [authentication](https://mailpit.axllent.org/docs/configuration/http/)
- [SMTP server](https://mailpit.axllent.org/docs/configuration/smtp/) with optional STARTTLS or SSL/TLS, authentication (including an "accept any" mode)
- A [REST API](https://mailpit.axllent.org/docs/api-v1/) for integration testing
- Real-time web UI updates using web sockets for new mail & optional [browser notifications](https://mailpit.axllent.org/docs/usage/notifications/) when new mail is received
- Optional [POP3 server](https://mailpit.axllent.org/docs/configuration/pop3/) to download captured message directly into your email client
- [HTML check](https://mailpit.axllent.org/docs/usage/html-check/) to test & score mail client compatibility with HTML emails
- [Link check](https://mailpit.axllent.org/docs/usage/link-check/) to test message links (HTML & text) & linked images
- [Spam check](https://mailpit.axllent.org/docs/usage/spamassassin/) to test message "spamminess" using a running SpamAssassin server
- [Create screenshots](https://mailpit.axllent.org/docs/usage/html-screenshots/) of HTML messages via web UI
- Mobile and tablet HTML preview toggle in desktop mode
- [Message tagging](https://mailpit.axllent.org/docs/usage/tagging/) including manual tagging or automated tagging using filtering and "plus addressing"
- [SMTP relaying](https://mailpit.axllent.org/docs/configuration/smtp-relay/) (message release) - relay messages via a different SMTP server including an optional allowlist of accepted recipients
- [SMTP forwarding](https://mailpit.axllent.org/docs/configuration/smtp-forward/) - automatically forward messages via a different SMTP server to predefined email addresses
- Fast message [storing & processing](https://mailpit.axllent.org/docs/configuration/email-storage/) - ingesting 100-200 emails per second over SMTP depending on CPU, network speed & email size,
easily handling tens of thousands of emails, with automatic email pruning (by default keeping the most recent 500 emails)
- [Chaos](https://mailpit.axllent.org/docs/integration/chaos/) feature to enable configurable SMTP errors to test application resilience
- `List-Unsubscribe` syntax validation
- Optional [webhook](https://mailpit.axllent.org/docs/integration/webhook/) for received messages
- **[Postmark API emulation](https://postmarkapp.com/developer)** - drop-in replacement for Postmark API during development & testing
- **[MCP server](https://spec.modelcontextprotocol.io/)** - enables AI assistants (like Claude Code) to read and analyze messages for debugging workflows


## Installation

The Mailpit web UI listens by default on `http://0.0.0.0:8025` and the SMTP port on `0.0.0.0:1025`.

Mailpit runs as a single binary and can be installed in different ways:


### Install via package managers

- **Mac**: `brew install mailpit` (to run automatically in the background: `brew services start mailpit`)
- **Arch Linux**: available in the AUR as `mailpit`
- **FreeBSD**: `pkg install mailpit`


### Install via script (Linux & Mac)

Linux & Mac users can install it directly to `/usr/local/bin/mailpit` with:

```shell
sudo sh < <(curl -sL https://raw.githubusercontent.com/axllent/mailpit/develop/install.sh)
```

You can also change the install path to something else by setting the `INSTALL_PATH` environment, for example:

```shell
INSTALL_PATH=/usr/bin sudo sh < <(curl -sL https://raw.githubusercontent.com/axllent/mailpit/develop/install.sh)
```


### Download static binary (Windows, Linux and Mac)

Static binaries can always be found on the [releases](https://github.com/axllent/mailpit/releases/latest). The `mailpit` binary can be extracted and copied to your `$PATH`, or simply run as `./mailpit`.


### Docker

See [Docker instructions](https://mailpit.axllent.org/docs/install/docker/) for 386, amd64 & arm64 images.


### Compile from source

To build Mailpit from source, see [Building from source](https://mailpit.axllent.org/docs/install/source/).


## Usage

Run `mailpit -h` to see options. More information can be seen in [the docs](https://mailpit.axllent.org/docs/configuration/runtime-options/).

If installed using homebrew, you may run `brew services start mailpit` to always run mailpit automatically.


### Testing Mailpit

Please refer to [the documentation](https://mailpit.axllent.org/docs/install/testing/) on how to easily test email delivery to Mailpit.


## Postmark API Emulation

Mailpit can emulate the [Postmark API](https://postmarkapp.com/developer) for seamless testing of applications that use Postmark for email delivery.

### Enable Postmark API

```bash
mailpit --postmark-api --postmark-token "your-secret-token"
```

Or using environment variables:
```bash
export MP_POSTMARK_API=true
export MP_POSTMARK_TOKEN="your-secret-token"
mailpit
```

### Available Endpoints

- **POST /email** - Send single email
- **POST /email/batch** - Send multiple emails 
- **POST /email/withTemplate** - Send template-based email

### Usage with Postmark SDKs

#### Node.js
```javascript
const postmark = require("postmark");
const client = new postmark.ServerClient("your-secret-token");

// Point to Mailpit instead of Postmark
client.apiUrl = "http://localhost:8025";

// Send email normally
client.sendEmail({
  From: "sender@example.com",
  To: "recipient@example.com", 
  Subject: "Test Email",
  TextBody: "Hello from Mailpit!",
  HtmlBody: "<p>Hello from <strong>Mailpit</strong>!</p>"
});
```

#### Python
```python
from postmarker.core import PostmarkClient

# Configure client to use Mailpit
client = PostmarkClient(
    server_token='your-secret-token',
    api_base='http://localhost:8025'
)

# Send email
client.emails.send(
    From='sender@example.com',
    To='recipient@example.com',
    Subject='Test Email',
    HtmlBody='<p>Hello from <strong>Mailpit</strong>!</p>'
)
```

#### PHP
```php
use Postmark\PostmarkClient;

$client = new PostmarkClient('your-secret-token');
// Set custom API URL for Mailpit
$client->setApiUrl('http://localhost:8025');

$client->sendEmail([
    'From' => 'sender@example.com',
    'To' => 'recipient@example.com', 
    'Subject' => 'Test Email',
    'HtmlBody' => '<p>Hello from <strong>Mailpit</strong>!</p>'
]);
```

### Configuration Options

```bash
--postmark-api                      # Enable Postmark API emulation
--postmark-token string             # Authentication token (required)  
--postmark-accept-any               # Accept any token (development mode)
```

## MCP Server for AI Assistants

Mailpit includes an [MCP (Model Context Protocol)](https://spec.modelcontextprotocol.io/) server that enables AI assistants to read and analyze messages during development and debugging.

### Enable MCP Server

```bash
mailpit --mcp-server --mcp-transport stdio
```

Or using environment variables:
```bash
export MP_MCP_SERVER=true
export MP_MCP_TRANSPORT=stdio  # or websocket
mailpit
```

### Integration with Claude Code

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mailpit": {
      "command": "mailpit",
      "args": [
        "--mcp-server", 
        "--mcp-transport", "stdio",
        "--database", "/path/to/your/mailpit.db"
      ],
      "env": {
        "MP_MCP_SERVER": "true",
        "MP_MCP_TRANSPORT": "stdio"
      }
    }
  }
}
```

### Available MCP Tools

The MCP server provides 4 tools for AI assistants:

1. **list_messages** - List and filter messages with optional search and tags
2. **get_message** - Retrieve full message content including headers and attachments  
3. **search_messages** - Advanced search with date filters and content matching
4. **analyze_message** - Comprehensive analysis including HTML compatibility, link checking, and spam scoring

### Usage Examples

Once configured, AI assistants can:

```
AI: List recent messages
→ Uses list_messages tool to show latest emails

AI: Show me the email about the password reset
→ Uses search_messages to find relevant email
→ Uses get_message to retrieve full content

AI: Analyze this message for deliverability issues  
→ Uses analyze_message to check HTML compatibility, 
  validate links, and assess spam score
```

### MCP Transport Options

**stdio** (recommended for local AI assistants):
```bash
mailpit --mcp-server --mcp-transport stdio
```

**WebSocket** (for remote access):
```bash
mailpit --mcp-server --mcp-transport websocket --mcp-http-addr :8026
```

### Configuration Options

```bash
--mcp-server                        # Enable MCP server
--mcp-transport string              # Transport type: stdio|websocket (default: stdio)
--mcp-http-addr string              # WebSocket address (default: :8026)
--mcp-auth-token string             # Authentication token for WebSocket transport
```

### WebSocket Access

When using WebSocket transport, the MCP server is available at:
```
ws://localhost:8026/mcp
```

Include authentication header if token is configured:
```javascript
const ws = new WebSocket('ws://localhost:8026/mcp', {
  headers: { 'Authorization': 'Bearer your-mcp-token' }
});
```

### MCP with Docker

When running Mailpit in Docker, the MCP server integration depends on your setup:

#### Docker with WebSocket Transport

**For AI assistants running on host** (recommended):
```bash
docker run -d \
  --name mailpit \
  -p 8025:8025 \
  -p 1025:1025 \
  -p 8026:8026 \
  -e MP_MCP_SERVER=true \
  -e MP_MCP_TRANSPORT=websocket \
  -e MP_MCP_AUTH_TOKEN=your-mcp-token \
  axllent/mailpit
```

**Claude Code Configuration** for Docker WebSocket:
```json
{
  "mcpServers": {
    "mailpit-docker": {
      "transport": {
        "type": "websocket",
        "host": "localhost",
        "port": 8026,
        "path": "/mcp"
      },
      "auth": {
        "type": "bearer",
        "token": "your-mcp-token"
      }
    }
  }
}
```

#### Docker with stdio Transport

**For containerized AI assistants**, use Docker networks:

```yaml
# docker-compose.yml
version: '3.8'
services:
  mailpit:
    image: axllent/mailpit
    ports:
      - "8025:8025"
      - "1025:1025" 
    environment:
      MP_MCP_SERVER: true
      MP_MCP_TRANSPORT: stdio
    volumes:
      - mailpit-data:/data
    networks:
      - ai-network

  claude-assistant:
    image: your-ai-assistant:latest
    depends_on:
      - mailpit
    environment:
      MAILPIT_MCP_COMMAND: "docker exec mailpit-container mailpit --mcp-server --mcp-transport stdio --database /data/mailpit.db"
    networks:
      - ai-network

networks:
  ai-network:
    driver: bridge

volumes:
  mailpit-data:
```

#### Docker Volume Considerations

**Persistent Data Access**: Ensure the database is accessible to MCP:
```bash
docker run -d \
  --name mailpit \
  -v mailpit-data:/data \
  -e MP_DATA_FILE=/data/mailpit.db \
  -e MP_MCP_SERVER=true \
  axllent/mailpit
```

**Host Directory Binding** for local AI assistants:
```bash
docker run -d \
  --name mailpit \
  -v $(pwd)/mailpit-data:/data \
  -p 8026:8026 \
  -e MP_MCP_SERVER=true \
  -e MP_MCP_TRANSPORT=websocket \
  axllent/mailpit
```

#### Security with Docker

**Production Docker Deployment**:
```bash
docker run -d \
  --name mailpit-prod \
  -p 127.0.0.1:8026:8026 \
  -e MP_MCP_SERVER=true \
  -e MP_MCP_TRANSPORT=websocket \
  -e MP_MCP_AUTH_TOKEN=$(openssl rand -hex 32) \
  --restart unless-stopped \
  axllent/mailpit
```

**Docker Network Isolation**:
```yaml
services:
  mailpit:
    image: axllent/mailpit
    networks:
      - internal
      - mcp-network
    # Don't expose MCP port externally
    expose:
      - "8026"
    environment:
      MP_MCP_SERVER: true
      MP_MCP_AUTH_TOKEN: ${MCP_TOKEN}

networks:
  internal:
    driver: bridge
    internal: true
  mcp-network:
    driver: bridge
```

#### Testing Docker MCP Setup

**Test WebSocket Connection**:
```bash
# Install wscat: npm install -g wscat
wscat -c ws://localhost:8026/mcp -H "Authorization: Bearer your-mcp-token"
```

**Test via curl** (HTTP endpoint for debugging):
```bash
curl -H "Authorization: Bearer your-mcp-token" \
     -H "Content-Type: application/json" \
     -d '{"method":"list_messages","params":{"limit":5}}' \
     http://localhost:8026/mcp/rpc
```

**Docker Health Check** with MCP:
```yaml
services:
  mailpit:
    image: axllent/mailpit
    healthcheck:
      test: [
        "CMD", 
        "sh", "-c",
        "/mailpit readyz && nc -z localhost 8026"
      ]
      interval: 15s
      timeout: 3s
      retries: 3
```


### Configuring sendmail

Mailpit's SMTP server (default on port 1025), so you will likely need to configure your sending application to deliver mail via that port. 
A common MTA (Mail Transfer Agent) that delivers system emails to an SMTP server is `sendmail`, used by many applications, including PHP. 
Mailpit can also act as substitute for sendmail. For instructions on how to set this up, please refer to the [sendmail documentation](https://mailpit.axllent.org/docs/install/sendmail/).
