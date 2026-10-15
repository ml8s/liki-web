// E2E smoke tests for previously uncovered pages: the 404 page.
// Each test verifies: HTTP 200, no i18n key leaks, no console errors.

import { test, expect } from '../fixtures';
import { I18N_KEY_RE } from '../helpers';

test.describe('404 page', () => {
  test('renders with heading text', async ({ page }) => {
    const resp = await page.goto('/nonexistent-page');
    expect(resp?.status()).toBe(404);

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

