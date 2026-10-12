import zhHant from '../i18n/zh-Hant.json';
import zhHans from '../i18n/zh-Hans.json';
import en from '../i18n/en.json';
import localeConfig from '../shared/locales.json';

const translations = { 'zh-Hant': zhHant, 'zh-Hans': zhHans, en };
// Single source of truth for the locale set lives in shared/locales.json;
// the SPA and the sitemap builder derive from the same file.
export const locales = localeConfig.locales;

export function t(locale, key) {
  return translations[locale]?.[key] ?? key;
}
