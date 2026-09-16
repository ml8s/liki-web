import { describe, it, expect } from 'vitest';
// 品牌门面契约（真相源：liki-skills/docs/brand.md v4.0）
// 本测试把品牌规范变成可执行门禁——改这些值 = 品牌变更，须同步更新 brand.md。
// 背景：品牌词在无门禁状态下三次漂移（Liki 起名自称 / 北极星句混入标题位 / 收束行啰嗦）。
import zh from '../i18n/zh-Hans.json';
import hant from '../i18n/zh-Hant.json';
import en from '../i18n/en.json';

const NS = '命理师的 Skill';          // 北极星句：限内部 + about 叙事段
const CATEGORY = '专业命理 Skill';        // 标题位品类词
const STANDARDS = '按命理师的专业标准构建'; // 标准句（北极星的无歧义展开式）

describe('brand facade contract (brand.md v4.0)', () => {
  it('skills 页 hero = 品类词（简繁），subtitle = 标准句', () => {
    expect(zh['skills.hero.title']).toBe(`Liki — ${CATEGORY}`);
    expect(hant['skills.hero.title']).toBe('Liki — 專業命理 Skill');
    expect(zh['skills.hero.subtitle']).toBe(STANDARDS);
    expect(hant['skills.hero.subtitle']).toBe('按命理師的專業標準構建');
    expect(en['skills.hero.subtitle']).toBe('Built to professional standards');
  });

  it('着陆页 hero = slogan + 价值句（C 端）', () => {
    expect(zh['index.hero.title']).toBe('懂命理，用 Liki');
    expect(hant['index.hero.title']).toBe('懂命理，用 Liki');
    expect(en['index.hero.title']).toBe('For Chinese Metaphysics, use Liki.');
    expect(zh['index.hero.subtitle']).toContain('确定性排盘');
    expect(hant['index.hero.subtitle']).toContain('確定性排盤');
  });

  it('英文标题保留 Professional（无属格歧义，修饰标准）', () => {
    expect(en['skills.hero.title']).toBe('Liki — Professional Skill for Chinese Metaphysics');
  });

  it('所有标题位禁用北极星句（简繁）', () => {
    const titleKeys = ['skills.hero.title', 'skills.title', 'skills.meta.ogTitle',
      'skills.meta.twitterTitle', 'index.hero.title', 'index.title',
      'naming.footer', 'naming.metaDesc',
      'disclaimer.metaDesc', 'about.metaDesc'];
    for (const dict of [zh, hant]) {
      for (const k of titleKeys) expect(dict[k], k).not.toContain(NS);
    }
  });

  it('北极星句仅存于 about 品牌叙事段，且必须紧随标准解释（消歧）', () => {
    // 简繁各自的北极星句与标准解释变体
    const variants = [
      [zh, '命理师的 Skill', '按照命理师真实工作标准构建'],
      [hant, '命理師的 Skill', '按照命理師真實工作標準構建'],
    ];
    for (const [dict, ns, explain] of variants) {
      expect(dict['about.s1.body']).toContain(ns);
      expect(dict['about.s1.body']).toContain(explain);
    }
  });

  it('收束行（歧义补丁）全面退役', () => {
    const retired = ['人人可用', '专业者可信赖', 'Trusted by professionals'];
    for (const dict of [zh, hant, en]) {
      for (const [k, v] of Object.entries(dict)) {
        if (typeof v === 'string') {
          for (const r of retired) expect(v, `${k} 含退役词「${r}」`).not.toContain(r);
        }
      }
    }
  });

  it('活跃 i18n 不再使用退役中文品牌名', () => {
    for (const dict of [zh, hant]) {
      for (const [k, v] of Object.entries(dict)) {
        if (typeof v === 'string') {
          expect(v, `${k} 含退役品牌词`).not.toMatch(/灵机|靈機/);
        }
      }
    }
  });

  it('chat 自称 = 品牌 + 起名顾问角色（产品名不作自称）', () => {
    expect(zh['chat.greeting']).toContain('我是 Liki，起名顾问');
    expect(zh['chat.greeting']).not.toContain('Liki 起名。');
  });
});
