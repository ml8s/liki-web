// liunian 页工具岛：出生信息 + 年份 → 共享八字目录流年盘（四柱 + 流年详览）。
// mock 形状对齐引擎契约（完整样本见 engine-contract.spec）。
import { test, expect } from '../fixtures';
import { fillBirthFields } from '../helpers';
import { mockCityCoords, mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart, baziFull, baziLiunian } from '../engine-fixtures';

test('工具岛：流年盘按八字目录面板渲染（四柱 + 流年详览）', async ({ page }) => {
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
  await tool.locator('.region-manual-toggle').click();
  await expect(tool.locator('input[name="birth-longitude"]')).toHaveValue('121.4726');
  await tool.locator('input[name="liunian-year"]').fill('2026');
  await tool.locator('button[type=submit]').click();

  // 完整八字目录盘四柱，流年目录盘承接 bazi_liunian 详览。
  await expect(tool.locator('.bazi-panel .pillar:not(.liunian)')).toHaveCount(4);
  await expect(tool.locator('.engine-bazi-liunian-panel')).toBeVisible();
  await expect(tool.locator('.engine-bazi-liunian-panel .pillar.liunian')).toContainText('丙午');
  await expect(tool.locator('.engine-bazi-liunian-panel .pillar.liunian')).toContainText('正官');
  await expect(tool.locator('.engine-bazi-liunian-panel .pillar.liunian')).toContainText('天河水');
  await expect(tool.locator('.engine-bazi-liunian-panel .pillar.liunian')).toContainText('生数 0 · 克数 1');
  await expect(tool.locator('.engine-bazi-liunian-panel .liunian-detail')).toContainText('天乙贵人');

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
  await tool.locator('.region-manual-toggle').click();
  await expect(tool.locator('input[name="birth-longitude"]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
