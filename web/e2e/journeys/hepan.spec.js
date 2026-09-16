// Hepan (Liki 合盘) landing page E2E tests.
// Covers: content rendering, order creation, checkout.
// Source: hepan.astro, payment.js

import { test, expect } from '../fixtures.js';

const heroChecks = [
  { path: '/zh-Hans/hepan', texts: ['Liki 合盘', '八字 · 紫微综合合盘报告'] },
  { path: '/zh-Hant/hepan', texts: ['Liki 合盤', '八字 · 紫微綜合合盤報告'] },
  { path: '/en/hepan',      texts: ['BaZi & ZiWei Compatibility'] },
];

test.describe('Hepan landing page — content', () => {
  for (const { path, texts } of heroChecks) {
    test(`${path} renders Chinese content`, async ({ page }) => {
      await page.goto(path);
      await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });
      for (const t of texts) {
        await expect(page.locator('body')).toContainText(t);
      }
    });
  }

  test('report example section is present', async ({ page }) => {
    await page.goto('/zh-Hans/hepan');
    await expect(page.getByRole('heading', { name: 'Liki 合盘', exact: true }).first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=张明')).toBeVisible();
    await expect(page.locator('text=李芳')).toBeVisible();
    await expect(page.getByRole('heading', { name: '五行互补', exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: '综合评价', exact: true })).toBeVisible();
  });

  test('form has person A and B sections', async ({ page }) => {
    await page.goto('/zh-Hans/hepan');
    await expect(page.getByRole('heading', { name: '甲方', exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: '乙方', exact: true })).toBeVisible();
  });
});

test.describe('Hepan — order creation and checkout', () => {
  const ORDER_ID = 'e2e-hp-001';

  async function mockCreateOrder(page) {
    await page.route('**/api/hepan/orders', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { order_id: ORDER_ID } }),
      });
    });
  }

  async function mockIntent(page) {
    await page.route('**/api/payments/intents', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { id: 'pi-' + ORDER_ID, business_order_id: ORDER_ID, product: 'hepan', provider: 'xunhu', status: 'created' } }),
      });
    });
  }

  async function mockCheckout(page) {
    await page.route('**/api/payments/intents/*/checkout', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { checkout_url: '/checkout/hp-test', qr_url: '' } }),
      });
    });
  }

  test('create order and redirect to checkout', async ({ page }) => {
    await mockCreateOrder(page);
    await mockIntent(page);
    await mockCheckout(page);

    await page.goto('/zh-Hans/hepan');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    // Fill person A
    await page.fill('#hp-a-name', '张三');
    await page.fill('#hp-a-birth', '1990-05-20T12:00');
    await page.fill('#hp-a-city', '上海');

    // Fill person B
    await page.fill('#hp-b-name', '李四');
    await page.fill('#hp-b-birth', '1992-08-15T18:30');
    await page.fill('#hp-b-city', '北京');

    // Fill email
    await page.fill('#hp-email', 'test@example.com');

    // Click buy button
    await page.click('button:has-text("获取报告")');

    // Should redirect to checkout URL
    await page.waitForURL('**/checkout/hp-test', { timeout: 10000 });
  });

  test('create order server error shows inline error', async ({ page }) => {
    await page.route('**/api/hepan/orders', async (route) => {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { message: 'server error' } }) });
    });
    await page.goto('/zh-Hans/hepan');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    await page.fill('#hp-a-name', '张三');
    await page.fill('#hp-a-birth', '1990-05-20T12:00');
    await page.fill('#hp-b-name', '李四');
    await page.fill('#hp-email', 'test@example.com');

    await page.click('button:has-text("获取报告")');

    await expect(page.locator('#form-error')).toBeVisible({ timeout: 5000 });
  });

  test('checkout API error is handled gracefully', async ({ page }) => {
    await page.route('**/api/hepan/orders', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { order_id: ORDER_ID } }) });
    });
    await page.route('**/api/payments/intents/*/checkout', async (route) => {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { message: 'payment error' } }) });
    });

    await page.goto('/zh-Hans/hepan');
    await expect(page.locator('html')).not.toHaveCSS('visibility', 'hidden', { timeout: 15000 });

    await page.fill('#hp-a-name', '张三');
    await page.fill('#hp-a-birth', '1990-05-20T12:00');
    await page.fill('#hp-b-name', '李四');
    await page.fill('#hp-email', 'test@example.com');

    await page.click('button:has-text("获取报告")');

    await expect(page.locator('#form-error')).toBeVisible({ timeout: 5000 });
  });
});
