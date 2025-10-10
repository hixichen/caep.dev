# SSF Hub Quick Reference

## API Endpoints

### SSF Standard Endpoints
```
POST /events                           # Receive security events from transmitters
GET  /.well-known/ssf_configuration    # SSF metadata discovery
```

### Management Endpoints
```
GET  /health                          # Health check
GET  /ready                           # Readiness check
GET  /metrics                         # Prometheus metrics
```

### Receiver Management API
```
POST   /api/v1/receivers              # Register new receiver
GET    /api/v1/receivers              # List all receivers
GET    /api/v1/receivers/{id}         # Get receiver details
PUT    /api/v1/receivers/{id}         # Update receiver
DELETE /api/v1/receivers/{id}         # Unregister receiver
```

## Quick Setup

### 1. Start the Broker
```bash
export GCP_PROJECT_ID="your-project"
export PUBSUB_TOPIC_PREFIX="ssf-events"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"

go run cmd/server/main.go
```

### 2. Register a Receiver
```bash
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-app",
    "webhook_url": "https://my-app.com/events",
    "event_types": ["https://schemas.openid.net/secevent/caep/event-type/session-revoked"],
    "delivery": {"method": "webhook"},
    "auth": {"type": "bearer", "token": "my-token"}
  }'
```

### 3. Send Test Event
```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "Authorization: Bearer transmitter-token" \
  -d "$SECURITY_EVENT_TOKEN"
```

## Configuration

### Environment Variables
```bash
GCP_PROJECT_ID                # Google Cloud Project ID (required)
PUBSUB_TOPIC_PREFIX          # Pub/Sub topic prefix (default: "ssf-events")
GOOGLE_APPLICATION_CREDENTIALS # Path to service account JSON
SERVER_PORT                  # HTTP server port (default: 8080)
LOG_LEVEL                    # Logging level (default: info)
JWT_SECRET                   # JWT signing secret
```

### Event Types
```
session-revoked              # Session revocation events
credential-change            # Credential change events
assurance-level-change       # Assurance level changes
device-compliance-change     # Device compliance changes
verification                 # Verification events
```

### Delivery Methods
```
webhook                      # HTTP POST to receiver URL
pull                        # Receiver pulls from Pub/Sub subscription
push                        # Pub/Sub pushes to receiver URL
```

### Authentication Types
```
none                        # No authentication
bearer                      # Bearer token
oauth2                      # OAuth2 client credentials
hmac                        # HMAC signature
```

## Common Commands

### Check Service Health
```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### List Receivers
```bash
curl http://localhost:8080/api/v1/receivers | jq .
```

### View Metrics
```bash
curl http://localhost:8080/metrics
```

### Test Receiver Webhook
```bash
curl -X POST https://receiver.com/events \
  -H "Content-Type: application/json" \
  -d '{"test": "event"}'
```

## Docker Commands

### Build Image
```bash
docker build -t ssf-hub:latest .
```

### Run Container
```bash
docker run -p 8080:8080 \
  -e GCP_PROJECT_ID="your-project" \
  -e GOOGLE_APPLICATION_CREDENTIALS="/app/credentials.json" \
  -v /path/to/credentials.json:/app/credentials.json \
  ssf-hub:latest
```

## Kubernetes Commands

### Deploy
```bash
kubectl apply -f deployments/kubernetes/
```

### Check Status
```bash
kubectl get pods -l app=ssf-hub
kubectl logs -l app=ssf-hub
```

### Scale
```bash
kubectl scale deployment ssf-hub --replicas=5
```

## Pub/Sub Commands

### List Topics
```bash
gcloud pubsub topics list --filter="name:ssf-events"
```

### List Subscriptions
```bash
gcloud pubsub subscriptions list --filter="name:ssf-events"
```

### Check Subscription Lag
```bash
gcloud pubsub subscriptions describe ssf-events-receiver-id-session-revoked
```

## Troubleshooting

### Check Logs
```bash
kubectl logs -l app=ssf-hub --tail=100
```

### Debug Receiver
```bash
curl http://localhost:8080/api/v1/receivers/receiver-id
```

### Test Connectivity
```bash
# Test receiver endpoint
curl -I https://receiver.com/events

# Test Pub/Sub
gcloud pubsub topics publish ssf-events-test --message="test"
```

### Common Issues
```
401 Unauthorized      → Check authentication tokens
404 Not Found         → Verify receiver registration
500 Internal Error    → Check logs and Pub/Sub connectivity
High Latency          → Review concurrent processing settings
Memory Issues         → Check max_outstanding_messages setting
```

## Sample Receiver Registration

### Basic Webhook Receiver
```json
{
  "id": "basic-receiver",
  "webhook_url": "https://app.com/events",
  "event_types": ["https://schemas.openid.net/secevent/caep/event-type/session-revoked"],
  "delivery": {"method": "webhook"},
  "auth": {"type": "bearer", "token": "secret-token"}
}
```

### Advanced Filtered Receiver
```json
{
  "id": "filtered-receiver",
  "webhook_url": "https://app.com/events",
  "event_types": ["*"],
  "delivery": {"method": "webhook", "timeout": "30s"},
  "auth": {"type": "oauth2", "client_id": "...", "client_secret": "...", "token_url": "..."},
  "filters": [
    {"field": "subject.format", "operator": "equals", "value": "email"},
    {"field": "source", "operator": "contains", "value": "trusted-issuer"}
  ],
  "retry": {"max_retries": 5, "initial_interval": "1s", "max_interval": "60s"}
}
```

### Pull-based Receiver
```json
{
  "id": "batch-processor",
  "event_types": ["https://schemas.openid.net/secevent/caep/event-type/session-revoked"],
  "delivery": {
    "method": "pull",
    "topic_name": "ssf-events-session-revoked",
    "subscription": "batch-processor-sub",
    "batch_size": 100
  }
}
```

## Monitoring Queries

### Prometheus Queries
```promql
# Event throughput
rate(ssf_broker_events_received_total[5m])

# Error rate
rate(ssf_broker_events_failed_total[5m]) / rate(ssf_broker_events_received_total[5m])

# Processing latency
histogram_quantile(0.95, ssf_broker_event_processing_duration_seconds)

# Active receivers
ssf_broker_active_receivers_total
```

### Log Queries (if using structured logging)
```json
{
  "level": "ERROR",
  "component": "broker"
}

{
  "event_type": "session-revoked",
  "receiver_id": "my-app"
}

{
  "processing_duration": {"$gt": 1000}
}
```