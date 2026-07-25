import zhHant from '../i18n/zh-Hant.json';
import zhHans from '../i18n/zh-Hans.json';
import en from '../i18n/en.json';

const translations = { 'zh-Hant': zhHant, 'zh-Hans': zhHans, en };
export const locales = ['zh-Hant', 'zh-Hans', 'en'];

export function t(locale, key) {
  return translations[locale]?.[key] ?? key;
}
