// divination 页工具岛：免费起卦 → 本卦/变卦推演与六爻爻线渲染。
import { test, expect } from '../fixtures';
import { mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';

// 初爻老阳（动）+ 少阳 + 少阴 + 少阳 + 少阴 + 少阳
// 本卦：下兑（110）上离（101）→ 火泽睽；变卦：下坎（010）上离 → 火水未济。
const casting = {
  mode: 'coins',
  order: 'bottom_up',
  yaos: [9, 7, 8, 7, 8, 7],
  dong_yao: [1],
  casting_id: 'cast_test_1',
};

test('工具岛：起卦渲染本卦/变卦与六爻', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'liuyao_qigua', { casting });
  const qiguaArguments: Record<string, unknown>[] = [];
  page.on('request', request => {
    if (!/\/engine\/mcp$/.test(new URL(request.url()).pathname)) return;
    const body = JSON.parse(request.postData() ?? '{}') as { params?: { name?: string; arguments?: Record<string, unknown> } };
    if (body.params?.name === 'liuyao_qigua') qiguaArguments.push(body.params.arguments ?? {});
  });
  await page.goto('/zh-Hans/divination/');

  const tool = page.locator('#divination-tool');
  await tool.locator('textarea').fill('这次换工作是否顺利？');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.gua-name')).toHaveText('火泽睽');
  await expect(tool.locator('.gua-sub')).toContainText('火水未济');
  await expect(tool.locator('.yao')).toHaveCount(6);
  await expect(tool.locator('.yao-dong')).toHaveCount(1);
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=divination/);
  expect(qiguaArguments).toEqual([{ mode: 'auto' }]);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'liuyao_qigua', 'engine_unavailable: boom');
  await page.goto('/en/divination/');

  const tool = page.locator('#divination-tool');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.el-form-item__error')).toHaveText('Enter the question you want to ask');
  await expect(tool.locator('.tool-error')).toHaveCount(0);
});

test('工具岛：完整问题遇引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'liuyao_qigua', 'engine_unavailable: boom');
  await page.goto('/en/divination/');

  const tool = page.locator('#divination-tool');
  await tool.locator('textarea').fill('Will this migration land safely?');
  await tool.locator('button[type=submit]').click();
  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
