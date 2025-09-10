FROM golang:alpine AS builder

ARG VERSION=dev

COPY . /app

WORKDIR /app

RUN  apk upgrade && apk add git npm && \
npm install && npm run package && \
CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/axllent/mailpit/config.Version=${VERSION}" -o /mailpit

FROM alpine:latest

LABEL org.opencontainers.image.title="Mailpit" \
  org.opencontainers.image.description="An email and SMTP testing tool with API for developers, Postmark API emulation, and MCP server support" \
  org.opencontainers.image.source="https://github.com/axllent/mailpit" \
  org.opencontainers.image.url="https://mailpit.axllent.org" \
  org.opencontainers.image.documentation="https://mailpit.axllent.org/docs/" \
  org.opencontainers.image.licenses="MIT"

COPY --from=builder /mailpit /mailpit

RUN apk upgrade --no-cache && apk add --no-cache tzdata

EXPOSE 1025/tcp 1110/tcp 8025/tcp 8026/tcp

# Environment variables for new features:
# MP_POSTMARK_API=true                    - Enable Postmark API emulation
# MP_POSTMARK_TOKEN=your-secret-token     - Postmark authentication token  
# MP_POSTMARK_ACCEPT_ANY=true             - Accept any Postmark token (dev mode)
# MP_MCP_SERVER=true                      - Enable MCP server for AI assistants
# MP_MCP_TRANSPORT=stdio|websocket        - MCP transport type (default: stdio)
# MP_MCP_HTTP_ADDR=:8026                  - MCP HTTP/WebSocket address
# MP_MCP_AUTH_TOKEN=your-mcp-token        - MCP authentication token

HEALTHCHECK --interval=15s --start-period=10s --start-interval=1s CMD ["/mailpit", "readyz"]

ENTRYPOINT ["/mailpit"]
