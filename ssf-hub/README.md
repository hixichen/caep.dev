# SSF Hub Service

A centralized SSF (Shared Signals Framework) broker service that acts as an event distribution hub using Google Cloud Pub/Sub as the backend.

## Overview

The SSF Hub Service provides:

- **SSF Receiver**: Standards-compliant SSF receiver that accepts events from any SSF transmitter
- **Event Broker**: Centralized hub for distributing security events to multiple consumers
- **Registration API**: Allows receivers to register for specific event types
- **Pub/Sub Backend**: Uses Google Cloud Pub/Sub for reliable event distribution
- **Deployment Ready**: Kubernetes-ready service with health checks and monitoring

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           SSF Hub Service Architecture                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  [SSF Transmitters] ──HTTP──► [Event Processor] ──► [Unified Topic]            │
│                                      │                    │                    │
│                                      ▼                    ▼                    │
│                              [Registration API]  [Hub Receiver] ──────┐        │
│                                                          │             │        │
│                                                          ▼             ▼        │
│                                                   [Event Distributor]  │        │
│                                                          │             │        │
│                                                          ▼             │        │
│  [External Receivers] ◄──HTTP Webhooks──── [Delivery Service]         │        │
│  • SIEM Systems                                                        │        │
│  • Auth Services                                                       │        │
│  • Analytics                                                           │        │
│                                                                         │        │
│  [Advanced Integrations] ◄──Direct Pub/Sub──────────────────────────────┘        │
│  • Custom Receivers                                                             │
│  • Federated Hubs                                                               │
└─────────────────────────────────────────────────────────────────────────────────┘

Key Components:
• Event Processor: Receives events from transmitters, validates, and routes
• Unified Topic: Single Pub/Sub topic with structured internal messages
• Hub Receiver: Hub acts as receiver to consume from its own topic
• Event Distributor: Delivers events to external receivers via webhooks
• Registration API: Manages receiver registration and configuration
```

## Features

### SSF Compliance
- Standard SSF receiver endpoints (`POST /events`, `/.well-known/ssf_configuration`)
- Security Event Token (SET) validation and parsing
- Support for CAEP event types (session-revoked, credential-change, etc.)
- OAuth2 and Bearer token authentication

### Event Brokering
- Centralized event ingestion from multiple transmitters
- Single unified topic with structured internal message schema
- Hub-as-receiver pattern for consistent event processing
- Event filtering and routing based on receiver preferences
- Dead letter handling for failed deliveries
- Event transformation and enrichment

### Receiver Management
- REST API for receiver registration and management
- Hub-managed webhook delivery (recommended)
- Direct Pub/Sub access for advanced integrations
- Event type subscriptions and filtering
- Authentication and authorization for receivers
- Hub federation support for multi-region deployments

### Operational Excellence
- Health and readiness endpoints
- Prometheus metrics
- Structured logging
- Graceful shutdown
- Kubernetes deployment manifests

## Quick Start

### Prerequisites

- Go 1.21+
- Google Cloud Project with Pub/Sub enabled
- Kubernetes cluster (optional, for deployment)

### Local Development

```bash
# Clone and navigate to the service
cd ssf-broker

# Install dependencies
go mod tidy

# Set environment variables
export GCP_PROJECT_ID="your-project"
export PUBSUB_TOPIC_PREFIX="ssf-events"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"

# Run the service
go run cmd/server/main.go
```

### Using the Service

1. **Register as SSF Receiver with Transmitters**:
   ```bash
   # Configure transmitters to send to:
   # https://your-ssf-broker.com/events
   ```

2. **Register Receivers for Event Distribution**:
   ```bash
   curl -X POST https://your-ssf-broker.com/api/v1/receivers \
     -H "Content-Type: application/json" \
     -d '{
       "id": "my-service",
       "webhook_url": "https://my-service.com/events",
       "event_types": ["session-revoked", "credential-change"],
       "auth": {"type": "bearer", "token": "..."}
     }'
   ```

3. **Events Flow Automatically**:
   - Transmitters send events to broker
   - Broker validates and processes events
   - Events distributed to registered receivers via Pub/Sub

## API Documentation

### SSF Endpoints (Standard)
- `POST /events` - Receive security events from transmitters
- `GET /.well-known/ssf_configuration` - SSF metadata discovery

### Management Endpoints
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /metrics` - Prometheus metrics

