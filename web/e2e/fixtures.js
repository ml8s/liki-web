// app/admin journey 的共享 fixture：凡是会 mock /api 的 spec 都从这里取 test。
// 未 mock 的 /api 调用由 page 兜底直接判定失败，不再落到 dev/preview 代理上。
import { test as base, expect, chromium } from '@playwright/test';

const test = base.extend({
  browser: async ({}, use) => {
    // 使用 Playwright 管理的 chromium（CI 已 `playwright install chromium`）。
    // 不用 google-chrome-stable 硬编码路径，避免与 config/CI 的 chromium 不一致。
    // context 交给 Playwright 按 use/project 选项建，用例级的 test.use({ locale })
    // 这类设置才会生效。
    const browser = await chromium.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });
    await use(browser);
    await browser.close();
  },

  page: async ({ page }, use) => {
    // 未 mock 的 /api 调用直接失败，而不是落到 dev/preview 代理上刷 ECONNREFUSED。
    // Playwright 的路由是后注册优先，所以用例自己的 page.route 依然生效；
    // live-system 打的是真实后端，不装这层兜底。
    if (!process.env.LIVE_SYSTEM_E2E) {
      await page.route('/api/**', route => route.abort('failed'));
    }
    await use(page);
  },
});

export { test, expect };
