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

Add Mailpit as an MCP server using the Claude Code CLI:

```bash
# Add Mailpit MCP server for email analysis
claude mcp add mailpit --env MP_MCP_SERVER=true --env MP_MCP_TRANSPORT=stdio -- mailpit --mcp-server --database /path/to/your/mailpit.db

# Or use a local configuration file (.mcp.json)
claude mcp add --scope project mailpit --env MP_MCP_SERVER=true -- mailpit --mcp-server --database ./mailpit.db
```

**Alternative: Local Configuration File**

Create a `.mcp.json` file in your project:

```json
{
  "mcpServers": {
    "mailpit": {
      "command": "mailpit",
      "args": [
        "--mcp-server",
        "--database", "./mailpit.db"
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

**stdio** (recommended for local development):
```bash
mailpit --mcp-server --mcp-transport stdio
```

**HTTP** (for remote access - when supported by Claude Code):
```bash
mailpit --mcp-server --mcp-transport http --mcp-http-addr :8025
```

### Configuration Options

```bash
--mcp-server                        # Enable MCP server
--mcp-transport string              # Transport type: stdio|http (default: stdio)
--mcp-http-addr string              # HTTP server address (default: :8025)
--mcp-auth-token string             # Authentication token for HTTP transport
```

### Remote Access

When using HTTP transport, the MCP server endpoints are available at:
```
http://localhost:8025/mcp
```

Note: Remote MCP access depends on Claude Code client support for HTTP transport. Check the latest Claude Code documentation for current transport support.

### MCP with Docker

When running Mailpit in Docker, both MCP server and Postmark API features can be enabled together for comprehensive email testing and AI assistant integration:

#### Docker with stdio Transport

**For local development** (recommended):
```bash
docker run -d \
  --name mailpit \
  -p 8025:8025 \
  -p 1025:1025 \
  -e MP_MCP_SERVER=true \
  -e MP_MCP_TRANSPORT=stdio \
  -e MP_POSTMARK_API=true \
  -e MP_POSTMARK_TOKEN=dev-token-123 \
  -e MP_POSTMARK_ACCEPT_ANY=true \
  axllent/mailpit
```

**Claude Code Configuration** for Docker:
```bash
# Add Docker-based MCP server
claude mcp add mailpit-docker -- docker exec -i mailpit mailpit --mcp-server --mcp-transport stdio --database /data/mailpit.db
```

**Alternative: Docker Exec Configuration**
```json
{
  "mcpServers": {
    "mailpit-docker": {
      "command": "docker",
      "args": [
        "exec", "-i", "mailpit",
        "mailpit", "--mcp-server", "--mcp-transport", "stdio",
        "--database", "/data/mailpit.db"
      ]
    }
  }
}
```

#### Docker Compose Configuration

**For complex environments**, use Docker Compose:

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
      MP_POSTMARK_API: true
      MP_POSTMARK_TOKEN: dev-token-123
      MP_POSTMARK_ACCEPT_ANY: true
    volumes:
      - mailpit-data:/data

volumes:
  mailpit-data:
```

**Claude Code Integration with Docker Compose**:
```bash
# Add Docker Compose-based MCP server
claude mcp add mailpit-compose -- docker-compose exec -T mailpit mailpit --mcp-server --mcp-transport stdio --database /data/mailpit.db
```

#### Docker Volume Considerations

**Persistent Data Access**: Ensure the database is accessible to MCP:
```bash
docker run -d \
  --name mailpit \
  -v mailpit-data:/data \
  -e MP_DATA_FILE=/data/mailpit.db \
  -e MP_MCP_SERVER=true \
  -e MP_POSTMARK_API=true \
  -e MP_POSTMARK_TOKEN=dev-token-123 \
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
  -e MP_POSTMARK_API=true \
  -e MP_POSTMARK_TOKEN=dev-token-123 \
  axllent/mailpit
```

#### Security with Docker

**Production Docker Deployment**:
```bash
docker run -d \
  --name mailpit-prod \
  -p 127.0.0.1:8025:8025 \
  -p 127.0.0.1:8026:8026 \
  -p 1025:1025 \
  -e MP_MCP_SERVER=true \
  -e MP_MCP_TRANSPORT=websocket \
  -e MP_MCP_AUTH_TOKEN=$(openssl rand -hex 32) \
  -e MP_POSTMARK_API=true \
  -e MP_POSTMARK_TOKEN=$(openssl rand -hex 32) \
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
      MP_POSTMARK_API: true
      MP_POSTMARK_TOKEN: ${POSTMARK_TOKEN}

networks:
  internal:
    driver: bridge
    internal: true
  mcp-network:
    driver: bridge
```

#### Testing Docker MCP Setup

**Test MCP Connection**:
```bash
# Test that Mailpit MCP server is working
claude mcp test mailpit-docker
```

**Verify Docker Container**:
```bash
# Check container is running
docker ps --filter name=mailpit

# Check container logs
docker logs mailpit --tail 10

# Test SMTP port
telnet localhost 1025

# Test web interface
curl http://localhost:8025/api/v1/messages
```

**Docker Health Check**:
```yaml
services:
  mailpit:
    image: axllent/mailpit
    healthcheck:
      test: [
        "CMD", 
        "sh", "-c",
        "/mailpit readyz"
      ]
      interval: 15s
      timeout: 3s
      retries: 3
```

#### Using Postmark API with Docker

Once Postmark API is enabled in your Docker container, configure your application to use the containerized Mailpit:

**Node.js Application**:
```javascript
const postmark = require("postmark");
const client = new postmark.ServerClient("dev-token-123");

// Point to Docker container
client.apiUrl = "http://localhost:8025";

client.sendEmail({
  From: "test@example.com",
  To: "user@example.com",
  Subject: "Docker Test",
  TextBody: "Testing Postmark API via Docker!"
});
```

**Docker Compose Application Integration**:
```yaml
version: '3.8'
services:
  mailpit:
    image: axllent/mailpit
    environment:
      MP_POSTMARK_API: true
      MP_POSTMARK_TOKEN: dev-token-123
      MP_MCP_SERVER: true
    ports:
      - "8025:8025"
      - "8026:8026"
  
  app:
    build: .
    environment:
      POSTMARK_SERVER_TOKEN: dev-token-123
      POSTMARK_API_URL: http://mailpit:8025
    depends_on:
      - mailpit
```


### Configuring sendmail

Mailpit's SMTP server (default on port 1025), so you will likely need to configure your sending application to deliver mail via that port. 
A common MTA (Mail Transfer Agent) that delivers system emails to an SMTP server is `sendmail`, used by many applications, including PHP. 
Mailpit can also act as substitute for sendmail. For instructions on how to set this up, please refer to the [sendmail documentation](https://mailpit.axllent.org/docs/install/sendmail/).
