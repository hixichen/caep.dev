# SSF Hub Local Development Guide

This guide helps developers set up, debug, and develop the SSF Hub locally.

## Quick Start

### Prerequisites

- Go 1.21+
- `curl` or `httpie` for API testing
- **Optional:** Docker (for Google Pub/Sub emulator)
- **Optional:** `gcloud` CLI (for real GCP)

### Option A: In-Memory Mock (No GCP Required) 🚀

**Perfect when you can't get GCP service accounts!**

```bash
# Clone and build
git clone <repository-url>
cd ssf-hub
go mod download

# Run with in-memory mock (no GCP needed!)
make run-mock
```

The hub will start on `http://localhost:8080` with everything running in memory!

### Option B: Google Pub/Sub Emulator

If you want to test with real Pub/Sub behavior:

```bash
# Start the Google Cloud Pub/Sub emulator
docker run -d --name pubsub-emulator \
  -p 8085:8085 \
  gcr.io/google.com/cloudsdktool/cloud-sdk:emulators \
  /bin/sh -c "gcloud beta emulators pubsub start --host-port=0.0.0.0:8085"

# Set environment variable to use emulator
export PUBSUB_EMULATOR_HOST=localhost:8085

# Run with emulator
make run-dev
```

### Option C: Real GCP (If You Have Service Account)

Create `.env.local`:
```bash
# Local development configuration
GCP_PROJECT_ID=your-real-project
GOOGLE_APPLICATION_CREDENTIALS=./service-account-key.json
LOG_LEVEL=debug
SERVER_PORT=8080
JWT_SECRET=local-dev-secret-change-in-production
REQUIRE_AUTH=false
```

```bash
# Load environment and run
source .env.local
make run-dev
```

## Development Workflow

### Architecture Overview

```
[SSF Transmitter] → [SSF Hub:8080] → [ssf-hub-events topic] → [Hub Receiver] → [Webhook to Services]
                                            ↑
                                    Single unified topic
                                   (auto-created locally)
```

### Core Components

1. **HTTP Handlers** (`internal/handlers/`) - REST API endpoints
2. **Broker/Controller** (`internal/controller/`) - Main business logic
3. **Pub/Sub Client** (`internal/pubsub/`) - Unified topic management
4. **Hub Receiver** (`internal/hubreceiver/`) - Internal event consumption
5. **Event Distributor** (`internal/distributor/`) - Webhook delivery
6. **Registry** (`internal/registry/`) - Receiver management

## Debugging Guide

### 1. Verify Hub is Running

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","timestamp":"2023-12-01T12:00:00Z"}

# SSF configuration endpoint
curl http://localhost:8080/.well-known/ssf_configuration
```

### 2. Register a Test Receiver

```bash
# Register a test webhook receiver
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-receiver",
    "name": "Test Receiver",
    "webhook_url": "http://httpbin.org/post",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
    ],
    "delivery": {
      "method": "webhook"
    },
    "auth": {
      "type": "none"
    }
  }'

# List receivers
curl http://localhost:8080/api/v1/receivers
```

### 2.1. Mock-Specific Debug Endpoints

When using the in-memory mock (`make run-mock`), you get additional debug endpoints:

```bash
# Check mock statistics
curl http://localhost:8080/debug/mock/stats | jq .

# Clear all mock messages (useful for testing)
curl -X POST http://localhost:8080/debug/mock/clear
```

**Example mock stats output:**
```json
{
  "project_id": "mock-local-project",
  "unified_topic": "ssf-hub-events",
  "hub_instance_id": "hub_1699123456_abc123",
  "topics_count": 1,
  "subscriptions_count": 1,
  "topics": {
    "ssf-hub-events": {
      "message_count": 5,
      "created_at": "2023-12-01T12:00:00Z"
    }
  },
  "subscriptions": {
    "ssf-hub-subscription-hub_123_abc": {
      "topic_name": "ssf-hub-events",
      "message_count": 5,
      "created_at": "2023-12-01T12:00:00Z"
    }
  }
}
```

### 3. Complete End-to-End Testing with Mock

Here's a step-by-step guide to test the full event flow:

#### Step 1: Start Mock Hub
```bash
make run-mock
# Hub starts on http://localhost:8080
```

#### Step 2: Register a Test Receiver
```bash
# Register a receiver that will catch our events
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-receiver",
    "name": "Test Receiver",
    "webhook_url": "https://httpbin.org/post",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
    ],
    "delivery": {
      "method": "webhook"
    },
    "auth": {
      "type": "none"
    }
  }'
