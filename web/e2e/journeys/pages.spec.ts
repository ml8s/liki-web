// Smoke test — visits every page and asserts no console errors/warnings + framework renders.
// Catches: source map 404s, i18n load failures,
//          unresolved i18n keys (raw key text leaked to page).
// Render markers are null for static resources. Framework check uses FOUC guard.

import { test, expect } from '../fixtures';
import { I18N_KEY_RE } from '../helpers';
import { mockCityCoords, mockEngineTool, mockPageview } from '../mocks';
import { fillBirthFields } from '../helpers';
import { baziChart, baziFull, baziLiunian } from '../engine-fixtures';

// 每个静态页首屏都会打一次 pageview 信标；声明成 204 空响应。
// （信标本就是非必需的；CI 共享 localhost 客户端还可能撞上它的限流。）
test.beforeEach(async ({ page }) => {
  await mockPageview(page);
});

// Pages to test with their framework-specific render marker and i18n content checks.
interface PageCase {
  path: string
  name: string
  marker?: string | null
  resource?: boolean
  skipI18nCheck?: boolean
  checks?: { selector: string; text: string }[]
}

const PAGES: PageCase[] = [
  { path: '/zh-Hans/naming', name: 'naming (static)',
    checks: [
      { selector: '.font-brand', text: 'Liki' },
      { selector: 'h1', text: 'AI 起名与姓名解读' },
    ] },
  { path: '/zh-Hans/disclaimer', name: 'disclaimer (static)',
    checks: [
      { selector: 'h1', text: '免责声明' },
    ] },
  { path: '/en/naming', name: 'naming EN',
    checks: [
      { selector: '.font-brand', text: 'Liki' },
      { selector: 'h1', text: 'Chinese Baby Names & Name Analysis' },
    ] },
  { path: '/zh-Hant/naming', name: 'naming ZH-Hant',
    checks: [
      { selector: '.font-brand', text: 'Liki' },
      { selector: 'h1', text: 'AI 起名與姓名解讀' },
    ] },
  // 着陆页（C 端）：slogan + 免费排盘 + 服务 + 信任
  { path: '/zh-Hans/',        name: 'index (C-end landing)',
    checks: [
      { selector: '.landing-chart', text: '免费排盘' },
      { selector: '.landing-services', text: '七项服务：命理、流年、合盘、问卦、择日、风水与起名' },
      { selector: '.landing-trust', text: '引擎排盘，结论可验证' },
    ] },
  // Legal pages — marker renders, i18n resolves, keys don't leak
  { path: '/zh-Hans/about',    name: 'about (static)' },
  { path: '/zh-Hans/contact',  name: 'contact (static)' },
  { path: '/zh-Hans/privacy',  name: 'privacy (static)' },
  { path: '/zh-Hans/terms',    name: 'terms (static)' },
  // Static resources
  { path: '/zh-Hans/natal',   name: 'natal (merged landing)',
    checks: [
      { selector: 'h1', text: '在线命理：八字、紫微与命书' },
    ] },
  { path: '/en/natal',       name: 'natal EN',
    checks: [
      { selector: 'h1', text: 'Chinese Astrology: BaZi, Zi Wei Dou Shu & Life Reading' },
    ] },
  { path: '/zh-Hans/hepan',   name: 'hepan (restored landing)',
    checks: [
      { selector: 'h1', text: '八字合盘：合婚、姻缘与关系解读' },
    ] },
  { path: '/zh-Hans/liunian', name: 'liunian (annual fortune landing)',
    checks: [
      { selector: 'h1', text: '流年运势：这一年，你的机会与注意' },
    ] },
  { path: '/zh-Hans/divination', name: 'divination (capability landing)',
    checks: [
      { selector: 'h1', text: '六爻问卦与应期解读' },
    ] },
  { path: '/zh-Hans/fengshui', name: 'fengshui (capability landing)',
    checks: [
      { selector: 'h1', text: '风水方位与宅盘解读' },
    ] },
  { path: '/zh-Hans/huangli', name: 'huangli (capability landing)',
    checks: [
      { selector: 'h1', text: '黄历择日与宜忌解读' },
    ] },
  { path: '/llms.txt',         marker: null,             name: 'llms.txt', resource: true },
];

