#!/usr/bin/env bash
# demo.sh — End-to-end flow demonstration for the Polyglot SMS Service
# Run from the repo root: bash demo.sh
set -euo pipefail

SMS_SENDER="http://localhost:8080"
SMS_STORE="http://localhost:8081"
REDIS_CLI="docker exec redis redis-cli"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log() { echo -e "${GREEN}[demo]${NC} $*"; }
warn() { echo -e "${YELLOW}[demo]${NC} $*"; }
fail() { echo -e "${RED}[demo]${NC} $*"; exit 1; }

# ── 0. Health checks ──────────────────────────────────────────────────────────
log "Step 0: Checking service health..."
curl -sf "$SMS_SENDER/v1/sms/health" | jq . || fail "sms-sender is not running"
curl -sf "$SMS_STORE/health"          | jq . || fail "sms-store is not running"

# ── 1. Send a successful SMS ──────────────────────────────────────────────────
log "Step 1: Sending a normal SMS..."
RESPONSE=$(curl -sf -X POST "$SMS_SENDER/v1/sms/send" \
  -H "Content-Type: application/json" \
  -d '{
        "userId": "user-demo-001",
        "phoneNumber": "+919876543210",
        "message": "Hello from the Polyglot SMS demo!"
      }')
echo "$RESPONSE" | jq .
STATUS=$(echo "$RESPONSE" | jq -r .status)
MSG_ID=$(echo "$RESPONSE" | jq -r .messageId)
log "SMS send result: status=$STATUS  messageId=$MSG_ID"

# ── 2. Block a user ───────────────────────────────────────────────────────────
log "Step 2: Blocking user 'user-blocked-999' in Redis..."
$REDIS_CLI SADD "blocked:users" "user-blocked-999"
log "Block list after add:"
$REDIS_CLI SMEMBERS "blocked:users"

# ── 3. Attempt to send to blocked user ───────────────────────────────────────
log "Step 3: Attempting to send SMS to blocked user (expect BLOCKED)..."
curl -s -X POST "$SMS_SENDER/v1/sms/send" \
  -H "Content-Type: application/json" \
  -d '{
        "userId": "user-blocked-999",
        "phoneNumber": "+911234567890",
        "message": "You should not see this"
      }' | jq .

# ── 4. Send a second SMS for the demo user ───────────────────────────────────
log "Step 4: Sending a second SMS for user-demo-001..."
curl -sf -X POST "$SMS_SENDER/v1/sms/send" \
  -H "Content-Type: application/json" \
  -d '{
        "userId": "user-demo-001",
        "phoneNumber": "+919876543210",
        "message": "Second message in the demo!"
      }' | jq .

# ── 5. Wait for Kafka consumer to process ────────────────────────────────────
log "Step 5: Waiting 3 seconds for Kafka consumer to store events..."
sleep 3

# ── 6. Retrieve SMS history from GoLang SMS Store ────────────────────────────
log "Step 6: Retrieving SMS history for user-demo-001 from SMS Store..."
curl -sf "$SMS_STORE/v1/user/user-demo-001/messages" | jq .

# ── 7. Retrieve history for blocked user (BLOCKED event should be stored) ────
log "Step 7: Retrieving SMS history for blocked user..."
curl -sf "$SMS_STORE/v1/user/user-blocked-999/messages" | jq .

# ── 8. Cleanup — unblock user ─────────────────────────────────────────────────
log "Step 8: Removing user-blocked-999 from block list..."
$REDIS_CLI SREM "blocked:users" "user-blocked-999"

log "✅  End-to-end demo complete!"
