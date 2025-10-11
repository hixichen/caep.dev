#!/usr/bin/env node

/**
 * Simple JWT Token Generator for SSF Hub Testing
 *
 * Usage:
 *   node generate-token.js [transmitter-id]
 *
 * Environment variables:
 *   JWT_SECRET - Secret key for signing (default: "dev-secret-key")
 *   JWT_EXPIRY - Token expiry in seconds (default: 3600)
 */

const crypto = require('crypto');

function base64UrlEncode(data) {
  return Buffer.from(data)
    .toString('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
}

function createJWT(payload, secret) {
  const header = {
    alg: 'HS256',
    typ: 'JWT'
  };

  const encodedHeader = base64UrlEncode(JSON.stringify(header));
  const encodedPayload = base64UrlEncode(JSON.stringify(payload));

  const signature = crypto
    .createHmac('sha256', secret)
    .update(`${encodedHeader}.${encodedPayload}`)
    .digest('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');

  return `${encodedHeader}.${encodedPayload}.${signature}`;
}

function generateToken(transmitterId) {
  const secret = process.env.JWT_SECRET || 'dev-secret-key';
  const expiry = parseInt(process.env.JWT_EXPIRY || '3600');

  const now = Math.floor(Date.now() / 1000);

  const payload = {
    iss: `https://${transmitterId}.example.com`,
    sub: transmitterId,
    aud: 'ssf-hub',
    iat: now,
    exp: now + expiry,
    transmitter_id: transmitterId
  };

  return createJWT(payload, secret);
}

// Command line usage
if (require.main === module) {
  const transmitterId = process.argv[2] || 'test-transmitter';
  const token = generateToken(transmitterId);

  console.log('Generated JWT Token for SSF Hub:');
  console.log('=====================================');
  console.log(`Transmitter ID: ${transmitterId}`);
  console.log(`Token: ${token}`);
  console.log('');
  console.log('Usage with curl:');
  console.log(`curl -X POST http://localhost:8080/events \\`);
  console.log(`  -H "Content-Type: application/json" \\`);
  console.log(`  -H "Authorization: Bearer ${token}" \\`);
  console.log(`  -d '{"your": "event", "payload": "here"}'`);
  console.log('');
  console.log('Token expires in:', process.env.JWT_EXPIRY || '3600', 'seconds');
}

module.exports = { generateToken };