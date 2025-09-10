# Mailpit Project Overview

## Purpose
Mailpit is an email and SMTP testing tool for developers. It acts as an SMTP server to capture emails, provides a modern web interface to view and test captured emails, and includes an API for automated integration testing. It's designed to be fast, lightweight, and easy to use for local development and testing.

## Tech Stack
- **Backend**: Go (1.24.3)
- **Frontend**: Vue 3, Bootstrap 5, TypeScript
- **Build Tools**: esbuild (for frontend), go build (for backend)
- **Database**: SQLite (via modernc.org/sqlite) or RQLite for distributed setups
- **Testing**: Go standard testing framework
- **Version Control**: Git (develop branch for active development)

## Key Features
- SMTP server (port 1025 by default)
- Web UI (port 8025 by default)
- POP3 server (optional, port 1110)
- REST API for automation
- Real-time updates via WebSockets
- Message tagging and search
- HTML/Link/Spam checking capabilities
- SMTP relay and forwarding options

## Architecture
- Single static binary deployment
- Embedded web UI assets
- Modular internal package structure
- WebSocket support for real-time updates