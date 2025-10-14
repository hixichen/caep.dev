# SSF Hub Transmitter Examples

This directory contains examples for transmitters to send security events to the SSF Hub.

**🎯 Complete Standards Coverage**: This SDK now supports all standardized CAEP, RISC, and SSF event types from the [Shared Signals Guide](https://sharedsignals.guide/#eventdefinitions).

## SDK Methods Available

### CAEP Events
- `sendSessionRevoked(userEmail, reason)`
- `sendCredentialChange(userEmail, changeType)`
- `sendAssuranceLevelChange(userEmail, previousLevel, newLevel)`
- `sendTokenClaimsChange(userEmail, previousClaims, currentClaims)` ✨ NEW

### RISC Events
- `sendAccountCredentialChangeRequired(userEmail, reason)` ✨ NEW
- `sendAccountPurged(userEmail, reason)` ✨ NEW
- `sendAccountDisabled(userEmail, reason)` ✨ NEW
- `sendAccountEnabled(userEmail, reason)` ✨ NEW
- `sendIdentifierChanged(oldEmail, newEmail, changeType)` ✨ NEW
- `sendIdentifierRecycled(userEmail, previousSubject)` ✨ NEW
- `sendCredentialCompromise(userEmail, credentialType, reasonCode)` ✨ NEW
- `sendOptIn(userEmail)` ✨ NEW
- `sendOptOut(userEmail)` ✨ NEW
- `sendRecoveryActivated(userEmail, recoveryMethod)` ✨ NEW
- `sendRecoveryInformationChanged(userEmail, changedField)` ✨ NEW

### Generic
- `sendCustomEvent(eventType, eventData, subject)` - For any custom event type

## Quick Start with curl

### 1. Basic Event Submission

```bash
# Send a session revoked event
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -H "X-Transmitter-ID: my-app" \
  -d '{
    "iss": "https://my-app.example.com",
    "jti": "event-123",
    "iat": 1640995200,
    "events": {
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
        "subject": {
          "format": "email",
          "email": "user@example.com"
        },
        "reason": "administrative"
      }
    }
  }'
```

### 2. With Bearer Token Authentication

```bash
# Using Authorization header
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token-here" \
  -d '{
    "iss": "https://my-app.example.com",
    "jti": "event-456",
    "iat": 1640995200,
    "events": {
      "https://schemas.openid.net/secevent/caep/event-type/credential-change": {
        "subject": {
          "format": "email",
          "email": "user@example.com"
        },
        "change_type": "create"
      }
    }
  }'
```

### 3. Development Mode (DEV_DEBUG bypass)

```bash
# In development mode, you can bypass auth completely
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -H "X-Dev-Mode: true" \
  -H "X-Transmitter-ID: dev-transmitter" \
  -d '{
    "iss": "https://dev.example.com",
    "jti": "dev-event-789",
    "iat": 1640995200,
    "events": {
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
        "subject": {
          "format": "email",
          "email": "dev-user@example.com"
        }
      }
    }
  }'
```

## Event Types

### CAEP Events (Continuous Access Evaluation Profile)

#### Session Revoked
```json
{
  "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "reason": "administrative"
  }
}
```

#### Credential Change
```json
{
  "https://schemas.openid.net/secevent/caep/event-type/credential-change": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "change_type": "create"
  }
}
```

#### Assurance Level Change
```json
{
  "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "previous_level": "nist-aal-1",
    "new_level": "nist-aal-2"
  }
}
```

#### Device Compliance Change
```json
{
  "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "current_status": "compliant",
    "previous_status": "non_compliant"
  }
}
```

#### Token Claims Change
```json
{
  "https://schemas.openid.net/secevent/caep/event-type/token-claims-change": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "previous_claims": {
      "role": "user"
    },
    "current_claims": {
      "role": "admin"
    }
  }
}
```

### RISC Events (Risk Incident Sharing and Coordination)

#### Account Credential Change Required
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/account-credential-change-required": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "reason": "security_policy"
  }
}
```

#### Account Purged
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/account-purged": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "reason": "policy_violation"
  }
}
```

#### Account Disabled
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/account-disabled": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "reason": "administrative"
  }
}
```

#### Account Enabled
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/account-enabled": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "reason": "administrative"
  }
}
```

