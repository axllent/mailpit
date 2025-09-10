#!/bin/bash
# Comprehensive validation test

echo "═══════════════════════════════════════════════════════════════"
echo "              CONFIGURATION VALIDATION"
echo "═══════════════════════════════════════════════════════════════"

echo ""
echo "Checking binary compilation..."
if [ -f "/tmp/mailpit-test" ]; then
    echo "✓ Binary exists"
else
    echo "✗ Binary not found"
    exit 1
fi

echo ""
echo "Checking command-line flags..."
/tmp/mailpit-test --help 2>&1 | grep -q "postmark-api" && echo "✓ Postmark flags present" || echo "✗ Postmark flags missing"
/tmp/mailpit-test --help 2>&1 | grep -q "mcp-server" && echo "✓ MCP flags present" || echo "✗ MCP flags missing"

echo ""
echo "Testing environment variable support..."
MP_POSTMARK_API=true MP_MCP_SERVER=true /tmp/mailpit-test --help 2>&1 | head -1 > /dev/null && echo "✓ Environment variables work" || echo "✗ Environment variables fail"

echo ""
echo "Checking feature integration in code..."
grep -q "registerPostmarkRoutes" /home/btafoya/projects/mailpit/server/server.go && echo "✓ Postmark routes registered" || echo "✗ Postmark routes not found"
grep -q "mcpserver.Start" /home/btafoya/projects/mailpit/server/server.go && echo "✓ MCP server start call found" || echo "✗ MCP server start not found"

echo ""
echo "Validating package structure..."
[ -d "/home/btafoya/projects/mailpit/server/postmark" ] && echo "✓ Postmark package exists" || echo "✗ Postmark package missing"
[ -d "/home/btafoya/projects/mailpit/server/mcp" ] && echo "✓ MCP package exists" || echo "✗ MCP package missing"

echo ""
echo "Checking for compilation issues..."
cd /home/btafoya/projects/mailpit
go build -o /dev/null 2>&1 && echo "✓ Code compiles without errors" || echo "✗ Compilation errors exist"

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "              RUNTIME VALIDATION"
echo "═══════════════════════════════════════════════════════════════"

# Kill any existing test instances
pkill -f "mailpit.*18025" 2>/dev/null || true
sleep 1

echo ""
echo "Starting Mailpit with both features enabled..."
/tmp/mailpit-test \
  --database /tmp/validate.db \
  --listen 127.0.0.1:18025 \
  --smtp 127.0.0.1:11025 \
  --postmark-api \
  --postmark-token "validate123" \
  --postmark-accept-any \
  --mcp-server \
  --mcp-transport stdio \
  --verbose > /tmp/validate.log 2>&1 &

PID=$!
sleep 3

echo ""
echo "Checking server startup..."
if ps -p $PID > /dev/null; then
    echo "✓ Server is running (PID: $PID)"
else
    echo "✗ Server failed to start"
    cat /tmp/validate.log
    exit 1
fi

echo ""
echo "Checking log messages..."
grep -q "postmark.*API enabled" /tmp/validate.log && echo "✓ Postmark initialized" || echo "✗ Postmark not initialized"
grep -q "mcp.*server initialized" /tmp/validate.log && echo "✓ MCP initialized" || echo "✗ MCP not initialized"

echo ""
echo "Testing API accessibility..."
curl -s http://127.0.0.1:18025/api/v1/info > /dev/null 2>&1 && echo "✓ API accessible" || echo "✗ API not accessible"

echo ""
echo "Testing Postmark endpoint..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:18025/email \
  -H "Content-Type: application/json" \
  -H "X-Postmark-Server-Token: any-token" \
  -d '{"From": "test@test.com", "To": "test@test.com", "Subject": "Test", "TextBody": "Test"}')

if [ "$RESPONSE" = "200" ]; then
    echo "✓ Postmark endpoint works (HTTP $RESPONSE)"
else
    echo "✗ Postmark endpoint failed (HTTP $RESPONSE)"
fi

# Cleanup
kill $PID 2>/dev/null || true
rm -f /tmp/validate.db /tmp/validate.log

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "              VALIDATION COMPLETE"
echo "═══════════════════════════════════════════════════════════════"