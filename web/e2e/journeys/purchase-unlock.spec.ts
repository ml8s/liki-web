// 付费解锁的 UI 表现：未购买 → 开通引导可见；已购买 → 引导消失、Chat 可用。
// 后端支付全链由 payment_runtime_test.go（Go 单测）覆盖；此处保护前端解锁逻辑。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { mockPageview, mockWorkspace, testUser } from '../mocks'

const user = testUser({ id: 'usr_unlock', email: 'unlock@example.com' })

const baseProduct = {
  id: 'liki-chat', name: 'Liki Chat', description: 'Unified metaphysics chat.',
  skills: ['liki'], display_currency: 'CNY', display_amount_minor: 100,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 100 } },
  ],
}

async function mockPurchaseState(page: Page, { purchased, productPurchased }: { purchased: boolean; productPurchased: boolean }) {
  await mockWorkspace(page, {
    user,
    entitlements: purchased ? [{ id: 'ent_1', skill: 'liki', status: 'active' }] : [],
    allowedSkills: purchased ? ['liki'] : [],
    products: [{ ...baseProduct, purchased: productPurchased }],
  })
  await mockPageview(page)
}

test('locked state shows the purchase entry', async ({ page }) => {
  await mockPurchaseState(page, { purchased: false, productPurchased: false })
  await page.goto('/app/chat')
  await expect(page.locator('#chat-purchase')).toBeVisible()
  await expect(page.locator('#chat-purchase')).toContainText('單次訪問')
})

test('purchased state hides the purchase entry', async ({ page }) => {
  await mockPurchaseState(page, { purchased: true, productPurchased: true })
  await page.goto('/app/chat')
  await expect(page.locator('#chat-purchase')).toBeHidden()
  // Core workspace still present (locale-independent structural markers).
  await expect(page.locator('.profile-picker')).toBeVisible()
  await expect(page.locator('.skill-grid')).toBeVisible()
})
