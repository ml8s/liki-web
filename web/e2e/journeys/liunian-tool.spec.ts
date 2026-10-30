// liunian 页工具岛：出生信息 + 起始年份 → 十年概览 + 选中年份八字/紫微详情。
// mock 形状对齐引擎 count 契约：返回从 anchor 开始的连续数组。
import { test, expect } from '../fixtures';
import { fillBirthFields } from '../helpers';
import { mockCityCoords, mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart, baziFull, baziLiunian } from '../engine-fixtures';

const START_YEAR = 2027;
const DECADE_YEARS = Array.from({ length: 10 }, (_, index) => START_YEAR + index);
const ZHI = ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥'];
const PALACES = ['命宫', '兄弟', '夫妻', '子女', '财帛', '疾厄', '迁移', '仆役', '官禄', '田宅', '福德', '父母'];
const baziLiunianDecade = {
  items: DECADE_YEARS.map((year, index) => ({
    ...baziLiunian,
    year,
    nian_name: index % 2 === 0 ? '丙午' : '丁未',
  })),
};
const ziweiLiunianDecade = {
  items: DECADE_YEARS.map((year, index) => ({
    ming_gong: '午',
    ming_gong_name: index % 2 === 0 ? '命宫' : '财帛宫',
    zhi: '午',
    si_hua: { 太阳: '禄' },
    gong_wei: PALACES.map((name, i) => ({
      name,
      zhi: ZHI[(i + index) % 12],
      xing_yao: i === 0 ? ['紫微'] : [],
      is_liu_ming: i === 0,
    })),
  })),
};
const ziweiChart = {
  gong_wei: PALACES.map((name, i) => ({ name, zhi: ZHI[i], xing_yao: [] })),
  ju_shu: '火六局',
  ming_gong: '命宫',
};

test('工具岛：十年概览渲染，选中年份显示八字与紫微详情', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'bazi_chart', baziChart);
  await mockEngineTool(page, 'bazi_fullchart', baziFull);
  await mockEngineTool(page, 'bazi_liunian', baziLiunianDecade);
  await mockEngineTool(page, 'ziwei_chart', ziweiChart);
  await mockEngineTool(page, 'ziwei_fullchart', ziweiChart);
  await mockEngineTool(page, 'ziwei_liunian', ziweiLiunianDecade);
  await mockEngineTool(page, 'ziwei_daxian', { items: [] });
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/zh-Hans/liunian/');

  const tool = page.locator('#liunian-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name="birth-place"]').fill('上海');
  await tool.locator('input[name="birth-place"]').blur();
  await tool.locator('.region-manual-toggle').click();
  await tool.locator('input[name="birth-timezone"]').fill('Asia/Shanghai');
  await tool.locator('input[name="birth-longitude"]').fill('121.4726');
  await tool.locator('input[name="liunian-start-year"]').fill(String(START_YEAR));
  await tool.locator('button[type=submit]').click();

  const overview = tool.locator('.annual-decade__grid');
  await expect(overview).toBeVisible();
  await expect(overview.locator('.annual-year')).toHaveCount(10);
  await expect(overview.locator('.annual-year').first()).toContainText(String(START_YEAR));
  await expect(overview.locator('.annual-year').first()).toHaveClass(/is-selected/);

  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText(String(START_YEAR));
  await expect(tool.locator('.engine-ziwei-liunian-panel .zw-center')).toContainText('流年命宫：命宫');

  await overview.locator('.annual-year', { hasText: String(START_YEAR + 4) }).click();
  await expect(overview.locator('.annual-year', { hasText: String(START_YEAR + 4) }))
    .toHaveClass(/is-selected/);
  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText(String(START_YEAR + 4));
  await expect(tool.locator('.engine-ziwei-liunian-panel .zw-center'))
    .toContainText('流年命宫：命宫');

  await overview.locator('.annual-year', { hasText: String(START_YEAR + 5) }).click();
  await expect(overview.locator('.annual-year', { hasText: String(START_YEAR + 5) }))
    .toHaveClass(/is-selected/);
  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText('丁未');
  await expect(tool.locator('.engine-ziwei-liunian-panel .zw-center'))
    .toContainText('流年命宫：财帛宫');

  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /birth-profiles/);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'bazi_chart', 'engine_unavailable: boom');
  await mockEngineTool(page, 'ziwei_chart', ziweiChart);
  await mockEngineTool(page, 'ziwei_fullchart', ziweiChart);
  await mockEngineTool(page, 'ziwei_liunian', ziweiLiunianDecade);
  await mockEngineTool(page, 'ziwei_daxian', { items: [] });
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await page.goto('/en/liunian/');

  const tool = page.locator('#liunian-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'male' });
  await tool.locator('input[name="birth-place"]').fill('Shanghai');
  await tool.locator('input[name="birth-place"]').blur();
  await tool.locator('.region-manual-toggle').click();
  await tool.locator('input[name="birth-timezone"]').fill('Asia/Shanghai');
  await tool.locator('input[name="birth-longitude"]').fill('121.4726');
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.engine-result-boundary__state[role="alert"]').first()).toContainText('Chart processing failed. Check the input and try again.');
});
