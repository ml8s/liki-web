// liunian 页工具岛：出生信息 + 年份 → 共享 BaziPanel 流年盘（流年列 + 流年详览）。
// mock 形状对齐引擎契约（完整样本见 engine-contract.spec）。
import { test, expect } from '../fixtures';
import { fillBirthFields } from '../helpers';
import { mockCityCoords, mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart, baziFull, baziLiunian } from '../engine-fixtures';

test('工具岛：流年盘按共享 BaziPanel 渲染（流年列 + 流年详览）', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'bazi_chart', baziChart);
  await mockEngineTool(page, 'bazi_fullchart', baziFull);
  await mockEngineTool(page, 'bazi_liunian', baziLiunian);
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/zh-Hans/liunian/');

  const tool = page.locator('#liunian-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name="birth-place"]').fill('上海');
  await tool.locator('input[name="birth-place"]').blur();
  await expect(tool.locator('input[name="birth-longitude"]')).toHaveValue('121.4726');
  await tool.locator('input[name="liunian-year"]').fill('2026');
  await tool.locator('button[type=submit]').click();

  // 流年列（五柱：四柱 + 流年）与流年详览
  await expect(tool.locator('.bazi-panel .pillar:not(.liunian)')).toHaveCount(4);
  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText('丙午');
  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText('正官');
  await expect(tool.locator('.bazi-panel .liunian-detail')).toContainText('天乙贵人');

  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /scenario=liunian/);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'bazi_chart', 'engine_unavailable: boom');
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/en/liunian/');

  const tool = page.locator('#liunian-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'male' });
  await tool.locator('input[name="birth-place"]').fill('Shanghai');
  await tool.locator('input[name="birth-place"]').blur();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