```

#### Step 3: Send a Test Event
```bash
# Send a simplified Security Event (mock accepts JSON format for testing)
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "X-Transmitter-ID: test-transmitter" \
  -d '{
    "iss": "https://test-transmitter.example.com",
    "jti": "test-event-123",
    "iat": 1699123456,
    "events": {
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
        "subject": {
          "format": "email",
          "identifier": "user@example.com"
        }
      }
    }
  }'
```

**Note:** The mock currently expects JWT format but may fail parsing. For local testing, events get processed through the mock pipeline regardless.

#### Step 4: Verify Event Processing

**Check if event reached the unified topic:**
```bash
curl http://localhost:8080/debug/mock/stats | jq '.topics."ssf-hub-events".message_count'
# Should show: 1 (or more if you sent multiple events)
```

**Check if hub subscription received the event:**
```bash
curl http://localhost:8080/debug/mock/stats | jq '.subscriptions | keys'
# Should show: ["ssf-hub-subscription-hub_123456_abc"]

curl http://localhost:8080/debug/mock/stats | jq '.subscriptions | to_entries[0].value.message_count'
# Should show: 1 (or more)
```

**Check receiver registration:**
```bash
curl http://localhost:8080/api/v1/receivers | jq '.[0].id'
# Should show: "test-receiver"
```

#### Step 5: Monitor Webhook Delivery (Advanced)

Since the mock delivers webhooks to real URLs, you can monitor the delivery:

**Using httpbin.org:**
```bash
# Your events should appear at: https://httpbin.org/post
# Check the hub logs for delivery attempts
```

**Using webhook.site (recommended for testing):**
```bash
# 1. Go to https://webhook.site and get a unique URL
# 2. Register receiver with that URL:
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "webhook-site-receiver",
    "name": "Webhook Site Receiver",
    "webhook_url": "https://webhook.site/your-unique-id",
    "event_types": ["*"],
    "delivery": {"method": "webhook"},
    "auth": {"type": "none"}
  }'

# 3. Send test event (as above)
# 4. Check webhook.site to see the delivered event payload
```

#### Step 6: Full Event Flow Verification

**Complete verification script:**
```bash
#!/bin/bash
echo "🚀 Testing SSF Hub Mock End-to-End..."

# 1. Check hub health
echo "1. Checking hub health..."
curl -s http://localhost:8080/health | jq '.status'

# 2. Register receiver
echo "2. Registering test receiver..."
curl -s -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-receiver",
    "webhook_url": "https://httpbin.org/post",
    "event_types": ["https://schemas.openid.net/secevent/caep/event-type/session-revoked"],
    "delivery": {"method": "webhook"},
    "auth": {"type": "none"}
  }' | jq '.id'

# 3. Clear mock state
echo "3. Clearing mock state..."
curl -s -X POST http://localhost:8080/debug/mock/clear | jq '.status'

# 4. Send test event
echo "4. Sending test event..."
curl -s -X POST http://localhost:8080/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "X-Transmitter-ID: test-transmitter" \
  -d '{"iss":"test","jti":"event-123","events":{"https://schemas.openid.net/secevent/caep/event-type/session-revoked":{"subject":{"identifier":"user@test.com"}}}}'

# 5. Check results
sleep 1  # Give time for processing
echo "5. Checking mock stats..."
curl -s http://localhost:8080/debug/mock/stats | jq '{
  topic_messages: .topics."ssf-hub-events".message_count,
  subscription_messages: (.subscriptions | to_entries[0].value.message_count // 0),
  total_receivers: (.subscriptions | length)
}'

echo "✅ Test complete! Check the output above."
```

#### Quick Testing Commands

**One-liner to test everything:**
```bash
# Run the automated test script
./scripts/test-mock.sh
```

**Manual quick tests:**
```bash
# 1. Check health
curl http://localhost:8080/health | jq .

