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

test.describe('Activate page', () => {
  const LOCALES = [
    { path: '/zh-Hans/naming/activate', label: '激活码' },
    { path: '/zh-Hant/naming/activate', label: '激活碼' },
    { path: '/en/naming/activate',      label: 'Activation Code' },
  ];

  for (const { path, label } of LOCALES) {
    test(`${path} renders activation form`, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

      // Form must be present with activation code input and email input.
      await expect(page.locator('#activate-form')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('body')).toContainText(label);
    });
  }

  test('form has code and email inputs', async ({ page }) => {
    await page.goto('/zh-Hans/naming/activate');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
    await expect(page.locator('#activate-code')).toBeVisible();
    await expect(page.locator('#activate-email')).toBeVisible();
    await expect(page.locator('#activate-form button[type="submit"]')).toBeVisible();
  });

  test('no i18n key leaks', async ({ page }) => {
    await page.goto('/zh-Hans/naming/activate');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    const bodyText = await page.locator('body').innerText();
    const leaked = bodyText.split('\n').filter(line => I18N_KEY_RE.test(line.trim()));
    if (leaked.length > 0) {
      throw new Error(`Unresolved i18n keys on activate:\n  ${leaked.join('\n  ')}`);
    }
    if (/\{\{/.test(bodyText)) {
      throw new Error("Raw template syntax '{{ }}' found on activate");
    }
  });

  test('server error during redeem shows error message', async ({ page }) => {
    await page.route('**/api/naming/verification-codes/redeem', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'server error' } }),
      });
    });

    await page.goto('/zh-Hans/naming/activate');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    await page.locator('input[placeholder]').first().fill('TEST-CODE');
    await page.locator('input[type="email"]').fill('user@example.com');
    await page.locator('button[type="submit"]').click();

    await expect(page.locator('#activate-error')).toBeVisible({ timeout: 5000 });
  });
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
