# SSF Hub Service Deployment Guide

This guide covers deploying the SSF Hub Service to Kubernetes with Google Cloud Pub/Sub backend.

## Overview

The SSF Hub Service acts as a centralized hub for Shared Signals Framework (SSF) events:

```
[SSF Transmitters] --HTTP--> [SSF Hub] --Pub/Sub--> [Registered Receivers]
                                 |
                                 v
                        [Registration API]
```

## Prerequisites

- Kubernetes cluster (GKE recommended)
- Google Cloud Project with Pub/Sub API enabled
- kubectl configured to access your cluster
- Docker registry access for pushing images

## Quick Start

### 1. Build and Push Docker Image

```bash
# Clone the repository
cd ssf-hub

# Build the Docker image
docker build -t your-registry/ssf-hub:latest .

# Push to your registry
docker push your-registry/ssf-hub:latest
```

### 2. Set up Google Cloud Resources

```bash
# Set your project ID
export PROJECT_ID="your-gcp-project"
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable pubsub.googleapis.com
gcloud services enable container.googleapis.com

# Create service account for the hub
gcloud iam service-accounts create ssf-hub \
    --display-name="SSF Hub Service Account"

# Grant Pub/Sub permissions
export SERVICE_ACCOUNT_EMAIL="ssf-hub@$PROJECT_ID.iam.gserviceaccount.com"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
    --role="roles/pubsub.admin"

# Create and download service account key
gcloud iam service-accounts keys create ./service-account-key.json \
    --iam-account=$SERVICE_ACCOUNT_EMAIL
```

### 3. Deploy to Kubernetes

```bash
# Create Kubernetes secrets
kubectl create secret generic ssf-hub-sa \
    --from-file=service-account-key.json

# Update deployment.yaml with your registry and project ID
sed -i 's/your-registry/my-registry.com/g' deployments/kubernetes/deployment.yaml
sed -i 's/your-project-id/my-project/g' deployments/kubernetes/deployment.yaml

# Apply the deployment
kubectl apply -f deployments/kubernetes/deployment.yaml

# Check deployment status
kubectl get pods -l app=ssf-hub
kubectl get svc ssf-hub-external
```

### 4. Configure DNS (Optional)

```bash
# Get external IP
EXTERNAL_IP=$(kubectl get svc ssf-hub-external -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Configure your DNS to point ssf-hub.your-domain.com to $EXTERNAL_IP
# Then apply ingress configuration
kubectl apply -f deployments/kubernetes/ingress.yaml
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GCP_PROJECT_ID` | Google Cloud Project ID | Required |
| `PUBSUB_TOPIC_PREFIX` | Prefix for Pub/Sub topics | `ssf-events` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `LOG_LEVEL` | Logging level | `info` |
| `JWT_SECRET` | JWT signing secret | Required |

### Pub/Sub Configuration

The hub uses a single unified Pub/Sub topic for all events:
- All security events → `ssf-hub-events`

The hub automatically creates this topic if it doesn't exist. All event types are published to this single topic with routing information in the message metadata.

## Usage

### 1. SSF Transmitter Configuration

Configure your SSF transmitters to send events to the hub:

```yaml
# transmitter-config.yaml
ssf_receiver:
  base_url: "https://ssf-hub.your-domain.com"
  auth:
    type: "bearer"
    token: "your-auth-token"

events:
  - "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
  - "https://schemas.openid.net/secevent/caep/event-type/credential-change"

delivery:
  method: "push"
  endpoint_url: "https://ssf-hub.your-domain.com/events"
```

### 2. Register Event Receivers

Register services that want to receive security events:

```bash
# Register a receiver via API
curl -X POST https://ssf-hub.your-domain.com/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-security-service",
    "name": "Security Monitoring Service",
    "webhook_url": "https://my-service.com/security-events",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
    ],
    "delivery": {
      "method": "webhook",
      "timeout": "30s"
    },
    "auth": {
      "type": "bearer",
      "token": "my-service-token"
    }
  }'
```

