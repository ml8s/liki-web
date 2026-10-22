import zhHant from '../i18n/zh-Hant.json';
import zhHans from '../i18n/zh-Hans.json';
import en from '../i18n/en.json';
import localeConfig from '@shared/locales.json';
import { createBirthFieldLabels } from '@shared/components/form/birth-fields';

// JSON 模块的类型是逐字面量推导的，翻译表按 locale/key 取用，这里按索引签名收敛。
const translations: Record<string, Record<string, string>> = { 'zh-Hant': zhHant, 'zh-Hans': zhHans, en };

// Single source of truth for the locale set lives in shared/locales.json;
// the SPA and the sitemap builder derive from the same file.
export const locales: string[] = localeConfig.locales;

// 未知 locale / key 回落到 key 本身——模板渲染不该出现空白。
// params 用于带占位符的文案（如「了解{service}」）：未提供的占位符原样保留。
export function t(locale: string, key: string, params?: Record<string, string | number>): string {
  const value = translations[locale]?.[key] ?? key;
  if (!params) return value;
  return value.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in params ? String(params[name]) : match,
  );
}

export function birthFieldLabels(locale: string) {
  return createBirthFieldLabels((key, params) => t(locale, key, params as Record<string, string | number>))
}
