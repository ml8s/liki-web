import { expect, type Page } from '@playwright/test'
import { mcpOk, mcpToolError } from '../mocks'
import type { JsonRecord } from './types'
import { mockFlowWorkspace } from './intent-flow'


export const castingQuestion = '这次换工作是否顺利？'
export const castingInstantPattern = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/

export const deterministicCasting = {
  mode: 'coins',
  order: 'bottom_up',
  yaos: [9, 7, 8, 7, 8, 7],
  dong_yao: [1],
  casting_id: 'cast_flow',
}

export const deterministicChart = {
  name: '火泽睽',
  ben_gua: '火泽睽',
  bian_gua: '火水未济',
  gong: '艮',
  gong_wuxing: '土',
  ri_chen_gan: '甲',
  ri_chen_zhi: '子',
  yue_jian_zhi: '子',
  yue_jian_gan: '甲',
  yong_shen: { name: '官鬼' },
  gua_ci: { name: '睽', gua_ci: '', yao_ci: ['', '', '', '', '', ''] },
  wang_shuai: ['', '', '', '', '', ''],
  ri_chen_relations: [{ relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }, { relations: [] }],
  xun_kong: ['戌', '亥'],
  lines: [{}, {}, {}, {}, {}, {}],
  bian_yao: [{}, {}, {}, {}, {}, {}],
  casting: deterministicCasting,
}

export interface EngineToolCall {
  tool: string
  args: JsonRecord
}

/** 用简体中文固定界面文案；query locale 优先于 mocked user preferences。 */
export async function mockCastingWorkspace(page: Page) {
  await mockFlowWorkspace(page)
}

export async function mockDivinationEngine(
  page: Page,
  options: { chartError?: string } = {},
): Promise<EngineToolCall[]> {
  const calls: EngineToolCall[] = []

  await page.route('**/engine/mcp', async route => {
    const request = route.request()
    const body = request.postDataJSON() as {
      params?: { name?: string; arguments?: JsonRecord }
    }
    const headerTool = await request.headerValue('mcp-name')
    const tool = body.params?.name ?? headerTool ?? ''
    calls.push({ tool, args: body.params?.arguments ?? {} })

    if (tool === 'liuyao_qigua') {
      await route.fulfill({ json: mcpOk({ casting: deterministicCasting }) })
      return
    }
    if (tool === 'liuyao_chart') {
      if (options.chartError) {
        await route.fulfill({ json: mcpToolError(options.chartError) })
        return
      }
      await route.fulfill({ json: mcpOk(deterministicChart) })
      return
    }
    await route.fulfill({ status: 500, json: { error: { code: -32000, message: 'unexpected engine tool' } } })
  })

  return calls
}

export function castingForm(page: Page) {
  return page.locator('.flow-main form').filter({
    has: page.locator('textarea[name="divination-question"]'),
  })
}

export async function openDivination(page: Page) {
  const url = new URL(page.url())
  url.searchParams.set('group', 'divination')
  url.searchParams.set('seed', 'divination')
  await page.goto(url.toString())
  const form = castingForm(page)
  await expect(form.locator('textarea[name="divination-question"]')).toBeVisible()
  await expect(form.getByRole('button', { name: '摇卦' })).toBeVisible()
  return form
}

export async function fillCastingForm(page: Page, question = castingQuestion) {
  const form = castingForm(page)
  await form.locator('textarea[name="divination-question"]').fill(question)
  await form.getByRole('button', { name: '手动填写经度' }).click()
  await form.locator('input[name="casting-longitude"]').fill('121.4726')
  await form.locator('input[name="casting-timezone"]').fill('Asia/Shanghai')
  await expect(form.locator('.el-form-item__error')).toHaveCount(0)
  return form
}
