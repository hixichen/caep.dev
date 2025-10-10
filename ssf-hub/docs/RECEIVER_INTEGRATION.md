# SSF Hub Receiver Integration Guide

This guide explains how receivers can consume security events from the SSF Hub and how the hub itself operates as a receiver.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  SSF Transmitter│    │    SSF Hub      │    │  Event Receivers│
│                 │    │                 │    │                 │
│  • OIDC Provider│───▶│  • Event Router │───▶│  • SIEM Systems │
│  • Identity Mgmt│    │  • Hub Receiver │    │  • Auth Services│
│  • Security Svc │    │  • Distribution │    │  • Audit Systems│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Message Flow Architecture

### 1. Internal Hub Processing
```
┌─────────────────────────────────────────────────────────────────┐
│                        SSF Hub Internal                         │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ Transmitter │  │     Hub     │  │    Unified Topic        │ │
│  │   Events    │─▶│  Processor  │─▶│   (ssf-hub-events)     │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│                                                ▲                │
│  ┌─────────────┐  ┌─────────────┐              │                │
│  │     Hub     │  │  Webhook    │              │                │
│  │  Subscriber │◀─│ Distributor │◀─────────────┘                │
│  └─────────────┘  └─────────────┘                               │
│         │                  │                                    │
│         ▼                  ▼                                    │
│  ┌─────────────┐  ┌─────────────┐                               │
│  │ Internal    │  │ External    │                               │
│  │ Processing  │  │ Receivers   │                               │
│  └─────────────┘  └─────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
```

### 2. Message Format in Unified Topic

All events in the unified topic use the `InternalMessage` schema:

```json
{
  "message_id": "msg_20231201120000_123456",
  "message_type": "security_event",
  "version": "1.0",
  "timestamp": "2023-12-01T12:00:00Z",
  "event": {
    "id": "evt_session_revoked_789",
    "type": "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "source": "https://idp.example.com",
    "time": "2023-12-01T12:00:00Z",
    "subject": {
      "format": "email",
      "identifier": "user@example.com"
    },
    "data": {
      "reason": "admin_revocation"
    },
    "metadata": {
      "received_at": "2023-12-01T12:00:00Z",
      "transmitter_id": "idp-123"
    }
  },
  "routing": {
    "target_receivers": ["siem-001", "auth-service-002"],
    "event_type": "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "subject": "user@example.com",
    "priority": 0,
    "ttl": "24h",
    "tags": {
      "environment": "production",
      "tenant": "acme-corp"
    }
  },
  "metadata": {
    "hub_instance_id": "hub_1701432000_123456",
    "processing_id": "proc_abc123",
    "retry_count": 0,
    "created_at": "2023-12-01T12:00:00Z"
  }
}
```

## Receiver Consumption Patterns

### 1. Hub-Managed Webhook Delivery (Recommended)

**How it works:**
- Receivers register with the hub via REST API
- Hub delivers events via HTTP webhooks
- Hub handles retries, filtering, and routing automatically

**Registration:**
```bash
curl -X POST https://ssf-hub.example.com/api/v1/receivers \\
  -H "Content-Type: application/json" \\
  -d '{
    "id": "my-service-001",
    "name": "My Security Service",
    "webhook_url": "https://my-service.example.com/ssf/events",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
      "https://schemas.openid.net/secevent/caep/event-type/credential-change"
    ],
    "delivery": {
      "method": "webhook",
      "batch_size": 1,
      "timeout": "30s"
    },
    "auth": {
      "type": "bearer",
      "token": "your-webhook-auth-token"
    },
    "filters": [
      {
        "field": "subject.format",
        "operator": "equals",
        "value": "email"
      }
    ]
  }'
```

**Webhook Endpoint Implementation:**
```go
func (s *MyService) HandleSSFEvent(w http.ResponseWriter, r *http.Request) {
    // Validate authentication
    if !s.validateAuth(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse the security event
    var event models.SecurityEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // Process the event
    if err := s.processSecurityEvent(&event); err != nil {
        http.Error(w, "Processing failed", http.StatusInternalServerError)
        return
    }

    // Acknowledge successful processing
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

### 2. Direct Pub/Sub Subscription (Advanced)

**When to use:**
- High-throughput scenarios requiring direct control
- Custom filtering logic beyond hub capabilities
- Integration with existing Pub/Sub infrastructure

**Implementation:**
```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "cloud.google.com/go/pubsub"
    "github.com/sgnl-ai/caep.dev/ssfreceiver/ssf-hub/pkg/models"
)

