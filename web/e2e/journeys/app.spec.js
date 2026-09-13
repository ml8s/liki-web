// @ts-check
import { test, expect } from '@playwright/test';

const protectedPages = [
  { path: '/app', heading: '工作台' },
  { path: '/app/reports', heading: '报告' },
  { path: '/app/orders', heading: '订单' },
  { path: '/app/account', heading: '账户' },
];

test.describe('Authenticated app shell', () => {
  test('login page mounts without runtime errors', async ({ page }) => {
    const errors = [];
    page.on('pageerror', error => errors.push(error));
    await page.goto('/app/login');
    await expect(page).toHaveURL(/\/app\/login$/);
    await expect(page.locator('input[type="email"]')).toBeVisible();
    expect(errors).toEqual([]);
  });

  test('protected routes redirect unauthenticated users to login', async ({ page }) => {
    for (const { path } of protectedPages) {
      await page.goto(path);
      await expect(page).toHaveURL(/\/app\/login/);
    }
  });

  test('authenticated pages render their headings', async ({ page }) => {
    await page.route('**/api/me', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: {
          id: 'usr_test',
          email: 'user@example.com',
          status: 'active',
          locale: 'zh-Hant',
        } }),
      });
    });

    for (const { path, heading } of protectedPages) {
      await page.goto(path);
      await expect(page.getByRole('heading', { name: heading })).toBeVisible();
    }
  });
});
