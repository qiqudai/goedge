export function parseBool(value, fallback = false) {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase();
    return v === '1' || v === 'true' || v === 'on';
  }
  if (typeof value === 'number') return value !== 0;
  return fallback;
}

export function splitLines(value) {
  if (!value) return [];
  return String(value)
    .split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean);
}

export function formatDate(value) {
  if (!value) return '-';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '-';
  return parsed.toLocaleString();
}

export function formatDateInTimezone(value, timeZone = 'Asia/Shanghai') {
  if (!value) return '-';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '-';
  try {
    return new Intl.DateTimeFormat('zh-CN', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    }).format(parsed).replace(/\//g, '-');
  } catch (e) {
    return parsed.toLocaleString();
  }
}

export function getCertDays(cert, certList) {
    if (!cert || !cert.id) return 0;
    const fullCert = certList.find(c => c.id === cert.id);
    if (!fullCert || !fullCert.expire_at) return 0;
    
    const now = new Date().getTime();
    const expire = new Date(fullCert.expire_at.replace(/-/g, '/')).getTime();
    if (isNaN(expire)) return 0;
    
    return Math.max(0, Math.floor((expire - now) / (1000 * 60 * 60 * 24)));
}
