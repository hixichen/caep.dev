# Quick Start: Receiver Integration

This guide helps you quickly integrate your service as an SSF receiver with the SSF Hub.

## Option 1: Hub-Managed Webhooks (Recommended)

### 1. Register Your Receiver

```bash
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-service",
    "name": "My Security Service",
    "webhook_url": "https://my-service.example.com/ssf/webhook",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
      "https://schemas.openid.net/secevent/caep/event-type/credential-change"
    ],
    "delivery": {
      "method": "webhook"
    },
    "auth": {
      "type": "bearer",
      "token": "your-secret-token"
    }
  }'
```

### 2. Implement Webhook Endpoint

```go
func HandleSSFWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Verify authentication
    authHeader := r.Header.Get("Authorization")
    if authHeader != "Bearer your-secret-token" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 2. Parse the security event
    var event SecurityEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // 3. Process the event
    switch event.Type {
    case "https://schemas.openid.net/secevent/caep/event-type/session-revoked":
        handleSessionRevoked(&event)
    case "https://schemas.openid.net/secevent/caep/event-type/credential-change":
        handleCredentialChange(&event)
    default:
        log.Printf("Unknown event type: %s", event.Type)
    }

    // 4. Acknowledge success
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

type SecurityEvent struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Source      string                 `json:"source"`
    Time        time.Time              `json:"time"`
    Subject     Subject                `json:"subject"`
    Data        map[string]interface{} `json:"data"`
}

type Subject struct {
    Format     string `json:"format"`
    Identifier string `json:"identifier"`
}
```

### 3. Test Your Integration

```bash
# Check receiver status
curl http://localhost:8080/api/v1/receivers/my-service

# Send a test event (if you're a transmitter)
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "X-Transmitter-ID: test-transmitter" \
  -d "eyJ0eXAiOiJzZWNldmVudCtqd3QiLCJhbGciOiJub25lIn0..."
```


## Common Patterns

### 1. Event Filtering

```go
func shouldProcessEvent(event *SecurityEvent) bool {
    // Filter by subject format
    if event.Subject.Format != "email" {
        return false
    }

    // Filter by event data
    if severity, ok := event.Data["severity"].(string); ok {
        return severity == "high" || severity == "critical"
    }

    return true
}
```

### 2. Idempotent Processing

```go
var processedEvents = make(map[string]bool)
var mutex sync.RWMutex

func processEventIdempotent(event *SecurityEvent) {
    mutex.Lock()
    defer mutex.Unlock()

    if processedEvents[event.ID] {
        log.Printf("Event %s already processed", event.ID)
        return
    }

    // Process the event
    doProcessEvent(event)

    // Mark as processed
    processedEvents[event.ID] = true
}
```

### 3. Error Handling

```go
func handleSSFWebhook(w http.ResponseWriter, r *http.Request) {
    event, err := parseEvent(r)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }

    if err := processEvent(event); err != nil {
        // Log error but return 200 to prevent retries for permanent failures
        log.Printf("Failed to process event %s: %v", event.ID, err)

        // Return 500 for temporary failures that should be retried
        if isTemporaryError(err) {
            http.Error(w, "Temporary Error", http.StatusInternalServerError)
            return
        }
    }

    w.WriteHeader(http.StatusOK)
}
```

## Testing Your Receiver

### 1. Unit Tests

```go
func TestHandleSessionRevoked(t *testing.T) {
    event := &SecurityEvent{
        ID:   "test-event-123",
        Type: "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
        Subject: Subject{
            Format:     "email",
            Identifier: "user@example.com",
        },
        Data: map[string]interface{}{
            "reason": "admin_revocation",
        },
    }

    err := handleSessionRevoked(event)
    assert.NoError(t, err)

    // Verify expected side effects
    // e.g., user session was terminated
}
```

### 2. Integration Tests

```go
func TestWebhookIntegration(t *testing.T) {
    // Start test server
    server := httptest.NewServer(http.HandlerFunc(HandleSSFWebhook))
    defer server.Close()

    // Register test receiver
    registerReceiver(t, server.URL+"/ssf/webhook")

    // Send test event
    sendTestEvent(t)

    // Verify event was processed
    // Check logs, database, etc.
}
```

## Monitoring Your Receiver

### 1. Health Checks

```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now(),
        "events_processed": eventCounter,
        "last_event_at": lastEventTime,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}
```

### 2. Metrics

```go
var (
    eventsReceived = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ssf_events_received_total",
            Help: "Total number of SSF events received",
        },
        []string{"event_type", "source"},
    )

    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ssf_event_processing_duration_seconds",
            Help: "Time spent processing SSF events",
        },
        []string{"event_type"},
    )
)

func processEventWithMetrics(event *SecurityEvent) {
    start := time.Now()

    eventsReceived.WithLabelValues(event.Type, event.Source).Inc()

    processEvent(event)

    processingDuration.WithLabelValues(event.Type).Observe(time.Since(start).Seconds())
}
```

## Best Practices

1. **Authentication**: Always validate webhook authentication
2. **Idempotency**: Handle duplicate events gracefully
3. **Error Handling**: Distinguish between temporary and permanent failures
4. **Monitoring**: Track processing metrics and health
5. **Security**: Use HTTPS for webhook endpoints
6. **Filtering**: Apply client-side filtering to reduce processing load
7. **Async Processing**: Consider async processing for heavy operations

## Next Steps

- Read the full [Receiver Integration Guide](RECEIVER_INTEGRATION.md)
- Check the [API Reference](API_REFERENCE.md) for complete API documentation
- See [Example Integrations](examples/) for language-specific examples