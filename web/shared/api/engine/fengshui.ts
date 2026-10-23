import { mcpTool } from '../mcp/client'
import { ENGINE_TOOLS } from './tools'

export interface BazhaiLiuNianXing {
  year?: number
  /** Engine boundary contract; current value is `gregorian_calendar_year`. */
  year_boundary?: string
  ru_zhong?: string
  gong_wei?: Array<{ gong_num: number; xing_name: string; wuxing?: string; ru_zhong?: boolean }>
}

export interface BazhaiChart {
  ming_gua: {
    gua: { name: string; wuxing?: string }
    group: string
    /** Engine boundary contract; current value is `gregorian_calendar_year`. */
    year_boundary?: string
  }
  ba_zhai_dirs: {
    sheng_qi: string
    tian_yi: string
    yan_nian: string
    fu_wei: string
    huo_hai: string
    wu_gui: string
    liu_sha: string
    jue_ming: string
  }
  liu_nian_xing?: BazhaiLiuNianXing
}

export function engineBazhaiChart(birthYear: number, gender: string): Promise<BazhaiChart> {
  return mcpTool<BazhaiChart>(ENGINE_TOOLS.BazhaiChart, { birth_year: birthYear, gender })
}

export interface XuankongStar {
  number: number
  color?: string
  name: string
  wuxing?: string
}

export interface XuankongPalace {
  /** 洛书宫数 1-9。 */
  gong_num: number
  yun_xing: XuankongStar
  shan_xing: XuankongStar
  xiang_xing: XuankongStar
}

export interface XuankongYun {
  year: number
  yuan?: string
  yun_number: number
  yun_name: string
  start_year: number
  end_year: number
}

export interface XuankongFourSituation {
  name: string
  sit_mountain_timely: boolean
  face_facing_timely: boolean
  sit_facing_timely: boolean
  face_mountain_timely: boolean
}

export interface XuankongShouShanChuSha {
  zheng_shen: number
  ling_shen: number
  shou_shan: boolean
  chu_sha: boolean
  assessment?: string
}

export interface XuankongXingJiaHui {
  shan_num: number
  xiang_num: number
  name: string
  meaning: string
}

export interface XuankongChart {
  yun: XuankongYun
  /** 九宫（洛书宫数 1-9，非渲染顺序）。 */
  gong_wei: XuankongPalace[]
  four_situation: XuankongFourSituation
  fu_yin_layers?: string[]
  fan_yin_layers?: string[]
  xing_jia_hui?: XuankongXingJiaHui[]
  zuo_shan: number
  zuo_shan_name: string
  xiang_shan: number
  xiang_shan_name: string
  shou_shan_chu_sha: XuankongShouShanChuSha
}

/** 玄空宅盘：period_date=立极日期（YYYY-MM-DD），坐山/向山为二十四山 index（0=子…23=壬，向=坐+12）。 */
export function engineXuankongChart(periodDate: string, zuoShan: number, xiangShan: number): Promise<XuankongChart> {
  return mcpTool<XuankongChart>(ENGINE_TOOLS.XuankongChart, {
    period_date: periodDate,
    zuo_shan: zuoShan,
    xiang_shan: xiangShan,
  })
}

// ── bazhai.layout（八宅门主灶布局判断）──

export interface BazhaiDoorStove {
  direction: string
  gua_name: string
  wuxing?: string
  /** 游年九星（伏位/生气/延年/天医/祸害/六煞/五鬼/绝命）。 */
  youxing: string
  rating: string
  group?: string
  /** 与命卦同组/异组。 */
  match: string
}

export interface BazhaiLayout {
  group: string
  ming_gua: string
  door: BazhaiDoorStove
  master: BazhaiDoorStove
  stove: BazhaiDoorStove
}

/** 八宅布局：命卦 + 门/主/灶三卦（乾坎艮震巽离坤兑）。 */
export function engineBazhaiLayout(mingGua: string, doorGua: string, masterGua: string, stoveGua: string): Promise<BazhaiLayout> {
  return mcpTool<BazhaiLayout>(ENGINE_TOOLS.BazhaiLayout, {
    ming_gua: mingGua,
    door_gua: doorGua,
    master_gua: masterGua,
    stove_gua: stoveGua,
  })
}
