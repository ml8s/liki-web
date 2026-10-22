// hepan 页工具岛：双方出生信息 → 共享 HepanPanel 合盘盘面渲染。
// mock 形状对齐引擎契约（完整样本见 engine-contract.spec 的 bazi_bond 用例）。
import { test, expect } from '../fixtures';
import { mockCityCoords, mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart, bond } from '../engine-fixtures';
import { fillBirthFields } from '../helpers';

test('工具岛：合盘盘面按共享 HepanPanel 渲染', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'bazi_chart', baziChart);
  await mockEngineTool(page, 'bazi_bond', bond);
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/zh-Hans/hepan/');

  const tool = page.locator('#hepan-tool');
  await expect(tool.locator('fieldset')).toHaveCount(2);

  const personA = tool.locator('fieldset').nth(0);
  const personB = tool.locator('fieldset').nth(1);
  await fillBirthFields(page, personA, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await fillBirthFields(page, personB, { date: '1992-08-20', time: '10:15', gender: 'male' });
  await personA.locator('input[name="person-a-place"]').fill('上海');
  await personA.locator('input[name="person-a-place"]').blur();
  await expect(personA.locator('input[name="person-a-longitude"]')).toHaveValue('121.4726');
  await personB.locator('input[name="person-b-place"]').fill('上海');
  await personB.locator('input[name="person-b-place"]').blur();
  await expect(personB.locator('input[name="person-b-longitude"]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  // 两盘四柱对照（A/B 各一行，mock 两侧同盘）
  await expect(tool.locator('.pillars')).toHaveCount(2);
  await expect(tool.locator('.pillars').first().locator('.pillar .gl').first()).toContainText('庚午');
  await expect(tool.locator('.chart-row .person').first()).toHaveText('你');

  // 日主关系（a_zhu=b_zhu=ri 的 pair → dayMasterValue 模板）
  const dayMaster = tool.locator('.rel-sec', { hasText: '日主关系' });
  await expect(dayMaster.locator('.rel-text')).toContainText('庚生壬');
  await expect(dayMaster.locator('.rel-text')).toContainText('申申相冲');

  // 关键地支 / 纳音 / 神煞 / 十神互看
  await expect(tool.locator('.rel-chip').first()).toContainText('申申相冲');
  await expect(tool.locator('.rel-chip.nayin')).toContainText('剑锋金');
  await expect(tool.locator('.rel-chip.shensha')).toContainText('天乙贵人');
  const shiShen = tool.locator('.rel-sec', { hasText: '十神互看' });
  await expect(shiShen.locator('.rel-chip')).toContainText(['你年柱看对方：七杀', '对方年柱看你：偏印']);

  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=hepan/);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'bazi_chart', 'engine_unavailable: boom');
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/en/hepan/');

  const tool = page.locator('#hepan-tool');
  await fillBirthFields(page, tool.locator('fieldset').nth(0), { date: '1990-05-12', time: '08:30', gender: 'female' });
  await fillBirthFields(page, tool.locator('fieldset').nth(1), { date: '1992-08-20', time: '10:15', gender: 'male' });
  await tool.locator('fieldset').nth(0).locator('input[name="person-a-place"]').fill('Shanghai');
  await tool.locator('fieldset').nth(0).locator('input[name="person-a-place"]').blur();
  await tool.locator('fieldset').nth(1).locator('input[name="person-b-place"]').fill('Shanghai');
  await tool.locator('fieldset').nth(1).locator('input[name="person-b-place"]').blur();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
