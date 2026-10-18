// 共享 API mock。
//
// 契约（见 fixtures.ts）：每个用例必须显式声明它依赖的接口，未声明的 /api 调用会被
// 拦下并判失败。这里是多个用例复用的那部分声明；用例特有的数据留在用例里。
//
// Playwright 的 page 路由是后注册优先，所以顺序是「先调这里的 mock，用例再
// page.route 覆盖个别接口」，不要用 `**/api/**` 之类的兜底——那会把契约一起兜掉。
import type { Page } from '@playwright/test'

// 后端统一信封 { data }；序列化与 content-type 交给 Playwright 的 fulfill({ json })。
export const json = (data: unknown) => ({ json: { data } })

export const noContent = { status: 204 }

export interface TestUser {
  id: string
  email: string
  status: string
  locale: string
}

export const testUser = (overrides: Partial<TestUser> = {}): TestUser => ({
  id: 'usr_test',
  email: 'user@example.com',
  status: 'active',
  locale: 'zh-Hant',
  ...overrides,
})

export const adminUser: TestUser = {
  id: 'user-admin',
  email: 'admin@example.com',
  status: 'active',
  locale: 'zh-Hans',
}

// 静态页首屏都会打一次即发即忘的 pageview 信标（BaseLayout 的 analytics 脚本）；
// 它不参与渲染，声明成 204 空响应。
export async function mockPageview(page: Page) {
  await page.route('**/api/analytics/pageview', route => route.fulfill(noContent))
}

// 后台壳：/api/me + /api/admin/session（能力决定能看到哪些后台视图）。
export async function mockAdmin(page: Page, capabilities: string[]) {
  await mockPageview(page)
  await page.route('**/api/me', route => route.fulfill(json(adminUser)))
  await page.route('**/api/admin/session', route => route.fulfill(json({
    email: adminUser.email,
    capabilities,
  })))
}

export interface WorkspaceOptions {
  user?: TestUser
  entitlements?: unknown[]
  allowedSkills?: string[]
  profiles?: unknown[]
  sessions?: unknown[]
  products?: unknown[]
  orders?: unknown[]
  records?: unknown[]
}

// 已登录工作台壳：登录态、能力、以及工作台首屏会拉取的列表（默认空集合）。
export async function mockWorkspace(page: Page, options: WorkspaceOptions = {}) {
  const user = options.user ?? testUser()
  await page.route('**/api/me', route => route.fulfill(json(user)))
  await page.route('**/api/me/entitlements', route => route.fulfill(json({
    entitlements: options.entitlements ?? [],
    allowed_skills: options.allowedSkills ?? [],
  })))
  await page.route('**/api/birth-profiles**', route => route.fulfill(json({
    profiles: options.profiles ?? [],
    next_cursor: '',
  })))
  await page.route('**/api/chat/sessions', route => route.fulfill(json(options.sessions ?? [])))
  await page.route('**/api/sales/products', route => route.fulfill(json({ products: options.products ?? [] })))
  await page.route('**/api/sales/orders', route => route.fulfill(json({ orders: options.orders ?? [] })))
  // 四术数服务页在载入时拉各自的历史记录；用例需要真实回放时自行覆盖此路由。
  await page.route('**/api/records**', route => route.fulfill(json({ records: options.records ?? [] })))
}
