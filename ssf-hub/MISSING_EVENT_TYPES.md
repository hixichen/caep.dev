# Missing Event Types - Implementation Plan

Based on https://sharedsignals.guide/#eventdefinitions, here are the standardized event types not yet implemented:

## 1. Add Constants to `pkg/models/event.go`

```go
// Add to existing constants section:

// CAEP Events (Missing)
EventTypeTokenClaimsChange = "https://schemas.openid.net/secevent/caep/event-type/token-claims-change"

// RISC Events (All Missing)
EventTypeAccountCredentialChangeRequired = "https://schemas.openid.net/secevent/risc/event-type/account-credential-change-required"
EventTypeAccountPurged                   = "https://schemas.openid.net/secevent/risc/event-type/account-purged"
EventTypeAccountDisabled                 = "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
EventTypeAccountEnabled                  = "https://schemas.openid.net/secevent/risc/event-type/account-enabled"
EventTypeIdentifierChanged               = "https://schemas.openid.net/secevent/risc/event-type/identifier-changed"
EventTypeIdentifierRecycled              = "https://schemas.openid.net/secevent/risc/event-type/identifier-recycled"
EventTypeCredentialCompromise            = "https://schemas.openid.net/secevent/risc/event-type/credential-compromise"
EventTypeOptIn                          = "https://schemas.openid.net/secevent/risc/event-type/opt-in"
EventTypeOptOut                         = "https://schemas.openid.net/secevent/risc/event-type/opt-out"
EventTypeRecoveryActivated              = "https://schemas.openid.net/secevent/risc/event-type/recovery-activated"
EventTypeRecoveryInformationChanged     = "https://schemas.openid.net/secevent/risc/event-type/recovery-information-changed"
```

## 2. Add SDK Helper Methods to `examples/transmitter/simple-sdk.js`

```javascript
/**
 * Send a token claims change event
 */
async sendTokenClaimsChange(userEmail, previousClaims, currentClaims) {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/caep/event-type/token-claims-change': {
                subject: { format: 'email', email: userEmail },
                previous_claims: previousClaims,
                current_claims: currentClaims
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an account purged event
 */
async sendAccountPurged(userEmail, reason = 'policy_violation') {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/account-purged': {
                subject: { format: 'email', email: userEmail },
                reason: reason
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an account disabled event
 */
async sendAccountDisabled(userEmail, reason = 'administrative') {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/account-disabled': {
                subject: { format: 'email', email: userEmail },
                reason: reason
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an account enabled event
 */
async sendAccountEnabled(userEmail, reason = 'administrative') {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/account-enabled': {
                subject: { format: 'email', email: userEmail },
                reason: reason
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an identifier changed event
 */
async sendIdentifierChanged(oldEmail, newEmail, changeType = 'user_initiated') {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/identifier-changed': {
                subject: { format: 'email', email: oldEmail },
                new_value: newEmail,
                change_type: changeType
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an identifier recycled event
 */
async sendIdentifierRecycled(userEmail, previousSubject) {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/identifier-recycled': {
                subject: { format: 'email', email: userEmail },
                previous_subject: previousSubject
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send a credential compromise event
 */
async sendCredentialCompromise(userEmail, credentialType, reasonCode) {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/credential-compromise': {
                subject: { format: 'email', email: userEmail },
                credential_type: credentialType,
                reason_code: reasonCode
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an opt-in event
 */
async sendOptIn(userEmail) {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/opt-in': {
                subject: { format: 'email', email: userEmail }
            }
        }
    };
    return this._sendEvent(event);
}

/**
 * Send an opt-out event
 */
async sendOptOut(userEmail) {
    const event = {
        iss: `https://${this.transmitterId}.example.com`,
        jti: this._generateEventId(),
        iat: Math.floor(Date.now() / 1000),
        events: {
            'https://schemas.openid.net/secevent/risc/event-type/opt-out': {
                subject: { format: 'email', email: userEmail }
            }
        }
    };
    return this._sendEvent(event);
}
```

## 3. Update README Documentation

Add examples for all new event types to `examples/transmitter/README.md`:

```markdown
### Token Claims Change
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

### Account Purged
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

### Account Disabled/Enabled
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

### Identifier Changed
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

### Credential Compromise
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
```

## Implementation Effort: ~2-3 hours

1. **Constants**: 5 minutes
2. **SDK Methods**: 1-2 hours
3. **Documentation**: 30 minutes
4. **Testing**: 30 minutes

The existing architecture already supports all these event types without any core changes needed!