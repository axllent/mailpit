# Mailpit Integration Examples

This directory contains examples of how to integrate Mailpit with various development environments and tools.

## VS Code with MCP

### Setup Instructions

1. **Install Mailpit** following the [installation guide](../README.md#installation)

2. **Configure MCP Server** - Add the configuration to your Claude Code settings:

   Copy the contents of `vscode-mcp-config.json` to your Claude Code configuration file:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%/Claude/claude_desktop_config.json`
   - **Linux**: `~/.config/Claude/claude_desktop_config.json`

3. **Create Mailpit Directory** in your workspace:
   ```bash
   mkdir -p .mailpit
   ```

4. **Start Development** - The MCP server will automatically start when Claude Code needs to access it

### Available Configurations

#### Basic MCP Only
Use the `mailpit` configuration for basic message reading and analysis:
```json
{
  "mcpServers": {
    "mailpit": { /* ... basic config ... */ }
  }
}
```

#### MCP + Postmark API  
Use the `mailpit-with-postmark` configuration when you need both features:
```json
{
  "mcpServers": {
    "mailpit-with-postmark": { /* ... full config ... */ }
  }
}
```

### Usage Examples

Once configured, you can ask Claude Code to:

- **"Show me recent test emails"** - Lists messages from your test suite
- **"Analyze the signup email for issues"** - Checks HTML compatibility and links  
- **"Find emails containing 'password reset'"** - Searches message content
- **"What emails were sent in the last hour?"** - Time-based filtering
- **"Check if the welcome email has any broken links"** - Link validation

## Node.js Project Integration

### Package.json Scripts

Add these scripts to your `package.json`:

```json
{
  "scripts": {
    "mailpit:start": "mailpit --postmark-api --postmark-accept-any",
    "mailpit:dev": "mailpit --postmark-api --postmark-token dev-token-123 --mcp-server",
    "test:mail": "npm run mailpit:start & npm run test && pkill mailpit"
  }
}
```

### Environment Configuration

Create a `.env.test` file:
```bash
# Postmark configuration for testing
POSTMARK_API_TOKEN=dev-token-123
POSTMARK_API_URL=http://localhost:8025

# Database
DATABASE_URL=postgresql://localhost/myapp_test
```

### Testing with Jest

```javascript
// tests/setup.js
const { execSync } = require('child_process');

beforeAll(async () => {
  // Start Mailpit for testing
  execSync('mailpit --postmark-api --postmark-accept-any --database /tmp/test-mailpit.db &');
  
  // Wait for startup
  await new Promise(resolve => setTimeout(resolve, 2000));
});

afterAll(() => {
  // Clean up
  execSync('pkill mailpit');
});
```

## Python Project Integration

### Requirements

Add to `requirements-dev.txt`:
```
postmarker>=0.15.0  # For Postmark API client
```

### Django Settings

```python
# settings/test.py
EMAIL_BACKEND = 'postmarker.django.EmailBackend'
POSTMARK = {
    'TOKEN': 'dev-token-123',
    'API_URL': 'http://localhost:8025',  # Point to Mailpit
}
```

### Pytest Configuration

```python
# conftest.py
import subprocess
import pytest
import time

@pytest.fixture(scope="session", autouse=True)
def mailpit_server():
    """Start Mailpit server for testing"""
    process = subprocess.Popen([
        'mailpit',
        '--postmark-api',
        '--postmark-accept-any',
        '--database', '/tmp/pytest-mailpit.db',
        '--listen', '127.0.0.1:8025'
    ])
    
    time.sleep(2)  # Wait for startup
    
    yield
    
    process.terminate()
    process.wait()
```

## PHP Project Integration

### Composer Dependencies

```json
{
  "require-dev": {
    "wildbit/postmark-php": "^4.0"
  }
}
```

### PHPUnit Configuration

```php
// tests/MailpitTestCase.php
use Postmark\PostmarkClient;

abstract class MailpitTestCase extends TestCase
{
    protected function setUp(): void
    {
        parent::setUp();
        
        // Configure Postmark client for Mailpit
        $this->postmarkClient = new PostmarkClient('dev-token-123');
        $this->postmarkClient->setApiUrl('http://localhost:8025');
    }
    
    protected function getMailpitMessages(): array
    {
        $response = file_get_contents('http://localhost:8025/api/v1/messages');
        return json_decode($response, true)['messages'] ?? [];
    }
}
```

## Docker Development

### MCP with Docker

#### Option 1: WebSocket Transport (Recommended)

Use the provided Docker Compose configuration:

```bash
# Start Mailpit with MCP WebSocket support
docker-compose -f examples/docker-compose.mcp.yml up
```

**Claude Code Configuration** (`claude_desktop_config.json`):
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
        "token": "mcp-secret-token-123"
      }
    }
  }
}
```

#### Option 2: Docker Exec with stdio

**Start Container**:
```bash
docker run -d --name mailpit-mcp \
  -p 8025:8025 -p 1025:1025 \
  -e MP_MCP_SERVER=true \
  -e MP_POSTMARK_API=true \
  -e MP_POSTMARK_TOKEN=dev-token-123 \
  -e MP_POSTMARK_ACCEPT_ANY=true \
  axllent/mailpit
```

**Claude Code Configuration**:
```json
{
  "mcpServers": {
    "mailpit-docker": {
      "command": "docker",
      "args": [
        "exec", "-i", "mailpit-mcp",
        "mailpit", "--mcp-server", "--mcp-transport", "stdio",
        "--database", "/data/mailpit.db"
      ]
    }
  }
}
```

#### Option 3: Docker Compose Exec

**Claude Code Configuration**:
```json
{
  "mcpServers": {
    "mailpit-compose": {
      "command": "docker-compose",
      "args": [
        "-f", "examples/docker-compose.mcp.yml",
        "exec", "-T", "mailpit",
        "mailpit", "--mcp-server", "--mcp-transport", "stdio"
      ]
    }
  }
}
```

### Docker Compose for Development

See the comprehensive example in `examples/docker-compose.mcp.yml` which includes:

- Basic development setup with MCP WebSocket
- Production configuration with security
- Example application integration
- Health checks and networking

### Usage
```bash
# Development mode
docker-compose -f examples/docker-compose.mcp.yml up

# With example application
docker-compose -f examples/docker-compose.mcp.yml --profile example up

# Production mode  
docker-compose -f examples/docker-compose.mcp.yml --profile production up
```

## Continuous Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Mailpit
        run: |
          sudo sh < <(curl -sL https://raw.githubusercontent.com/axllent/mailpit/develop/install.sh)
      
      - name: Start Mailpit
        run: |
          mailpit --postmark-api --postmark-accept-any --database /tmp/ci-mailpit.db &
          sleep 2
      
      - name: Run tests
        run: npm test
        env:
          POSTMARK_API_URL: http://localhost:8025
          POSTMARK_API_TOKEN: ci-token-123
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**: Use different ports if defaults are taken
   ```bash
   mailpit --listen :18025 --smtp :11025 --mcp-http-addr :18026
   ```

2. **Permission Issues**: Ensure Mailpit can write to database location
   ```bash
   mkdir -p ~/.mailpit
   mailpit --database ~/.mailpit/mailpit.db
   ```

3. **MCP Not Connecting**: Check Claude Code logs and configuration
   ```bash
   # Check if Mailpit MCP is listed
   # In Claude Code, it should show "mailpit" in available servers
   ```

4. **Postmark API Not Responding**: Verify configuration
   ```bash
   curl -X POST http://localhost:8025/email \
     -H "Content-Type: application/json" \
     -H "X-Postmark-Server-Token: test" \
     -d '{"From":"test@test.com","To":"test@test.com","Subject":"Test","TextBody":"Test"}'
   ```