### 3. Discover Hub Capabilities

```bash
# Get SSF configuration
curl https://ssf-hub.your-domain.com/.well-known/ssf_configuration

# List registered receivers
curl https://ssf-hub.your-domain.com/api/v1/receivers

# Get metrics
curl https://ssf-hub.your-domain.com/metrics
```

## Event Flow

1. **Transmitter sends event** → `POST /events`
2. **Hub validates and processes** the Security Event Token (SET)
3. **Hub identifies interested receivers** based on event type and filters
4. **Hub publishes to Pub/Sub topics** for distribution
5. **Receivers get events** via webhooks or Pub/Sub subscriptions

## Monitoring

### Health Checks

- Health: `GET /health` - Basic service health
- Readiness: `GET /ready` - Service ready to handle requests
- Metrics: `GET /metrics` - Prometheus metrics

### Prometheus Metrics

```
# Total receivers
ssf_hub_receivers_total

# Receivers by status
ssf_hub_receivers_by_status{status="active"}

# Event type subscribers
ssf_hub_event_type_subscribers{event_type="session-revoked"}
```

### Logging

Structured JSON logs include:
- Event processing details
- Receiver registration/updates
- Pub/Sub operations
- Error conditions

## Scaling

### Horizontal Scaling

```bash
# Scale deployment
kubectl scale deployment ssf-hub --replicas=5

# Enable autoscaling
kubectl autoscale deployment ssf-hub \
    --min=3 --max=10 --cpu-percent=70
```

### Pub/Sub Scaling

Pub/Sub automatically scales based on load. Configure:
- `max_concurrent_handlers` for message processing
- `max_outstanding_messages` for throughput
- Topic partitioning for very high loads

## Security

### Authentication

- **Transmitters**: Use Bearer tokens or OAuth2
- **Receivers**: Configure webhook authentication
- **Management API**: Use proper RBAC

### Network Security

```yaml
# Example NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ssf-hub-netpol
spec:
  podSelector:
    matchLabels:
      app: ssf-hub
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS for GCP APIs
```

### Secrets Management

- Use Kubernetes secrets for sensitive data
- Rotate JWT secrets regularly
- Use Workload Identity on GKE

## Troubleshooting

### Common Issues

#### Pod CrashLoopBackOff
```bash
# Check logs
kubectl logs -l app=ssf-hub --tail=50

# Check events
kubectl describe pod <pod-name>
```

#### Pub/Sub Permissions
```bash
# Test service account
gcloud auth activate-service-account --key-file=service-account-key.json
gcloud pubsub topics list
```

#### High Memory Usage
```bash
# Check metrics
kubectl top pods -l app=ssf-hub

# Adjust resource limits
kubectl patch deployment ssf-hub -p '{"spec":{"template":{"spec":{"containers":[{"name":"ssf-hub","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

### Debug Mode

Enable debug logging:
```bash
kubectl set env deployment/ssf-hub LOG_LEVEL=debug
```

## Production Checklist

- [ ] **Resource Limits**: Set appropriate CPU/memory limits
- [ ] **Health Checks**: Configure liveness/readiness probes
- [ ] **Monitoring**: Set up Prometheus/Grafana dashboards
- [ ] **Alerting**: Configure alerts for failures
- [ ] **Backup**: Backup receiver configurations (if using persistent storage)
- [ ] **Security**: Enable RBAC, network policies, and secrets management
- [ ] **SSL/TLS**: Use valid certificates for external endpoints
- [ ] **Rate Limiting**: Configure appropriate rate limits
- [ ] **Log Aggregation**: Set up centralized logging
- [ ] **Documentation**: Document your specific receiver integrations

## Migration from Standalone SSF Receiver

If migrating from individual SSF receiver implementations:

1. **Deploy SSF Hub** using this guide
2. **Register your services** as receivers via the API
3. **Update transmitters** to send to the hub instead of individual services
4. **Verify event flow** using monitoring and logs
5. **Decommission old receivers** once verified

This centralized approach provides better scalability, monitoring, and management of SSF events.