# 2. Register receiver
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{"id":"quick-test","webhook_url":"https://httpbin.org/post","event_types":["*"],"delivery":{"method":"webhook"},"auth":{"type":"none"}}'

# 3. Send event
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "X-Transmitter-ID: test" \
  -d '{"iss":"test","jti":"quick-'$(date +%s)'","events":{"https://schemas.openid.net/secevent/caep/event-type/session-revoked":{"subject":{"identifier":"user@test.com"}}}}'

# 4. Check results
curl http://localhost:8080/debug/mock/stats | jq '{topic_messages: .topics."ssf-hub-events".message_count, receivers: (.subscriptions | length)}'
```

**Event validation checklist:**
```bash
# ✅ Event reached topic?
curl -s http://localhost:8080/debug/mock/stats | jq '.topics."ssf-hub-events".message_count'

# ✅ Hub subscription processed it?
curl -s http://localhost:8080/debug/mock/stats | jq '.subscriptions | to_entries[0].value.message_count'

# ✅ Receivers registered?
curl -s http://localhost:8080/api/v1/receivers | jq 'length'

# ✅ Hub healthy?
curl -s http://localhost:8080/health | jq '.status'
```

### 4. Monitor Local Pub/Sub

```bash
# Install the Pub/Sub CLI tool
pip install google-cloud-pubsub

# List topics (should show ssf-hub-events)
gcloud pubsub topics list --project=local-dev-project

# List subscriptions
gcloud pubsub subscriptions list --project=local-dev-project

# Pull messages manually (for debugging)
gcloud pubsub subscriptions pull ssf-hub-subscription-<hub-id> \
  --auto-ack --limit=10 --project=local-dev-project
```

### 5. Debug Logs

Enable debug logging to see detailed event flow:

```bash
# Set debug level
export LOG_LEVEL=debug

# Watch logs in real-time
make run-dev | jq .  # Pretty print JSON logs
```

**Key log patterns to watch for:**
- `"event_id"` - Track specific event processing
- `"receiver_id"` - See which receivers are being targeted
- `"unified_topic"` - Pub/Sub operations
- `"webhook_delivery"` - Outbound webhook attempts

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test file
go test -v ./internal/controller

# Run specific test
go test -v ./internal/controller -run TestController_ProcessSecurityEvent
```

### Integration Testing

```bash
# Start dependencies
docker-compose up -d  # If you have docker-compose.yml

# Run integration tests
make test-integration

# Manual integration test
./scripts/integration-test.sh  # If available
```

### Load Testing

```bash
# Simple load test with curl
for i in {1..100}; do
  curl -X POST http://localhost:8080/events \
    -H "Content-Type: application/secevent+jwt" \
    -d "test-event-$i" &
done
wait

# Check performance
curl http://localhost:8080/metrics | grep ssf_
```

## Common Issues & Solutions

### 1. Mock Mode Issues

**Error:** `cannot find package "github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/internal/pubsub"`

**Solution:**
```bash
# Make sure you're in the ssf-hub directory
cd ssf-hub
go mod tidy
make run-mock
```

**No messages appearing in mock stats:**
```bash
# Check if events are being sent
curl http://localhost:8080/debug/mock/stats | jq '.topics'

# Clear messages and try again
curl -X POST http://localhost:8080/debug/mock/clear
# Send test event...
curl http://localhost:8080/debug/mock/stats | jq '.topics'
```

### 2. Pub/Sub Emulator Connection Failed

**Error:** `transport: Error while dialing dial tcp [::1]:8085: connect: connection refused`

**Solution:**
```bash
# Check if emulator is running
docker ps | grep pubsub-emulator

# Restart emulator
docker restart pubsub-emulator

# Verify connection
export PUBSUB_EMULATOR_HOST=localhost:8085
echo $PUBSUB_EMULATOR_HOST
```

### 2. JWT Parsing Errors

**Error:** `failed to parse token: token is malformed`

**Solutions:**
1. Use proper JWT format for events
2. For local testing, temporarily disable verification in `broker.go`:
   ```go
   // Change from:
   secEvent, err := b.parser.ParseSecEvent(rawSET)
   // To:
   secEvent, err := b.parser.ParseSecEventNoVerify(rawSET)
   ```

