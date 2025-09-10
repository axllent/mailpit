# Postmark API and MCP Server Features

## Overview

Mailpit now includes two powerful new features:

1. **Postmark API Emulation** - Allow applications using the Postmark SDK to send test emails to Mailpit
2. **MCP (Model Context Protocol) Server** - Enable AI assistants to read and analyze messages during debugging sessions

## Postmark API Emulation

### Configuration

Enable the Postmark API emulation with the following flags:

```bash
mailpit --postmark-api --postmark-token "your-secret-token"
```

Environment variables:
- `MP_POSTMARK_API=true` - Enable Postmark API
- `MP_POSTMARK_TOKEN=your-secret-token` - Set authentication token
- `MP_POSTMARK_ACCEPT_ANY=true` - Accept any token (development mode)

### Endpoints

- `POST /email` - Send a single email
- `POST /email/batch` - Send multiple emails in batch
- `POST /email/withTemplate` - Send template-based email

### Authentication

Include the authentication token in the request header:
```
X-Postmark-Server-Token: your-secret-token
```

### Example Usage

```javascript
// Using Postmark.js client
const postmark = require("postmark");
const client = new postmark.ServerClient("your-secret-token");

// Point to your local Mailpit instance
client.apiUrl = "http://localhost:8025";

// Send email
client.sendEmail({
  From: "sender@example.com",
  To: "recipient@example.com",
  Subject: "Test Email",
  TextBody: "Hello from Postmark emulation!",
  HtmlBody: "<html><body><p>Hello from <strong>Postmark</strong> emulation!</p></body></html>"
});
```

### cURL Example

```bash
curl -X POST http://localhost:8025/email \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: your-secret-token" \
  -d '{
    "From": "sender@example.com",
    "To": "recipient@example.com",
    "Subject": "Test Email",
    "TextBody": "Plain text content",
    "HtmlBody": "<p>HTML content</p>",
    "Attachments": [{
      "Name": "document.pdf",
      "Content": "base64-encoded-content",
      "ContentType": "application/pdf"
    }]
  }'
```

## MCP Server

### Configuration

Enable the MCP server with the following flags:

```bash
mailpit --mcp-server --mcp-transport stdio
```

Environment variables:
- `MP_MCP_SERVER=true` - Enable MCP server
- `MP_MCP_TRANSPORT=stdio|websocket` - Transport type
- `MP_MCP_HTTP_ADDR=:8026` - HTTP/WebSocket address
- `MP_MCP_AUTH_TOKEN=secret` - Authentication token for HTTP transport

### Available Tools

The MCP server provides four tools for AI assistants:

1. **list_messages** - List recent messages with optional filtering
   ```json
   {
     "limit": 50,
     "search": "error",
     "tag": "important"
   }
   ```

2. **get_message** - Retrieve a specific message by ID
   ```json
   {
     "id": "message-id",
     "includeRaw": false
   }
   ```

3. **search_messages** - Search messages with date filters
   ```json
   {
     "query": "invoice",
     "dateFrom": "2024-01-01T00:00:00Z",
     "dateTo": "2024-12-31T23:59:59Z",
     "limit": 100
   }
   ```

4. **analyze_message** - Analyze message for issues
   ```json
   {
     "id": "message-id"
   }
   ```

### Integration with Claude

Add to your Claude Code configuration:

```json
{
  "mcpServers": {
    "mailpit": {
      "command": "mailpit",
      "args": ["--mcp-server", "--mcp-transport", "stdio"],
      "env": {
        "MP_DATABASE": "/path/to/mailpit.db"
      }
    }
  }
}
```

### WebSocket Transport

For WebSocket transport, the MCP server is available at:
```
ws://localhost:8025/mcp
```

Include authentication if configured:
```javascript
const ws = new WebSocket('ws://localhost:8025/mcp', {
  headers: {
    'Authorization': 'Bearer your-auth-token'
  }
});
```

## Security Considerations

### Postmark API
- Always use authentication tokens in production
- Rotate tokens regularly
- Use HTTPS in production environments
- Limit API access to trusted networks

### MCP Server
- Use authentication for HTTP/WebSocket transports
- Restrict stdio access to local processes
- Implement rate limiting for production use
- Monitor and log access attempts

## Troubleshooting

### Common Issues

1. **Postmark API returns 401 Unauthorized**
   - Check that the authentication token matches
   - Ensure the header name is correct: `X-Postmark-Server-Token`

2. **MCP server not responding**
   - Verify the transport type matches your client configuration
   - Check that the database is accessible
   - Review logs for connection errors

3. **Messages not appearing**
   - Ensure Mailpit is running and accessible
   - Check the SMTP server is configured correctly
   - Verify database permissions

### Debug Logging

Enable debug logging for more detailed information:
```bash
mailpit --verbose
```

## Examples

### Python Example (Postmark)

```python
import requests
import json

url = "http://localhost:8025/email"
headers = {
    "Content-Type": "application/json",
    "X-Postmark-Server-Token": "your-secret-token"
}

data = {
    "From": "python@example.com",
    "To": "recipient@example.com",
    "Subject": "Python Test",
    "TextBody": "Sent from Python",
    "HtmlBody": "<p>Sent from <strong>Python</strong></p>"
}

response = requests.post(url, headers=headers, json=data)
print(response.json())
```

### Node.js Example (MCP)

```javascript
const { MCPClient } = require('@modelcontextprotocol/sdk');

async function analyzeEmails() {
  const client = new MCPClient({
    transport: 'stdio',
    command: 'mailpit',
    args: ['--mcp-server', '--mcp-transport', 'stdio']
  });

  await client.connect();

  // List recent messages
  const messages = await client.call('list_messages', {
    limit: 10
  });

  // Analyze each message
  for (const msg of messages.messages) {
    const analysis = await client.call('analyze_message', {
      id: msg.id
    });
    console.log(`Message ${msg.subject}: ${analysis.deliverability}`);
  }

  await client.close();
}

analyzeEmails();
```

## Contributing

To contribute to these features:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

These features are part of Mailpit and follow the same MIT license.