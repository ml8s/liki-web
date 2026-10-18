import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })


const user = testUser()
const entitlements = { entitlements: [], allowed_skills: ['liki'] }
const profile = {
  id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', gender: 'female', relation: '家人',
  calendar: 'lunar', birth_date: '1958-闰5-12', birth_time: '06:00', birth_time_known: 'known',
  birth_place: '上海', note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}
const session = {
  id: 'chat_1', user_id: 'usr_test', profile_id: 'p_1', skill: 'liki', expert: 'natal', title: '妈妈 · 八字紫微',
  status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', last_message_at: null,
}
const message = {
  id: 'msg_1', session_id: 'chat_1', user_id: 'usr_test', role: 'user', kind: 'text', content: '你好', created_at: '2026-01-01T00:01:00Z',
}
const product = {
  id: 'liki-chat', name: 'Liki Chat', description: 'Unified metaphysics chat.',
  skills: ['liki'],
  display_currency: 'CNY', display_amount_minor: 100, purchased: true,
  purchase_options: [
    { billing_mode: 'one_time', price: { currency_code: 'CNY', minor_units: 100 } },
  ],
}
const order = {
  id: 'order_1', user_id: 'usr_test', email: 'user@example.com', product_id: 'liki-chat',
  amount_minor: 2990, currency: 'CNY', status: 'paid', created_at: '2026-01-01T00:00:00Z', paid_at: '2026-01-01T00:05:00Z',
}

const pages = [
  { path: '/app/login', name: 'login', marker: 'input[type="email"]' },
  { path: '/app/', name: 'dashboard', marker: '.services' },
  { path: '/app/birth-profiles', name: 'profiles', marker: '.profile-card' },
  { path: '/app/birth-profiles/new', name: 'profile-new', marker: '.birth-form' },
  { path: '/app/birth-profiles/p_1', name: 'profile-detail', marker: '.paipan-preview' },
  { path: '/app/synastry', name: 'synastry', marker: '.b-form' },
  { path: '/app/divination', name: 'divination', marker: '.el-input' },
  { path: '/app/fengshui', name: 'fengshui', marker: '.el-select' },
  { path: '/app/huangli', name: 'huangli', marker: '.el-select' },
  { path: '/app/chat', name: 'chat-list', marker: '.skill-grid' },
  { path: '/app/chat/chat_1', name: 'chat-session', marker: '.thread' },
  { path: '/app/orders', name: 'orders', marker: '.orders-table, .order-cards' },
  { path: '/app/account', name: 'account', marker: '.app-card--danger' },
  { path: '/app/partners', name: 'partner-center', marker: 'main' },
  { path: '/app/partners/kol', name: 'partner-kol', marker: 'main' },
  { path: '/app/partners/scout', name: 'partner-scout', marker: 'main' },
  { path: '/app/this-route-does-not-exist', name: 'not-found', marker: '.not-found' },
]

async function mockProductWorkspace(page: Page) {
  await mockWorkspace(page, {
    user,
    allowedSkills: entitlements.allowed_skills,
    profiles: [profile],
    sessions: [session],
    products: [product],
    orders: [order],
  })
  // 详情页按 id 取单个档案，覆盖工作台壳里的列表响应。
  await page.route('**/api/birth-profiles**', r => {
    if (r.request().url().endsWith('/birth-profiles/p_1')) return r.fulfill(json(profile))
    return r.fulfill(json({ profiles: [profile], next_cursor: '' }))
  })
  await page.route('**/api/chat/sessions/chat_1', r => r.fulfill(json(session)))
  await page.route('**/api/chat/sessions/chat_1/messages', r => r.fulfill(json([message])))
  await page.route('**/api/sales/subscriptions', r => r.fulfill(json([])))
  await page.route('**/api/sales/subscription-payments', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  // Partner 面：kol/scout 深状态 mock 从简，覆盖骨架的错误/空态即可。
  await page.route('**/api/partner-portal', r => r.fulfill(json({
    kol: { status: '', application: null, can_submit: true },
    scout: { status: '', application: null, can_submit: true },
  })))
  await page.route('**/api/partner-portal/kol/**', r => r.fulfill(json({})))
  await page.route('**/api/partner-portal/scout/**', r => r.fulfill(json({})))
}

async function audit(page: Page, path: string) {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze()
  const criticalSerious = results.violations.filter(v => v.impact === 'critical' || v.impact === 'serious')
  expect(criticalSerious, `a11y violations on ${path}:\n` + criticalSerious
    .map(v => `${v.id}: ${v.nodes.length} nodes [${v.nodes.map(n => n.target.join(' ')).join('; ')}]`)
    .join('\n')).toEqual([])
}

test.describe('App accessibility audit', () => {
  for (const { path, name, marker } of pages) {
    test(`${name}: ${path}`, async ({ page }) => {
      await mockProductWorkspace(page)

      await page.goto(path)
      await page.waitForSelector(marker, { timeout: 10000 })
      await page.waitForTimeout(200)

      await audit(page, path)
    })
  }
})

test.describe('App mobile accessibility smoke', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  for (const { path, name, marker } of pages.filter(page => [
    'dashboard', 'profiles', 'chat-list', 'chat-session', 'orders', 'account', 'partner-center',
  ].includes(page.name))) {
    test(`mobile ${name}: ${path}`, async ({ page }) => {
      await mockProductWorkspace(page)
      await page.goto(path)
      await page.waitForSelector(marker, { timeout: 10000 })
      await page.waitForTimeout(200)
      await audit(page, path)
    })
  }
})