### Registration API
- `POST /api/v1/receivers` - Register a new receiver
- `GET /api/v1/receivers` - List registered receivers
- `GET /api/v1/receivers/{id}` - Get receiver details
- `PUT /api/v1/receivers/{id}` - Update receiver configuration
- `DELETE /api/v1/receivers/{id}` - Unregister receiver

## Configuration

The service is configured via environment variables and/or YAML configuration files.

### Environment Variables
- `GCP_PROJECT_ID` - Google Cloud Project ID (required)
- `PUBSUB_TOPIC_PREFIX` - Prefix for Pub/Sub topic names (default: "ssf-events")
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account JSON
- `SERVER_PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)
- `JWT_SECRET` - JWT signing secret for internal tokens

### Google Cloud Pub/Sub Configuration

The SSF Hub uses Google Cloud Pub/Sub as its event distribution backbone. Here's how to configure it:

#### 1. GCP Project Setup

```bash
# Set your project
export PROJECT_ID="your-ssf-project"
gcloud config set project $PROJECT_ID

# Enable Pub/Sub API
gcloud services enable pubsub.googleapis.com

# Enable IAM API (for service accounts)
gcloud services enable iam.googleapis.com
```

#### 2. Service Account Configuration

Create a service account with appropriate Pub/Sub permissions:

```bash
# Create service account for SSF Hub
gcloud iam service-accounts create ssf-broker \
    --display-name="SSF Hub Service Account" \
    --description="Service account for SSF Hub Pub/Sub operations"

export SERVICE_ACCOUNT="ssf-broker@${PROJECT_ID}.iam.gserviceaccount.com"

# Grant Pub/Sub Admin role (for topic/subscription management)
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/pubsub.admin"

# Alternatively, use more specific roles:
# For production, use these minimal permissions:
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/pubsub.publisher"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/pubsub.subscriber"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/pubsub.viewer"

# Create and download service account key
gcloud iam service-accounts keys create ssf-broker-key.json \
    --iam-account=$SERVICE_ACCOUNT
```

#### 3. Topic and Subscription Naming

The broker automatically creates topics based on event types:

| Event Type | Topic Name |
|------------|------------|
| Session Revoked | `ssf-events-session-revoked` |
| Credential Change | `ssf-events-credential-change` |
| Assurance Level Change | `ssf-events-assurance-level-change` |
| Device Compliance Change | `ssf-events-device-compliance-change` |
| Verification | `ssf-events-verification` |

Subscriptions are created per receiver:
- Format: `{topic-prefix}-{receiver-id}-{event-name}`
- Example: `ssf-events-my-service-session-revoked`

#### 4. Pub/Sub Performance Configuration

```yaml
pubsub:
  project_id: "your-project"
  topic_prefix: "ssf-events"

  # Performance tuning
  max_concurrent_handlers: 50        # Concurrent message processors
  max_outstanding_messages: 5000     # Buffer size for unprocessed messages
  max_outstanding_bytes: 5000000000  # 5GB buffer for message data

  # Message handling
  enable_message_ordering: true      # Maintain message order (slower but consistent)
  ack_deadline: 60                   # Seconds to acknowledge messages
  retention_duration: 168            # Hours to retain messages (7 days)

  # Delivery settings
  receive_timeout: "30s"             # Timeout for receiving messages
```

#### 5. Receiver Delivery Methods

##### Webhook Delivery (Recommended)
The broker pushes events to receiver HTTP endpoints:

```json
{
  "id": "my-service",
  "webhook_url": "https://my-service.com/events",
  "delivery": {
    "method": "webhook",
    "timeout": "30s",
    "batch_size": 1
  },
  "auth": {
    "type": "bearer",
    "token": "your-receiver-token"
  }
}
```

##### Pull Delivery
Receivers pull events from Pub/Sub subscriptions:

```json
{
  "id": "my-service",
  "delivery": {
    "method": "pull",
    "topic_name": "ssf-events-session-revoked",
    "subscription": "my-service-subscription",
    "batch_size": 10
  }
}
```

##### Push Delivery
Pub/Sub pushes directly to receiver endpoints:

```json
{
  "id": "my-service",
  "delivery": {
    "method": "push",
    "topic_name": "ssf-events-session-revoked"
  }
}
```

#### 6. Authentication Configuration

For receivers that need Google Cloud credentials:

```bash
# Create receiver-specific service account
gcloud iam service-accounts create my-service-receiver \
    --display-name="My Service SSF Receiver"

export RECEIVER_SA="my-service-receiver@${PROJECT_ID}.iam.gserviceaccount.com"

