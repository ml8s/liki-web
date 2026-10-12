// @ts-check
// 管理台会话状态：未登录显示登录表单；已登录但无管理能力必须明确告知，
// 不能落入「已登录却反复看到登录表单」的循环。
import { test, expect } from '@playwright/test'

const json = (status, data) => ({
  status,
  contentType: 'application/json',
  body: JSON.stringify(data),
})

test('403 session shows the no-permission state, not the login form', async ({ page }) => {
  await page.route('**/api/admin/session', route => route.fulfill(json(403, {
    type: 'about:blank', title: 'forbidden', status: 403, detail: '无权访问',
  })))
  await page.goto('/admin.html')
  await expect(page.locator('#no-permission')).toBeVisible()
  await expect(page.locator('#no-permission')).toContainText('没有管理后台权限')
  await expect(page.locator('#auth-section')).toBeHidden()
  await expect(page.locator('#main-section')).toBeHidden()
})

test('401 session keeps the login form', async ({ page }) => {
  await page.route('**/api/admin/session', route => route.fulfill(json(401, {
    type: 'about:blank', title: 'unauthenticated', status: 401,
  })))
  await page.goto('/admin.html')
  await expect(page.locator('#auth-section')).toBeVisible()
  await expect(page.locator('#no-permission')).toBeHidden()
})
