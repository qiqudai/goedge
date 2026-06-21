import type { GuardPageDefinition, GuardPageMap } from '@/types/guardPages'
import type { ErrorPageI18nSettings } from '@/types/errorPages'
import { DEFAULT_ERROR_PAGE_I18N } from '@/constants/errorPageLocales'
import guardDefaults from '@cdn-common/i18n/guard_default_strings.json'
import clickTemplate from '@cdn-common/i18n/guard_templates/click.html?raw'
import slideTemplate from '@cdn-common/i18n/guard_templates/slide.html?raw'
import captchaTemplate from '@cdn-common/i18n/guard_templates/captcha.html?raw'
import delayJumpTemplate from '@cdn-common/i18n/guard_templates/delay_jump.html?raw'
import rotateTemplate from '@cdn-common/i18n/guard_templates/rotate.html?raw'
import { extractTemplateKeys, normalizeLocaleTag } from '@/services/errorPageService'

const PLACEHOLDER_RE = /\{\{([a-zA-Z0-9_]+)\}\}/g

type GuardDefaultStrings = Record<string, Record<string, Record<string, string>>>

const DEFAULT_STRINGS = (guardDefaults as { strings: GuardDefaultStrings }).strings || {}

export const GUARD_PAGE_KEYS = ['click', 'slide', 'captcha', 'delay_jump', 'rotate'] as const

export type GuardPageKey = typeof GUARD_PAGE_KEYS[number]

export const ANTI_CC_TYPE_TO_PAGE_KEY: Record<string, GuardPageKey> = {
  slide: 'slide',
  slide_simple: 'slide',
  captcha: 'captcha',
  click: 'click',
  click_simple: 'click',
  '5s': 'delay_jump',
  rotate: 'rotate'
}

const DEFAULT_TEMPLATES: Record<GuardPageKey, string> = {
  click: clickTemplate,
  slide: slideTemplate,
  captcha: captchaTemplate,
  delay_jump: delayJumpTemplate,
  rotate: rotateTemplate
}

const mergeMissingStringValues = (
  target: Record<string, string>,
  fallback: Record<string, string> | undefined
): Record<string, string> => {
  const next = { ...target }
  if (!fallback) return next
  Object.entries(fallback).forEach(([key, value]) => {
    if (!String(next[key] || '').trim()) {
      next[key] = value
    }
  })
  return next
}

export const fillMissingGuardPageStrings = (
  pageKey: string,
  strings: Record<string, Record<string, string>> | undefined,
  langs: string[]
): Record<string, Record<string, string>> => {
  const next = { ...(strings || {}) }
  const pageDefaults = DEFAULT_STRINGS[pageKey] || {}
  langs.forEach(lang => {
    next[lang] = mergeMissingStringValues(next[lang] || {}, pageDefaults[lang])
  })
  Object.entries(pageDefaults).forEach(([lang, values]) => {
    next[lang] = mergeMissingStringValues(next[lang] || {}, values)
  })
  return next
}

export const resolveGuardPageKey = (antiCcType: string): GuardPageKey => {
  return ANTI_CC_TYPE_TO_PAGE_KEY[antiCcType] || 'click'
}

export const ensureGuardPageStructure = (
  pages: GuardPageMap | undefined,
  i18n: ErrorPageI18nSettings | undefined
): GuardPageMap => {
  const enabledLangs = Array.isArray(i18n?.enabled_langs) && i18n.enabled_langs.length
    ? [...new Set(i18n.enabled_langs.map(normalizeLocaleTag).filter(Boolean))]
    : [...DEFAULT_ERROR_PAGE_I18N.enabled_langs]

  const nextPages: GuardPageMap = {}
  GUARD_PAGE_KEYS.forEach(key => {
    const existing = pages?.[key]
    nextPages[key] = {
      template: existing?.template || DEFAULT_TEMPLATES[key] || '',
      strings: fillMissingGuardPageStrings(key, existing?.strings, enabledLangs)
    }
  })
  return nextPages
}

export const renderGuardPagePreview = (
  template: string,
  strings: Record<string, string>,
  lang = 'zh-CN'
): string => {
  if (!template) return ''
  const withStrings = template.replace(PLACEHOLDER_RE, (_, key) => {
    if (key === 'html_lang') {
      return lang
    }
    return strings?.[key] ?? `{{${key}}}`
  })
  return withStrings
}

export const resolveGuardPreviewStrings = (
  def: GuardPageDefinition,
  lang: string,
  defaultLang: string
): Record<string, string> => {
  const candidates = [lang, lang.split('-')[0], defaultLang, 'zh-CN', 'en']
  for (const candidate of candidates) {
    const strings = candidate ? def.strings?.[candidate] : undefined
    if (strings) {
      return strings
    }
  }
  const first = Object.values(def.strings || {})[0] as Record<string, string> | undefined
  return first || {}
}

export const extractGuardTemplateKeys = (template: string): string[] => extractTemplateKeys(template)

export const migrateLegacyAntiCCPageCustom = (
  guardPages: GuardPageMap,
  antiCcType: string,
  legacyCustom: string
): GuardPageMap => {
  const custom = String(legacyCustom || '').trim()
  if (!custom) {
    return guardPages
  }
  const pageKey = resolveGuardPageKey(antiCcType)
  const page = guardPages[pageKey]
  if (page?.template && page.template !== DEFAULT_TEMPLATES[pageKey]) {
    return guardPages
  }
  return {
    ...guardPages,
    [pageKey]: {
      ...(page || { template: DEFAULT_TEMPLATES[pageKey], strings: {} }),
      template: custom
    }
  }
}
