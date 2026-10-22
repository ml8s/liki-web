// fengshui 页工具岛：出生年 + 性别 → 命卦/方位/流年飞星 + 玄空飞星盘 + 八宅布局（门/主/灶）。
import { test, expect } from '../fixtures';
import { mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';

const bazhai = {
  ming_gua: { gua: { index: 8, name: '艮', wuxing: '土', yin_yang: '阳' }, group: '西四命' },
  ba_zhai_dirs: {
    sheng_qi: '西南', tian_yi: '西北', yan_nian: '西', fu_wei: '东北',
    huo_hai: '南', wu_gui: '北', liu_sha: '东', jue_ming: '东南',
  },
};

const xuankongChart = {
  yun: { year: 2026, yuan: '下元', yun_number: 9, yun_name: '九运', start_year: 2024, end_year: 2043 },
  gong_wei: [
    { gong_num: 1, yun_xing: { number: 5, name: '五黄廉贞', wuxing: '土' }, shan_xing: { number: 9, name: '九紫右弼', wuxing: '火' }, xiang_xing: { number: 9, name: '九紫右弼', wuxing: '火' } },
    { gong_num: 2, yun_xing: { number: 6, name: '六白武曲', wuxing: '金' }, shan_xing: { number: 8, name: '八白左辅', wuxing: '土' }, xiang_xing: { number: 1, name: '一白贪狼', wuxing: '水' } },
    { gong_num: 3, yun_xing: { number: 7, name: '七赤破军', wuxing: '金' }, shan_xing: { number: 7, name: '七赤破军', wuxing: '金' }, xiang_xing: { number: 2, name: '二黑巨门', wuxing: '土' } },
    { gong_num: 4, yun_xing: { number: 8, name: '八白左辅', wuxing: '土' }, shan_xing: { number: 6, name: '六白武曲', wuxing: '金' }, xiang_xing: { number: 3, name: '三碧禄存', wuxing: '木' } },
    { gong_num: 5, yun_xing: { number: 9, name: '九紫右弼', wuxing: '火' }, shan_xing: { number: 5, name: '五黄廉贞', wuxing: '土' }, xiang_xing: { number: 4, name: '四绿文曲', wuxing: '木' } },
    { gong_num: 6, yun_xing: { number: 1, name: '一白贪狼', wuxing: '水' }, shan_xing: { number: 4, name: '四绿文曲', wuxing: '木' }, xiang_xing: { number: 5, name: '五黄廉贞', wuxing: '土' } },
    { gong_num: 7, yun_xing: { number: 2, name: '二黑巨门', wuxing: '土' }, shan_xing: { number: 3, name: '三碧禄存', wuxing: '木' }, xiang_xing: { number: 6, name: '六白武曲', wuxing: '金' } },
    { gong_num: 8, yun_xing: { number: 3, name: '三碧禄存', wuxing: '木' }, shan_xing: { number: 2, name: '二黑巨门', wuxing: '土' }, xiang_xing: { number: 7, name: '七赤破军', wuxing: '金' } },
    { gong_num: 9, yun_xing: { number: 4, name: '四绿文曲', wuxing: '木' }, shan_xing: { number: 1, name: '一白贪狼', wuxing: '水' }, xiang_xing: { number: 8, name: '八白左辅', wuxing: '土' } },
  ],
  four_situation: { name: '双星会坐', sit_mountain_timely: true, face_facing_timely: false, sit_facing_timely: true, face_mountain_timely: false },
  fu_yin_layers: [],
  fan_yin_layers: ['山星反吟'],
  xing_jia_hui: [
    { shan_num: 7, xiang_num: 2, name: '七二同道', meaning: '二七同宫，传统论火灾风险；成灾与否须核元运、生旺与峦头' },
    { shan_num: 2, xiang_num: 7, name: '二七同道', meaning: '二七同宫，传统论火灾风险；成灾与否须核元运、生旺与峦头' },
  ],
  zuo_shan: 0,
  zuo_shan_name: '子',
  xiang_shan: 12,
  xiang_shan_name: '午',
  shou_shan_chu_sha: { zheng_shen: 9, ling_shen: 1, shou_shan: true, chu_sha: false, assessment: '收山得宜（坐宫山星当令）；拨水入零堂未得，向首零神不见水' },
};

const bazhaiLayout = {
  group: '西四命',
  ming_gua: '艮',
  door: { direction: '西南', gua_name: '坤', wuxing: '土', youxing: '生气', rating: '大吉', group: '西四卦', match: '吉' },
  master: { direction: '西', gua_name: '兑', wuxing: '金', youxing: '延年', rating: '吉', group: '西四卦', match: '吉' },
  stove: { direction: '西北', gua_name: '乾', wuxing: '金', youxing: '天医', rating: '吉', group: '西四卦', match: '吉' },
};

test('工具岛：命卦与八宅方位按共享组件渲染', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'bazhai_chart', bazhai);
  await mockEngineTool(page, 'xuankong_chart', xuankongChart);
  await mockEngineTool(page, 'bazhai_layout', bazhaiLayout);
  await page.goto('/zh-Hans/fengshui/');

  const tool = page.locator('#fengshui-tool');
  await tool.getByRole('spinbutton', { name: '出生年份' }).fill('1990');
  const genderInput = tool.getByRole('combobox', { name: '性别' });
  await genderInput.click({ force: true });
  await page.locator(`[id="${await genderInput.getAttribute('aria-controls')}"]`)
    .getByRole('option', { name: '女', exact: true })
    .click();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.mingua')).toHaveText('艮');
  await expect(tool.locator('.dirs .dir')).toHaveCount(8);
  await expect(tool.locator('.dirs .dir.good')).toHaveCount(4);
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=fengshui/);

  // 玄空飞星盘（xuankong_chart）：标准九宫 + 坐/向角标 + 补充判断
  await expect(tool.locator('.chart-grid .xk-cell')).toHaveCount(9);
  await expect(tool.locator('.xk-head')).toContainText('九运');
  await expect(tool.locator('.xk-cell.sit')).toContainText('坐');
  await expect(tool.locator('.xk-cell.face')).toContainText('向');
  await expect(tool.locator('.xuankong-panel')).toContainText('双星会坐');
  await expect(tool.locator('.chart-chip', { hasText: '山星反吟' })).toHaveCount(1);

  // 八宅布局（bazhai_layout）：方位图 + 门/主/灶三要判断
  await expect(tool.locator('.chart-grid .bz-cell')).toHaveCount(9);
  const pickGua = async (label: string, gua: string) => {
    const input = tool.getByRole('combobox', { name: label });
    await input.click({ force: true });
    const listbox = page.locator(`[id="${await input.getAttribute('aria-controls')}"]`);
    await listbox
      .getByRole('option', { name: `${label} · ${gua}` })
      .click();
    await expect(listbox).toBeHidden();
  };
  await pickGua('门', '坤');
  await pickGua('主卧', '兑');
  await pickGua('灶', '乾');
  await expect(tool.locator('.bz-verdict')).toBeVisible();
  await expect(tool.locator('.bz-table tbody tr')).toHaveCount(3);
  await expect(tool.locator('.bz-table')).toContainText('生气');
  await expect(tool.locator('.bz-table')).toContainText('大吉');
  await expect(tool.locator('.bz-badge')).toContainText(['门', '主卧', '灶']);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'bazhai_chart', 'engine_unavailable: boom');
  await page.goto('/en/fengshui/');

  const tool = page.locator('#fengshui-tool');
  await tool.getByRole('spinbutton', { name: 'Birth year' }).fill('1990');
  const genderInput = tool.getByRole('combobox', { name: 'Gender' });
  await genderInput.click({ force: true });
  await page.locator(`[id="${await genderInput.getAttribute('aria-controls')}"]`)
    .getByRole('option', { name: 'Male', exact: true })
    .click();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
