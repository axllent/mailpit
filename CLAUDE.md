# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mailpit is an email and SMTP testing tool for developers. It captures emails via SMTP (port 1025), provides a web UI (port 8025) for viewing/testing, and includes a REST API for automation.

**Tech Stack**: Go 1.24.3 backend, Vue 3 + Bootstrap 5 frontend, SQLite storage, WebSockets for real-time updates

## Essential Commands

### Running Mailpit
```bash
# Run locally with default settings
go run main.go

# Run with custom ports
go run main.go --smtp 0.0.0.0:1025 --ui 0.0.0.0:8025
```

### Development Workflow
```bash
# Frontend development (watch mode)
npm install
npm run watch

# Build for production
npm run package
CGO_ENABLED=0 go build -ldflags "-s -w" -o mailpit
```

### Testing
```bash
# Run Go tests for specific packages
go test ./internal/storage ./server -v

# Run all tests (use -p 1 to avoid parallel execution issues)
go test -p 1 ./internal/storage ./server ./internal/smtpd ./internal/pop3 ./internal/tools ./internal/html2text ./internal/htmlcheck ./internal/linkcheck -v

# Run a single test
go test ./internal/storage -v -run TestSpecificFunction
```

### Code Quality
```bash
# Format Go code (REQUIRED before commits)
gofmt -s -w .

# Check Go formatting
gofmt -s -d .

# Lint JavaScript/Vue
npm run lint

# Fix JS/Vue linting issues
npm run lint-fix
```

## Architecture

```
/internal/          - Private packages (core business logic)
  /smtpd/          - SMTP server implementation
  /storage/        - Database layer (SQLite/RQLite)
  /auth/           - Authentication logic
  /htmlcheck/      - HTML compatibility checking
  /linkcheck/      - Link validation
  /pop3/           - POP3 server (optional)
/server/           - HTTP server and API
  /apiv1/          - REST API v1 endpoints
  /ui-src/         - Vue.js source code
  /ui/             - Compiled/embedded UI assets
  /websockets/     - WebSocket handlers for real-time updates
/config/           - Application configuration
main.go            - Application entry point
```

### Key Components
- **SMTP Server**: `internal/smtpd/` - Handles incoming mail
- **Storage Layer**: `internal/storage/` - Message persistence and retrieval
- **Web UI**: `server/ui-src/` - Vue 3 SPA for message viewing
- **API**: `server/apiv1/` - RESTful API for automation
- **WebSockets**: `server/websockets/` - Real-time message updates

## Code Style Requirements

### Go
- Use `gofmt -s` formatting (enforced by CI)
- Follow standard Go idioms and error handling
- Internal packages under `internal/`
- Test files end with `_test.go`

### Frontend
- Vue 3 Composition API preferred
- Tabs (4 spaces width) for indentation
- Line width: 120 characters max
- ESLint and Prettier configured in package.json

## Before Committing

1. **Format Go code**: `gofmt -s -w .`
2. **Run tests**: `go test -p 1 ./internal/storage ./server ./internal/smtpd -v`
3. **Lint frontend**: `npm run lint`
4. **Build assets**: `npm run package`
5. **Verify build**: `go build`

## Common Development Tasks

### Adding a New API Endpoint
1. Add handler in `server/apiv1/`
2. Update OpenAPI spec in `server/ui/api/v1/swagger.json`
3. Add tests in corresponding `_test.go` file
4. Update frontend if needed in `server/ui-src/`

### Modifying SMTP Behavior
1. Core SMTP logic in `internal/smtpd/`
2. Message processing in `internal/smtpd/process.go`
3. Storage operations in `internal/storage/`
4. Test with: `go test ./internal/smtpd -v`

### Updating the Web UI
1. Vue components in `server/ui-src/components/`
2. Run `npm run watch` for development
3. Build with `npm run package` before testing with Go binary
4. WebSocket updates in `server/ui-src/mixins/websocket.js`

## Important Notes

- **Branch Strategy**: Always work against `develop` branch, not `main`
- **Parallel Test Issues**: Use `-p 1` flag to prevent test conflicts
- **Static Binary**: Build with `CGO_ENABLED=0` for portability
- **Embedded Assets**: UI assets are embedded in the binary - rebuild after frontend changes
- **Real-time Updates**: WebSocket connection required for live message updates
- **Database**: Default SQLite, supports RQLite for distributed setups