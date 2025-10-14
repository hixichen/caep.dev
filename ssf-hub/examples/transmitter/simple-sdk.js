/**
 * Simple SSF Hub Transmitter SDK for Node.js
 *
 * Usage:
 *   const transmitter = new SSFTransmitter('http://localhost:8080', 'my-app');
 *   await transmitter.sendSessionRevoked('user@example.com');
 */

class SSFTransmitter {
  constructor(hubUrl, transmitterId, options = {}) {
    this.hubUrl = hubUrl.replace(/\/$/, ''); // Remove trailing slash
    this.transmitterId = transmitterId;
    this.bearerToken = options.bearerToken;
    this.devMode = options.devMode || process.env.DEV_DEBUG === 'true';
  }

  /**
   * Send a session revoked event
   */
  async sendSessionRevoked(userEmail, reason = 'administrative') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/caep/event-type/session-revoked': {
          subject: {
            format: 'email',
            email: userEmail
          },
          reason: reason
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send a credential change event
   */
  async sendCredentialChange(userEmail, changeType = 'create') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/caep/event-type/credential-change': {
          subject: {
            format: 'email',
            email: userEmail
          },
          change_type: changeType
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send an assurance level change event
   */
  async sendAssuranceLevelChange(userEmail, previousLevel, newLevel) {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/caep/event-type/assurance-level-change': {
          subject: {
            format: 'email',
            email: userEmail
          },
          previous_level: previousLevel,
          new_level: newLevel
        }
      }
    };

    return this._sendEvent(event);
  }

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
          subject: {
            format: 'email',
            email: userEmail
          },
          previous_claims: previousClaims,
          current_claims: currentClaims
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send an account credential change required event
   */
  async sendAccountCredentialChangeRequired(userEmail, reason = 'security_policy') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/risc/event-type/account-credential-change-required': {
          subject: {
            format: 'email',
            email: userEmail
          },
          reason: reason
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
          subject: {
            format: 'email',
            email: userEmail
          },
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
          subject: {
            format: 'email',
            email: userEmail
          },
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
          subject: {
            format: 'email',
            email: userEmail
          },
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
          subject: {
            format: 'email',
            email: oldEmail
          },
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
          subject: {
            format: 'email',
            email: userEmail
          },
          previous_subject: previousSubject
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send a credential compromise event
   */
  async sendCredentialCompromise(userEmail, credentialType, reasonCode = 'data_breach') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/risc/event-type/credential-compromise': {
          subject: {
            format: 'email',
            email: userEmail
          },
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
          subject: {
            format: 'email',
            email: userEmail
          }
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
          subject: {
            format: 'email',
            email: userEmail
          }
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send a recovery activated event
   */
  async sendRecoveryActivated(userEmail, recoveryMethod = 'email') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/risc/event-type/recovery-activated': {
          subject: {
            format: 'email',
            email: userEmail
          },
          recovery_method: recoveryMethod
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send a recovery information changed event
   */
  async sendRecoveryInformationChanged(userEmail, changedField = 'recovery_email') {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/risc/event-type/recovery-information-changed': {
          subject: {
            format: 'email',
            email: userEmail
          },
          changed_field: changedField
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Send a custom event
   */
  async sendCustomEvent(eventType, eventData, subject) {
    const event = {
      iss: `https://${this.transmitterId}.example.com`,
      jti: this._generateEventId(),
      iat: Math.floor(Date.now() / 1000),
      events: {
        [eventType]: {
          subject: subject,
          ...eventData
        }
      }
    };

    return this._sendEvent(event);
  }

  /**
   * Internal method to send events to the hub
   */
  async _sendEvent(event) {
    const headers = {
      'Content-Type': 'application/json'
    };

    // Add authentication
    if (this.devMode) {
      headers['X-Dev-Mode'] = 'true';
      headers['X-Transmitter-ID'] = this.transmitterId;
    } else if (this.bearerToken) {
      headers['Authorization'] = `Bearer ${this.bearerToken}`;
    } else {
      headers['X-Transmitter-ID'] = this.transmitterId;
    }

    try {
      const response = await fetch(`${this.hubUrl}/events`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(event)
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const result = await response.json();
      console.log('Event sent successfully:', {
        eventId: event.jti,
        transmitterId: this.transmitterId,
        status: result.status
      });

      return result;
    } catch (error) {
      console.error('Failed to send event:', error.message);
      throw error;
    }
  }

  /**
   * Generate a unique event ID
   */
  _generateEventId() {
    return `evt_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export for Node.js
if (typeof module !== 'undefined' && module.exports) {
  module.exports = SSFTransmitter;
}

// Example usage
if (require.main === module) {
  async function example() {
    // Development mode example
    const transmitter = new SSFTransmitter('http://localhost:8080', 'my-app', {
      devMode: true
    });

    try {
      // Send a session revoked event
      await transmitter.sendSessionRevoked('user@example.com', 'admin_action');

      // Send a credential change event
      await transmitter.sendCredentialChange('user@example.com', 'update');

      // Send an assurance level change
      await transmitter.sendAssuranceLevelChange('user@example.com', 'nist-aal-1', 'nist-aal-2');

      console.log('All events sent successfully!');
    } catch (error) {
      console.error('Example failed:', error.message);
    }
  }

  example();
}