# Grant subscriber permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:${RECEIVER_SA}" \
    --role="roles/pubsub.subscriber"

# Create key for receiver
gcloud iam service-accounts keys create my-service-key.json \
    --iam-account=$RECEIVER_SA
```

#### 7. Monitoring and Quotas

Configure monitoring and check quotas:

```bash
# Check Pub/Sub quotas
gcloud compute project-info describe --format="value(quotas[].limit,quotas[].metric)" | grep pubsub

# Enable monitoring
gcloud services enable monitoring.googleapis.com

# Create alert policies for message processing
gcloud alpha monitoring policies create --policy-from-file=monitoring/pubsub-alerts.yaml
```

#### 8. Network Configuration

For private GKE clusters or VPC networks:

```bash
# Allow egress to Pub/Sub
gcloud compute firewall-rules create allow-pubsub-egress \
    --allow tcp:443 \
    --destination-ranges 199.36.153.8/30,199.36.153.4/30 \
    --direction EGRESS \
    --target-tags ssf-broker

# For private Google access
gcloud compute networks subnets update my-subnet \
    --enable-private-ip-google-access \
    --region us-central1
```

### Complete Configuration Example

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

pubsub:
  project_id: "my-ssf-project"
  topic_prefix: "ssf-events"
  credentials_file: "/var/secrets/google/service-account-key.json"

  # Performance settings
  max_concurrent_handlers: 20
  max_outstanding_messages: 2000
  max_outstanding_bytes: 2000000000  # 2GB

  # Message settings
  enable_message_ordering: true
  ack_deadline: 90                   # seconds
  retention_duration: 336            # 14 days
  receive_timeout: "60s"

auth:
  jwt_secret: "${JWT_SECRET}"
  token_expiration: 24               # hours
  require_auth: true
  allowed_issuers:
    - "https://my-idp.example.com"
    - "https://partner-idp.example.com"

logging:
  level: "info"
  format: "json"

retry:
  max_retries: 5
  initial_interval: "1s"
  max_interval: "30s"
  backoff_multiplier: 2.0
```

## Deployment

### Docker
```bash
docker build -t ssf-broker:latest .
docker run -p 8080:8080 ssf-broker:latest
```

### Kubernetes
```bash
kubectl apply -f deployments/kubernetes/
```

## Development

### Project Structure
```
ssf-broker/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── broker/          # Core broker logic
│   ├── handlers/        # HTTP handlers
│   ├── registry/        # Receiver registry
│   └── pubsub/          # Pub/Sub integration
├── pkg/
│   ├── api/             # API models and validation
│   └── models/          # Domain models
├── configs/             # Configuration files
├── deployments/         # Deployment manifests
└── docs/                # Additional documentation
```

### Testing
```bash
# Run unit tests
go test ./...

# Run integration tests (requires GCP setup)
go test -tags=integration ./...
```

## Documentation

### Getting Started
- **[Quick Start - Receiver Integration](docs/QUICK_START_RECEIVER.md)** - Get your service receiving SSF events in 5 minutes
- **[Usage Guide](docs/USAGE_GUIDE.md)** - Comprehensive usage examples and patterns
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment instructions

### Integration Guides
- **[Receiver Integration Guide](docs/RECEIVER_INTEGRATION.md)** - Complete guide for implementing receivers
- **[API Reference](docs/API_REFERENCE.md)** - REST API documentation

### Operations
- **[Quick Reference](docs/QUICK_REFERENCE.md)** - Common commands and configurations
- **[Monitoring](docs/MONITORING.md)** - Metrics, alerts, and observability

## Key Features Summary

### 🏗️ Hub-as-Receiver Architecture
- SSF Hub itself follows the receiver model for consistency
- Single unified Pub/Sub topic with structured internal messages
- Hub subscribes to its own topic for event distribution

### 🔌 Multiple Integration Patterns
1. **Hub-Managed Webhooks** (Recommended)
   - Register via REST API
   - Hub delivers events via HTTP POST
   - Built-in retry logic and monitoring

2. **Direct Pub/Sub Access** (Advanced)
   - Subscribe directly to unified topic
   - Full control over message processing
   - Custom filtering and routing

3. **Hub Federation**
   - Multi-region hub deployments
   - Cross-hub event sharing
   - Disaster recovery scenarios

### 📊 Operational Excellence
- Comprehensive metrics and monitoring
- Health checks and readiness probes
- Kubernetes-native deployment
- Production-ready configurations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.