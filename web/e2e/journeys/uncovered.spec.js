// E2E smoke tests for previously uncovered pages:
// 404, activate, hepan, mingshu report pages.
// Each test verifies: HTTP 200, no i18n key leaks, no console errors.

import { test, expect } from '../fixtures.js';
import { I18N_KEY_RE } from '../helpers.js';

test.describe('404 page', () => {
  test('renders with heading text', async ({ page }) => {
    const resp = await page.goto('/nonexistent-page');
    expect(resp.status()).toBe(404);

    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
    await expect(page.locator('body')).toContainText('404');
  });

  test('no i18n key leaks', async ({ page }) => {
    await page.goto('/nonexistent-page');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    const bodyText = await page.locator('body').innerText();
    const leaked = bodyText.split('\n').filter(line => I18N_KEY_RE.test(line.trim()));
    if (leaked.length > 0) {
      throw new Error(`Unresolved i18n keys on 404:\n  ${leaked.join('\n  ')}`);
    }
  });
});

test.describe('Retired naming entry points', () => {
  for (const path of ['/zh-Hans/naming/activate', '/zh-Hans/naming/chat']) {
    test(`${path} redirects to Liki Chat`, async ({ page }) => {
      // The redirect is a static meta refresh. Fulfill the app document so the
      // Vue router cannot race this assertion by sending an unauthenticated
      // visitor from /app to /app/login.
      await page.route('**/app**', async route => {
        await route.fulfill({
          status: 200,
          contentType: 'text/html',
          body: '<main id="app"></main>',
        });
      });
      await page.goto(path);
      await expect(page).toHaveURL(/\/app\/?\?scenario=naming&locale=zh-Hans/);
    });
  }
});

test.describe('Report pages — hepan', () => {
  const LOCALES = [
    { path: '/zh-Hans/hepan/report?order_id=demo' },
    { path: '/zh-Hant/hepan/report?order_id=demo' },
    { path: '/en/hepan/report?order_id=demo' },
  ];

  for (const { path } of LOCALES) {
    test(`${path} loads without console errors`, async ({ page }) => {
      const errors = [];
      page.on('pageerror', e => errors.push(e.message));
      await page.goto(path);
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
      if (errors.length > 0) throw new Error(errors.join('\n'));
    });
  }
});

test.describe('Report pages — mingshu', () => {
  const LOCALES = [
    { path: '/zh-Hans/mingshu/report?order_id=demo' },
    { path: '/zh-Hant/mingshu/report?order_id=demo' },
    { path: '/en/mingshu/report?order_id=demo' },
  ];

  for (const { path } of LOCALES) {
    test(`${path} loads without console errors`, async ({ page }) => {
      const errors = [];
      page.on('pageerror', e => errors.push(e.message));
      await page.goto(path);
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
      if (errors.length > 0) throw new Error(errors.join('\n'));
    });
  }
});
