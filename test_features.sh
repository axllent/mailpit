#!/bin/bash
# Test script for new Mailpit features

echo "Testing Postmark and MCP features..."

# Start Mailpit with both features enabled
echo "Starting Mailpit with Postmark API and MCP server..."
/tmp/mailpit-test \
  --postmark-api \
  --postmark-token "test-token-123" \
  --mcp-server \
  --mcp-transport stdio \
  --listen 127.0.0.1:8025 \
  --smtp 127.0.0.1:1025 &

MAILPIT_PID=$!
echo "Mailpit started with PID: $MAILPIT_PID"

# Wait for server to start
sleep 2

# Test 1: Send email via Postmark API
echo ""
echo "Test 1: Sending email via Postmark API..."
curl -X POST http://127.0.0.1:8025/email \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: test-token-123" \
  -d '{
    "From": "sender@example.com",
    "To": "recipient@example.com",
    "Subject": "Test via Postmark API",
    "TextBody": "This is a test email sent via Postmark API emulation.",
    "HtmlBody": "<html><body><p>This is a <strong>test email</strong> sent via Postmark API emulation.</p></body></html>"
  }' 2>/dev/null | python3 -m json.tool

# Test 2: Check if email was received
echo ""
echo "Test 2: Checking messages via API..."
curl -s http://127.0.0.1:8025/api/v1/messages | python3 -m json.tool | head -20

# Clean up
echo ""
echo "Stopping Mailpit..."
kill $MAILPIT_PID 2>/dev/null

echo ""
echo "Tests completed!"