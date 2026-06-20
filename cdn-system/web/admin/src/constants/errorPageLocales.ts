export const COMMON_ERROR_PAGE_LOCALES = [
  { value: 'zh-CN', label: '简体中文 (zh-CN)' },
  { value: 'zh-TW', label: '繁体中文 (zh-TW)' },
  { value: 'en', label: 'English (en)' },
  { value: 'ja', label: '日本語 (ja)' },
  { value: 'ko', label: '한국어 (ko)' },
  { value: 'fr', label: 'Français (fr)' },
  { value: 'de', label: 'Deutsch (de)' },
  { value: 'es', label: 'Español (es)' },
  { value: 'pt', label: 'Português (pt)' },
  { value: 'ru', label: 'Русский (ru)' },
  { value: 'ar', label: 'العربية (ar)' },
  { value: 'th', label: 'ไทย (th)' },
  { value: 'vi', label: 'Tiếng Việt (vi)' },
  { value: 'id', label: 'Bahasa Indonesia (id)' },
  { value: 'ms', label: 'Bahasa Melayu (ms)' },
  { value: 'it', label: 'Italiano (it)' },
  { value: 'nl', label: 'Nederlands (nl)' },
  { value: 'pl', label: 'Polski (pl)' },
  { value: 'tr', label: 'Türkçe (tr)' },
  { value: 'hi', label: 'हिन्दी (hi)' }
]

export const ERROR_PAGE_CODES = [
  { key: '400', label: '400 错误页面' },
  { key: '403', label: '403 错误页面' },
  { key: '502', label: '502 错误页面' },
  { key: '504', label: '504 错误页面' },
  { key: 'traffic_limit', label: '流量超限' },
  { key: 'site_locked', label: '网站被锁' },
  { key: 'domain_invalid', label: '域名无效' },
  { key: 'conn_limit', label: '连接数超限' },
  { key: 'timeout', label: '套餐到期' },
  { key: 'ip', label: '限制IP访问' }
]

export const DEFAULT_ERROR_PAGE_I18N = {
  default_lang: 'zh-CN',
  lang_mode: 'browser',
  enabled_langs: ['zh-CN', 'en']
} as const

export const SITE_ERROR_PAGE_LANG_OPTIONS = [
  { value: '', label: '继承全局' },
  { value: 'browser', label: '跟随浏览器' }
]
