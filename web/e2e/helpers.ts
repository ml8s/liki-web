// Shared test helpers — regex, assertions, utilities.
// Route mocks live in mocks.ts.
import type { Locator, Page } from '@playwright/test'
import { expect } from '@playwright/test'

// i18n key pattern: if a raw key like "site.name" or "services.bazi.label" appears
// as visible text, the i18n lookup failed and leaked the key.
// i18n keys are always [a-z]+.[a-z]+ with optional third segment.
// Domain "liki.hk" and the standard filename "llms.txt" happen to match — exclude explicitly.
export const I18N_KEY_RE = /\b(?!liki\.hk\b|llms\.txt\b)[a-z]{2,}\.[a-z]{2,}(?:\.[a-z]+)?\b/;

// waitForSettled — 等待应用进入稳定态：所有标记 aria-busy 的加载区（骨架屏/加载中）
// 清空。替代裸 waitForTimeout：a11y 审计与后续断言都需要「渲染完成」这一条件，
// 而非固定时长猜测（负载下固定等待必然 flaky）。
export async function waitForSettled(page: Page): Promise<void> {
  await page.waitForFunction(
    () => !document.querySelector('[aria-busy="true"]'),
    undefined,
    { timeout: 10_000 },
  );
  // Element Plus can still run short enter/leave transitions after loading
  // states clear. Accessibility audits must observe final CSS, not a mid-
  // transition foreground/background pair.
  await page.waitForFunction(
    () =>
      ![...document.querySelectorAll('.el-tag')].some(element =>
        element.getAnimations({ subtree: true }).some(animation => animation.playState === 'running'),
      ),
    undefined,
    { timeout: 10_000 },
  );
}

// 统一 BirthFields 已使用 Element Plus；E2E 也按同一交互契约填写。
export async function fillBirthFields(
  page: Page,
  scope: Page | Locator,
  options: {
    date: string
    time: string
    gender: 'female' | 'male'
    place?: string
  },
) {
  const dateField = scope.getByRole('combobox', { name: /出生日期|Birth date/ })
  await dateField.click()
  await dateField.fill(options.date)
  await dateField.press('Enter')
  await dateField.blur()
  await expect(dateField).toHaveValue(options.date)
  const timeField = scope.getByRole('combobox', { name: /出生时间|出生時間|Birth time/ })
  await timeField.click()
  await timeField.fill(options.time)
  await timeField.press('Enter')
  await timeField.blur()
  await expect(timeField).toHaveValue(options.time)
  await scope.locator('.el-radio-button').nth(options.gender === 'female' ? 0 : 1).click()
  if (options.place) {
    await scope.getByRole('textbox', { name: /出生地|Birth place/ }).fill(options.place)
    await scope.getByRole('textbox', { name: /出生地|Birth place/ }).blur()
  }
}