test.describe('Static page interactions', () => {
  test('requests carry a correlation id header', async ({ request }) => {
    const response = await request.get('/zh-Hans/')
    const id = response.headers()['x-request-id'] ?? ''
    expect(id.length).toBeGreaterThan(8)
  });

  test('bare capability entries negotiate a localized landing page', async ({ request }) => {
    const cases = [
      ['/divination', 'zh-TW,zh;q=0.9', '/zh-Hant/divination/'],
      ['/fengshui', 'ja-JP,ja;q=0.9', '/zh-Hant/fengshui/'],
      ['/huangli', 'zh-Hans,zh;q=0.9', '/zh-Hans/huangli/'],
    ];

    for (const [path, language, location] of cases) {
      const response = await request.get(path, {
        headers: { 'Accept-Language': language },
        maxRedirects: 0,
      });
      expect(response.status()).toBe(302);
      expect(response.headers().location).toBe(location);
    }
  });

  test('home analytics sends a first-party path-only pageview', async ({ page }) => {
    const pageviews: unknown[] = [];
    await page.route('**/api/analytics/pageview', async (route) => {
      pageviews.push(route.request().postDataJSON());
      await route.fulfill({ status: 204 });
    });

    await page.goto('/zh-Hans/');
    await expect.poll(() => pageviews).toHaveLength(1);
    expect(pageviews[0]).toEqual({ path: '/zh-Hans/' });
  });

  test('language switcher stores the locale and routes to its static page', async ({ page }) => {
    await page.route('**/en/', route => route.fulfill({
      status: 200,
      contentType: 'text/html',
      body: '<!doctype html><html lang="en"><title>Liki</title></html>',
    }));

    await page.goto('/zh-Hans/');
    await page.click('#lang-toggle');
    await expect(page.locator('#lang-dropdown')).toBeVisible();
    await expect(page.locator('#lang-toggle')).toHaveAttribute('aria-expanded', 'true');

    await page.click('[data-lang="en"]');
    await page.waitForURL('**/en/');
    const cookies = await page.context().cookies();
    expect(cookies.find(({ name }) => name === 'liki_locale')).toMatchObject({ value: 'en' });
  });

  test('homepage free-chart CTAs stay on the marketing site', async ({ page }) => {
    await page.goto('/zh-Hans/')

    await expect(page.locator('.landing-hero .landing-actions a').first())
      .toHaveAttribute('href', '#chart-tool')
    await expect(page.locator('.landing-cta a'))
      .toHaveAttribute('href', '#chart-tool')

    // 七个服务卡的“去体验”必须进入营销站免费工具，不得直接进 App。
    const cardLinks = page.locator('#services .card > a')
    await expect(cardLinks.first()).toHaveAttribute('href', '/zh-Hans/natal/#chart-tool')
    const hrefs = await cardLinks.evaluateAll(nodes => nodes.map(node => node.getAttribute('href') ?? ''))
    expect(hrefs.some(href => href.startsWith('/app'))).toBe(false)
  })

  test('free chart form renders charts and hands off only on the interpretation CTA', async ({ page }) => {
    const palaces = ['命宫', '兄弟', '夫妻', '子女', '财帛', '疾厄', '迁移', '仆役', '官禄', '田宅', '福德', '父母']
      .map((name, index) => ({
        name,
        index,
        gan: '甲',
        zhi: ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥'][index],
        xing_yao: [],
      }));
    const ziwei = {
      gong_wei: palaces,
      ju_shu: '火六局',
      ming_gong: '命宫',
      ming_zhu: '巨门',
      shen_zhu: '文昌',
      ziwei_pos: '命宫',
      shen_gong: '命宫',
      nian_gan: '庚',
      nian_zhi: '午',
      shi_zhi: '卯',
      si_hua: {},
    };

    await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
    await mockEngineTool(page, 'bazi_chart', baziChart);
    await mockEngineTool(page, 'bazi_fullchart', baziFull);
    await mockEngineTool(page, 'bazi_liunian', { items: [baziLiunian] });
    await mockEngineTool(page, 'ziwei_chart', ziwei);
    await mockEngineTool(page, 'ziwei_fullchart', ziwei);
    await mockEngineTool(page, 'ziwei_daxian', { items: [] });
    await mockEngineTool(page, 'ziwei_liunian', { items: [{
      ming_gong: '命宫',
      ming_gong_name: '命宫',
      zhi: '子',
      si_hua: {},
      gong_wei: palaces.map(({ name, zhi }) => ({ name, zhi, xing_yao: [], is_liu_ming: name === '命宫' })),
    }] });

    await page.goto('/zh-Hans/');
    const tool = page.locator('#quick-birth-tool');
    await fillBirthFields(page, tool, { date: '2000-01-01', time: '12:00', gender: 'female', place: '上海' });
    await tool.locator('input[name="birth-timezone"]').fill('Asia/Shanghai');
    await tool.locator('input[name="birth-longitude"]').fill('121.4726');
    await tool.locator('button[type=submit]').click();

    // 免费排盘留在营销站完成；不跳 App。
    await expect(tool.locator('.bazi-panel').first()).toBeVisible();
    await expect(tool.locator('.ziwei-panel').first()).toBeVisible();
    await expect(tool.locator('.tool-cta')).toBeVisible();
    expect(page.url()).toContain('/zh-Hans/');

    // 只有“解读”CTA 才交接给 App。
    await tool.locator('.tool-cta').evaluate(element => element.removeAttribute('href'));
    await tool.locator('.tool-cta').click();

    const raw = await page.evaluate(() => sessionStorage.getItem('liki:form-drafts') ?? '{}');
    const drafts = JSON.parse(raw) as Record<string, { data: Record<string, unknown>; createdAt: number }>;
    const birth = drafts['handoff:birth'];
    expect(birth?.data).toMatchObject({
      date: '2000-01-01',
      time: '12:00',
      gender: 'female',
      location: {
        place: '上海市',
        longitude: 121.4726,
        timezone: 'Asia/Shanghai',
        source: 'city',
      },
    });
    expect(typeof birth?.createdAt).toBe('number');
    expect(drafts['handoff:consultation']?.data).toMatchObject({
      version: 1,
      scenario: 'natal',
    });
  });
});

