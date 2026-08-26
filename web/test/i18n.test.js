import { describe, it, expect } from 'vitest';
// 直接 import 生产实现——此前本文件复制了一份"加强版"（params 插值 + detectLang/setLang），
// 而生产 i18n.js 并无这些函数（语言检测实际在 I18nData 组件 inline script），
// 幽灵测试给了虚假覆盖。现在只测真实契约。
import { t, locales } from '../src/i18n.js';
import zh from '../i18n/zh-Hans.json';
import en from '../i18n/en.json';
import hant from '../i18n/zh-Hant.json';

const dict = { 'zh-Hans': zh, en, 'zh-Hant': hant };

describe('t()', () => {
  it('returns translation for each locale', () => {
    for (const locale of locales) {
      expect(t(locale, 'site.name')).toBe(dict[locale]['site.name']);
    }
  });

  it('falls back to the key itself when missing', () => {
    expect(t('zh-Hans', 'no.such.key')).toBe('no.such.key');
    expect(t('en', 'no.such.key')).toBe('no.such.key');
  });

  it('falls back to the key for unknown locale', () => {
    // 生产契约：未知 locale → undefined?.[key] ?? key（无 zh-Hans 兜底链）
    expect(t('fr', 'site.name')).toBe('site.name');
  });

  it('locales covers all three site languages', () => {
    expect(locales).toEqual(['zh-Hant', 'zh-Hans', 'en']);
  });
});
