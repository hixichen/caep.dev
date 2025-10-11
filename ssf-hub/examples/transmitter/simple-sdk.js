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