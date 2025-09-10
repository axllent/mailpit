# Mailpit Code Style and Conventions

## Go Code Style
- Use standard Go formatting with `gofmt -s`
- Follow Go best practices and idioms
- Package names are lowercase, single-word
- Internal packages are under `internal/` directory
- Error handling: return errors, don't panic
- Use meaningful variable and function names
- Add comments for exported functions and types

## Go Project Structure
```
/cmd              - Command line tools (if any)
/config           - Configuration package
/internal         - Private application code
  /auth           - Authentication logic
  /htmlcheck      - HTML compatibility checking
  /linkcheck      - Link validation
  /logger         - Logging utilities
  /pop3           - POP3 server implementation
  /smtpd          - SMTP server implementation
  /storage        - Database/storage layer
  /tools          - Utility functions
/server           - HTTP server and API
  /apiv1          - API v1 endpoints
  /handlers       - HTTP handlers
  /ui             - Embedded UI assets
  /ui-src         - UI source code
  /websockets     - WebSocket handlers
/sendmail         - Sendmail replacement utility
```

## Frontend Code Style
- Vue 3 Composition API preferred
- TypeScript for type safety where beneficial
- Bootstrap 5 for UI components
- Use tabs (4 spaces width) for indentation
- Max line width: 120 characters
- ESLint for JavaScript/Vue linting
- Prettier for formatting

## JavaScript/Vue Conventions
```json
{
  "tabWidth": 4,
  "useTabs": true,
  "printWidth": 120
}
```

## Git Workflow
- Main branch: `develop` (active development)
- Feature branches: `feature/*`
- All PRs target `develop` branch
- Tests must pass before merge
- Linting must pass before merge

## Testing Conventions
- Test files end with `_test.go`
- Use table-driven tests where appropriate
- Run tests with `-p 1` to avoid parallel execution issues
- Benchmark functions start with `Benchmark`

## Build Conventions
- Frontend assets must be built before Go binary
- Use `CGO_ENABLED=0` for static binary builds
- Embed UI assets in binary for single-file deployment
- Strip debug info with `-ldflags "-s -w"`