export async function sha256Hex(value) {
  if (!window.crypto || !window.crypto.subtle) {
    throw new Error('secure context required');
  }
  const data = new TextEncoder().encode(value);
  const hash = await window.crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(hash))
    .map(byte => byte.toString(16).padStart(2, '0'))
    .join('');
}