type DirectReceiver struct {
    client       *pubsub.Client
    subscription *pubsub.Subscription
}

func (r *DirectReceiver) Start(ctx context.Context) error {
    return r.subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
        // Parse internal message
        var internalMsg models.InternalMessage
        if err := json.Unmarshal(msg.Data, &internalMsg); err != nil {
            log.Printf("Failed to parse internal message: %v", err)
            msg.Nack()
            return
        }

        // Check if this message is for us
        if !r.isTargetReceiver(internalMsg.Routing.TargetReceivers) {
            msg.Ack() // Acknowledge but don't process
            return
        }

        // Apply additional filtering
        if !r.shouldProcess(&internalMsg) {
            msg.Ack()
            return
        }

        // Process the security event
        if err := r.processEvent(internalMsg.Event); err != nil {
            log.Printf("Failed to process event: %v", err)
            msg.Nack()
            return
        }

        msg.Ack()
    })
}

func (r *DirectReceiver) isTargetReceiver(targets []string) bool {
    myID := "my-receiver-id"
    for _, target := range targets {
        if target == myID {
            return true
        }
    }
    return false
}
```

### 3. Hub-to-Hub Federation

**Scenario:** Multiple SSF hubs in different regions/environments

```go
type FederatedHubReceiver struct {
    localHub  *Hub
    remoteHub string
}

func (f *FederatedHubReceiver) RegisterWithRemote() error {
    // Register this hub as a receiver with the remote hub
    receiverReq := &models.ReceiverRequest{
        ID:         "hub-us-west",
        Name:       "US West Hub",
        WebhookURL: "https://hub-us-west.example.com/api/v1/federation/events",
        EventTypes: []string{"*"}, // All event types
        Delivery: models.DeliveryConfig{
            Method: models.DeliveryMethodWebhook,
        },
        Auth: models.AuthConfig{
            Type:  models.AuthTypeBearer,
            Token: f.getFederationToken(),
        },
        Tags: map[string]string{
            "type":   "federation",
            "region": "us-west",
        },
    }

    return f.registerWithHub(f.remoteHub, receiverReq)
}

func (f *FederatedHubReceiver) HandleFederatedEvent(event *models.SecurityEvent) error {
    // Process federated event locally
    return f.localHub.ProcessSecurityEvent(context.Background(),
        event.Metadata.RawSET, "federated-"+event.Source)
}
```

## Hub as a Receiver Model

### 1. Hub Internal Receiver Registration

The SSF Hub itself should be registered as a receiver to consume from the unified topic:

```go
type HubReceiver struct {
    hubID        string
    subscription string
    distributor  *EventDistributor
}

func (h *Hub) InitializeAsReceiver() error {
    // Create hub's internal subscription to unified topic
    subscription := h.generateHubSubscriptionName()

    if err := h.pubsubClient.CreateHubSubscription(context.Background(), subscription); err != nil {
        return fmt.Errorf("failed to create hub subscription: %w", err)
    }

    // Start consuming internal messages
    go h.consumeInternalMessages(subscription)

    return nil
}

func (h *Hub) consumeInternalMessages(subscription string) {
    ctx := context.Background()

    for {
        err := h.pubsubClient.PullInternalMessages(ctx, subscription, 100, h.processInternalMessage)
        if err != nil {
            h.logger.Error("Failed to pull internal messages", "error", err)
            time.Sleep(5 * time.Second)
            continue
        }
    }
}

func (h *Hub) processInternalMessage(msg *models.InternalMessage) error {
    // Distribute to external receivers
    return h.distributor.DistributeToReceivers(msg)
}
```

### 2. Event Distribution Service

```go
type EventDistributor struct {
    registry    registry.Registry
    httpClient  *http.Client
    retryPolicy *RetryPolicy
}

