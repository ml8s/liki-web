// 引擎契约夹具（共享单源）：排盘/合盘工具的 mock 数据。
// 修改此处即同时影响 natal / liunian / hepan / naming 工具岛测试与契约断言。
import type { BaziChartData, BaziFullChartData, BaziLiuNianOutput } from '@shared/api/engine/bazi'

export const baziChart: BaziChartData = {
  nian: { gan: '庚', zhi: '午', na_yin: '路旁土' },
  yue: { gan: '辛', zhi: '巳', na_yin: '白蜡金' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1990,
  gender: 'female',
  zi_shi_rule: '晚子时属本日',
  da_yun: {
    start_date: '1996-02-04',
    start_year_after: 6,
    start_month_after: 7,
    start_day_after: 12,
    direction: '顺排',
    steps: [
      { gan: '壬', zhi: '午', name: '壬午', wuxing: '木', shi_shen: '比肩', start_date: '1996-02-04', end_date: '2005-02-03', start_year: 1996, end_year: 2005 },
      { gan: '癸', zhi: '未', name: '癸未', wuxing: '木', shi_shen: '劫财', start_date: '2006-02-04', end_date: '2015-02-03', start_year: 2006, end_year: 2015 },
      { gan: '甲', zhi: '申', name: '甲申', wuxing: '木', shi_shen: '食神', start_date: '2016-02-04', end_date: '2025-02-03', start_year: 2016, end_year: 2025 },
    ],
  },
};

export const baziFull: BaziFullChartData = {
  ...baziChart,
  nian: { ...baziChart.nian, cang_gan: { main: '丁', mid: '己' }, shi_shens: [{ shi_shen: '正官' }], shen_sha: [{ name: '天乙贵人' }], is_void: false },
  yue: { ...baziChart.yue, cang_gan: { main: '丙', mid: '庚', minor: '戊' }, shi_shens: [{ shi_shen: '正印' }], shen_sha: [{ name: '驿马' }], is_void: false },
  ri: { ...baziChart.ri, cang_gan: { main: '庚', mid: '壬', minor: '戊' }, shi_shens: [{ shi_shen: '日主' }], shen_sha: [], is_void: false },
  shi: { ...baziChart.shi, cang_gan: { main: '乙' }, shi_shens: [{ shi_shen: '伤官' }], shen_sha: [{ name: '羊刃' }], is_void: true },
  san_yuan: { tai_yuan: { gan: '壬', zhi: '午' }, ming_gong: { gan: '癸', zhi: '未' }, shen_gong: { gan: '甲', zhi: '申' } },
  fu_yi: { qiangruo: '身旺', yong: '木', xi: '水', ji: '土' },
  tiao_hou: { primary_wuxing: '木', secondary_wuxing: '水' },
  ge_ju: { ge_ju: '正财格', yong_fa: '顺用' },
};

export const baziLiunian: BaziLiuNianOutput = {
  year: 2026,
  nian_name: '丙午',
  shi_shen: '正官',
  nian_gan: '丙',
  nian_zhi: '午',
  wuxing: '火',
  na_yin: '天河水',
  sheng: 0,
  ke: 1,
  shensha: [{ name: '天乙贵人', category: '吉', description: '流运天乙贵人，有贵人相助' }],
  fuyin_fanyin: [],
  dayun_interactions: [],
  natal_interactions: [],
};

// bond 用部分关系字段（HepanPanel 按 relation||detail||type 容错），故不加类型标注。
export const bond = {
  zhu_cross: {
    pairs: [
      {
        a_zhu: 'ri', b_zhu: 'ri', a_gan: '壬', b_gan: '庚', a_zhi: '申', b_zhi: '申',
        gan_guan_xi: { relation: '庚生壬' },
        zhi_guan_xi: { type: '六冲', detail: '申申相冲' },
      },
    ],
  },
  nayin_cross: {
    pairs: [
      { a_zhu: 'ri', b_zhu: 'ri', a_na_yin: '剑锋金', b_na_yin: '剑锋金', relation: '同类' },
    ],
  },
  shensha_cross: {
    tian_yi: { a_present_in_b: true, b_present_in_a: false },
  },
  shi_shen_cross: {
    a_to_b: { nian_gan: '七杀' },
    b_to_a: { nian_gan: '偏印' },
  },
};
