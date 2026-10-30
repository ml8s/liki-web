import type { Page, Route } from '@playwright/test'
import { expect, test } from '../fixtures'
import {
  castingForm,
  castingInstantPattern,
  castingQuestion,
  deterministicCasting,
  deterministicChart,
  fillCastingForm,
  mockCastingWorkspace,
  mockDivinationEngine,
  openDivination,
  type EngineToolCall,
  type JsonRecord,
} from '../helpers/domain-flow-casting'
import { flowSession } from '../helpers/domain-flow'
import { mockBFFChatMessage } from '../helpers/chat'
import { mcpOk, mcpToolError } from '../mocks'

async function seedDivinationHandoff(page: Page) {
  await page.addInitScript(context => {
    try {
      const key = 'liki:form-drafts'
      const raw = sessionStorage.getItem(key)
      const drafts = raw ? JSON.parse(raw) as Record<string, unknown> : {}
      drafts['handoff:consultation'] = { data: context, createdAt: Date.now() }
      sessionStorage.setItem(key, JSON.stringify(drafts))
    } catch {
      // sessionStorage unavailable: the test's navigation below will fail visibly.
    }
  }, {
    version: 1,
    scenario: 'divination',
    question: castingQuestion,
    casting: deterministicCasting,
    casting_instant: '2026-01-01T00:00:00Z',
    location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
  })
}

