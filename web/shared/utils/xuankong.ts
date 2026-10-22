// 玄空飞星：二十四山 → 洛书宫映射与中文数字（运星按传统用中文数字排布）。

/** 二十四山名（index 0-23，与引擎 zuo_shan/xiang_shan 对齐；向山 = 坐山 +12）。 */
export const MOUNTAINS_24 = [
  '子', '癸', '丑', '艮', '寅', '甲', '卯', '乙', '辰', '巽', '巳', '丙',
  '午', '丁', '未', '坤', '申', '庚', '酉', '辛', '戌', '乾', '亥', '壬',
] as const

/** 八卦名（玄空/八宅布局共用，与引擎 gua.name 对齐）。 */
export const GUA_8 = ['乾', '坎', '艮', '震', '巽', '离', '坤', '兑'] as const

/** 二十四山 → 洛书宫数（坎1 坤2 震3 巽4 乾6 兑7 艮8 离9）。 */
export function mountainGong(mountain: number): number {
  if (mountain < 0 || mountain > 23) return 0
  const gua = Math.floor(((mountain + 1) % 24) / 3)
  return [1, 8, 3, 4, 9, 2, 7, 6][gua]
}

const CN_NUMBERS = ['', '一', '二', '三', '四', '五', '六', '七', '八', '九']

/** 紫白星数 → 中文数字（运星传统写法）。 */
export function cnNumber(n: number): string {
  return CN_NUMBERS[n] ?? String(n)
}
