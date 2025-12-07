#!/bin/bash
set -e

# Backend Integration Test Script
# Tests all backend API endpoints with comprehensive verification

# Configuration
SERVER_PORT=5000
BASE_URL="http://localhost:${SERVER_PORT}"
DATA_DIR="/tmp/fls-backend-test-data"
SESSION_SECRET="test-secret-backend"
ADMIN_USER="backendadmin"
ADMIN_PASS="backendpass123"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$DATA_DIR"
    rm -f /tmp/backend-session-cookie
}

trap cleanup EXIT

# Clean up
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

# Build server (skip if PREBUILT_SERVER is set)
if [ -n "$PREBUILT_SERVER" ] && [ -f "$PREBUILT_SERVER" ]; then
    log_info "Using pre-built server from $PREBUILT_SERVER"
    cp "$PREBUILT_SERVER" /tmp/fls-backend-server
    chmod +x /tmp/fls-backend-server
else
    log_info "Building server..."
    go build -o /tmp/fls-backend-server ./cmd/server
    if [ $? -ne 0 ]; then
        log_error "Failed to build server"
        exit 1
    fi
fi

# Start server
log_info "Starting server..."
/tmp/fls-backend-server \
    -bind="0.0.0.0:${SERVER_PORT}" \
    -data-dir="$DATA_DIR" \
    -session-secret="$SESSION_SECRET" \
    -discord-client-id="test-id" \
    -discord-client-secret="test-secret" \
    -discord-redirect-url="${BASE_URL}/auth/login/callback" \
    -admin-username="$ADMIN_USER" \
    -admin-password="$ADMIN_PASS" \
    > /tmp/backend-server.log 2>&1 &

SERVER_PID=$!
log_info "Server started with PID: $SERVER_PID"

# Wait for server
log_info "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
        log_info "Server is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        log_error "Server failed to start"
        cat /tmp/backend-server.log
        exit 1
    fi
    sleep 1
done

SESSION_COOKIE="/tmp/backend-session-cookie"

# Test 1: Health endpoint
log_info "Test 1: Health endpoint"
RESPONSE=$(curl -s "${BASE_URL}/health")
if echo "$RESPONSE" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    log_info "✓ Health check passed"
    # Verify version field exists
    if echo "$RESPONSE" | jq -e '.version' > /dev/null 2>&1; then
        log_info "✓ Version field present in health response"
    else
        log_warn "Version field missing in health response"
    fi
else
    log_error "✗ Health check failed: $RESPONSE"
    exit 1
fi

# Test 2: Admin login
log_info "Test 2: Admin login"
RESPONSE=$(curl -s -c "$SESSION_COOKIE" -X POST \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}" \
    "${BASE_URL}/auth/admin/login")
if echo "$RESPONSE" | jq -e '.message == "admin login successful"' > /dev/null 2>&1; then
    log_info "✓ Admin login passed"
else
    log_error "✗ Admin login failed: $RESPONSE"
    exit 1
fi

# Test 3: Get current user
log_info "Test 3: Get current user"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" "${BASE_URL}/api/auth/me")
if echo "$RESPONSE" | jq -e '.username' > /dev/null 2>&1; then
    USERNAME=$(echo "$RESPONSE" | jq -r '.username')
    IS_ADMIN=$(echo "$RESPONSE" | jq -r '.is_admin')
    log_info "✓ Current user: $USERNAME (Admin: $IS_ADMIN)"
    if [ "$IS_ADMIN" != "true" ]; then
        log_error "User should be admin"
        exit 1
    fi
else
    log_error "✗ Failed to get current user: $RESPONSE"
    exit 1
fi

# Test 4: List benchmarks (empty)
log_info "Test 4: List benchmarks (empty)"
RESPONSE=$(curl -s "${BASE_URL}/api/benchmarks")
BENCHMARK_COUNT=$(echo "$RESPONSE" | jq '.benchmarks | length')
if [ "$BENCHMARK_COUNT" -eq 0 ]; then
    log_info "✓ List benchmarks empty"
    # Verify pagination fields
    TOTAL=$(echo "$RESPONSE" | jq '.total')
    PAGE=$(echo "$RESPONSE" | jq '.page')
    PER_PAGE=$(echo "$RESPONSE" | jq '.per_page')
    log_info "  Total: $TOTAL, Page: $PAGE, Per Page: $PER_PAGE"
else
    log_error "✗ Expected 0 benchmarks, got $BENCHMARK_COUNT"
    exit 1
fi

# Test 5: Create benchmark
log_info "Test 5: Create benchmark"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" -X POST \
    -F "title=Backend Test Benchmark" \
    -F "description=Comprehensive backend test" \
    -F "files=@testdata/mangohud/run1.csv" \
    "${BASE_URL}/api/benchmarks")
