# SSF Hub Usage Guide

## Table of Contents

1. [Overview](#overview)
2. [Architecture & Components](#architecture--components)
3. [Usage Scenarios](#usage-scenarios)
4. [Deployment Patterns](#deployment-patterns)
5. [Integration Guide](#integration-guide)
6. [Configuration Examples](#configuration-examples)
7. [Best Practices](#best-practices)
8. [Monitoring & Operations](#monitoring--operations)
9. [Troubleshooting](#troubleshooting)

## Overview

The SSF Hub is a centralized hub for Shared Signals Framework (SSF) events that enables secure, real-time distribution of security events across multiple systems. It acts as both an SSF receiver (accepting events from transmitters) and an event broker (distributing events to registered receivers).

### Key Benefits

- **Centralized Event Hub**: Single point for collecting and distributing security events
- **Standards Compliance**: Full SSF specification compliance
- **Scalable Architecture**: Google Cloud Pub/Sub backend for high throughput
- **Flexible Delivery**: Multiple delivery methods (webhook, pull, push)
- **Event Filtering**: Advanced filtering and routing capabilities
- **Operational Excellence**: Built-in monitoring, logging, and health checks

## Architecture & Components

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ SSF Transmitter │    │ SSF Transmitter │    │ SSF Transmitter │
│   (Identity)    │    │   (Security)    │    │   (Compliance)  │
│   Provider      │    │     Tool        │    │     System      │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │         POST /events │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────▼───────────┐
                    │    SSF Hub Service  │
                    │                        │
                    │  ┌─────────────────┐   │
                    │  │ Event Parser &  │   │
                    │  │   Validator     │   │
                    │  └─────────────────┘   │
                    │  ┌─────────────────┐   │
                    │  │ Receiver        │   │
                    │  │ Registry        │   │
                    │  └─────────────────┘   │
                    │  ┌─────────────────┐   │
                    │  │ Event Router &  │   │
                    │  │   Filter        │   │
                    │  └─────────────────┘   │
                    └────────────┬───────────┘
                                 │
                    ┌────────────▼───────────┐
                    │  Google Cloud Pub/Sub  │
                    │                        │
                    │  ┌─────────────────┐   │
                    │  │ Topic:          │   │
                    │  │ session-revoked │   │
                    │  └─────────────────┘   │
                    │  ┌─────────────────┐   │
                    │  │ Topic:          │   │
                    │  │ credential-chg  │   │
                    │  └─────────────────┘   │
                    └────────────┬───────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
   ┌──────▼─────┐        ┌───────▼────┐        ┌───────▼────┐
   │ Application│        │ Security   │        │ Compliance │
   │ Receiver   │        │ Service    │        │ Dashboard  │
   │            │        │ Receiver   │        │ Receiver   │
   └────────────┘        └────────────┘        └────────────┘
```

### Core Components

1. **SSF Receiver Interface**: Standards-compliant endpoints for receiving events
2. **Event Parser**: Validates and parses Security Event Tokens (SETs)
3. **Receiver Registry**: Manages registration and configuration of event receivers
4. **Event Router**: Filters and routes events to interested receivers
5. **Pub/Sub Client**: Manages Google Cloud Pub/Sub topics and subscriptions
6. **Management API**: REST API for receiver registration and monitoring

## Usage Scenarios

### 1. Multi-Tenant SaaS Platform

**Scenario**: A SaaS platform needs to share security events across tenant applications and services.

**Configuration**:
```yaml
# Multiple transmitters send events to the broker
transmitters:
  - identity_provider: "https://idp.saas.com"
  - security_service: "https://security.saas.com"
  - compliance_engine: "https://compliance.saas.com"

# Tenant applications register as receivers
receivers:
  - tenant_app_1: webhook delivery to tenant endpoints
  - tenant_app_2: pull-based delivery for batch processing
  - security_dashboard: real-time webhook for monitoring
```

**Benefits**:
- Centralized event distribution
- Tenant isolation through filtering
- Scalable to thousands of tenants

### 2. Enterprise Security Hub

**Scenario**: Large enterprise with multiple identity providers and security tools needs consolidated event sharing.

**Configuration**:
```yaml
# Enterprise transmitters
transmitters:
  - active_directory: "https://ad.enterprise.com"
  - okta_instance: "https://enterprise.okta.com"
  - security_tool: "https://siem.enterprise.com"

# Enterprise receivers
receivers:
  - hr_system: session events for access management
  - security_operations: all event types for monitoring
  - compliance_audit: filtered events for reporting
```

**Benefits**:
- Unified security event view
- Compliance and audit capabilities
- Reduced integration complexity

### 3. Cross-Domain Security Alliance

**Scenario**: Multiple organizations sharing security signals for threat intelligence.

**Configuration**:
```yaml
# Partner organization transmitters
transmitters:
  - partner_a_idp: "https://idp.partner-a.com"
  - partner_b_security: "https://security.partner-b.com"
  - threat_intel_feed: "https://intel.security-alliance.com"

# Subscriber organization receivers
receivers:
  - org_1_security: threat intelligence consumption
  - org_2_compliance: regulatory event monitoring
  - shared_soc: collaborative security operations
```

**Benefits**:
- Cross-organizational security intelligence
- Standardized event formats
- Secure, controlled sharing

### 4. Cloud-Native Microservices

**Scenario**: Cloud-native application with microservices needing real-time security event coordination.

**Configuration**:
```yaml
# Microservice transmitters
transmitters:
  - auth_service: "https://auth.app.com"
  - user_service: "https://users.app.com"
  - session_service: "https://sessions.app.com"

# Microservice receivers
receivers:
  - fraud_detection: real-time event analysis
  - user_analytics: behavioral analysis
  - audit_service: compliance logging
```

**Benefits**:
- Real-time event coordination
- Microservices decoupling
- Scalable event processing

## Deployment Patterns

### 1. Centralized Hub (Recommended)

**Architecture**: Single broker instance serving multiple transmitters and receivers.

```yaml
deployment:
  type: centralized
  instances: 1
  scaling: horizontal (multiple pods)
  benefits:
    - Simple management
    - Centralized monitoring
    - Cost effective
  considerations:
    - Single point of failure (mitigated by HA)
    - Cross-region latency
```

**Use Cases**:
- Small to medium enterprises
- Single cloud region deployments
- Centralized security operations

### 2. Federated Brokers

**Architecture**: Multiple broker instances with event federation.

```yaml
deployment:
  type: federated
  regions:
    - us-east: primary broker
    - us-west: secondary broker
    - eu-central: regional broker
  benefits:
    - Geographic distribution
    - Reduced latency
    - Regional compliance
  considerations:
    - Complex configuration
    - Event synchronization
    - Higher operational overhead
```

**Use Cases**:
- Global enterprises
- Regulatory compliance requirements
- High availability needs

### 3. Hybrid Cloud

**Architecture**: Broker deployed across multiple cloud providers or on-premises.

```yaml
deployment:
  type: hybrid
  components:
    - gcp_pubsub: Google Cloud Pub/Sub backend
    - k8s_broker: Kubernetes deployment (any cloud)
    - on_prem_receivers: On-premises event consumers
  benefits:
    - Cloud flexibility
    - Gradual migration
    - Compliance options
  considerations:
    - Network connectivity
    - Security boundaries
    - Operational complexity
```

**Use Cases**:
- Cloud migration scenarios
- Regulatory data residency
- Multi-cloud strategies

## Integration Guide

### For SSF Transmitters

#### 1. Configure SSF Transmitter

```go
// Example using ssfreceiver library
import "github.com/sgnl-ai/caep.dev/ssfreceiver"

// Configure transmitter to send to broker
transmitterConfig := &TransmitterConfig{
    BrokerURL: "https://ssf-hub.company.com",
    Events: []string{
        "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
        "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    },
    Auth: AuthConfig{
        Type: "bearer",
        Token: "transmitter-token",
    },
}
```

#### 2. Send Events to Broker

```bash
# Send security event to broker
curl -X POST https://ssf-hub.company.com/events \
  -H "Content-Type: application/secevent+jwt" \
  -H "Authorization: Bearer transmitter-token" \
  -d "$SECURITY_EVENT_TOKEN"
```

#### 3. Discover Broker Configuration

```bash
# Get broker SSF configuration
curl https://ssf-hub.company.com/.well-known/ssf_configuration
```

### For Event Receivers

#### 1. Register with Broker

```bash
# Register as event receiver
curl -X POST https://ssf-hub.company.com/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-application",
    "name": "My Application",
    "webhook_url": "https://my-app.com/events",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
    ],
    "delivery": {
      "method": "webhook",
      "timeout": "30s",
      "batch_size": 1
    },
    "auth": {
      "type": "bearer",
      "token": "my-app-token"
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

#### 2. Handle Webhook Events

```go
// Handle incoming events via webhook
func handleSecurityEvent(w http.ResponseWriter, r *http.Request) {
    // Read event from request body
    eventData, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read event", http.StatusBadRequest)
        return
    }

    // Parse the security event
    var event models.SecurityEvent
    if err := json.Unmarshal(eventData, &event); err != nil {
        http.Error(w, "Invalid event format", http.StatusBadRequest)
        return
    }

    // Process the event
    switch event.Type {
    case models.EventTypeSessionRevoked:
        handleSessionRevoked(&event)
    case models.EventTypeCredentialChange:
        handleCredentialChange(&event)
    }

    // Acknowledge receipt
    w.WriteHeader(http.StatusOK)
}
```

#### 3. Pull Events (Alternative)

```bash
# Configure for pull delivery
curl -X POST https://ssf-hub.company.com/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "batch-processor",
    "delivery": {
      "method": "pull",
      "topic_name": "ssf-events-session-revoked",
      "batch_size": 10
    }
  }'
```

### For Administrators

#### 1. Monitor Broker Health

```bash
# Check broker health
curl https://ssf-hub.company.com/health

# Check readiness
curl https://ssf-hub.company.com/ready

# Get metrics
curl https://ssf-hub.company.com/metrics
```

#### 2. Manage Receivers

```bash
# List all receivers
curl https://ssf-hub.company.com/api/v1/receivers

# Get receiver details
curl https://ssf-hub.company.com/api/v1/receivers/my-application

# Update receiver configuration
curl -X PUT https://ssf-hub.company.com/api/v1/receivers/my-application \
  -H "Content-Type: application/json" \
  -d '{ "name": "Updated Application Name" }'

# Remove receiver
curl -X DELETE https://ssf-hub.company.com/api/v1/receivers/my-application
```

## Configuration Examples

### Complete Deployment Configuration

```yaml
# complete-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ssf-hub-config
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8080
      read_timeout: "30s"
      write_timeout: "30s"
      idle_timeout: "120s"

    pubsub:
      project_id: "my-company-security"
      topic_prefix: "ssf-events"
      credentials_file: "/var/secrets/google/service-account-key.json"
      max_concurrent_handlers: 50
      max_outstanding_messages: 5000
      enable_message_ordering: true
      ack_deadline: 90
      retention_duration: 336  # 14 days

    auth:
      jwt_secret: "${JWT_SECRET}"
      token_expiration: 24  # hours
      require_auth: true
      allowed_issuers:
        - "https://identity.company.com"
        - "https://security.company.com"

    logging:
      level: "info"
      format: "json"

    retry:
      max_retries: 5
      initial_interval: "1s"
      max_interval: "30s"
      backoff_multiplier: 2.0
```

### Receiver Configuration Examples

#### High-Volume Application

```json
{
  "id": "fraud-detection",
  "name": "Fraud Detection System",
  "webhook_url": "https://fraud.company.com/api/security-events",
  "event_types": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change"
  ],
  "delivery": {
    "method": "webhook",
    "timeout": "10s",
    "batch_size": 1
  },
  "auth": {
    "type": "oauth2",
    "client_id": "fraud-detection-client",
    "client_secret": "${FRAUD_CLIENT_SECRET}",
    "token_url": "https://auth.company.com/oauth/token",
    "scopes": ["security-events.read"]
  },
  "filters": [
    {
      "field": "subject.format",
      "operator": "in",
      "value": ["email", "phone_number"]
    },
    {
      "field": "data.risk_score",
      "operator": "exists",
      "value": null
    }
  ],
  "retry": {
    "max_retries": 5,
    "initial_interval": "1s",
    "max_interval": "30s",
    "multiplier": 2.0,
    "enable_jitter": true
  },
  "tags": {
    "environment": "production",
    "team": "security",
    "criticality": "high"
  }
}
```

#### Batch Processing System

```json
{
  "id": "audit-processor",
  "name": "Audit Log Processor",
  "event_types": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
  ],
  "delivery": {
    "method": "pull",
    "topic_name": "ssf-events-audit",
    "subscription": "audit-processor-subscription",
    "batch_size": 100
  },
  "filters": [
    {
      "field": "metadata.transmitter_id",
      "operator": "in",
      "value": ["identity-provider", "security-service"]
    }
  ],
  "tags": {
    "environment": "production",
    "team": "compliance",
    "processing": "batch"
  }
}
```

#### Development/Testing Receiver

```json
{
  "id": "dev-test-receiver",
  "name": "Development Test Receiver",
  "webhook_url": "https://webhook.site/unique-id",
  "event_types": ["*"],
  "delivery": {
    "method": "webhook",
    "timeout": "60s",
    "batch_size": 1
  },
  "auth": {
    "type": "none"
  },
  "tags": {
    "environment": "development",
    "purpose": "testing"
  }
}
```

## Best Practices

### Security

1. **Authentication & Authorization**
   ```yaml
   # Use strong authentication for transmitters
   transmitter_auth:
     - type: "oauth2"
     - validate_issuer: true
     - require_https: true

   # Secure receiver webhooks
   receiver_auth:
     - type: "hmac"
     - algorithm: "sha256"
     - rotate_secrets: "monthly"
   ```

2. **Network Security**
   ```yaml
   network:
     - use_tls: true
     - minimum_tls_version: "1.2"
     - firewall_rules: "restrictive"
     - vpc_endpoints: "enabled"
   ```

3. **Data Protection**
   ```yaml
   data_protection:
     - encrypt_at_rest: true
     - encrypt_in_transit: true
     - pii_handling: "careful"
     - retention_policies: "defined"
   ```

### Performance

1. **Scaling Configuration**
   ```yaml
   scaling:
     horizontal_pod_autoscaler:
       min_replicas: 3
       max_replicas: 20
       cpu_threshold: 70
       memory_threshold: 80

   pubsub:
     max_concurrent_handlers: 50
     max_outstanding_messages: 5000
   ```

2. **Resource Limits**
   ```yaml
   resources:
     requests:
       cpu: "500m"
       memory: "1Gi"
     limits:
       cpu: "2000m"
       memory: "4Gi"
   ```

3. **Caching Strategy**
   ```yaml
   caching:
     receiver_registry:
       type: "memory"
       ttl: "5m"
     metadata_cache:
       type: "redis"
       ttl: "1h"
   ```

### Reliability

1. **Error Handling**
   ```yaml
   error_handling:
     retry_policy:
       max_retries: 5
       backoff: "exponential"
       jitter: true
     circuit_breaker:
       failure_threshold: 10
       timeout: "30s"
   ```

2. **Health Checks**
   ```yaml
   health_checks:
     liveness:
       path: "/health"
       interval: "10s"
     readiness:
       path: "/ready"
       interval: "5s"
   ```

3. **Backup & Recovery**
   ```yaml
   backup:
     receiver_registry:
       frequency: "daily"
       retention: "30d"
     configuration:
       version_control: true
       automated_backup: true
   ```

### Monitoring

1. **Key Metrics**
   ```yaml
   metrics:
     event_throughput:
       - events_received_total
       - events_published_total
       - events_delivered_total
       - events_failed_total

     performance:
       - event_processing_duration
       - webhook_delivery_duration
       - pubsub_publish_duration

     reliability:
       - receiver_health_status
       - subscription_lag
       - error_rate
   ```

2. **Alerting Rules**
   ```yaml
   alerts:
     high_error_rate:
       condition: "error_rate > 5%"
       duration: "5m"
       severity: "warning"

     service_down:
       condition: "up == 0"
       duration: "1m"
       severity: "critical"

     high_latency:
       condition: "processing_duration > 1s"
       duration: "2m"
       severity: "warning"
   ```

### Operational Excellence

1. **Logging Strategy**
   ```yaml
   logging:
     structured: true
     format: "json"
     fields:
       - timestamp
       - level
       - message
       - event_id
       - receiver_id
       - transmitter_id

     sensitive_data:
       - redact_pii: true
       - hash_identifiers: true
   ```

2. **Configuration Management**
   ```yaml
   config_management:
     version_control: "git"
     environment_separation: true
     secret_management: "vault"
     config_validation: true
   ```

3. **Deployment Strategy**
   ```yaml
   deployment:
     strategy: "rolling_update"
     max_unavailable: "25%"
     max_surge: "25%"
     rollback_enabled: true
     canary_deployment: true
   ```

## Monitoring & Operations

### Prometheus Metrics

The broker exposes comprehensive metrics via `/metrics` endpoint:

```prometheus
# Event processing metrics
ssf_broker_events_received_total{transmitter_id="...", event_type="..."}
ssf_broker_events_published_total{event_type="..."}
ssf_broker_events_delivered_total{receiver_id="...", event_type="..."}
ssf_broker_events_failed_total{receiver_id="...", reason="..."}

# Performance metrics
ssf_broker_event_processing_duration_seconds{quantile="0.5"}
ssf_broker_webhook_delivery_duration_seconds{quantile="0.95"}
ssf_broker_pubsub_publish_duration_seconds{quantile="0.99"}

# System metrics
ssf_broker_active_receivers_total
ssf_broker_subscriptions_total
ssf_broker_memory_usage_bytes
ssf_broker_cpu_usage_percent
```

### Grafana Dashboard Example

```json
{
  "dashboard": {
    "title": "SSF Hub Overview",
    "panels": [
      {
        "title": "Event Throughput",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(ssf_broker_events_received_total[5m])",
            "legendFormat": "Events Received/sec"
          },
          {
            "expr": "rate(ssf_broker_events_delivered_total[5m])",
            "legendFormat": "Events Delivered/sec"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(ssf_broker_events_failed_total[5m]) / rate(ssf_broker_events_received_total[5m])",
            "legendFormat": "Error Rate %"
          }
        ]
      },
      {
        "title": "Receiver Health",
        "type": "table",
        "targets": [
          {
            "expr": "ssf_broker_receiver_health_status",
            "format": "table"
          }
        ]
      }
    ]
  }
}
```

### Operational Runbooks

#### Event Processing Failure

```bash
# 1. Check broker health
curl https://ssf-hub.company.com/health

# 2. Check recent logs
kubectl logs -l app=ssf-hub --tail=100 | grep ERROR

# 3. Check Pub/Sub subscription lag
gcloud pubsub subscriptions describe ssf-events-receiver-id-session-revoked

# 4. Verify receiver configuration
curl https://ssf-hub.company.com/api/v1/receivers/receiver-id

# 5. Test receiver endpoint
curl -X POST https://receiver.company.com/events \
  -H "Content-Type: application/json" \
  -d '{"test": "event"}'
```

#### High Latency Investigation

```bash
# 1. Check processing metrics
curl https://ssf-hub.company.com/metrics | grep duration

# 2. Check Pub/Sub metrics
gcloud monitoring metrics list --filter="metric.type:pubsub"

# 3. Check resource utilization
kubectl top pods -l app=ssf-hub

# 4. Review concurrent processing settings
kubectl get configmap ssf-hub-config -o yaml
```

## Troubleshooting

### Common Issues

#### 1. Events Not Being Delivered

**Symptoms**:
- Events received but not delivered to receivers
- High error rates in metrics

**Investigation**:
```bash
# Check receiver status
curl https://ssf-hub.company.com/api/v1/receivers/receiver-id

# Verify Pub/Sub subscriptions
gcloud pubsub subscriptions list --filter="name:ssf-events"

# Check receiver webhook endpoint
curl -I https://receiver.company.com/events
```

**Solutions**:
- Verify receiver webhook URL is accessible
- Check authentication configuration
- Validate event type subscriptions
- Review filtering rules

#### 2. High Memory Usage

**Symptoms**:
- Pod memory usage increasing over time
- Out of memory errors

**Investigation**:
```bash
# Check memory metrics
kubectl top pods -l app=ssf-hub

# Review concurrent processing settings
kubectl describe configmap ssf-hub-config
```

**Solutions**:
- Reduce `max_outstanding_messages`
- Implement proper message acknowledgment
- Increase pod memory limits
- Review receiver processing speed

#### 3. Pub/Sub Connection Issues

**Symptoms**:
- Connection errors in logs
- Events not being published

**Investigation**:
```bash
# Check service account permissions
gcloud projects get-iam-policy PROJECT_ID

# Verify Pub/Sub API is enabled
gcloud services list --enabled --filter="name:pubsub.googleapis.com"

# Test Pub/Sub connectivity
gcloud pubsub topics list
```

**Solutions**:
- Verify service account has proper permissions
- Check network connectivity
- Validate credentials configuration
- Review firewall rules

#### 4. Authentication Failures

**Symptoms**:
- 401/403 errors from transmitters
- Webhook delivery failures

**Investigation**:
```bash
# Check authentication configuration
kubectl get secret ssf-hub-secrets -o yaml

# Review logs for auth errors
kubectl logs -l app=ssf-hub | grep -i auth
```

**Solutions**:
- Verify JWT secrets are correctly configured
- Check OAuth2 client credentials
- Validate token expiration settings
- Review issuer allowlist

### Performance Tuning

#### Event Processing Optimization

```yaml
# Optimize for high throughput
pubsub:
  max_concurrent_handlers: 100
  max_outstanding_messages: 10000
  max_outstanding_bytes: 10000000000  # 10GB

# Optimize for low latency
pubsub:
  max_concurrent_handlers: 20
  max_outstanding_messages: 1000
  enable_message_ordering: false
```

#### Webhook Delivery Optimization

```yaml
# Parallel delivery configuration
delivery:
  max_concurrent_webhooks: 50
  timeout: "10s"
  retry_strategy: "exponential_backoff"

# Batch delivery for high volume
delivery:
  batch_size: 10
  batch_timeout: "5s"
  compression: "gzip"
```

#### Resource Optimization

```yaml
# Kubernetes resource tuning
resources:
  requests:
    cpu: "1000m"      # Increase for high load
    memory: "2Gi"     # Increase for large messages
  limits:
    cpu: "4000m"
    memory: "8Gi"

# JVM tuning (if applicable)
jvm_opts:
  - "-Xmx4g"
  - "-XX:+UseG1GC"
  - "-XX:MaxGCPauseMillis=200"
```

### Debug Mode

Enable debug mode for detailed troubleshooting:

```yaml
# Debug configuration
logging:
  level: "debug"
  debug_mode: true
  verbose_errors: true

# Additional debug endpoints
debug:
  enabled: true
  endpoints:
    - "/debug/receivers"
    - "/debug/subscriptions"
    - "/debug/events"
```

Access debug information:

```bash
# Get receiver debug info
curl https://ssf-hub.company.com/debug/receivers

# Get subscription details
curl https://ssf-hub.company.com/debug/subscriptions

# Get recent event processing info
curl https://ssf-hub.company.com/debug/events
```

This completes the comprehensive usage guide for the SSF Hub service. The guide provides practical guidance for all stakeholders - from developers integrating with the broker to operators managing it in production.