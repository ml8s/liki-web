import { createI18n } from 'vue-i18n'

export const appLocales = ['zh-Hans', 'zh-Hant', 'en'] as const
export type AppLocale = typeof appLocales[number]

export const LOCALE_COOKIE = 'liki_locale'

function normalizeLocale(value: unknown): AppLocale | null {
  const locale = String(value ?? '').trim()
  if ((appLocales as readonly string[]).includes(locale)) return locale as AppLocale
  const normalized = locale.toLowerCase()
  if (normalized.startsWith('zh')) {
    if (normalized.includes('cn') || normalized.includes('hans') || normalized.includes('sg')) return 'zh-Hans'
    return 'zh-Hant'
  }
  if (normalized.startsWith('en')) return 'en'
  return null
}

function readCookieLocale(): AppLocale | null {
  if (typeof document === 'undefined') return null
  const match = document.cookie.match(/(?:^|;\s*)liki_locale=([^;]+)/)
  return normalizeLocale(match?.[1])
}

export function savedLocalePreference(): AppLocale | null {
  return readCookieLocale()
}

function readQueryLocale(): AppLocale | null {
  if (typeof window === 'undefined') return null
  return normalizeLocale(new URLSearchParams(window.location.search).get('locale'))
}

function readBrowserLocale(): AppLocale | null {
  if (typeof navigator === 'undefined') return null
  return normalizeLocale(navigator.language)
}

export function initialLocale(): AppLocale {
  return readQueryLocale() ?? readCookieLocale() ?? readBrowserLocale() ?? 'zh-Hant'
}

export function normalizeUnknownLocale(value: unknown): AppLocale | null {
  return normalizeLocale(value)
}

export const requestedLocale = readQueryLocale()

const localeLoaders = {
  'zh-Hans': () => import('@/locales/zh-Hans.json'),
  'zh-Hant': () => import('@/locales/zh-Hant.json'),
  en: () => import('@/locales/en.json'),
} as const satisfies Record<AppLocale, () => Promise<{ default: Record<string, unknown> }>>

const loadedLocales = new Set<AppLocale>()

async function loadLocaleMessages(locale: AppLocale) {
  if (loadedLocales.has(locale)) return
  const messages = await localeLoaders[locale]()
  i18n.global.setLocaleMessage(locale, messages.default)
  loadedLocales.add(locale)
}

export async function setAppLocale(locale: AppLocale, options: { persist?: boolean } = {}) {
  await loadLocaleMessages(locale)
  i18n.global.locale.value = locale
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
    if (options.persist !== false) {
      const secure = globalThis.location?.protocol === 'https:'
      document.cookie = `${LOCALE_COOKIE}=${locale}; Path=/; Max-Age=31536000; SameSite=Lax${secure ? '; Secure' : ''}`
    }
  }
}

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: 'zh-Hant',
  missingWarn: true,
  fallbackWarn: true,
  messages: {},
})

export async function initializeAppLocale() {
  const locale = i18n.global.locale.value as AppLocale
  await setAppLocale(locale, { persist: false })
  return locale
}
