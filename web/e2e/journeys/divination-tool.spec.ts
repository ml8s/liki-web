// divination 页工具岛：免费起卦 → casting receipt 渲染本卦/变卦与六爻爻线。
import { test, expect } from '../fixtures';
import { mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { mockCityCoords } from '../mocks';

// 初爻老阳（动）+ 少阳 + 少阴 + 少阳 + 少阴 + 少阳
// 本卦：下兑（110）上离（101）→ 火泽睽；变卦：下坎（010）上离 → 火水未济。
const casting = {
  mode: 'coins',
  order: 'bottom_up',
  yaos: [9, 7, 8, 7, 8, 7],
  dong_yao: [1],
  casting_id: 'cast_test_1',
};

const chart = {
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
  casting,
};

async function fillDivinationForm(page: Parameters<typeof mockPageview>[0]) {
  await page.locator('#divination-tool textarea').fill('这次换工作是否顺利？');
  await page.locator('#divination-tool input[name="casting-place"]').fill('上海');
  await page.locator('#divination-tool input[name="casting-place"]').blur();
  await page.locator('#divination-tool .region-manual-toggle').click();
  await page.locator('#divination-tool input[name="casting-timezone"]').fill('Asia/Shanghai');
  await page.locator('#divination-tool input[name="casting-longitude"]').fill('121.4726');
}

test('工具岛：起卦 receipt 渲染本卦/变卦与六爻', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'liuyao_qigua', { casting });
  await mockEngineTool(page, 'liuyao_chart', chart);
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  const qiguaArguments: Record<string, unknown>[] = [];
  page.on('request', request => {
    if (!/\/engine\/mcp$/.test(new URL(request.url()).pathname)) return;
    const body = JSON.parse(request.postData() ?? '{}') as { params?: { name?: string; arguments?: Record<string, unknown> } };
    if (body.params?.name === 'liuyao_qigua') qiguaArguments.push(body.params.arguments ?? {});
  });
  await page.goto('/zh-Hans/divination/');

  const tool = page.locator('#divination-tool');
  await fillDivinationForm(page);
  await tool.locator('button[type=submit]').click();

  const castingPanel = tool.locator('.liuyao-panel');
  await expect(castingPanel.locator('.gua-name')).toHaveText('火泽睽');
  await expect(castingPanel.locator('.gua-sub').first()).toContainText('火水未济');
  await expect(castingPanel.locator('.yao')).toHaveCount(6);
  await expect(castingPanel.locator('.yao-dong')).toHaveCount(1);
  await expect(tool.locator('.liuyao-chart-panel')).toBeVisible();
  await expect(tool.locator('.liuyao-chart-panel .gua-name').first()).toHaveText('火泽睽');
  await expect(tool.locator('.liuyao-chart-panel .lines-table').first()).toBeVisible();
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=divination/);
  expect(qiguaArguments).toEqual([{ mode: 'auto' }]);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'liuyao_qigua', 'engine_unavailable: boom');
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/en/divination/');

  const tool = page.locator('#divination-tool');
  await fillDivinationForm(page);
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
  await expect(tool.locator('.tool-error')).toHaveCount(1);
});

test('工具岛：完整问题遇引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'liuyao_qigua', 'engine_unavailable: boom');
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/en/divination/');

  const tool = page.locator('#divination-tool');
  await fillDivinationForm(page);
  await tool.locator('button[type=submit]').click();
  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
