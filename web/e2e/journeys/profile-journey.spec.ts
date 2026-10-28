// 建档主路径：登录 → 新建档案 → 详情排盘 → 列表可见。真实后端（profiles API +
// engine 排盘），登录走 DEV_MODE 验证码。CI 与本地 e2e 环境均注入 DEV_MODE。
import { test, expect } from '../fixtures'
import { fillBirthFields } from '../helpers'

// 真实后端（profiles API + engine 排盘）：显式退出 API 契约。
test.use({ liveBackend: true })

const email = `profile-journey-${Date.now()}@test.liki`

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

  // 新建档案。出生地失焦会调用 city_coords（candidates 模式）；CI 不启动 engine，
  // 这里只对城市查询给出确定性响应，其余引擎请求照走（详情排盘允许降级）。
  await page.route(/\/(engine|counsel)\/mcp$/, route => {
    const body = route.request().postDataJSON() as { params?: { name?: string } }
    if (body?.params?.name === 'city_coords') {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          result: {
            content: [{
              type: 'text',
              text: JSON.stringify({
                name: '杭州市', longitude: 120.1551, latitude: 30.2741, country: '浙江省',
                candidates: [{
                  name: '杭州市', region: '浙江省', country: '浙江省', level: 'city',
                  longitude: 120.1551, latitude: 30.2741, timezone: 'Asia/Shanghai',
                }],
              }),
            }],
          },
        }),
      })
    }
    return route.continue()
  })
  await page.goto('/app/birth-profiles/new')
  await page.getByLabel(/稱呼/).fill('旅程測試')
  // Element Plus renders native radios visually covered by styled spans;
  // click the visible control instead of the hidden original input.
  await fillBirthFields(page, page, {
    date: '1990-06-15',
    time: '08:30',
    gender: 'male',
    place: '杭州',
  })
  await page.locator('.el-checkbox').last().click()
  const saveResponsePromise = page.waitForResponse(r =>
    r.url().includes('/api/birth-profiles') && r.request().method() === 'POST'
  )
  await page.getByRole('button', { name: /儲存檔案/ }).click()
  const saveResponse = await saveResponsePromise
  if (!saveResponse.ok()) {
    const requestBody = await saveResponse.request().postDataJSON()
    throw new Error(`profile create failed: ${saveResponse.status()} ${await saveResponse.text()} body=${JSON.stringify(requestBody)}`)
  }
  await page.waitForURL(/\/app\/classic\/birth-profiles\/[a-f0-9-]+/, { timeout: 15000 })

  // 详情页：档案信息已持久化。排盘渲染依赖 engine 集成，由
  // engine-contract.spec（mock 形状）与 deploy assembly E2E（真实引擎）覆盖，
  // CI 的 e2e job 不启动 engine。
  await expect(page.getByText('旅程測試').first()).toBeVisible()
  await expect(page.getByText(/出生|上海/).first()).toBeVisible()

  // 列表页可见该档案。
  await page.goto('/app/birth-profiles')
  await expect(page.getByText('旅程測試').first()).toBeVisible()
})
