const crypto = require('crypto');
const fs = require('fs');

const privateKey = fs.readFileSync('private.pem', 'utf8');

const header = {
  alg: 'RS256',
  typ: 'JWT'
};

const claims = {
  tier: 'enterprise',
  tenant_id: '00000000-0000-0000-0000-000000000001',
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + (365 * 24 * 60 * 60), // 1 year
  features: ['subprocesses', 'sso', 'branding', 'secrets', 'audit_logs', 'analytics']
};

function base64url(obj) {
  return Buffer.from(JSON.stringify(obj))
    .toString('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
}

const encodedHeader = base64url(header);
const encodedPayload = base64url(claims);

const signatureInput = `${encodedHeader}.${encodedPayload}`;
const signer = crypto.createSign('RSA-SHA256');
signer.update(signatureInput);
const signature = signer.sign(privateKey, 'base64')
  .replace(/=/g, '')
  .replace(/\+/g, '-')
  .replace(/\//g, '_');

const token = `${encodedHeader}.${encodedPayload}.${signature}`;

console.log('\n--- PHAXA ENTERPRISE LICENSE TOKEN ---');
console.log(token);
console.log('--------------------------------------');
console.log('Issued To: 00000000-0000-0000-0000-000000000001');
console.log('Tier:      enterprise');
console.log('Features:  ' + claims.features.join(', '));
console.log('--------------------------------------\n');
