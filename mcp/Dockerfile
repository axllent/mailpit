# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates (needed for HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mailpit-mcp-server ./cmd/mailpit-mcp-server

# Final stage - minimal alpine for healthcheck support
FROM alpine:3.21

# Install ca-certificates for HTTPS connections
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 mailpit && \
    adduser -D -u 1000 -G mailpit mailpit

# Copy the binary
COPY --from=builder /build/mailpit-mcp-server /usr/local/bin/mailpit-mcp-server

# Set environment defaults
ENV MAILPIT_URL=http://localhost:8025 \
    MCP_TRANSPORT=http \
    MCP_HTTP_HOST=0.0.0.0 \
    MCP_HTTP_PORT=3000

# Expose the HTTP port
EXPOSE 3000

# Switch to non-root user
USER mailpit

# Health check against the /health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

ENTRYPOINT ["mailpit-mcp-server"]
