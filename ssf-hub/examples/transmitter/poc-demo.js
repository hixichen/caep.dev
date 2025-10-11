#!/usr/bin/env node

/**
 * SSF Hub POC Demo Script
 *
 * This script demonstrates three ways to authenticate with SSF Hub:
 * 1. Development mode (DEV_DEBUG bypass)
 * 2. Simple X-Transmitter-ID header
 * 3. Bearer token authentication
 *
 * Usage:
 *   # Development mode
 *   DEV_DEBUG=true node poc-demo.js
 *
 *   # Bearer token mode
 *   node poc-demo.js --bearer-token
 *
 *   # Simple header mode (default)
 *   node poc-demo.js
 */

const { generateToken } = require('./generate-token.js');

class SSFHubDemo {
  constructor(hubUrl = 'http://localhost:8080') {
    this.hubUrl = hubUrl;
    this.devMode = process.env.DEV_DEBUG === 'true';
  }

  async sendTestEvent(method = 'simple') {
    const eventPayload = {
      iss: 'https://poc-demo.example.com',
      jti: `demo_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      iat: Math.floor(Date.now() / 1000),
      events: {
        'https://schemas.openid.net/secevent/caep/event-type/session-revoked': {
          subject: {
            format: 'email',
            email: 'demo-user@example.com'
          },
          reason: 'poc_demonstration'
        }
      }
    };

    let headers = {
      'Content-Type': 'application/json'
    };

    console.log(`\n🚀 Sending test event using ${method} authentication:`);
    console.log('=' .repeat(60));

    switch (method) {
      case 'dev-mode':
        headers['X-Dev-Mode'] = 'true';
        headers['X-Transmitter-ID'] = 'poc-demo-transmitter';
        console.log('✓ Using development mode bypass (X-Dev-Mode: true)');
        break;

      case 'bearer-token':
        const token = generateToken('poc-demo-transmitter');
        headers['Authorization'] = `Bearer ${token}`;
        console.log('✓ Using Bearer token authentication');
        console.log(`  Token: ${token.substring(0, 50)}...`);
        break;

      case 'simple':
      default:
        headers['X-Transmitter-ID'] = 'poc-demo-transmitter';
        console.log('✓ Using simple X-Transmitter-ID header');
        break;
    }

    try {
      console.log(`📡 Sending to: ${this.hubUrl}/events`);
      console.log(`📦 Event ID: ${eventPayload.jti}`);

      const response = await fetch(`${this.hubUrl}/events`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(eventPayload)
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const result = await response.json();
      console.log('✅ Event sent successfully!');
      console.log('📊 Response:', JSON.stringify(result, null, 2));

      return result;
    } catch (error) {
      console.error('❌ Failed to send event:', error.message);
      throw error;
    }
  }

  async registerTestReceiver() {
    const receiver = {
      id: 'poc-demo-receiver',
      name: 'POC Demo Webhook Receiver',
      webhook_url: 'https://webhook.site/unique-url-here', // Replace with actual webhook.site URL
      event_types: [
        'https://schemas.openid.net/secevent/caep/event-type/session-revoked',
        'https://schemas.openid.net/secevent/caep/event-type/credential-change'
      ],
      delivery: {
        method: 'webhook'
      },
      auth: {
        type: 'none'
      }
    };

    console.log('\n📝 Registering test receiver:');
    console.log('=' .repeat(40));

    try {
      const response = await fetch(`${this.hubUrl}/api/v1/receivers`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(receiver)
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const result = await response.json();
      console.log('✅ Receiver registered successfully!');
      console.log('📊 Response:', JSON.stringify(result, null, 2));

      return result;
    } catch (error) {
      console.error('❌ Failed to register receiver:', error.message);
      throw error;
    }
  }

  async runDemo() {
    console.log('🎯 SSF Hub POC Demo');
    console.log('==================');
    console.log(`Hub URL: ${this.hubUrl}`);
    console.log(`Dev Mode: ${this.devMode ? 'ENABLED' : 'disabled'}`);

    try {
      // Register a test receiver first
      await this.registerTestReceiver();

      // Determine which authentication method to use
      const args = process.argv.slice(2);
      let authMethod = 'simple';

      if (this.devMode) {
        authMethod = 'dev-mode';
      } else if (args.includes('--bearer-token')) {
        authMethod = 'bearer-token';
      }

      // Send a test event
      await this.sendTestEvent(authMethod);

      console.log('\n🎉 POC Demo completed successfully!');
      console.log('\n💡 Tips:');
      console.log('   - Set DEV_DEBUG=true for development mode bypass');
      console.log('   - Use --bearer-token flag to test JWT authentication');
      console.log('   - Check your webhook.site URL to see the delivered event');
      console.log('   - Visit http://localhost:8080/metrics to see hub metrics');

    } catch (error) {
      console.error('\n💥 Demo failed:', error.message);
      process.exit(1);
    }
  }
}

// Run the demo if this script is executed directly
if (require.main === module) {
  const demo = new SSFHubDemo();
  demo.runDemo();
}

module.exports = SSFHubDemo;