test.describe('Smoke — all pages render without errors or warnings', () => {
  for (const { path, marker, name, checks, resource, skipI18nCheck } of PAGES) {
    test(`${name}: ${path}`, async ({ page }) => {
      const consoleProblems: string[] = [];
      page.on('console', msg => {
        if ((msg.type() === 'error' || msg.type() === 'warning') && !(msg.location().url || '').includes('favicon')) {
          consoleProblems.push(`[${msg.type()}] ${msg.text()} (${msg.location().url || 'no-url'})`);
        }
      });
      page.on('requestfailed', req => {
        // Ignore aborted requests (page close during navigation)
        if (req.failure()?.errorText !== 'net::ERR_ABORTED') {
          consoleProblems.push(`[requestfailed] ${req.url()} — ${req.failure()?.errorText || 'unknown'}`);
        }
      });
      page.on('pageerror', err => {
        consoleProblems.push(`[uncaught] ${err.message}`);
      });

      const resp = await page.goto(path);

      // Static resources: just verify HTTP 200.
      if (resource) {
        expect(resp?.status()).toBe(200);
        return;
      }

      // Wait for i18n to finish (FOUC guard removes html visibility:hidden).
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

      // i18n content must resolve — raw key like "site.name" means fetch failed.
      if (checks) for (const { selector, text } of checks) {
        await expect(page.locator(selector).first()).toContainText(text, { timeout: 5000 });
      }

      // No unresolved i18n keys leaked to the page body.
      if (!skipI18nCheck) {
        const bodyText = await page.locator('body').innerText();
        const leakedKeys = bodyText.split('\n').filter(line => I18N_KEY_RE.test(line.trim()));
        if (leakedKeys.length > 0) {
          throw new Error(`Unresolved i18n keys on ${path}:\n  ${leakedKeys.join('\n  ')}`);
        }
        // Raw template syntax must not leak.
        if (/\{\{/.test(bodyText)) {
          throw new Error(`Raw template syntax '{{ }}' found on ${path}`);
        }
      }

      // Vue pages: v-cloak must be removed after mount.
      if (marker === '.chat-shell') {
        await expect(page.locator('#app:not([v-cloak])')).toBeAttached({ timeout: 10000 });
      }

      if (consoleProblems.length > 0) {
        throw new Error(`Console errors on ${path}:\n  ${consoleProblems.join('\n  ')}`);
      }
    });
  }
});

// ── OG meta tags: social sharing cards ──

test.describe('OG / Twitter meta tags', () => {
  test('naming page has correct OG tags', async ({ page }) => {
    await page.goto('/zh-Hans/naming');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    const tags = {
      'og:title':       await page.locator('meta[property="og:title"]').getAttribute('content'),
      'og:description': await page.locator('meta[property="og:description"]').getAttribute('content'),
      'og:image':       await page.locator('meta[property="og:image"]').getAttribute('content'),
      'og:url':         await page.locator('meta[property="og:url"]').getAttribute('content'),
      'og:type':        await page.locator('meta[property="og:type"]').getAttribute('content'),
      'twitter:card':   await page.locator('meta[name="twitter:card"]').getAttribute('content'),
    };

    expect(tags['og:title']).toBeTruthy();
    expect(tags['og:description']).toBeTruthy();
    expect(tags['og:image']).toMatch(/^https:\/\/liki\.hk\/img\/og-image\.png$/);
    expect(tags['og:url']).toBe('https://liki.hk/zh-Hans/naming/');
    expect(tags['og:type']).toBe('website');
    expect(tags['twitter:card']).toBe('summary_large_image');
  });

  test('OG image is accessible', async ({ page }) => {
    const resp = await page.goto('/img/og-image.png');
    expect(resp?.status()).toBe(200);
    const ct = resp?.headers()['content-type'] ?? '';
    expect(ct).toMatch(/^image\/png/);
  });
});

test.describe('Images load without error', () => {
  for (const page of ['naming', 'natal']) {
    test(`${page} page images are valid`, async ({ page: p }) => {
      await p.goto(`/zh-Hans/${page}`);
      await expect(p.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

      const imgs = p.locator('img');
      const count = await imgs.count();
      for (let i = 0; i < count; i++) {
        const img = imgs.nth(i);
        const src = await img.getAttribute('src');
        if (!src || src.startsWith('data:')) continue;
        const w = await img.evaluate(el => (el as HTMLImageElement).naturalWidth);
        if (w === 0) throw new Error(`Broken image on /zh-Hans/${page}: src="${src}"`);
      }
    });
  }
});

// ── Mobile viewport: critical pages render on mobile ──

test.describe('Mobile viewport', () => {
  test('naming renders at 375x812', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto('/zh-Hans/naming');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
    await expect(page.locator('[data-cta="naming"]').first()).toBeVisible();
  });
});