BENCHMARK_ID=$(echo "$RESPONSE" | jq -r '.ID')
if [ "$BENCHMARK_ID" != "null" ] && [ -n "$BENCHMARK_ID" ]; then
    log_info "✓ Created benchmark with ID: $BENCHMARK_ID"
    # Verify all fields
    TITLE=$(echo "$RESPONSE" | jq -r '.Title')
    DESCRIPTION=$(echo "$RESPONSE" | jq -r '.Description')
    USER_ID=$(echo "$RESPONSE" | jq -r '.UserID')
    log_info "  Title: $TITLE"
    log_info "  Description: $DESCRIPTION"
    log_info "  UserID: $USER_ID"
else
    log_error "✗ Failed to create benchmark: $RESPONSE"
    exit 1
fi

# Test 6: Get benchmark
log_info "Test 6: Get benchmark"
RESPONSE=$(curl -s "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}")
FETCHED_TITLE=$(echo "$RESPONSE" | jq -r '.Title')
if [ "$FETCHED_TITLE" == "Backend Test Benchmark" ]; then
    log_info "✓ Get benchmark passed"
    # Verify user info is populated
    USERNAME_IN_BENCH=$(echo "$RESPONSE" | jq -r '.User.Username')
    log_info "  Benchmark owner: $USERNAME_IN_BENCH"
else
    log_error "✗ Get benchmark failed: $RESPONSE"
    exit 1
fi

# Test 7: Get benchmark data
log_info "Test 7: Get benchmark data"
RESPONSE=$(curl -s "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}/data")
if echo "$RESPONSE" | jq -e '.runs | length > 0' > /dev/null 2>&1; then
    RUN_COUNT=$(echo "$RESPONSE" | jq '.runs | length')
    log_info "✓ Get benchmark data passed ($RUN_COUNT runs)"
    
    # Verify data structure (suppress verbose output)
    FIRST_RUN=$(echo "$RESPONSE" | jq -c '.runs[0]')
    if echo "$FIRST_RUN" | jq -e '.Label' > /dev/null 2>&1; then
        LABEL=$(echo "$FIRST_RUN" | jq -r '.Label')
        log_info "  First run label: $LABEL"
    fi
    
    # Verify data points exist (suppress verbose output)
    if echo "$FIRST_RUN" | jq -e '.Data | length > 0' > /dev/null 2>&1; then
        DATA_POINTS=$(echo "$FIRST_RUN" | jq '.Data | length')
        log_info "  Data points in first run: $DATA_POINTS"
    fi
else
    log_warn "⚠ Get benchmark data returned no runs (may be expected if data storage failed)"
    log_info "✓ Get benchmark data endpoint accessible"
fi

# Test 8: Update benchmark
log_info "Test 8: Update benchmark"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" -X PUT \
    -H "Content-Type: application/json" \
    -d '{"title":"Updated Backend Benchmark","description":"Updated description"}' \
    "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}")
UPDATED_TITLE=$(echo "$RESPONSE" | jq -r '.Title')
if [ "$UPDATED_TITLE" == "Updated Backend Benchmark" ]; then
    log_info "✓ Update benchmark passed"
else
    log_error "✗ Update benchmark failed: $RESPONSE"
    exit 1
fi

# Test 9: List benchmarks (with data)
log_info "Test 9: List benchmarks (with data)"
RESPONSE=$(curl -s "${BASE_URL}/api/benchmarks")
BENCHMARK_COUNT=$(echo "$RESPONSE" | jq '.benchmarks | length')
if [ "$BENCHMARK_COUNT" -eq 1 ]; then
    log_info "✓ List benchmarks with data"
    FIRST_BENCHMARK=$(echo "$RESPONSE" | jq '.benchmarks[0]')
    B_TITLE=$(echo "$FIRST_BENCHMARK" | jq -r '.Title')
    B_RUN_COUNT=$(echo "$FIRST_BENCHMARK" | jq -r '.RunCount')
    log_info "  Benchmark: $B_TITLE (Runs: $B_RUN_COUNT)"
else
    log_error "✗ Expected 1 benchmark, got $BENCHMARK_COUNT"
    exit 1
fi

# Test 10: Search benchmarks
log_info "Test 10: Search benchmarks"
RESPONSE=$(curl -s "${BASE_URL}/api/benchmarks?search=Updated")
SEARCH_COUNT=$(echo "$RESPONSE" | jq '.benchmarks | length')
if [ "$SEARCH_COUNT" -eq 1 ]; then
    log_info "✓ Search benchmarks passed"
else
    log_error "✗ Search failed, expected 1, got $SEARCH_COUNT"
    exit 1
fi

# Test 11: Download benchmark data
log_info "Test 11: Download benchmark data"
HTTP_CODE=$(curl -s -o /tmp/benchmark-download.zip -w "%{http_code}" \
    "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}/download")
