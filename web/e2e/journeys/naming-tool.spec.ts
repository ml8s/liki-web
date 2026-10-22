// naming 页工具岛：出生信息 → bazi_chart（engine）+ bazi_yongshen（counsel）→ 用神方向。
import { test, expect } from '../fixtures';
import { fillBirthFields } from '../helpers';
import type { Page } from '@playwright/test';
import { mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart } from '../engine-fixtures';

const yongShen = {
  selected: { yong: '木', xi: ['水', '木'], ji: ['土'], source: 'tiao_hou', confidence: 'high' },
  status: 'ok',
  rule_id: 'YS-TH-1',
};

const yongShenReview = {
  selected: { yong: null, xi: [], ji: [], source: 'tiao_hou', confidence: 'low', known_fields: [] },
  status: 'needs_review',
  conflicts: [{ code: 'tiaohou_fuyi_conflict', message: '调候用神与扶抑忌神直接相反，不硬判' }],
};

async function mockNamingEngine(page: Page) {
  const calls: Array<{ tool: string; arguments: Record<string, unknown> }> = []
  await page.route(/\/engine\/mcp$/, async route => {
    const body = JSON.parse(route.request().postData() ?? '{}') as {
      params?: { name?: string; arguments?: Record<string, unknown> }
    }
    const tool = body.params?.name ?? ''
    const args = body.params?.arguments ?? {}
    calls.push({ tool, arguments: args })
    const data = tool === 'city_coords'
      ? { candidates: [{ name: '上海市', region: '上海市', country: '中国', level: 'city', longitude: 121.4726, latitude: 31.2304, timezone: 'Asia/Shanghai' }], ambiguous: false }
        : tool === 'bazi_chart'
          ? baziChart
          : {}
    await route.fulfill({ json: { jsonrpc: '2.0', id: 1, result: { content: [{ type: 'text', text: JSON.stringify(data) }] } } })
  })
  return calls
}

test('工具岛：用神/喜忌按共享组件渲染，低置信度显示复核提示', async ({ page }) => {
  await mockPageview(page);
  const engineCalls = await mockNamingEngine(page);
  await mockEngineTool(page, 'bazi_yongshen', yongShenReview);
  await page.goto('/zh-Hans/naming/');

  const tool = page.locator('#naming-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name=naming-birth-place]').fill('上海');
  await tool.locator('input[name=naming-birth-place]').blur();
  await expect(tool.locator('input[name=naming-birth-longitude]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.ys-value')).toHaveText('—');
  await expect(tool.locator('.ys-review')).toContainText('复核');
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=naming/);
  expect(engineCalls.find(call => call.tool === 'bazi_chart')?.arguments).toEqual({
    instant: '1990-05-11T23:30:00Z', timezone: 'Asia/Shanghai',
    longitude: 121.4726,
    gender: 'female',
  })
});

test('工具岛：正常盘渲染喜/忌五行', async ({ page }) => {
  await mockPageview(page);
  const engineCalls = await mockNamingEngine(page);
  await mockEngineTool(page, 'bazi_yongshen', yongShen);
  await page.goto('/en/naming/');

  const tool = page.locator('#naming-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name=naming-birth-place]').fill('上海');
  await tool.locator('input[name=naming-birth-place]').blur();
  await expect(tool.locator('input[name=naming-birth-longitude]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.ys-value')).toHaveText('木');
  await expect(tool.locator('.ys-chip.good')).toHaveCount(2);
  await expect(tool.locator('.ys-chip.bad')).toHaveCount(1);
  await expect(tool.locator('.ys-review')).toHaveCount(0);
  expect(engineCalls.find(call => call.tool === 'bazi_chart')?.arguments).toEqual({
    instant: '1990-05-11T23:30:00Z', timezone: 'Asia/Shanghai',
    longitude: 121.4726,
    gender: 'female',
  })
});

test('工具岛：counsel 失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockNamingEngine(page);
  await mockMcpToolError(page, '**/counsel/mcp', 'bazi_yongshen', 'engine_unavailable: boom');
  await page.goto('/en/naming/');

  const tool = page.locator('#naming-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name=naming-birth-place]').fill('上海');
  await tool.locator('input[name=naming-birth-place]').blur();
  await expect(tool.locator('input[name=naming-birth-longitude]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
