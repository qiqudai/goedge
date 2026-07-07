const PEM_BLOCK_RE = /-----BEGIN ([A-Z0-9 ]+)-----[\s\S]*?-----END \1-----/g
const PRIVATE_KEY_TYPES = new Set(['RSA PRIVATE KEY', 'PRIVATE KEY', 'EC PRIVATE KEY'])

export function splitPemBundle (pemText) {
  const text = String(pemText || '').trim()
  if (!text) {
    return { cert: '', key: '' }
  }
  const certs = []
  let key = ''
  let match
  PEM_BLOCK_RE.lastIndex = 0
  while ((match = PEM_BLOCK_RE.exec(text)) !== null) {
    const block = match[0]
    const type = match[1]
    if (type === 'CERTIFICATE') {
      certs.push(block)
    } else if (PRIVATE_KEY_TYPES.has(type) && !key) {
      key = block
    }
  }
  return {
    cert: certs.join('\n').trim(),
    key: key.trim()
  }
}

export function looksLikePrivateKeyPem (pemText) {
  const text = String(pemText || '')
  return /-----BEGIN (?:RSA )?PRIVATE KEY-----/.test(text) ||
    /-----BEGIN EC PRIVATE KEY-----/.test(text)
}

export function looksLikeCertificatePem (pemText) {
  return /-----BEGIN CERTIFICATE-----/.test(String(pemText || ''))
}

export function normalizeUploadPemFields (cert, key) {
  let certValue = String(cert || '').trim()
  let keyValue = String(key || '').trim()
  const fromCert = splitPemBundle(certValue)
  if (fromCert.key && !keyValue) {
    keyValue = fromCert.key
    certValue = fromCert.cert || certValue
  }
  const fromKey = splitPemBundle(keyValue)
  if (fromKey.key) {
    keyValue = fromKey.key
    // Full paste in key field (private key + chain) overrides stale cert textarea.
    if (fromKey.cert) {
      certValue = fromKey.cert
    }
  }
  return { cert: certValue, key: keyValue }
}
