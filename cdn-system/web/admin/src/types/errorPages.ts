export interface ErrorPageI18nSettings {
  default_lang: string
  lang_mode: 'browser' | 'fixed'
  enabled_langs: string[]
}

export interface ErrorPageDefinition {
  template: string
  strings: Record<string, Record<string, string>>
}

export type ErrorPageMap = Record<string, ErrorPageDefinition>

export interface GlobalConfigPayload {
  waf?: Record<string, unknown>
  nginx?: Record<string, unknown>
  default_config?: Record<string, unknown>
  resources?: Record<string, unknown>
  error_page_i18n: ErrorPageI18nSettings
  error_pages: ErrorPageMap
}

export type ErrorPageLangMode = 'browser' | 'fixed'

export type SiteErrorPageLang = '' | 'browser' | string