function mockSessionCreation(page: Page, sessions: JsonRecord[]) {
  return page.route('**/api/chat/sessions', async (route: Route) => {
    if (route.request().method() !== 'POST') {
      await route.fulfill({ json: { data: [] } })
      return
    }

    sessions.push(route.request().postDataJSON() as JsonRecord)
    await route.fulfill({ json: { data: flowSession('chat_casting', {
      profile_id: '',
      expert: 'divination',
      title: '问卦',
      context: {
        version: 1,
        scenario: 'divination',
        question: castingQuestion,
        casting: deterministicCasting,
        casting_instant: '2026-01-01T00:00:00Z',
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
    }) } })
  })
}

test.describe('divination casting flow', () => {
  test('exposes the four domains and the shared casting form without a task hierarchy', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 840 })
    await mockCastingWorkspace(page)

    await page.goto('/app/?locale=zh-Hans')
    await openDivination(page)

    const form = castingForm(page)
    await expect(form.getByText('起卦方式')).toBeVisible()
    await expect(form.getByText('系统自动起卦')).toBeVisible()
    await expect(form.getByRole('button', { name: '使用当前定位' })).toBeVisible()

    const legacyTaskLinks = page.locator('a[href="/app/divination"], a[href="/app/hepan"], a[href="/app/fengshui"], a[href="/app/huangli"], a[href="/app/natal"], a[href="/app/liunian"], a[href="/app/naming"]')
    await expect(legacyTaskLinks).toHaveCount(0)
  })

  test('restores a marketing casting receipt without drawing a new hexagram', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await mockCastingWorkspace(page)
    const calls = await mockDivinationEngine(page)
    await seedDivinationHandoff(page)

    await page.goto('/app/?service=divination&locale=zh-Hans')
    await expect(page.locator('.flow-domain-strip [role="tab"]', { hasText: '问卦' }))
      .toHaveAttribute('aria-selected', 'true')
    await expect(page.locator('.artifact-capsule')).toBeVisible()

    const tools = calls.map(call => call.tool)
    expect(tools).toContain('liuyao_chart')
    expect(tools).not.toContain('liuyao_qigua')
    expect(calls.find(call => call.tool === 'liuyao_chart')?.args.casting).toMatchObject({
      casting_id: deterministicCasting.casting_id,
    })
  })

  test('retries the folded casting chart without drawing a new hexagram', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 })
    await mockCastingWorkspace(page)
    await seedDivinationHandoff(page)

    let chartAttempts = 0
    const tools: string[] = []
    await page.route('**/engine/mcp', async route => {
      const body = route.request().postDataJSON() as { params?: { name?: string } }
      const tool = body.params?.name ?? ''
      tools.push(tool)

      if (tool === 'liuyao_qigua') {
        await route.fulfill({ json: mcpOk({ casting: deterministicCasting }) })
        return
      }
      if (tool === 'liuyao_chart') {
        chartAttempts += 1
        if (chartAttempts === 1) {
          await route.fulfill({ json: mcpToolError('engine_unavailable: boom') })
        } else {
          await route.fulfill({ json: mcpOk(deterministicChart) })
        }
        return
      }
      await route.fulfill({ status: 500, json: { error: { code: -32000, message: 'unexpected engine tool' } } })
    })

    await page.goto('/app/?service=divination&locale=zh-Hans')
    await expect(page.locator('.flow-error')).toBeVisible()
    await page.getByRole('button', { name: '重试' }).click()

    await expect(page.locator('.artifact-capsule')).toBeVisible()
    expect(chartAttempts).toBe(2)
    expect(tools).toContain('liuyao_chart')
    expect(tools).not.toContain('liuyao_qigua')
  })

  test('requires question and location before calling a casting engine', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 840 })
    await mockCastingWorkspace(page)
    const engineCalls = await mockDivinationEngine(page)

    await page.goto('/app/?locale=zh-Hans')
    const form = await openDivination(page)
    await form.getByRole('button', { name: '摇卦' }).click()

    const errors = form.locator('.el-form-item__error')
    await expect(errors).toHaveText(['请先输入你想问的问题', '请填写起卦地点'])
    expect(engineCalls).toEqual([])

    await form.locator('textarea[name="divination-question"]').fill(castingQuestion)
    await form.getByRole('button', { name: '摇卦' }).click()
    await expect(errors).toHaveText(['请填写起卦地点'])
    expect(engineCalls).toEqual([])
  })

  test('does not fabricate an artifact when chart assembly fails', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 840 })
    await mockCastingWorkspace(page)
    const engineCalls = await mockDivinationEngine(page, {
      chartError: 'engine_unavailable: deterministic chart failure',
    })

    await page.goto('/app/?locale=zh-Hans')
    await openDivination(page)
    await fillCastingForm(page)
    await castingForm(page).getByRole('button', { name: '摇卦' }).click()

    await expect(page.locator('.flow-main [role="alert"]').first()).toBeVisible()
    await expect(page.locator('.artifact-capsule')).toHaveCount(0)
    expect(engineCalls.map(call => call.tool)).toEqual(['liuyao_qigua', 'liuyao_chart'])
  })

  test('casts, renders the artifact, and puts the casting receipt on the divination session', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 1200 })
    await mockCastingWorkspace(page)

    const engineCalls = await mockDivinationEngine(page)
    const sessionCreations: JsonRecord[] = []
    await mockSessionCreation(page, sessionCreations)
    await page.route('**/api/birth-profiles/**', route =>
      route.fulfill({ json: { data: { profiles: [], next_cursor: '' } } }))

    const session = flowSession('chat_casting', {
      profile_id: '',
      expert: 'divination',
      title: '问卦',
      context: {
        version: 1,
        scenario: 'divination',
        question: castingQuestion,
        casting: deterministicCasting,
        casting_instant: '2026-01-01T00:00:00Z',
        location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
    })
    const runtimeRequests = await mockBFFChatMessage(page, {
      session,
      answer: '动爻在初爻，事宜缓办。',
    })

    await page.goto('/app/?locale=zh-Hans')
    await openDivination(page)
    await fillCastingForm(page)
    await castingForm(page).getByRole('button', { name: '摇卦' }).click()

    await expect(page.locator('.artifact-capsule')).toHaveCount(1)
    expect(engineCalls.map(call => call.tool)).toEqual(['liuyao_qigua', 'liuyao_chart'])
    expect(engineCalls[0]?.args).toEqual({ mode: 'auto' })
    expect(engineCalls[1]?.args).toMatchObject({
      longitude: 121.4726,
      standard_time: expect.any(String),
      casting: { casting_id: 'cast_flow' },
    })
    expect(String(engineCalls[1]?.args.standard_time)).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+08:00$/)
    expect(engineCalls[1]?.args).not.toHaveProperty('timezone')

    await page.locator('.artifact-capsule').click()
    const sheet = page.locator('.flow-sheet__body')
    await expect(sheet).toBeVisible()
    await expect(sheet.locator('.liuyao-chart-panel .gua-name')).toHaveText('火泽睽')
    await expect(sheet.locator('.liuyao-chart-panel .gua-sub')).toContainText('火水未济')
    await expect(sheet.locator('.liuyao-chart-panel .lines-table').first().locator('.line-row:not(.line-header)')).toHaveCount(6)
    await expect(sheet.locator('.liuyao-chart-panel .line-row.dong')).toHaveCount(1)
    await page.keyboard.press('Escape')
    await expect(sheet).toBeHidden()

    const followUp = '这卦对下个月的面试意味着什么？'
    await page.getByRole('textbox', { name: '直接提出你的问题' }).fill(followUp)
    await page.getByRole('button', { name: '发送' }).click()

    await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('动爻在初爻，事宜缓办。')
    expect(sessionCreations).toHaveLength(1)
    const creation = sessionCreations[0] as {
      profile_id: string
      skill: string
      expert: string
      scenario: string
      context: {
        question?: string
        casting?: { casting_id?: string }
        casting_instant?: string
        location?: { longitude?: number; timezone?: string }
      }
    }
    expect(creation).toMatchObject({
      profile_id: '',
      skill: 'liki',
      expert: 'divination',
      scenario: 'divination',
    })
    expect(creation.context.question).toBe(castingQuestion)
    expect(creation.context.casting?.casting_id).toBe('cast_flow')
    expect(creation.context.casting_instant).toMatch(castingInstantPattern)
    expect(creation.context.location).toEqual({ longitude: 121.4726, timezone: 'Asia/Shanghai' })

    expect(runtimeRequests).toHaveLength(1)
    expect(runtimeRequests[0]?.url).toContain('/api/chat/sessions/chat_casting/ag-ui')
    // The browser sends only protocol-safe input. The BFF revalidates persisted
    // consultation context server-side, so raw casting facts must not leak here.
    const runtimeBodies = runtimeRequests.map(request => request.body)
    const forwardedProps = runtimeBodies[0]?.forwardedProps as { locale?: string }
    expect(forwardedProps.locale).toBe('zh-Hans')
    expect(runtimeBodies[0]).not.toHaveProperty('consultation_context')
  })

  test('uses a desktop sidecar for the deterministic divination artifact', async ({ page }) => {
    await mockCastingWorkspace(page)
    const engineCalls = await mockDivinationEngine(page)
    await page.route('**/api/chat/sessions', route =>
      route.fulfill({ json: { data: [] } }))

    await page.goto('/app/?locale=zh-Hans')
    await openDivination(page)
    await fillCastingForm(page)
    await castingForm(page).getByRole('button', { name: '摇卦' }).click()

    await page.locator('.artifact-capsule').click()
    const sidecar = page.locator('.artifact-sidecar')
    await expect(sidecar).toBeVisible()
    await expect(sidecar.getByRole('heading', { name: '问卦' })).toBeVisible()
    await expect(sidecar.locator('.liuyao-chart-panel .gua-name')).toHaveText('火泽睽')
    await expect(sidecar.locator('.liuyao-chart-panel .lines-table').first().locator('.line-row:not(.line-header)')).toHaveCount(6)
    expect(engineCalls.map((call: EngineToolCall) => call.tool)).toEqual([
      'liuyao_qigua',
      'liuyao_chart',
    ])
  })

  test('redirects the legacy app divination route into Consultation', async ({ page }) => {
    await mockCastingWorkspace(page)

    await page.goto('/app/divination?locale=zh-Hans')
    await expect(page).toHaveURL(/domain=divination&seed=divination/)
    await expect(page.locator('.flow-divination-form textarea[name="divination-question"]')).toBeVisible()
  })

  test('keeps the compatibility-owned Classic divination route reachable', async ({ page }) => {
    await mockCastingWorkspace(page)
    await page.route('**/api/records**', route =>
      route.fulfill({ json: { data: { records: [], next_cursor: '' } } }))

    await page.goto('/app/classic/divination?locale=zh-Hans')
    await expect(page.getByRole('heading', { name: '问卦', level: 1 })).toBeVisible()
    await expect(page.locator('textarea[name="divination-question"]')).toBeVisible()
    await expect(page.locator('form')).toContainText('起卦地点')
  })
})
