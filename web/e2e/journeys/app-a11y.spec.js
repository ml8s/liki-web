// @ts-check
import { test, expect } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'

const json = data => ({ status: 200, contentType: 'application/json', body: JSON.stringify({ data }) })

const user = { id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant' }
const entitlements = { entitlements: [], allowed_skills: ['liki'] }
const profile = {
  id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', gender: 'female', relation: '家人',
  calendar: 'lunar', birth_date: '1958-闰5-12', birth_time: '06:00', birth_time_known: 'known',
  birth_place: '上海', note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}
const session = {
  id: 'chat_1', user_id: 'usr_test', profile_id: 'p_1', skill: 'liki', title: '妈妈 · 八字紫微',
  status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', last_message_at: null,
}
const message = {
  id: 'msg_1', session_id: 'chat_1', user_id: 'usr_test', role: 'user', kind: 'text', content: '你好', created_at: '2026-01-01T00:01:00Z',
}
const product = {
  id: 'liki-chat', name: 'Liki Chat', description: 'Unified metaphysics chat.',
  skills: ['liki'], prices: { CNY: 2990, USD: 2990 }, purchased: true,
}
const order = {
  id: 'order_1', user_id: 'usr_test', email: 'user@example.com', product_id: 'liki-chat',
  amount_minor: 2990, currency: 'CNY', status: 'paid', created_at: '2026-01-01T00:00:00Z', paid_at: '2026-01-01T00:05:00Z',
}

const pages = [
  { path: '/app/login', name: 'login', marker: 'input[type="email"]' },
  { path: '/app/', name: 'dashboard', marker: '.services' },
  { path: '/app/profiles', name: 'profiles', marker: '.profile-card' },
  { path: '/app/profiles/new', name: 'profile-new', marker: '.birth-form' },
  { path: '/app/profiles/p_1', name: 'profile-detail', marker: '.paipan-preview' },
  { path: '/app/synastry', name: 'synastry', marker: '.b-form' },
  { path: '/app/naming', name: 'naming', marker: '.el-select' },
  { path: '/app/divination', name: 'divination', marker: 'input[placeholder*="项目"]' },
  { path: '/app/fengshui', name: 'fengshui', marker: '.el-select' },
  { path: '/app/huangli', name: 'huangli', marker: '.el-select' },
  { path: '/app/chat', name: 'chat-list', marker: '.picker-row' },
  { path: '/app/chat/chat_1', name: 'chat-session', marker: '.thread' },
  { path: '/app/orders', name: 'orders', marker: '.orders-table, .order-cards' },
  { path: '/app/account', name: 'account', marker: '.danger-zone' },
]

test.describe('App accessibility audit', () => {
  for (const { path, name, marker } of pages) {
    test(`${name}: ${path}`, async ({ page }) => {
      await page.route('**/api/me', r => r.fulfill(json(user)))
      await page.route('**/api/me/entitlements', r => r.fulfill(json(entitlements)))
      await page.route('**/api/profiles**', r => {
        if (r.request().url().endsWith('/profiles/p_1')) return r.fulfill(json(profile))
        return r.fulfill(json({ profiles: [profile], next_cursor: '' }))
      })
      await page.route('**/api/chat/sessions', r => r.fulfill(json([session])))
      await page.route('**/api/chat/sessions/chat_1', r => r.fulfill(json(session)))
      await page.route('**/api/chat/sessions/chat_1/messages', r => r.fulfill(json([message])))
      await page.route('**/api/chat/products', r => r.fulfill(json({ products: [product] })))
      await page.route('**/api/chat/orders', r => r.fulfill(json({ orders: [order] })))
      await page.route('**/api/me/sessions', r => r.fulfill(json([])))
      await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))

      await page.goto(path)
      await page.waitForSelector(marker, { timeout: 10000 })
      await page.waitForTimeout(200)

      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .analyze()
      const criticalSerious = results.violations.filter(v => v.impact === 'critical' || v.impact === 'serious')
      expect(criticalSerious, `a11y violations on ${path}:\n` + criticalSerious
        .map(v => `${v.id}: ${v.nodes.length} nodes [${v.nodes.map(n => n.target.join(' ')).join('; ')}]`)
        .join('\n')).toEqual([])
    })
  }
})