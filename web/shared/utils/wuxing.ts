// 五行（Five Elements）：幸运元素表（五行 → 幸运数字 + 传统色）。
// 用于 natal 岛工具结果区的"幸运数字/颜色"展示；五行计数与旺衰由引擎
// （fu_yi.wuxing_count / element_states）提供，经共享 BaziPanel 展示。
export const WUXING_LUCKY: Record<string, { numbers: string; colors: string[] }> = {
  木: { numbers: '3 / 8', colors: ['#3f9d63', '#2f7f8f'] }, // 绿 / 青
  火: { numbers: '2 / 7', colors: ['#d64545', '#8a4fb5'] }, // 红 / 紫
  土: { numbers: '5 / 10', colors: ['#c9a227', '#8a6b4f'] }, // 黄 / 棕
  金: { numbers: '4 / 9', colors: ['#d9d9d9', '#c0c8d0'] }, // 白 / 银
  水: { numbers: '1 / 6', colors: ['#2b3b52', '#3f6ea5'] }, // 黑 / 蓝
}
