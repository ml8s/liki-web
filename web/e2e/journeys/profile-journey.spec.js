// @ts-check
// 建档主路径：登录 → 新建档案 → 详情排盘 → 列表可见。真实后端（profiles API +
// engine 排盘），登录走 DEV_MODE 验证码。CI 与本地 e2e 环境均注入 DEV_MODE。
import { test, expect } from '@playwright/test'

const email = `profile-journey-${Date.now()}@test.liki`
const H = { Origin: process.env.BASE_URL || 'http://localhost:8080' }

test.describe.configure({ mode: 'serial' })

test('create profile, see chart, see it listed', async ({ page, browser }) => {
  const ctx = await browser.newContext({ locale: 'zh-Hant' })
  page = await ctx.newPage()
  // 统一账号登录（页面内完成，覆盖登录表单 → dev code → 自动验证）。
  await page.goto('/app/login')
  await page.getByLabel(/郵箱|邮箱/i).fill(email)
  await page.getByRole('button', { name: /取得驗證碼|获取验证码|get code/i }).click()
  const codeResponse = await page.waitForResponse(r => r.url().includes('/api/auth/email/code'))
  const { data } = await codeResponse.json()
  expect(data?.dev_code, 'DEV_MODE must echo a dev code').toBeTruthy()
  await page.getByLabel(/驗證碼|验证码|verification code/i).fill(data.dev_code)
  await page.waitForURL(url => !url.href.includes('/login'), { timeout: 15000 })

  // 新建档案。
  await page.goto('/app/profiles/new')
  await page.getByLabel(/稱呼/).fill('旅程測試')
  // Element Plus renders native radios visually covered by styled spans;
  // click the visible control instead of the hidden original input.
  await page.locator('.el-radio-button:has-text("男")').click()
  await page.getByRole('combobox', { name: /選擇日期/ }).fill('1990-06-15')
  await page.getByRole('combobox', { name: /精確時間/ }).fill('08:30')
  await page.getByLabel(/出生地/).fill('杭州')
  await page.locator('.el-checkbox').last().click()
  await page.getByRole('button', { name: /儲存檔案/ }).click()
  await page.waitForURL(/\/app\/profiles\/[a-f0-9-]+/, { timeout: 15000 })

  // 详情页：排盘渲染（免费确定性计算，无需任何购买）。
  await expect(page.locator('.paipan-preview, [class*=paipan]').first()).toBeVisible()
  await expect(page.getByText(/年柱/).first()).toBeVisible()
  await expect(page.getByText(/日柱/).first()).toBeVisible()

  // 列表页可见该档案。
  await page.goto('/app/profiles')
  await expect(page.getByText('旅程測試').first()).toBeVisible()
})