#### Identifier Changed
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/identifier-changed": {
    "subject": {
      "format": "email",
      "email": "old-email@example.com"
    },
    "new_value": "new-email@example.com",
    "change_type": "user_initiated"
  }
}
```

#### Identifier Recycled
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/identifier-recycled": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "previous_subject": {
      "format": "iss_sub",
      "iss": "https://old-provider.example.com",
      "sub": "old-user-123"
    }
  }
}
```

#### Credential Compromise
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/credential-compromise": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "credential_type": "password",
    "reason_code": "data_breach"
  }
}
```

#### Opt In
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/opt-in": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    }
  }
}
```

#### Opt Out
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/opt-out": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    }
  }
}
```

#### Recovery Activated
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/recovery-activated": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "recovery_method": "email"
  }
}
```

#### Recovery Information Changed
```json
{
  "https://schemas.openid.net/secevent/risc/event-type/recovery-information-changed": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    },
    "changed_field": "recovery_email"
  }
}
```

### SSF Events (Shared Signals Framework)

#### Verification
```json
{
  "https://schemas.openid.net/secevent/ssf/event-type/verification": {
    "subject": {
      "format": "email",
      "email": "user@example.com"
    }
  }
}
```

## Testing with Sample Receivers

### 1. Register a Test Receiver
```bash
curl -X POST http://localhost:8080/api/v1/receivers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-receiver",
    "name": "Test Webhook Receiver",
    "webhook_url": "https://webhook.site/your-unique-url",
    "event_types": [
      "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
      "https://schemas.openid.net/secevent/caep/event-type/credential-change"
    ],
    "delivery": {
      "method": "webhook"
    },
    "auth": {
      "type": "none"
    }
  }'
```

### 2. Send Event and Watch Webhook
After registering the receiver above, send an event and watch your webhook.site URL to see it delivered.

## POC Demo Scripts

### Quick Demo
Run the complete POC demo that registers a receiver and sends events:

```bash
# Development mode (bypasses auth)
DEV_DEBUG=true node poc-demo.js

# Bearer token mode
node poc-demo.js --bearer-token

# Simple header mode
node poc-demo.js

# Demonstrate ALL 17 standardized event types
DEV_DEBUG=true node poc-demo.js --all-events

# Full demo with bearer token and all events
node poc-demo.js --bearer-token --all-events
```

The `--all-events` flag will demonstrate all CAEP, RISC, and SSF event types supported by the hub, giving you a comprehensive view of the complete standards compliance.

### Generate Bearer Tokens for Testing

Use the token generator for JWT authentication:

```bash
# Generate token for specific transmitter
node generate-token.js my-transmitter-id

# Generate token with custom settings
JWT_SECRET=my-secret JWT_EXPIRY=7200 node generate-token.js my-app
```

Example output:
```
Generated JWT Token for SSF Hub:
=====================================
Transmitter ID: my-app
Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL215LWFwcC5leGFtcGxlLmNvbSIsInN1YiI6Im15LWFwcCIsImF1ZCI6InNzZi1odWIiLCJpYXQiOjE3MDQwNjcyMDAsImV4cCI6MTcwNDA3MDgwMCwidHJhbnNtaXR0ZXJfaWQiOiJteS1hcHAifQ.signature

Usage with curl:
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"your": "event", "payload": "here"}'
```

## Environment Variables for Development

```bash
# Enable development mode (bypasses authentication)
export DEV_DEBUG=true

# Set default transmitter for development
export DEV_DEFAULT_TRANSMITTER=dev-app

# JWT settings for token generation
export JWT_SECRET=your-secret-key
export JWT_EXPIRY=3600
```

## Development Mode Authentication Bypass

When `DEV_DEBUG=true` is set or `X-Dev-Mode: true` header is sent, SSF Hub will:

1. Skip normal authentication validation
2. Use `X-Transmitter-ID` header if provided
3. Fall back to `DEV_DEFAULT_TRANSMITTER` environment variable
4. Default to `dev-transmitter` if nothing is specified

This makes it easy to test and develop without setting up proper JWT infrastructure.

**⚠️ Warning**: Never use development mode in production!