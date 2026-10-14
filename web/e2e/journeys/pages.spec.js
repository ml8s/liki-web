// Smoke test — visits every page and asserts no console errors/warnings + framework renders.
// Catches: source map 404s, i18n load failures,
//          unresolved i18n keys (raw key text leaked to page).
// Render markers are null for static resources. Framework check uses FOUC guard.

import { test, expect } from '../fixtures.js';
import { I18N_KEY_RE } from '../helpers.js';

// Analytics is non-essential; the shared localhost CI client can hit its rate limit.

// Pages to test with their framework-specific render marker and i18n content checks.
const PAGES = [
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
      { selector: 'h1', text: 'AI naming and name interpretation' },
    ] },
  { path: '/zh-Hant/naming', name: 'naming ZH-Hant',
    checks: [
      { selector: '.font-brand', text: 'Liki' },
      { selector: 'h1', text: 'AI 起名與姓名解讀' },
    ] },
  // 着陆页（C 端）：slogan + 免费排盘 + 服务 + 信任
  { path: '/zh-Hans/',        name: 'index (C-end landing)',
    checks: [
      { selector: '.landing-paipan', text: '免费排盘' },
      { selector: '.landing-services', text: '六大命理场景' },
      { selector: '.landing-trust', text: '为什么可信' },
    ] },
  // 开发者 skill 页（原品牌门面迁址）
  { path: '/zh-Hans/skills/', name: 'skills (developer)',
    checks: [
      { selector: '.hero-brand h1', text: 'Liki — 专业命理 Skill' },
      { selector: '.install-cmd', text: 'npx skills add ml8s/liki' },
    ] },
  // Legal pages — marker renders, i18n resolves, keys don't leak
  { path: '/zh-Hans/about',    name: 'about (static)' },
  { path: '/zh-Hans/contact',  name: 'contact (static)' },
  { path: '/zh-Hans/privacy',  name: 'privacy (static)' },
  { path: '/zh-Hans/terms',    name: 'terms (static)' },
  // Static resources
  { path: '/skills/liki/SKILL.md', marker: null,          name: 'skills',   resource: true },
  { path: '/zh-Hans/mingshu',   name: 'mingshu (static)',
    checks: [
      { selector: 'h1', text: '个人命书与终身运势解读' },
    ] },
  { path: '/zh-Hans/hepan',    name: 'hepan (static)',
    checks: [
      { selector: 'h1', text: '八字合盘与感情关系解读' },
    ] },
  { path: '/en/hepan',       name: 'hepan EN',
    checks: [
      { selector: 'h1', text: 'BaZi compatibility and relationship reading' },
    ] },
  { path: '/llms.txt',         marker: null,             name: 'llms.txt', resource: true },
];

test.describe('Static page interactions', () => {
  test('home analytics sends a first-party path-only pageview', async ({ page }) => {
    const pageviews = [];
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

  test('free chart form stores an onboarding draft', async ({ page }) => {
    await page.route('**/app/birth-profiles/new', route => route.fulfill({
      status: 200,
      contentType: 'text/html',
      body: '<!doctype html><html><title>New profile</title></html>',
    }));

    await page.goto('/zh-Hans/');
    await page.fill('#paipanHook [name=date]', '2000-01-01');
    await page.fill('#paipanHook [name=time]', '12:00');
    await page.selectOption('#paipanHook [name=gender]', 'female');
    await page.click('#paipanHook button[type=submit]');

    await page.waitForURL('**/app/birth-profiles/new');
    expect(await page.evaluate(() => sessionStorage.getItem('liki-draft'))).toEqual(JSON.stringify({
      birth_date: '2000-01-01',
      birth_time: '12:00',
      gender: 'female',
      calendar: 'solar',
    }));
  });

  test('skills page copies install commands with temporary feedback', async ({ page }) => {
    await page.addInitScript(() => {
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: (text) => Promise.resolve(text) },
      });
    });

    await page.goto('/zh-Hans/skills/');
    const button = page.locator('.copy-btn').first();
    const original = await button.innerHTML();
    await button.click();
    await expect(button).toHaveText('✓');
    await expect.poll(() => button.innerHTML()).toBe(original);
  });
});

test.describe('Smoke — all pages render without errors or warnings', () => {
  for (const { path, marker, name, checks, resource, skipI18nCheck } of PAGES) {
    test(`${name}: ${path}`, async ({ page }) => {
      const consoleProblems = [];
      await page.route('**/api/analytics/pageview', route => route.fulfill({ status: 204 }));
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
        expect(resp.status()).toBe(200);
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
    expect(resp.status()).toBe(200);
    const ct = resp.headers()['content-type'];
    expect(ct).toMatch(/^image\/png/);
  });
});

test.describe('Images load without error', () => {
  for (const page of ['naming', 'mingshu', 'hepan']) {
    test(`${page} page images are valid`, async ({ page: p }) => {
      await p.goto(`/zh-Hans/${page}`);
      await expect(p.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

      const imgs = p.locator('img');
      const count = await imgs.count();
      for (let i = 0; i < count; i++) {
        const img = imgs.nth(i);
        const src = await img.getAttribute('src');
        if (!src || src.startsWith('data:')) continue;
        const w = await img.evaluate(el => el.naturalWidth);
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
