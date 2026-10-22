// natal 页工具岛：SSR hydration + 共享排盘组件（BaziPanel/ZiweiPanel）渲染。
// mock 形状对齐引擎契约（完整样本见 engine-contract.spec）。
import { test, expect } from '../fixtures';
import { fillBirthFields } from '../helpers';
import { mockCityCoords, mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';
import { baziChart, baziFull, baziLiunian } from '../engine-fixtures';

const PALACES = ['命宫', '兄弟', '夫妻', '子女', '财帛', '疾厄', '迁移', '仆役', '官禄', '田宅', '福德', '父母'];
const ZHI = ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥'];
const gongWei = PALACES.map((name, i) => ({
  name,
  gan: '甲',
  zhi: ZHI[i],
  is_shen_gong: i === 0,
  xing_yao: i === 0
    ? [
        { name: '紫微', is_zhu_xing: true, liang_du: '庙', si_hua: '禄' },
        { name: '破军', is_zhu_xing: true, liang_du: '旺' },
        { name: '天魁', is_zhu_xing: false },
      ]
    : [],
  chang_sheng: '长生',
  za_yao: [],
  ages: [6, 18, 30],
}));

const ziweiChart = {
  gong_wei: gongWei.map(g => ({ name: g.name, gan: g.gan, zhi: g.zhi, is_shen_gong: g.is_shen_gong, xing_yao: g.xing_yao })),
  ju_shu: '火六局',
  ming_gong: '命宫',
  ming_zhu: '巨门',
  shen_zhu: '文昌',
  ziwei_pos: '命宫',
  shen_gong: '命宫',
  nian_gan: '庚',
  nian_zhi: '午',
  shi_zhi: '卯',
  si_hua: { 太阳: '禄', 武曲: '权', 太阴: '科', 天同: '忌' },
};

const ziweiFull = { ...ziweiChart, gong_wei: gongWei };

const ziweiDaXian = PALACES.map((name, i) => ({
  name, gong: name,
  qi_sui: 6 + i * 10, zhi_sui: 15 + i * 10,
  start_year: 2005 + i * 10, end_year: 2014 + i * 10,
}));

async function declareEngine(page: Parameters<typeof mockEngineTool>[0]) {
  // 出生地匹配（city_coords candidates 模式）→ 自动填经度。
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  // 真太阳时链路（无出生地时仍取农历，供紫微消费）。
  await mockEngineTool(page, 'bazi_chart', baziChart);
  await mockEngineTool(page, 'bazi_fullchart', baziFull);
  await mockEngineTool(page, 'bazi_liunian', baziLiunian);
  await mockEngineTool(page, 'ziwei_chart', ziweiChart);
  await mockEngineTool(page, 'ziwei_fullchart', ziweiFull);
  await mockEngineTool(page, 'ziwei_daxian', ziweiDaXian);
}

test('工具岛：八字盘 + 紫微盘按共享组件渲染', async ({ page }) => {
  await mockPageview(page);
  await declareEngine(page);
  await page.goto('/zh-Hans/natal/');

  const tool = page.locator('#natal-tool');
  await expect(tool.locator('button[type=submit]')).toHaveText('立即排盘');

  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'female' });
  await tool.locator('input[name="birth-place"]').fill('上海');
  await tool.locator('input[name="birth-place"]').blur();
  await expect(tool.locator('input[name="birth-longitude"]')).toHaveValue('121.4726');
  await tool.locator('button[type=submit]').click();

  // 八字盘（共享 BaziPanel）：四柱 + 大运 + 用神
  await expect(tool.locator('.bazi-panel .pillar:not(.liunian)')).toHaveCount(4);
  await expect(tool.locator('.bazi-panel .pillar').nth(0)).toContainText('庚午');
  await expect(tool.locator('.bazi-panel .dayun .dy')).toHaveCount(3);
  await expect(tool.locator('.bazi-panel .ys')).toContainText('身旺');

  // 流年列（bazi_liunian）：年份输入 + 五柱 + 流年详览
  await expect(tool.locator('input[name="natal-liunian-year"]')).toBeVisible();
  await expect(tool.locator('.bazi-panel .pillar.liunian')).toContainText('丙午');
  await expect(tool.locator('.bazi-panel .liunian-detail')).toContainText('天乙贵人');

  // 紫微盘（共享 ZiweiPanel）：12 宫 + 中心信息
  await expect(tool.locator('.ziwei-panel .zw-palace')).toHaveCount(12);
  await expect(tool.locator('.ziwei-panel .zw-center')).toContainText('火六局');

  // CTA 与隐私
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=natal/);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'bazi_chart', 'engine_unavailable: boom');
  await page.goto('/en/natal/');

  const tool = page.locator('#natal-tool');
  await fillBirthFields(page, tool, { date: '1990-05-12', time: '08:30', gender: 'male' });
  await mockCityCoords(page, { name: '上海市', region: '上海市', longitude: 121.4726, latitude: 31.2304 });
  await tool.locator('input[name="birth-place"]').fill('上海');
  await tool.locator('input[name="birth-place"]').blur();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
