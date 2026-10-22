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
  /** /api/me 的合作角色摘要：默认空数组 = 普通用户（无合作入口）。 */
  partner_roles: string[]
}

export const testUser = (overrides: Partial<TestUser> = {}): TestUser => ({
  id: 'usr_test',
  email: 'user@example.com',
  status: 'active',
  locale: 'zh-Hant',
  partner_roles: [],
  ...overrides,
})

export const adminUser: TestUser = {
  id: 'user-admin',
  email: 'admin@example.com',
  status: 'active',
  locale: 'zh-Hans',
  partner_roles: [],
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

/** MCP 直连的 JSON-RPC 成功信封（content[0].text 内为 JSON 字符串）。 */
export const mcpOk = (data: unknown) => ({
  jsonrpc: '2.0',
  id: 1,
  result: { content: [{ type: 'text', text: JSON.stringify(data) }] },
})

/** MCP 工具业务错误：HTTP 200 + isError（引擎/counsel 通用形态）。 */
export const mcpToolError = (message: string) => ({
  jsonrpc: '2.0',
  id: 1,
  result: { content: [{ type: 'text', text: message }], isError: true },
})

import { COUNSEL_TOOLS } from '../shared/api/engine'

/** 按请求体里的工具名声明单条 MCP 工具路由（同 URL 承载任意工具，用 fallback 串接）。 */
export async function mockMcpTool(page: Page, url: '**/engine/mcp' | '**/counsel/mcp', tool: string, data: unknown) {
  await page.route(url, async route => {
    const body = JSON.parse(route.request().postData() ?? '{}') as { params?: { name?: string } }
    if (body.params?.name !== tool) return route.fallback()
    return route.fulfill({ json: mcpOk(data) })
  })
}

/** 同 mockMcpTool，但返回 isError（引擎失败态用例）。 */
export async function mockMcpToolError(page: Page, url: '**/engine/mcp' | '**/counsel/mcp', tool: string, message: string) {
  await page.route(url, async route => {
    const body = JSON.parse(route.request().postData() ?? '{}') as { params?: { name?: string } }
    if (body.params?.name !== tool) return route.fallback()
    return route.fulfill({ json: mcpToolError(message) })
  })
}

/** city_coords candidates 模式：单一候选，表单自动填入经度。 */
export async function mockCityCoords(page: Page, options: {
  name: string; region: string; longitude: number; latitude: number; level?: string; timezone?: string
}) {
  await mockMcpTool(page, '**/engine/mcp', 'city_coords', {
    candidates: [{
      name: options.name,
      region: options.region,
      country: options.region,
      level: options.level ?? 'city',
      longitude: options.longitude,
      latitude: options.latitude,
      timezone: options.timezone ?? 'Asia/Shanghai',
    }],
  })
}

/** 声明单个引擎/counsel 工具（小用例使用；全量契约回归见 engine-contract.spec 的 routeEngine）。 */
export async function mockEngineTool(page: Page, tool: string, data: unknown) {
  const url = COUNSEL_TOOLS.has(tool) ? '**/counsel/mcp' : '**/engine/mcp'
  await mockMcpTool(page, url, tool, data)
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
