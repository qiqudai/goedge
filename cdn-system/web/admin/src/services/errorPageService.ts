import type { ErrorPageDefinition, ErrorPageI18nSettings, ErrorPageMap, GlobalConfigPayload } from '@/types/errorPages'
import { DEFAULT_ERROR_PAGE_I18N, ERROR_PAGE_CODES } from '@/constants/errorPageLocales'

const PLACEHOLDER_RE = /\{\{([a-zA-Z0-9_]+)\}\}/g

export function extractTemplateKeys(template: string): string[] {
  const keys = new Set<string>()
  let match
  const re = new RegExp(PLACEHOLDER_RE)
  while ((match = re.exec(template || '')) !== null) {
    if (match[1]) {
      keys.add(match[1])
    }
  }
  return Array.from(keys).sort()
}

export function renderErrorPagePreview(
  template: string,
  strings: Record<string, string>
): string {
  if (!template) return ''
  return template.replace(PLACEHOLDER_RE, (_, key) => strings?.[key] ?? `{{${key}}}`)
}

export function resolvePreviewStrings(
  def: ErrorPageDefinition,
  lang: string,
  defaultLang: string
): Record<string, string> {
  const candidates = [lang, lang.split('-')[0], defaultLang]
  for (const candidate of candidates) {
    const strings = candidate ? def.strings?.[candidate] : undefined
    if (strings) {
      return strings
    }
  }
  const first = Object.values(def.strings || {})[0] as Record<string, string> | undefined
  return first || {}
}

export function ensureErrorPageStructure(
  pages: ErrorPageMap | undefined,
  i18n: ErrorPageI18nSettings | undefined
): { pages: ErrorPageMap; i18n: ErrorPageI18nSettings } {
  const settings: ErrorPageI18nSettings = {
    default_lang: i18n?.default_lang || DEFAULT_ERROR_PAGE_I18N.default_lang,
    lang_mode: i18n?.lang_mode === 'fixed' ? 'fixed' : 'browser',
    enabled_langs: Array.isArray(i18n?.enabled_langs) && i18n.enabled_langs.length
      ? [...new Set(i18n.enabled_langs)]
      : [...DEFAULT_ERROR_PAGE_I18N.enabled_langs]
  }
  if (!settings.enabled_langs.includes(settings.default_lang)) {
    settings.enabled_langs.unshift(settings.default_lang)
  }

  const nextPages: ErrorPageMap = {}
  ERROR_PAGE_CODES.forEach(({ key }) => {
    const existing = pages?.[key]
    const page: ErrorPageDefinition = {
      template: existing?.template || '<h1>{{title}}</h1><p>{{subtitle}}</p>',
      strings: { ...(existing?.strings || {}) }
    }
    nextPages[key] = page
    settings.enabled_langs.forEach(lang => {
      if (!page.strings[lang]) {
        page.strings[lang] = {}
      }
    })
  })
  return { pages: nextPages, i18n: settings }
}

export function buildGlobalConfigPayload(
  fullConfig: Record<string, unknown>,
  pages: ErrorPageMap,
  i18n: ErrorPageI18nSettings
): GlobalConfigPayload {
  return {
    ...(fullConfig as unknown as GlobalConfigPayload),
    error_page_i18n: i18n,
    error_pages: pages
  }
}

export function normalizeLocaleTag(lang: string): string {
  const value = String(lang || '').trim()
  if (!value) return ''
  const parts = value.replace(/_/g, '-').split('-')
  if (!parts.length) return ''
  const head = parts[0]
  if (!head) return ''
  parts[0] = head.toLowerCase()
  for (let i = 1; i < parts.length; i += 1) {
    const part = parts[i] || ''
    parts[i] = part.length === 2 ? part.toUpperCase() : part.toLowerCase()
  }
  return parts.join('-')
}
