# Mailpit Development Commands

## Backend Development
```bash
# Run Mailpit locally
go run main.go

# Run with specific options
go run main.go --smtp 0.0.0.0:1025 --ui 0.0.0.0:8025

# Build the binary
CGO_ENABLED=0 go build -ldflags "-s -w" -o mailpit

# Run tests for specific packages
go test -p 1 ./internal/storage ./server ./internal/smtpd ./internal/pop3 ./internal/tools ./internal/html2text ./internal/htmlcheck ./internal/linkcheck -v

# Run tests for a single package
go test ./internal/storage -v

# Run benchmarks
go test -p 1 ./internal/storage ./internal/html2text -bench=.

# Format Go code
gofmt -s -w .

# Check formatting without changes
gofmt -s -d .
```

## Frontend Development
```bash
# Install dependencies
npm install

# Build frontend assets (minified for production)
npm run build

# Watch mode for development
npm run watch

# Package for production (same as build)
npm run package

# Run linting
npm run lint

# Fix linting issues
npm run lint-fix

# Update CanIEmail data
npm run update-caniemail
```

## Docker
```bash
# Build Docker image
docker build -t mailpit .

# Run Docker container
docker run -p 1025:1025 -p 8025:8025 mailpit
```

## Full Build Process
```bash
# 1. Install frontend dependencies and build
npm install
npm run package

# 2. Build Go binary
CGO_ENABLED=0 go build -ldflags "-s -w" -o mailpit

# 3. Run the built binary
./mailpit
```

## Testing Workflow
```bash
# Run Go tests
go test -p 1 ./internal/storage ./server ./internal/smtpd ./internal/pop3 ./internal/tools ./internal/html2text ./internal/htmlcheck ./internal/linkcheck -v

# Check Go formatting
gofmt -s -d .

# Run JavaScript linting
npm run lint

# Full CI test suite locally
gofmt -s -w . && git diff --exit-code && go test -p 1 ./... -v && npm run lint && npm run package
```