### 3. Receiver Registration Fails

**Error:** `receiver validation failed: webhook_url is required`

**Solution:** Ensure your receiver registration includes all required fields:
```json
{
  "id": "required",
  "webhook_url": "required-for-webhook-delivery",
  "event_types": ["required-array"],
  "delivery": {"method": "webhook"},
  "auth": {"type": "none"}
}
```

### 4. No Events Received by Webhook

**Debug steps:**
1. Check receiver registration: `curl http://localhost:8080/api/v1/receivers`
2. Verify hub logs for delivery attempts
3. Test webhook endpoint independently: `curl <your-webhook-url>`
4. Check if event type matches receiver subscription

### 5. High Memory Usage

**Solutions:**
```bash
# Monitor memory
go tool pprof http://localhost:8080/debug/pprof/heap

# Reduce Pub/Sub buffer sizes in config
export PUBSUB_MAX_OUTSTANDING_MESSAGES=100
export PUBSUB_MAX_OUTSTANDING_BYTES=10000000
```

## Development Tips

### 1. Use JSON Log Formatting

```bash
# Pretty print logs
make run-dev | jq 'select(.level == "ERROR")' # Only errors
make run-dev | jq 'select(.receiver_id != null)' # Only receiver logs
```

### 2. Mock External Services

For webhook testing, use:
- [httpbin.org](http://httpbin.org/post) - Simple webhook target
- [webhook.site](https://webhook.site) - Inspect webhook payload
- [ngrok](https://ngrok.com) - Expose local services

### 3. Database Debugging

The memory registry doesn't persist data. For debugging:
```bash
# Check current receivers
curl http://localhost:8080/api/v1/receivers | jq .

# Get hub statistics
curl http://localhost:8080/api/v1/stats | jq .
```

### 4. Performance Profiling

```bash
# Enable pprof endpoint (add to main.go)
import _ "net/http/pprof"

# Profile CPU
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=10

# Profile memory
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Configuration Reference

### Environment Variables

| Variable | Description | Default | Local Dev Value |
|----------|-------------|---------|-----------------|
| `GCP_PROJECT_ID` | GCP project | Required | `local-dev-project` |
| `PUBSUB_EMULATOR_HOST` | Pub/Sub emulator | None | `localhost:8085` |
| `LOG_LEVEL` | Logging level | `info` | `debug` |
| `SERVER_PORT` | HTTP port | `8080` | `8080` |
| `JWT_SECRET` | JWT signing key | Required | `local-dev-secret` |
| `REQUIRE_AUTH` | Enforce auth | `false` | `false` |

### Unified Topic Configuration

The hub uses a single topic: `ssf-hub-events`
- All event types are published here
- Hub creates topic automatically
- Subscription name: `ssf-hub-subscription-{hub-instance-id}`

## API Endpoints

### Core SSF Endpoints
- `POST /events` - Receive security events from transmitters
- `GET /.well-known/ssf_configuration` - SSF discovery

### Management API
- `GET /api/v1/receivers` - List receivers
- `POST /api/v1/receivers` - Register receiver
- `PUT /api/v1/receivers/{id}` - Update receiver
- `DELETE /api/v1/receivers/{id}` - Remove receiver
- `GET /api/v1/stats` - Hub statistics

### Health & Debug
- `GET /health` - Health check
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics
- `GET /debug/pprof/*` - Go profiling (if enabled)

## VS Code Setup

Recommended `.vscode/settings.json`:
```json
{
  "go.testFlags": ["-v"],
  "go.buildTags": "dev",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "files.associations": {
    "*.md": "markdown"
  }
}
```

Recommended `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch SSF Hub",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "GCP_PROJECT_ID": "local-dev-project",
        "PUBSUB_EMULATOR_HOST": "localhost:8085",
        "LOG_LEVEL": "debug",
        "REQUIRE_AUTH": "false"
      }
    }
  ]
}
```

---

## Getting Help

- Check logs first: `make run-dev | jq .`
- Verify Pub/Sub emulator: `docker logs pubsub-emulator`
- Test endpoints: Use the curl examples above
- Run tests: `make test`
- File issues: [GitHub Issues](https://github.com/your-repo/issues)

Happy debugging! 🚀