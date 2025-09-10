#!/bin/bash
# Comprehensive test suite for Postmark API functionality

set -e

echo "═══════════════════════════════════════════════════════════════"
echo "         POSTMARK API EMULATION TEST SUITE"
echo "═══════════════════════════════════════════════════════════════"

# Configuration
MAILPIT_BIN="/tmp/mailpit-test"
HOST="127.0.0.1"
HTTP_PORT="18025"
SMTP_PORT="11025"
API_TOKEN="test-token-abc123"
DB_FILE="/tmp/mailpit-test.db"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$MAILPIT_PID" ]; then
        kill $MAILPIT_PID 2>/dev/null || true
        wait $MAILPIT_PID 2>/dev/null || true
    fi
    rm -f $DB_FILE
}

# Only cleanup on actual exit, not on every test
trap cleanup EXIT INT TERM

# Start Mailpit with Postmark API enabled
echo -e "\n${YELLOW}Starting Mailpit with Postmark API...${NC}"
$MAILPIT_BIN \
    --database $DB_FILE \
    --listen $HOST:$HTTP_PORT \
    --smtp $HOST:$SMTP_PORT \
    --postmark-api \
    --postmark-token "$API_TOKEN" \
    --verbose &

MAILPIT_PID=$!
echo "Mailpit started with PID: $MAILPIT_PID"

# Wait for server to be ready
echo "Waiting for server to be ready..."
for i in {1..10}; do
    if curl -s http://$HOST:$HTTP_PORT/api/v1/info > /dev/null 2>&1; then
        echo -e "${GREEN}Server is ready!${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}Server failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Function to run a test
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_pattern="$3"
    
    echo -e "\n${YELLOW}TEST:${NC} $test_name"
    
    local result=$(eval "$test_cmd" 2>/dev/null)
    
    if echo "$result" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        echo "Expected pattern: $expected_pattern"
        echo "Got: $result"
        return 1
    fi
}

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0

# Test 1: Send single email with valid token
echo -e "\n${YELLOW}═══ Test Group: Authentication ═══${NC}"

test_cmd='curl -s -w "\n%{http_code}" -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "{
        \"From\": \"sender@example.com\",
        \"To\": \"recipient@example.com\",
        \"Subject\": \"Test Email\",
        \"TextBody\": \"This is a test\"
    }"'

if run_test "Valid authentication token" "$test_cmd" "200"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 2: Send email with invalid token
test_cmd='curl -s -w "\n%{http_code}" -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: wrong-token" \
    -d "{
        \"From\": \"sender@example.com\",
        \"To\": \"recipient@example.com\",
        \"Subject\": \"Test Email\",
        \"TextBody\": \"This is a test\"
    }"'

if run_test "Invalid authentication token returns 401" "$test_cmd" "401"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 3: Send email without token
test_cmd='curl -s -w "\n%{http_code}" -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -d "{
        \"From\": \"sender@example.com\",
        \"To\": \"recipient@example.com\",
        \"Subject\": \"Test Email\",
        \"TextBody\": \"This is a test\"
    }"'

if run_test "Missing authentication token returns 401" "$test_cmd" "401"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 4: Send email with HTML and Text body
echo -e "\n${YELLOW}═══ Test Group: Email Content ═══${NC}"

test_cmd='curl -s -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "{
        \"From\": \"html@example.com\",
        \"To\": \"recipient@example.com\",
        \"Subject\": \"HTML Email Test\",
        \"TextBody\": \"Plain text version\",
        \"HtmlBody\": \"<html><body><h1>HTML Version</h1></body></html>\"
    }" | python3 -m json.tool | grep -q "MessageID" && echo "Success"'

if run_test "Email with HTML and Text body" "$test_cmd" "Success"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 5: Send email with attachment
test_cmd='curl -s -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "{
        \"From\": \"attach@example.com\",
        \"To\": \"recipient@example.com\",
        \"Subject\": \"Attachment Test\",
        \"TextBody\": \"Email with attachment\",
        \"Attachments\": [{
            \"Name\": \"test.txt\",
            \"Content\": \"VGhpcyBpcyBhIHRlc3QgZmlsZQ==\",
            \"ContentType\": \"text/plain\"
        }]
    }" | python3 -m json.tool | grep -q "MessageID" && echo "Success"'

if run_test "Email with attachment" "$test_cmd" "Success"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 6: Send batch emails
echo -e "\n${YELLOW}═══ Test Group: Batch Sending ═══${NC}"

test_cmd='curl -s -X POST http://$HOST:$HTTP_PORT/email/batch \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "[
        {
            \"From\": \"batch1@example.com\",
            \"To\": \"recipient1@example.com\",
            \"Subject\": \"Batch Email 1\",
            \"TextBody\": \"First batch email\"
        },
        {
            \"From\": \"batch2@example.com\",
            \"To\": \"recipient2@example.com\",
            \"Subject\": \"Batch Email 2\",
            \"TextBody\": \"Second batch email\"
        }
    ]" | python3 -m json.tool | grep -q "MessageID" && echo "Success"'

if run_test "Batch email sending" "$test_cmd" "Success"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 7: Invalid JSON
echo -e "\n${YELLOW}═══ Test Group: Error Handling ═══${NC}"

test_cmd='curl -s -w "\n%{http_code}" -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "invalid json"'

if run_test "Invalid JSON returns 422" "$test_cmd" "422"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 8: Missing required fields
test_cmd='curl -s -w "\n%{http_code}" -X POST http://$HOST:$HTTP_PORT/email \
    -H "Content-Type: application/json" \
    -H "X-Postmark-Server-Token: $API_TOKEN" \
    -d "{
        \"Subject\": \"Missing From and To\",
        \"TextBody\": \"This should fail\"
    }"'

if run_test "Missing required fields returns 422" "$test_cmd" "422"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 9: Verify messages in Mailpit
echo -e "\n${YELLOW}═══ Test Group: Integration ═══${NC}"

test_cmd='curl -s http://$HOST:$HTTP_PORT/api/v1/messages | python3 -m json.tool | grep -q "messages" && echo "Success"'

if run_test "Messages stored in Mailpit" "$test_cmd" "Success"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test 10: Check for postmark-api tag
test_cmd='curl -s http://$HOST:$HTTP_PORT/api/v1/messages | python3 -c "
import sys, json
data = json.load(sys.stdin)
if data.get(\"messages\"):
    for msg in data[\"messages\"]:
        if \"postmark-api\" in msg.get(\"Tags\", []):
            print(\"Success\")
            sys.exit(0)
sys.exit(1)
" 2>/dev/null && echo "Found"'

if run_test "Messages tagged with 'postmark-api'" "$test_cmd" "Found"; then
    ((PASSED_TESTS++))
fi
((TOTAL_TESTS++))

# Test Summary
echo -e "\n${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}                        TEST SUMMARY${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"

if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED: $PASSED_TESTS/$TOTAL_TESTS${NC}"
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED: $PASSED_TESTS/$TOTAL_TESTS passed${NC}"
    exit 1
fi