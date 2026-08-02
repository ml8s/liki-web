// Smoke test — visits every page and asserts no console errors/warnings + framework renders.
// Catches: source map 404s, i18n load failures,
//          unresolved i18n keys (raw key text leaked to page).
// Render markers: .chat-shell (vanilla JS chat page), null (static resources). Framework check uses FOUC guard.

import { test, expect } from '../fixtures.js';
import { I18N_KEY_RE } from '../helpers.js';

// Pages to test with their framework-specific render marker and i18n content checks.
const PAGES = [
  { path: '/zh-Hans/naming', name: 'naming (vanilla)',
    checks: [
      { selector: '.font-brand', text: 'Liki 灵机' },
      { selector: '.cta-section h2', text: '找到你的名字' },
    ] },
  { path: '/zh-Hans/naming/chat', marker: '.chat-shell',     name: 'chat (vanilla)' },
  { path: '/zh-Hans/disclaimer', name: 'disclaimer (vanilla)',
    checks: [
      { selector: 'h1', text: '免责声明' },
    ] },
  { path: '/en/naming', name: 'naming EN',
    checks: [
      { selector: '.font-brand', text: 'Liki' },
      { selector: '.cta-section h2', text: 'Find Your Name' },
    ] },
  { path: '/en/naming/chat',     marker: '.chat-shell',     name: 'chat EN (vanilla)' },
  { path: '/zh-Hant/naming', name: 'naming ZH-Hant',
    checks: [
      { selector: '.font-brand', text: 'Liki 靈機' },
      { selector: '.cta-section h2', text: '找到你的名字' },
    ] },
  { path: '/zh-Hant/naming/chat',     marker: '.chat-shell',     name: 'chat ZH-Hant' },
  // Legal pages — marker renders, i18n resolves, keys don't leak
  { path: '/zh-Hans/about',    name: 'about (vanilla)' },
  { path: '/zh-Hans/contact',  name: 'contact (vanilla)' },
  { path: '/zh-Hans/privacy',  name: 'privacy (vanilla)' },
  { path: '/zh-Hans/terms',    name: 'terms (vanilla)' },
  // Static resources
  { path: '/skills/liki-master/SKILL.md', marker: null,        name: 'skills',   resource: true },
  { path: '/zh-Hans/mingshu',   name: 'mingshu (vanilla)',
    checks: [
      { selector: 'h1', text: '灵机命书' },
    ] },
  { path: '/zh-Hans/hepan',    name: 'hepan (vanilla)',
    checks: [
      { selector: 'h1', text: '灵机合盘' },
    ] },
  { path: '/en/hepan',       name: 'hepan EN',
    checks: [
      { selector: 'h1', text: 'BaZi & ZiWei Compatibility' },
    ] },
  { path: '/llms.txt',         marker: null,             name: 'llms.txt', resource: true },
  { path: '/zh-Hans/hepan/report?order_id=demo',  marker: '#report-container',  name: 'hepan report', resource: true },
  { path: '/zh-Hans/mingshu/report?order_id=demo', marker: '#report-container', name: 'mingshu report', resource: true },
];

test.describe('Smoke — all pages render without errors or warnings', () => {
  for (const { path, marker, name, checks, resource, skipI18nCheck } of PAGES) {
    test(`${name}: ${path}`, async ({ page }) => {
      const consoleProblems = [];
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
  test('naming page images are valid', async ({ page }) => {
    await page.goto('/zh-Hans/naming');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    const imgs = page.locator('img');
    const count = await imgs.count();
    for (let i = 0; i < count; i++) {
      const img = imgs.nth(i);
      const src = await img.getAttribute('src');
      if (!src || src.startsWith('data:')) continue;
      const w = await img.evaluate(el => el.naturalWidth);
      if (w === 0) throw new Error(`Broken image on /zh-Hans/naming: src="${src}"`);
    }
  });
});

// ── Mobile viewport: critical pages render on mobile ──

test.describe('Mobile viewport', () => {
  test('naming renders at 375x812', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto('/zh-Hans/naming');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
    await expect(page.locator('#purchase-form')).toBeVisible();
  });
});
