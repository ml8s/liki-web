import { test as base, expect, chromium } from '@playwright/test';

const test = base.extend({
  browser: async ({}, use) => {
    // 使用 Playwright 管理的 chromium（CI 已 `playwright install chromium`）。
    // 不用 google-chrome-stable 硬编码路径，避免与 config/CI 的 chromium 不一致。
    const browser = await chromium.launch({
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox'],
    });
    await use(browser);
    await browser.close();
  },

  context: async ({ browser, baseURL }, use, testInfo) => {
    const projectUse = testInfo.project.use;
    const context = await browser.newContext({
      baseURL,
      viewport: projectUse.viewport || { width: 1280, height: 720 },
      ignoreHTTPSErrors: true,
    });
    await use(context);
    await context.close();
  },
});

export { test, expect };
