const stripScheme = (value) => String(value || '').trim().replace(/^[a-z][a-z\d+.-]*:\/\//i, '')

export function isHttpsEnabled(value) {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value > 0

  const normalized = String(value ?? '').trim().toLowerCase()
  return normalized === '1' || normalized === 'true' || normalized === 'yes' || normalized === 'on'
}

export function getPrimarySiteDomain(row) {
  const raw = row?.domain_display || row?.domainDisplay || row?.domain || (Array.isArray(row?.domains) ? row.domains[0] : '')
  const first = String(raw || '')
    .split(',')
    .map(item => item.trim())
    .find(Boolean) || ''
  return stripScheme(first).replace(/^\*\./, '').replace(/\/.*$/, '')
}

export function buildSiteBrowseUrl(row) {
  const domain = getPrimarySiteDomain(row)
  if (!domain) return ''
  const state = String(row?.https_state || row?.httpsState || '').trim().toLowerCase()
  const httpsActive = state ? state === 'active' : isHttpsEnabled(row?.https)
  return `${httpsActive ? 'https' : 'http'}://${domain}`
}

export function openSiteInBrowser(row) {
  const url = buildSiteBrowseUrl(row)
  if (!url || typeof window === 'undefined') return false
  window.open(url, '_blank', 'noopener,noreferrer')
  return true
}
