// huangli 页工具岛：事项 + 日期范围 → 共享择日组件（宜/平/忌分组）渲染。
import { test, expect } from '../fixtures';
import { mockEngineTool, mockMcpToolError, mockPageview } from '../mocks';

const huangliDays = [
  {
    date: '2026-06-06',
    ri_zhu: { gan: '辛', zhi: '酉', na_yin: '石榴木' },
    na_yin: '石榴木', wuxing: '金', jian_chu: '建', huangdao: { index: 1, name: '青龙', path: '黄道', sequence: 1 },
    xi_shen: '寅', cai_shen: '卯', fu_shen: '午', gan_ji: '甲', zhi_ji: '子',
    mansion: { index: 1, name: '角', dong_wu: '蛟', wuxing: '木', group: '东方', group_idx: 1 },
    jie_qi: '', jie_qi_days: 0, ren_yuan: '',
    event: 'wedding', event_label: '嫁娶', suitability: 'recommended', reason: '',
  },
  {
    date: '2026-06-07',
    ri_zhu: { gan: '壬', zhi: '戌', na_yin: '大海水' },
    na_yin: '大海水', wuxing: '水', jian_chu: '除', huangdao: { index: 2, name: '明堂', path: '黄道', sequence: 2 },
    xi_shen: '卯', cai_shen: '辰', fu_shen: '未', gan_ji: '乙', zhi_ji: '丑',
    mansion: { index: 2, name: '亢', dong_wu: '龙', wuxing: '金', group: '东方', group_idx: 1 },
    jie_qi: '', jie_qi_days: 0, ren_yuan: '',
    event: 'wedding', event_label: '嫁娶', suitability: 'possible', reason: '次吉',
  },
  {
    date: '2026-06-08',
    ri_zhu: { gan: '癸', zhi: '亥', na_yin: '大海水' },
    na_yin: '大海水', wuxing: '水', jian_chu: '满', huangdao: { index: 3, name: '天刑', path: '黑道', sequence: 3 },
    xi_shen: '辰', cai_shen: '巳', fu_shen: '申', gan_ji: '丙', zhi_ji: '寅',
    mansion: { index: 3, name: '氐', dong_wu: '貉', wuxing: '土', group: '东方', group_idx: 1 },
    jie_qi: '', jie_qi_days: 0, ren_yuan: '',
    event: 'wedding', event_label: '嫁娶', suitability: 'unsuitable', reason: '冲煞',
  },
];

test('工具岛：择日结果按共享组件分组渲染', async ({ page }) => {
  await mockPageview(page);
  await mockEngineTool(page, 'huangli_days', huangliDays);
  await page.goto('/zh-Hans/huangli/');

  const tool = page.locator('#huangli-tool');
  const startField = tool.getByRole('combobox', { name: /起始日期|開始日期|Start date/ });
  const endField = tool.getByRole('combobox', { name: /结束日期|結束日期|End date/ });
  await startField.fill('2026-06-06');
  await startField.blur();
  await endField.fill('2026-06-08');
  await endField.blur();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.day.good')).toHaveCount(1);
  await expect(tool.locator('.day.good')).toContainText('2026-06-06');
  await expect(tool.locator('.day.possible')).toHaveCount(1);
  await expect(tool.locator('.day.bad')).toHaveCount(1);
  await expect(tool.locator('.range-info')).toContainText('2026-06-06');
  await expect(tool.locator('.tool-cta')).toHaveAttribute('href', /service=huangli/);
});

test('工具岛：引擎失败时展示错误文案', async ({ page }) => {
  await mockPageview(page);
  await mockMcpToolError(page, '**/engine/mcp', 'huangli_days', 'engine_unavailable: boom');
  await page.goto('/en/huangli/');

  const tool = page.locator('#huangli-tool');
  const startField = tool.getByRole('combobox', { name: /起始日期|開始日期|Start date/ });
  const endField = tool.getByRole('combobox', { name: /结束日期|結束日期|End date/ });
  await startField.fill('2026-06-06');
  await startField.blur();
  await endField.fill('2026-06-08');
  await endField.blur();
  await tool.locator('button[type=submit]').click();

  await expect(tool.locator('.tool-error')).toHaveText('Unable to calculate right now. Please try again.');
});
