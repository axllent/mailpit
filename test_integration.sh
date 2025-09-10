#!/bin/bash
# Integration test for Postmark API and MCP features

set -e

echo "═══════════════════════════════════════════════════════════════"
echo "         MAILPIT INTEGRATION TEST SUITE"
echo "═══════════════════════════════════════════════════════════════"

# Kill any existing Mailpit instances on test ports
pkill -f "mailpit.*18025" 2>/dev/null || true
sleep 1

# Start Mailpit with all features
echo "Starting Mailpit with Postmark API and MCP server..."
/tmp/mailpit-test \
  --database /tmp/test-mailpit.db \
  --listen 127.0.0.1:18025 \
  --smtp 127.0.0.1:11025 \
  --postmark-api \
  --postmark-token "test123" \
  --mcp-server \
  --mcp-transport stdio \
  --verbose > /tmp/mailpit-test.log 2>&1 &

MAILPIT_PID=$!
echo "Started Mailpit with PID: $MAILPIT_PID"

# Wait for startup
sleep 3

echo ""
echo "Testing endpoints..."
echo ""

# Test 1: Health check
echo -n "1. Health check: "
if curl -s http://127.0.0.1:18025/api/v1/info | grep -q "version"; then
    echo "✓ PASS"
else
    echo "✗ FAIL"
fi

# Test 2: Postmark API with valid token
echo -n "2. Postmark API (valid token): "
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST http://127.0.0.1:18025/email \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: test123" \
  -d '{
    "From": "test@example.com",
    "To": "recipient@example.com",
    "Subject": "Test Email",
    "TextBody": "Test content"
  }')

if echo "$RESPONSE" | grep -q "HTTP_CODE:200"; then
    echo "✓ PASS"
else
    echo "✗ FAIL"
    echo "$RESPONSE"
fi

# Test 3: Postmark API with invalid token
echo -n "3. Postmark API (invalid token): "
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST http://127.0.0.1:18025/email \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: wrong" \
  -d '{
    "From": "test@example.com",
    "To": "recipient@example.com",
    "Subject": "Test Email",
    "TextBody": "Test content"
  }')

if echo "$RESPONSE" | grep -q "HTTP_CODE:401"; then
    echo "✓ PASS"
else
    echo "✗ FAIL"
fi

# Test 4: Batch email endpoint
echo -n "4. Postmark batch endpoint: "
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST http://127.0.0.1:18025/email/batch \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: test123" \
  -d '[
    {"From": "test1@example.com", "To": "r1@example.com", "Subject": "Batch 1", "TextBody": "Test 1"},
    {"From": "test2@example.com", "To": "r2@example.com", "Subject": "Batch 2", "TextBody": "Test 2"}
  ]')

if echo "$RESPONSE" | grep -q "HTTP_CODE:200"; then
    echo "✓ PASS"
else
    echo "✗ FAIL"
fi

# Test 5: Template endpoint
echo -n "5. Postmark template endpoint: "
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST http://127.0.0.1:18025/email/withTemplate \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: test123" \
  -d '{
    "From": "template@example.com",
    "To": "recipient@example.com",
    "Subject": "Template Test",
    "TextBody": "Template content"
  }')

if echo "$RESPONSE" | grep -q "HTTP_CODE:200"; then
    echo "✓ PASS"
else
    echo "✗ FAIL"
fi

# Test 6: Check messages were stored
echo -n "6. Messages stored in database: "
MSG_COUNT=$(curl -s http://127.0.0.1:18025/api/v1/messages | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(len(data.get('messages', [])))
except:
    print('0')
")

if [ "$MSG_COUNT" -gt "0" ]; then
    echo "✓ PASS ($MSG_COUNT messages)"
else
    echo "✗ FAIL"
fi

# Test 7: Check for postmark-api tag
echo -n "7. Messages tagged correctly: "
TAGGED=$(curl -s http://127.0.0.1:18025/api/v1/messages | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    count = 0
    for msg in data.get('messages', []):
        if 'postmark-api' in msg.get('Tags', []):
            count += 1
    print(count)
except:
    print('0')
")

if [ "$TAGGED" -gt "0" ]; then
    echo "✓ PASS ($TAGGED tagged messages)"
else
    echo "✗ FAIL"
fi

# Test 8: MCP server configuration check
echo -n "8. MCP server configured: "
if grep -q "mcp.*server initialized" /tmp/mailpit-test.log; then
    echo "✓ PASS"
else
    echo "✗ FAIL (feature may not be activated in stdio mode)"
fi

# Clean up
echo ""
echo "Cleaning up..."
kill $MAILPIT_PID 2>/dev/null || true
rm -f /tmp/test-mailpit.db /tmp/mailpit-test.log

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "                    TEST COMPLETE"
echo "═══════════════════════════════════════════════════════════════"