func (d *EventDistributor) DistributeToReceivers(msg *models.InternalMessage) error {
    for _, receiverID := range msg.Routing.TargetReceivers {
        receiver, err := d.registry.Get(receiverID)
        if err != nil {
            d.logger.Error("Receiver not found", "receiver_id", receiverID)
            continue
        }

        // Deliver based on receiver's delivery method
        switch receiver.Delivery.Method {
        case models.DeliveryMethodWebhook:
            go d.deliverWebhook(receiver, msg.Event)
        default:
            d.logger.Warn("Unsupported delivery method", "method", receiver.Delivery.Method)
        }
    }

    return nil
}

func (d *EventDistributor) deliverWebhook(receiver *models.Receiver, event *models.SecurityEvent) {
    // Apply receiver-specific filtering
    if !d.shouldDeliverToReceiver(receiver, event) {
        return
    }

    // Create delivery attempt
    delivery := &models.EventDelivery{
        DeliveryID: generateDeliveryID(),
        ReceiverID: receiver.ID,
        EventID:    event.ID,
        Attempt:    1,
        Status:     models.DeliveryStatusPending,
    }

    // Perform delivery with retries
    d.performDeliveryWithRetries(receiver, event, delivery)
}
```

## Receiver Types and Use Cases

### 1. SIEM Integration
```yaml
# Example: Splunk SIEM
receiver_id: "splunk-siem-001"
name: "Splunk SIEM"
webhook_url: "https://splunk.example.com/services/collector/event"
event_types: ["*"]  # All events
auth:
  type: "bearer"
  token: "Splunk-Token"
filters:
  - field: "data.severity"
    operator: "in"
    value: ["high", "critical"]
```

### 2. Authentication Service
```yaml
# Example: Auth0 Integration
receiver_id: "auth0-service"
name: "Auth0 Authentication"
webhook_url: "https://auth0-webhook.example.com/ssf"
event_types:
  - "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
  - "https://schemas.openid.net/secevent/caep/event-type/credential-change"
filters:
  - field: "subject.format"
    operator: "equals"
    value: "email"
```

### 3. Real-time Analytics
```yaml
# Example: Real-time dashboard
receiver_id: "analytics-dashboard"
name: "Security Analytics Dashboard"
webhook_url: "https://dashboard.example.com/api/events"
event_types: ["*"]
delivery:
  batch_size: 10
  timeout: "5s"
tags:
  purpose: "analytics"
  priority: "real-time"
```

## Best Practices

### 1. Receiver Implementation
- **Idempotency**: Handle duplicate event deliveries gracefully
- **Authentication**: Always validate webhook authentication
- **Error Handling**: Return appropriate HTTP status codes
- **Filtering**: Implement additional client-side filtering as needed
- **Monitoring**: Track delivery success/failure rates

### 2. Hub Configuration
- **Subscription Naming**: Use consistent naming for hub subscriptions
- **Dead Letter Queues**: Configure DLQs for failed deliveries
- **Monitoring**: Monitor internal message processing latency
- **Scaling**: Auto-scale based on message volume

### 3. Security Considerations
- **Network Security**: Use TLS for all communications
- **Authentication**: Implement strong authentication for webhooks
- **Authorization**: Validate receiver permissions for event types
- **Audit Logging**: Log all delivery attempts and outcomes

## Monitoring and Observability

### Key Metrics
```yaml
# Hub Internal Metrics
ssf_hub_internal_messages_processed_total
ssf_hub_internal_messages_failed_total
ssf_hub_distribution_latency_seconds
ssf_hub_receiver_delivery_success_rate

# Receiver Metrics
ssf_receiver_events_received_total
ssf_receiver_events_processed_total
ssf_receiver_processing_duration_seconds
ssf_receiver_webhook_response_codes
```

### Health Checks
```go
func (h *Hub) HealthCheck() error {
    // Check hub subscription health
    if !h.isHubSubscriptionHealthy() {
        return fmt.Errorf("hub subscription unhealthy")
    }

    // Check receiver delivery health
    if !h.isDeliverySystemHealthy() {
        return fmt.Errorf("delivery system unhealthy")
    }

    return nil
}
```

This architecture ensures that the SSF Hub operates as both a receiver (consuming from the unified topic) and a distributor (delivering to external receivers), maintaining consistency with the receiver model throughout the system.