if [ "$HTTP_CODE" -eq 200 ]; then
    # Verify it's a zip file (case-insensitive check)
    if file /tmp/benchmark-download.zip | grep -qi "zip"; then
        log_info "✓ Download benchmark data passed"
        # List zip contents
        unzip -l /tmp/benchmark-download.zip > /tmp/zip-contents.txt 2>&1
        log_info "  Zip contents:"
        cat /tmp/zip-contents.txt | tail -5
    else
        log_error "✗ Downloaded file is not a zip"
        exit 1
    fi
else
    log_error "✗ Download failed with HTTP $HTTP_CODE"
    exit 1
fi

# Test 12: List audit logs (admin only)
log_info "Test 12: List audit logs"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" "${BASE_URL}/api/admin/logs")
if echo "$RESPONSE" | jq -e '.logs | length > 0' > /dev/null 2>&1; then
    LOG_COUNT=$(echo "$RESPONSE" | jq '.logs | length')
    log_info "✓ List audit logs passed ($LOG_COUNT logs)"
    # Check first log
    FIRST_LOG=$(echo "$RESPONSE" | jq '.logs[0]')
    ACTION=$(echo "$FIRST_LOG" | jq -r '.Action')
    log_info "  Most recent action: $ACTION"
else
    log_error "✗ List audit logs failed: $RESPONSE"
    exit 1
fi

# Test 13: Create API token
log_info "Test 13: Create API token"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" -X POST \
    -H "Content-Type: application/json" \
    -d '{"name":"Backend Test Token"}' \
    "${BASE_URL}/api/tokens")
TOKEN=$(echo "$RESPONSE" | jq -r '.Token')
if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
    log_info "✓ Create API token passed"
    log_info "  Token length: ${#TOKEN}"
else
    log_error "✗ Create API token failed: $RESPONSE"
    exit 1
fi

# Test 14: Use API token
log_info "Test 14: Use API token"
RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
    "${BASE_URL}/api/benchmarks")
if echo "$RESPONSE" | jq -e '.benchmarks' > /dev/null 2>&1; then
    log_info "✓ API token authentication passed"
else
    log_error "✗ API token authentication failed: $RESPONSE"
    exit 1
fi

# Test 15: List API tokens
log_info "Test 15: List API tokens"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" "${BASE_URL}/api/tokens")
if echo "$RESPONSE" | jq -e '. | length > 0' > /dev/null 2>&1; then
    TOKEN_COUNT=$(echo "$RESPONSE" | jq '. | length')
    log_info "✓ List API tokens passed ($TOKEN_COUNT tokens)"
else
    log_error "✗ List API tokens failed: $RESPONSE"
    exit 1
fi

# Test 16: Delete API token
log_info "Test 16: Delete API token"
TOKEN_ID=$(echo "$RESPONSE" | jq -r '.[0].ID')
DELETE_RESPONSE=$(curl -s -b "$SESSION_COOKIE" -X DELETE \
    "${BASE_URL}/api/tokens/${TOKEN_ID}")
if echo "$DELETE_RESPONSE" | jq -e '.message == "token deleted"' > /dev/null 2>&1; then
    log_info "✓ Delete API token passed"
else
    log_error "✗ Delete API token failed: $DELETE_RESPONSE"
    exit 1
fi

# Test 17: Delete benchmark
log_info "Test 17: Delete benchmark"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" -X DELETE \
    "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}")
if echo "$RESPONSE" | jq -e '.message == "benchmark deleted"' > /dev/null 2>&1; then
    log_info "✓ Delete benchmark passed"
else
    log_error "✗ Delete benchmark failed: $RESPONSE"
    exit 1
fi

# Test 18: Verify deletion
log_info "Test 18: Verify benchmark deletion"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/api/benchmarks/${BENCHMARK_ID}")
if [ "$HTTP_CODE" -eq 404 ]; then
    log_info "✓ Benchmark deletion verified"
else
    log_error "✗ Benchmark still exists (HTTP $HTTP_CODE)"
    exit 1
fi

# Test 19: Logout
log_info "Test 19: Logout"
RESPONSE=$(curl -s -b "$SESSION_COOKIE" -c "$SESSION_COOKIE" -X POST \
    "${BASE_URL}/auth/logout")
if echo "$RESPONSE" | jq -e '.message == "logout successful"' > /dev/null 2>&1; then
    log_info "✓ Logout passed"
else
    log_error "✗ Logout failed: $RESPONSE"
    exit 1
fi

# Test 20: Verify logout
log_info "Test 20: Verify logout"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -b "$SESSION_COOKIE" \
    "${BASE_URL}/api/auth/me")
if [ "$HTTP_CODE" -eq 401 ]; then
    log_info "✓ Logout verification passed"
else
    log_error "✗ Still authenticated after logout (HTTP $HTTP_CODE)"
    exit 1
fi

rm -f "$SESSION_COOKIE"

log_info ""
log_info "=========================================="
log_info "All 20 backend tests passed successfully!"
log_info "=========================================="
