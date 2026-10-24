// 引擎契约回归：用「真实引擎输出形状」渲染免费排盘与合盘，防止字段漂移被假 mock 掩盖。
// 排盘/合盘 = 前端调用 web BFF，BFF 服务端调用 engine-mcp（只读）。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { ENGINE_TOOLS } from '../../shared/api/engine/tools'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })


const user = testUser()
const profile = {
  id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', relation: '家人',
  birth: {
    gender: 'female', date: '1958-05-12', time: '06:00',
    instant: '1958-05-11T22:00:00Z',
    location: { place: '上海', longitude: 121.4726, timezone: 'Asia/Shanghai' },
  },
  note: '', third_party_consent: true, last_used_at: null,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

// ── 真实引擎输出形状（与 liki/engine Go 结构对齐，勿简化为字符串）──
const baziChart = {
  nian: { gan: '戊', zhi: '戌', na_yin: '平地木' },
  yue: { gan: '丁', zhi: '巳', na_yin: '沙中土' },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金' },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金' },
  birth_year: 1958,
  gender: 'female',
  da_yun: {
    direction: '逆排',
    steps: [
      { gan: '丙', zhi: '辰', name: '丙辰', wuxing: '火', shi_shen: '偏财', start_year: 1968, end_year: 1977, start_date: '1968-01-01', end_date: '1977-12-31' },
      { gan: '辛', zhi: '亥', name: '辛亥', wuxing: '金', shi_shen: '正印', start_year: 2018, end_year: 2027, start_date: '2018-01-01', end_date: '2027-12-31' },
    ],
  },
}
const baziFull = {
  nian: { gan: '戊', zhi: '戌', na_yin: '平地木', shi_shens: [{ shi_shen: '七杀', name: '七杀' }], cang_gan: { main: '戊', mid: '辛', minor: '丁' }, shen_sha: [{ name: '天乙贵人' }, { name: '华盖' }], is_void: false },
  yue: { gan: '丁', zhi: '巳', na_yin: '沙中土', shi_shens: [{ shi_shen: '正财', name: '正财' }], cang_gan: { main: '丙', mid: '戊', minor: '庚' }, shen_sha: [{ name: '驿马' }], is_void: false },
  ri: { gan: '壬', zhi: '申', na_yin: '剑锋金', shi_shens: [{ shi_shen: '日主', name: '日主' }], cang_gan: { main: '庚', mid: '壬', minor: '戊' }, shen_sha: [{ name: '驿马' }], is_void: false },
  shi: { gan: '癸', zhi: '卯', na_yin: '金箔金', shi_shens: [{ shi_shen: '劫财', name: '劫财' }], cang_gan: { main: '乙' }, shen_sha: [{ name: '羊刃' }], is_void: true },
  san_yuan: { tai_yuan: { gan: '壬', zhi: '申' }, ming_gong: { gan: '癸', zhi: '亥' }, shen_gong: { gan: '乙', zhi: '卯' } },
    fu_yi: { qiangruo: '身弱', yong: '金', xi: '金', ji: '土' },
    tiao_hou: { primary_wuxing: '金', secondary_wuxing: '水', season: '夏' },
    ge_ju: { ge_ju: '正财格', yong_fa: '财旺身弱宜比劫帮身' },
}
const ziweiChart = {"birth_lunar_month": 6, "birth_year": 2000, "gender": "male", "gong_wei": [{"gan": "己", "index": "命宫", "is_shen_gong": true, "name": "命宫", "xing_yao": [{"is_zhu_xing": true, "liang_du": "庙", "name": "紫微", "xing": "紫微"}, {"is_zhu_xing": true, "liang_du": "旺", "name": "破军", "xing": "破军"}, {"is_zhu_xing": false, "name": "天魁", "xing": "天魁"}], "zhi": "丑"}, {"gan": "戊", "index": "兄弟", "is_shen_gong": false, "name": "兄弟", "xing_yao": [{"is_zhu_xing": true, "liang_du": "庙", "name": "天机", "xing": "天机"}], "zhi": "子"}, {"gan": "丁", "index": "夫妻", "is_shen_gong": false, "name": "夫妻", "xing_yao": [], "zhi": "亥"}, {"gan": "丙", "index": "子女", "is_shen_gong": false, "name": "子女", "xing_yao": [{"is_zhu_xing": true, "liang_du": "不", "name": "太阳", "si_hua": "禄", "xing": "太阳"}, {"is_zhu_xing": false, "liang_du": "陷", "name": "文曲", "xing": "文曲"}], "zhi": "戌"}, {"gan": "乙", "index": "财帛", "is_shen_gong": false, "name": "财帛", "xing_yao": [{"is_zhu_xing": true, "liang_du": "利", "name": "武曲", "si_hua": "权", "xing": "武曲"}, {"is_zhu_xing": true, "liang_du": "旺", "name": "七杀", "xing": "七杀"}, {"is_zhu_xing": false, "name": "擎羊", "xing": "擎羊"}, {"is_zhu_xing": false, "name": "左辅", "xing": "左辅"}], "zhi": "酉"}, {"gan": "甲", "index": "疾厄", "is_shen_gong": false, "name": "疾厄", "xing_yao": [{"is_zhu_xing": true, "liang_du": "旺", "name": "天同", "si_hua": "忌", "xing": "天同"}, {"is_zhu_xing": true, "liang_du": "陷", "name": "天梁", "xing": "天梁"}, {"is_zhu_xing": false, "name": "禄存", "xing": "禄存"}, {"is_zhu_xing": false, "name": "火星", "xing": "火星"}], "zhi": "申"}, {"gan": "癸", "index": "迁移", "is_shen_gong": false, "name": "迁移", "xing_yao": [{"is_zhu_xing": true, "liang_du": "得", "name": "天相", "xing": "天相"}, {"is_zhu_xing": false, "name": "天钺", "xing": "天钺"}, {"is_zhu_xing": false, "name": "陀罗", "xing": "陀罗"}], "zhi": "未"}, {"gan": "壬", "index": "仆役", "is_shen_gong": false, "name": "仆役", "xing_yao": [{"is_zhu_xing": true, "liang_du": "旺", "name": "巨门", "xing": "巨门"}], "zhi": "午"}, {"gan": "辛", "index": "官禄", "is_shen_gong": false, "name": "官禄", "xing_yao": [{"is_zhu_xing": true, "liang_du": "陷", "name": "廉贞", "xing": "廉贞"}, {"is_zhu_xing": true, "liang_du": "陷", "name": "贪狼", "xing": "贪狼"}, {"is_zhu_xing": false, "name": "右弼", "xing": "右弼"}, {"is_zhu_xing": false, "name": "地空", "xing": "地空"}, {"is_zhu_xing": false, "name": "地劫", "xing": "地劫"}], "zhi": "巳"}, {"gan": "庚", "index": "田宅", "is_shen_gong": false, "is_yuan_gong": true, "name": "田宅", "xing_yao": [{"is_zhu_xing": true, "liang_du": "陷", "name": "太阴", "si_hua": "科", "xing": "太阴"}, {"is_zhu_xing": false, "liang_du": "得", "name": "文昌", "xing": "文昌"}, {"is_zhu_xing": false, "name": "铃星", "xing": "铃星"}], "zhi": "辰"}, {"gan": "己", "index": "福德", "is_shen_gong": false, "name": "福德", "xing_yao": [{"is_zhu_xing": true, "liang_du": "得", "name": "天府", "xing": "天府"}], "zhi": "卯"}, {"gan": "戊", "index": "父母", "is_shen_gong": false, "name": "父母", "xing_yao": [{"is_zhu_xing": false, "name": "天马", "xing": "天马"}], "zhi": "寅"}], "ju_shu": "火六局", "ju_shu_name": "火六局", "kong_gong": [{"dui_gong": "官禄", "gong_name": "夫妻", "jie_xing": ["廉贞", "贪狼"]}, {"dui_gong": "疾厄", "gong_name": "父母", "jie_xing": ["天同", "天梁"]}], "lunar_day": 15, "lunar_month": 6, "ming_gong": "命宫", "ming_zhu": "巨门", "nian_gan": "庚", "nian_zhi": "辰", "palace_facts": [{"kind": "star", "palace": "命宫", "star": "紫微", "target": "紫微"}, {"kind": "star", "palace": "命宫", "star": "紫微", "target": "紫微主星"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "庙", "value": "庙"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "紫微主星", "value": "庙"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "庙旺", "value": "庙"}, {"kind": "star", "palace": "命宫", "star": "破军", "target": "破军"}, {"kind": "star", "palace": "命宫", "star": "破军", "target": "紫微主星"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "命宫", "star": "天魁", "target": "天魁"}, {"kind": "star", "palace": "命宫", "star": "天魁", "target": "紫微六吉星"}, {"kind": "star", "palace": "兄弟", "star": "天机", "target": "天机"}, {"kind": "star", "palace": "兄弟", "star": "天机", "target": "紫微主星"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "庙", "value": "庙"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "紫微主星", "value": "庙"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "庙旺", "value": "庙"}, {"kind": "special", "palace": "兄弟", "star": "天机", "target": "唯一主星", "value": "true"}, {"kind": "special", "palace": "夫妻", "target": "无主星", "value": "true"}, {"kind": "star", "palace": "子女", "star": "太阳", "target": "太阳"}, {"kind": "star", "palace": "子女", "star": "太阳", "target": "紫微主星"}, {"kind": "si_hua", "palace": "子女", "star": "太阳", "target": "禄"}, {"kind": "brightness", "palace": "子女", "star": "太阳", "target": "不", "value": "不"}, {"kind": "brightness", "palace": "子女", "star": "太阳", "target": "紫微主星", "value": "不"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "文曲"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "紫微六吉星"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "紫微文星"}, {"kind": "brightness", "palace": "子女", "star": "文曲", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "子女", "star": "文曲", "target": "落陷", "value": "陷"}, {"kind": "special", "palace": "子女", "star": "太阳", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "财帛", "star": "武曲", "target": "武曲"}, {"kind": "star", "palace": "财帛", "star": "武曲", "target": "紫微主星"}, {"kind": "si_hua", "palace": "财帛", "star": "武曲", "target": "权"}, {"kind": "brightness", "palace": "财帛", "star": "武曲", "target": "利", "value": "利"}, {"kind": "brightness", "palace": "财帛", "star": "武曲", "target": "紫微主星", "value": "利"}, {"kind": "star", "palace": "财帛", "star": "七杀", "target": "七杀"}, {"kind": "star", "palace": "财帛", "star": "七杀", "target": "紫微主星"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "财帛", "star": "擎羊", "target": "擎羊"}, {"kind": "star", "palace": "财帛", "star": "擎羊", "target": "煞星"}, {"kind": "star", "palace": "财帛", "star": "左辅", "target": "左辅"}, {"kind": "star", "palace": "财帛", "star": "左辅", "target": "紫微六吉星"}, {"kind": "star", "palace": "疾厄", "star": "天同", "target": "天同"}, {"kind": "star", "palace": "疾厄", "star": "天同", "target": "紫微主星"}, {"kind": "si_hua", "palace": "疾厄", "star": "天同", "target": "忌"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "疾厄", "star": "天梁", "target": "天梁"}, {"kind": "star", "palace": "疾厄", "star": "天梁", "target": "紫微主星"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "疾厄", "star": "禄存", "target": "禄存"}, {"kind": "star", "palace": "疾厄", "star": "火星", "target": "火星"}, {"kind": "star", "palace": "疾厄", "star": "火星", "target": "煞星"}, {"kind": "star", "palace": "迁移", "star": "天相", "target": "天相"}, {"kind": "star", "palace": "迁移", "star": "天相", "target": "紫微主星"}, {"kind": "brightness", "palace": "迁移", "star": "天相", "target": "得", "value": "得"}, {"kind": "brightness", "palace": "迁移", "star": "天相", "target": "紫微主星", "value": "得"}, {"kind": "star", "palace": "迁移", "star": "天钺", "target": "天钺"}, {"kind": "star", "palace": "迁移", "star": "天钺", "target": "紫微六吉星"}, {"kind": "star", "palace": "迁移", "star": "陀罗", "target": "陀罗"}, {"kind": "star", "palace": "迁移", "star": "陀罗", "target": "煞星"}, {"kind": "special", "palace": "迁移", "star": "天相", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "仆役", "star": "巨门", "target": "巨门"}, {"kind": "star", "palace": "仆役", "star": "巨门", "target": "紫微主星"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "庙旺", "value": "旺"}, {"kind": "special", "palace": "仆役", "star": "巨门", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "官禄", "star": "廉贞", "target": "廉贞"}, {"kind": "star", "palace": "官禄", "star": "廉贞", "target": "紫微主星"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "官禄", "star": "贪狼", "target": "贪狼"}, {"kind": "star", "palace": "官禄", "star": "贪狼", "target": "紫微主星"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "官禄", "star": "右弼", "target": "右弼"}, {"kind": "star", "palace": "官禄", "star": "右弼", "target": "紫微六吉星"}, {"kind": "star", "palace": "官禄", "star": "地空", "target": "地空"}, {"kind": "star", "palace": "官禄", "star": "地空", "target": "煞星"}, {"kind": "star", "palace": "官禄", "star": "地劫", "target": "地劫"}, {"kind": "star", "palace": "官禄", "star": "地劫", "target": "煞星"}, {"kind": "star", "palace": "田宅", "star": "太阴", "target": "太阴"}, {"kind": "star", "palace": "田宅", "star": "太阴", "target": "紫微主星"}, {"kind": "si_hua", "palace": "田宅", "star": "太阴", "target": "科"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "文昌"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "紫微六吉星"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "紫微文星"}, {"kind": "brightness", "palace": "田宅", "star": "文昌", "target": "得", "value": "得"}, {"kind": "star", "palace": "田宅", "star": "铃星", "target": "铃星"}, {"kind": "star", "palace": "田宅", "star": "铃星", "target": "煞星"}, {"kind": "special", "palace": "田宅", "star": "太阴", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "福德", "star": "天府", "target": "天府"}, {"kind": "star", "palace": "福德", "star": "天府", "target": "紫微主星"}, {"kind": "brightness", "palace": "福德", "star": "天府", "target": "得", "value": "得"}, {"kind": "brightness", "palace": "福德", "star": "天府", "target": "紫微主星", "value": "得"}, {"kind": "special", "palace": "福德", "star": "天府", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "父母", "star": "天马", "target": "天马"}, {"kind": "special", "palace": "父母", "target": "无主星", "value": "true"}], "patterns": [{"description": "紫微坐命，三方四正得吉星朝拱", "name": "紫微朝垣"}, {"description": "七杀破军贪狼会聚命宫三方", "name": "杀破狼"}, {"description": "左辅右弼在命宫三方", "name": "辅弼拱主"}], "san_fang": [{"fu_xing": ["天魁"], "name": "命宫", "si_hua": [], "zhu_xing": ["紫微", "破军"]}, {"fu_xing": ["擎羊", "左辅"], "name": "财帛", "si_hua": ["权"], "zhu_xing": ["武曲", "七杀"]}, {"fu_xing": ["右弼", "地空", "地劫"], "name": "官禄", "si_hua": [], "zhu_xing": ["廉贞", "贪狼"]}, {"fu_xing": ["天钺", "陀罗"], "name": "迁移", "si_hua": [], "zhu_xing": ["天相"]}], "school": {"leap_month": "first_fifteen_days_current_month_next_fifteen_days_following_month", "source": "iztro_compatible"}, "shen_gong": "命宫", "shen_zhu": "文昌", "shi_zhi": "午", "si_hua": {"天同": "忌", "太阳": "禄", "太阴": "科", "武曲": "权"}, "ziwei_pos": "命宫"}
const ziweiFull = {"birth_lunar_month": 6, "birth_year": 2000, "gender": "male", "gong_wei": [{"ages": [4, 16, 28, 40, 52, 64, 76, 88, 100, 112], "bo_shi": "奏书", "chang_sheng": "养", "gan": "己", "index": "命宫", "is_shen_gong": true, "jiang_qian": "攀鞍", "name": "命宫", "sui_qian": "天德", "xing_yao": [{"is_zhu_xing": true, "liang_du": "庙", "name": "紫微", "xing": "紫微"}, {"is_zhu_xing": true, "liang_du": "旺", "name": "破军", "xing": "破军"}, {"is_zhu_xing": false, "name": "天魁", "xing": "天魁"}], "za_yao": ["天德", "寡宿", "破碎"], "zhi": "丑"}, {"ages": [3, 15, 27, 39, 51, 63, 75, 87, 99, 111], "bo_shi": "将军", "chang_sheng": "胎", "gan": "戊", "index": "兄弟", "is_shen_gong": false, "jiang_qian": "将星", "name": "兄弟", "sui_qian": "白虎", "xing_yao": [{"is_zhu_xing": true, "liang_du": "庙", "name": "天机", "xing": "天机"}], "za_yao": ["台辅", "解神"], "zhi": "子"}, {"ages": [2, 14, 26, 38, 50, 62, 74, 86, 98, 110], "bo_shi": "小耗", "chang_sheng": "绝", "gan": "丁", "index": "夫妻", "is_shen_gong": false, "jiang_qian": "亡神", "name": "夫妻", "sui_qian": "龙德", "xing_yao": [], "za_yao": ["三台", "天官", "天贵", "红鸾"], "zhi": "亥"}, {"ages": [1, 13, 25, 37, 49, 61, 73, 85, 97, 109], "bo_shi": "青龙", "chang_sheng": "墓", "gan": "丙", "index": "子女", "is_shen_gong": false, "jiang_qian": "月煞", "name": "子女", "sui_qian": "大耗", "xing_yao": [{"is_zhu_xing": true, "liang_du": "不", "name": "太阳", "si_hua": "禄", "xing": "太阳"}, {"is_zhu_xing": false, "liang_du": "陷", "name": "文曲", "xing": "文曲"}], "za_yao": ["天虚"], "zhi": "戌"}, {"ages": [12, 24, 36, 48, 60, 72, 84, 96, 108, 120], "bo_shi": "力士", "chang_sheng": "死", "gan": "乙", "index": "财帛", "is_shen_gong": false, "jiang_qian": "咸池", "name": "财帛", "sui_qian": "小耗", "xing_yao": [{"is_zhu_xing": true, "liang_du": "利", "name": "武曲", "si_hua": "权", "xing": "武曲"}, {"is_zhu_xing": true, "liang_du": "旺", "name": "七杀", "xing": "七杀"}, {"is_zhu_xing": false, "name": "擎羊", "xing": "擎羊"}, {"is_zhu_xing": false, "name": "左辅", "xing": "左辅"}], "za_yao": ["咸池", "月德"], "zhi": "酉"}, {"ages": [11, 23, 35, 47, 59, 71, 83, 95, 107, 119], "bo_shi": "博士", "chang_sheng": "病", "gan": "甲", "index": "疾厄", "is_shen_gong": false, "jiang_qian": "指背", "name": "疾厄", "sui_qian": "官符", "xing_yao": [{"is_zhu_xing": true, "liang_du": "旺", "name": "天同", "si_hua": "忌", "xing": "天同"}, {"is_zhu_xing": true, "liang_du": "陷", "name": "天梁", "xing": "天梁"}, {"is_zhu_xing": false, "name": "禄存", "xing": "禄存"}, {"is_zhu_xing": false, "name": "火星", "xing": "火星"}], "za_yao": ["天使", "天巫", "封诰", "旬空", "龙池"], "zhi": "申"}, {"ages": [10, 22, 34, 46, 58, 70, 82, 94, 106, 118], "bo_shi": "官府", "chang_sheng": "衰", "gan": "癸", "index": "迁移", "is_shen_gong": false, "jiang_qian": "天煞", "name": "迁移", "sui_qian": "贯索", "xing_yao": [{"is_zhu_xing": true, "liang_du": "得", "name": "天相", "xing": "天相"}, {"is_zhu_xing": false, "name": "天钺", "xing": "天钺"}, {"is_zhu_xing": false, "name": "陀罗", "xing": "陀罗"}], "za_yao": ["空亡"], "zhi": "未"}, {"ages": [9, 21, 33, 45, 57, 69, 81, 93, 105, 117], "bo_shi": "伏兵", "chang_sheng": "帝旺", "gan": "壬", "index": "仆役", "is_shen_gong": false, "jiang_qian": "灾煞", "name": "仆役", "sui_qian": "丧门", "xing_yao": [{"is_zhu_xing": true, "liang_du": "旺", "name": "巨门", "xing": "巨门"}], "za_yao": ["凤阁", "天伤", "天姚", "天福", "年解", "截路", "蜚廉"], "zhi": "午"}, {"ages": [8, 20, 32, 44, 56, 68, 80, 92, 104, 116], "bo_shi": "大耗", "chang_sheng": "临官", "gan": "辛", "index": "官禄", "is_shen_gong": false, "jiang_qian": "劫煞", "name": "官禄", "sui_qian": "晦气", "xing_yao": [{"is_zhu_xing": true, "liang_du": "陷", "name": "廉贞", "xing": "廉贞"}, {"is_zhu_xing": true, "liang_du": "陷", "name": "贪狼", "xing": "贪狼"}, {"is_zhu_xing": false, "name": "右弼", "xing": "右弼"}, {"is_zhu_xing": false, "name": "地空", "xing": "地空"}, {"is_zhu_xing": false, "name": "地劫", "xing": "地劫"}], "za_yao": ["天喜", "天寿", "天才", "天空", "孤辰", "恩光"], "zhi": "巳"}, {"ages": [7, 19, 31, 43, 55, 67, 79, 91, 103, 115], "bo_shi": "病符", "chang_sheng": "冠带", "gan": "庚", "index": "田宅", "is_shen_gong": false, "is_yuan_gong": true, "jiang_qian": "华盖", "name": "田宅", "sui_qian": "岁建", "xing_yao": [{"is_zhu_xing": true, "liang_du": "陷", "name": "太阴", "si_hua": "科", "xing": "太阴"}, {"is_zhu_xing": false, "liang_du": "得", "name": "文昌", "xing": "文昌"}, {"is_zhu_xing": false, "name": "铃星", "xing": "铃星"}], "za_yao": ["华盖", "阴煞"], "zhi": "辰"}, {"ages": [6, 18, 30, 42, 54, 66, 78, 90, 102, 114], "bo_shi": "喜神", "chang_sheng": "沐浴", "gan": "己", "index": "福德", "is_shen_gong": false, "jiang_qian": "息神", "name": "福德", "sui_qian": "病符", "xing_yao": [{"is_zhu_xing": true, "liang_du": "得", "name": "天府", "xing": "天府"}], "za_yao": ["八座", "天月"], "zhi": "卯"}, {"ages": [5, 17, 29, 41, 53, 65, 77, 89, 101, 113], "bo_shi": "飞廉", "chang_sheng": "长生", "gan": "戊", "index": "父母", "is_shen_gong": false, "jiang_qian": "岁驿", "name": "父母", "sui_qian": "吊客", "xing_yao": [{"is_zhu_xing": false, "name": "天马", "xing": "天马"}], "za_yao": ["天刑", "天厨", "天哭"], "zhi": "寅"}], "ju_shu": "火六局", "ju_shu_name": "火六局", "kong_gong": [{"dui_gong": "官禄", "gong_name": "夫妻", "jie_xing": ["廉贞", "贪狼"]}, {"dui_gong": "疾厄", "gong_name": "父母", "jie_xing": ["天同", "天梁"]}], "lunar_day": 15, "lunar_month": 6, "ming_gong": "命宫", "ming_zhu": "巨门", "nian_gan": "庚", "nian_zhi": "辰", "palace_facts": [{"kind": "star", "palace": "命宫", "star": "紫微", "target": "紫微"}, {"kind": "star", "palace": "命宫", "star": "紫微", "target": "紫微主星"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "庙", "value": "庙"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "紫微主星", "value": "庙"}, {"kind": "brightness", "palace": "命宫", "star": "紫微", "target": "庙旺", "value": "庙"}, {"kind": "star", "palace": "命宫", "star": "破军", "target": "破军"}, {"kind": "star", "palace": "命宫", "star": "破军", "target": "紫微主星"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "命宫", "star": "破军", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "命宫", "star": "天魁", "target": "天魁"}, {"kind": "star", "palace": "命宫", "star": "天魁", "target": "紫微六吉星"}, {"kind": "star", "palace": "兄弟", "star": "天机", "target": "天机"}, {"kind": "star", "palace": "兄弟", "star": "天机", "target": "紫微主星"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "庙", "value": "庙"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "紫微主星", "value": "庙"}, {"kind": "brightness", "palace": "兄弟", "star": "天机", "target": "庙旺", "value": "庙"}, {"kind": "special", "palace": "兄弟", "star": "天机", "target": "唯一主星", "value": "true"}, {"kind": "special", "palace": "夫妻", "target": "无主星", "value": "true"}, {"kind": "star", "palace": "子女", "star": "太阳", "target": "太阳"}, {"kind": "star", "palace": "子女", "star": "太阳", "target": "紫微主星"}, {"kind": "si_hua", "palace": "子女", "star": "太阳", "target": "禄"}, {"kind": "brightness", "palace": "子女", "star": "太阳", "target": "不", "value": "不"}, {"kind": "brightness", "palace": "子女", "star": "太阳", "target": "紫微主星", "value": "不"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "文曲"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "紫微六吉星"}, {"kind": "star", "palace": "子女", "star": "文曲", "target": "紫微文星"}, {"kind": "brightness", "palace": "子女", "star": "文曲", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "子女", "star": "文曲", "target": "落陷", "value": "陷"}, {"kind": "special", "palace": "子女", "star": "太阳", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "财帛", "star": "武曲", "target": "武曲"}, {"kind": "star", "palace": "财帛", "star": "武曲", "target": "紫微主星"}, {"kind": "si_hua", "palace": "财帛", "star": "武曲", "target": "权"}, {"kind": "brightness", "palace": "财帛", "star": "武曲", "target": "利", "value": "利"}, {"kind": "brightness", "palace": "财帛", "star": "武曲", "target": "紫微主星", "value": "利"}, {"kind": "star", "palace": "财帛", "star": "七杀", "target": "七杀"}, {"kind": "star", "palace": "财帛", "star": "七杀", "target": "紫微主星"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "财帛", "star": "七杀", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "财帛", "star": "擎羊", "target": "擎羊"}, {"kind": "star", "palace": "财帛", "star": "擎羊", "target": "煞星"}, {"kind": "star", "palace": "财帛", "star": "左辅", "target": "左辅"}, {"kind": "star", "palace": "财帛", "star": "左辅", "target": "紫微六吉星"}, {"kind": "star", "palace": "疾厄", "star": "天同", "target": "天同"}, {"kind": "star", "palace": "疾厄", "star": "天同", "target": "紫微主星"}, {"kind": "si_hua", "palace": "疾厄", "star": "天同", "target": "忌"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "疾厄", "star": "天同", "target": "庙旺", "value": "旺"}, {"kind": "star", "palace": "疾厄", "star": "天梁", "target": "天梁"}, {"kind": "star", "palace": "疾厄", "star": "天梁", "target": "紫微主星"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "疾厄", "star": "天梁", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "疾厄", "star": "禄存", "target": "禄存"}, {"kind": "star", "palace": "疾厄", "star": "火星", "target": "火星"}, {"kind": "star", "palace": "疾厄", "star": "火星", "target": "煞星"}, {"kind": "star", "palace": "迁移", "star": "天相", "target": "天相"}, {"kind": "star", "palace": "迁移", "star": "天相", "target": "紫微主星"}, {"kind": "brightness", "palace": "迁移", "star": "天相", "target": "得", "value": "得"}, {"kind": "brightness", "palace": "迁移", "star": "天相", "target": "紫微主星", "value": "得"}, {"kind": "star", "palace": "迁移", "star": "天钺", "target": "天钺"}, {"kind": "star", "palace": "迁移", "star": "天钺", "target": "紫微六吉星"}, {"kind": "star", "palace": "迁移", "star": "陀罗", "target": "陀罗"}, {"kind": "star", "palace": "迁移", "star": "陀罗", "target": "煞星"}, {"kind": "special", "palace": "迁移", "star": "天相", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "仆役", "star": "巨门", "target": "巨门"}, {"kind": "star", "palace": "仆役", "star": "巨门", "target": "紫微主星"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "旺", "value": "旺"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "紫微主星", "value": "旺"}, {"kind": "brightness", "palace": "仆役", "star": "巨门", "target": "庙旺", "value": "旺"}, {"kind": "special", "palace": "仆役", "star": "巨门", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "官禄", "star": "廉贞", "target": "廉贞"}, {"kind": "star", "palace": "官禄", "star": "廉贞", "target": "紫微主星"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "廉贞", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "官禄", "star": "贪狼", "target": "贪狼"}, {"kind": "star", "palace": "官禄", "star": "贪狼", "target": "紫微主星"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "官禄", "star": "贪狼", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "官禄", "star": "右弼", "target": "右弼"}, {"kind": "star", "palace": "官禄", "star": "右弼", "target": "紫微六吉星"}, {"kind": "star", "palace": "官禄", "star": "地空", "target": "地空"}, {"kind": "star", "palace": "官禄", "star": "地空", "target": "煞星"}, {"kind": "star", "palace": "官禄", "star": "地劫", "target": "地劫"}, {"kind": "star", "palace": "官禄", "star": "地劫", "target": "煞星"}, {"kind": "star", "palace": "田宅", "star": "太阴", "target": "太阴"}, {"kind": "star", "palace": "田宅", "star": "太阴", "target": "紫微主星"}, {"kind": "si_hua", "palace": "田宅", "star": "太阴", "target": "科"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "陷", "value": "陷"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "紫微主星", "value": "陷"}, {"kind": "brightness", "palace": "田宅", "star": "太阴", "target": "落陷", "value": "陷"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "文昌"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "紫微六吉星"}, {"kind": "star", "palace": "田宅", "star": "文昌", "target": "紫微文星"}, {"kind": "brightness", "palace": "田宅", "star": "文昌", "target": "得", "value": "得"}, {"kind": "star", "palace": "田宅", "star": "铃星", "target": "铃星"}, {"kind": "star", "palace": "田宅", "star": "铃星", "target": "煞星"}, {"kind": "special", "palace": "田宅", "star": "太阴", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "福德", "star": "天府", "target": "天府"}, {"kind": "star", "palace": "福德", "star": "天府", "target": "紫微主星"}, {"kind": "brightness", "palace": "福德", "star": "天府", "target": "得", "value": "得"}, {"kind": "brightness", "palace": "福德", "star": "天府", "target": "紫微主星", "value": "得"}, {"kind": "special", "palace": "福德", "star": "天府", "target": "唯一主星", "value": "true"}, {"kind": "star", "palace": "父母", "star": "天马", "target": "天马"}, {"kind": "special", "palace": "父母", "target": "无主星", "value": "true"}], "patterns": [{"description": "紫微坐命，三方四正得吉星朝拱", "name": "紫微朝垣"}, {"description": "七杀破军贪狼会聚命宫三方", "name": "杀破狼"}, {"description": "左辅右弼在命宫三方", "name": "辅弼拱主"}], "san_fang": [{"fu_xing": ["天魁"], "name": "命宫", "si_hua": [], "zhu_xing": ["紫微", "破军"]}, {"fu_xing": ["擎羊", "左辅"], "name": "财帛", "si_hua": ["权"], "zhu_xing": ["武曲", "七杀"]}, {"fu_xing": ["右弼", "地空", "地劫"], "name": "官禄", "si_hua": [], "zhu_xing": ["廉贞", "贪狼"]}, {"fu_xing": ["天钺", "陀罗"], "name": "迁移", "si_hua": [], "zhu_xing": ["天相"]}], "school": {"leap_month": "first_fifteen_days_current_month_next_fifteen_days_following_month", "source": "iztro_compatible"}, "shen_gong": "命宫", "shen_zhu": "文昌", "shi_zhi": "午", "si_hua": {"天同": "忌", "太阳": "禄", "太阴": "科", "武曲": "权"}, "ziwei_pos": "命宫"}
const ziweiDaXian = [{"end_year": 2014, "gong": "命宫", "name": "命宫", "qi_sui": 6, "start_year": 2005, "zhi_sui": 15}, {"end_year": 2024, "gong": "父母", "name": "父母", "qi_sui": 16, "start_year": 2015, "zhi_sui": 25}, {"end_year": 2034, "gong": "福德", "name": "福德", "qi_sui": 26, "start_year": 2025, "zhi_sui": 35}, {"end_year": 2044, "gong": "田宅", "name": "田宅", "qi_sui": 36, "start_year": 2035, "zhi_sui": 45}, {"end_year": 2054, "gong": "官禄", "name": "官禄", "qi_sui": 46, "start_year": 2045, "zhi_sui": 55}, {"end_year": 2064, "gong": "仆役", "name": "仆役", "qi_sui": 56, "start_year": 2055, "zhi_sui": 65}, {"end_year": 2074, "gong": "迁移", "name": "迁移", "qi_sui": 66, "start_year": 2065, "zhi_sui": 75}, {"end_year": 2084, "gong": "疾厄", "name": "疾厄", "qi_sui": 76, "start_year": 2075, "zhi_sui": 85}, {"end_year": 2094, "gong": "财帛", "name": "财帛", "qi_sui": 86, "start_year": 2085, "zhi_sui": 95}, {"end_year": 2104, "gong": "子女", "name": "子女", "qi_sui": 96, "start_year": 2095, "zhi_sui": 105}, {"end_year": 2114, "gong": "夫妻", "name": "夫妻", "qi_sui": 106, "start_year": 2105, "zhi_sui": 115}, {"end_year": 2124, "gong": "兄弟", "name": "兄弟", "qi_sui": 116, "start_year": 2115, "zhi_sui": 125}]
const ziweiLiuNian = {"fu_xing": {"天喜": "卯", "天钺": "酉", "天马": "申", "天魁": "亥", "擎羊": "午", "文昌": "申", "文曲": "午", "禄存": "巳", "红鸾": "酉", "陀罗": "辰"}, "gong_wei": [{"is_liu_ming": false, "name": "财帛", "xing_yao": [], "zhi": "寅"}, {"is_liu_ming": false, "name": "子女", "xing_yao": ["流喜"], "zhi": "卯"}, {"is_liu_ming": false, "name": "夫妻", "xing_yao": ["流陀", "年解"], "zhi": "辰"}, {"is_liu_ming": false, "name": "兄弟", "xing_yao": ["流禄"], "zhi": "巳"}, {"is_liu_ming": true, "name": "命宫", "xing_yao": ["流羊", "流曲"], "zhi": "午"}, {"is_liu_ming": false, "name": "父母", "xing_yao": [], "zhi": "未"}, {"is_liu_ming": false, "name": "福德", "xing_yao": ["流马", "流昌"], "zhi": "申"}, {"is_liu_ming": false, "name": "田宅", "xing_yao": ["流钺", "流鸾"], "zhi": "酉"}, {"is_liu_ming": false, "name": "官禄", "xing_yao": [], "zhi": "戌"}, {"is_liu_ming": false, "name": "仆役", "xing_yao": ["流魁"], "zhi": "亥"}, {"is_liu_ming": false, "name": "迁移", "xing_yao": [], "zhi": "子"}, {"is_liu_ming": false, "name": "疾厄", "xing_yao": [], "zhi": "丑"}], "ming_gong": "仆役", "ming_gong_name": "仆役", "si_hua": {"天同": "禄", "天机": "权", "廉贞": "忌", "文昌": "科"}, "si_hua_gong": {"天同": "疾厄", "天机": "兄弟", "廉贞": "官禄", "文昌": "田宅"}, "zhi": "午"}
const bazhai = {
  ming_gua: { gua: { index: 8, name: '艮', wuxing: '土', yin_yang: '阳' }, group: '西四命', year_boundary: 'gregorian_calendar_year' },
  ba_zhai_dirs: {
    sheng_qi: '西南', tian_yi: '西北', yan_nian: '西', fu_wei: '东北',
    huo_hai: '南', wu_gui: '北', liu_sha: '东', jue_ming: '东南',
  },
  liu_nian_xing: {
    year: 2026, year_boundary: 'gregorian_calendar_year', ru_zhong: '三碧禄存',
    gong_wei: [
      { gong_num: 1, xing: 4, xing_name: '四绿文曲', wuxing: '木', ru_zhong: false },
      { gong_num: 2, xing: 5, xing_name: '五黄廉贞', wuxing: '土', ru_zhong: false },
      { gong_num: 3, xing: 6, xing_name: '六白武曲', wuxing: '金', ru_zhong: false },
      { gong_num: 4, xing: 7, xing_name: '七赤破军', wuxing: '金', ru_zhong: false },
      { gong_num: 5, xing: 3, xing_name: '三碧禄存', wuxing: '木', ru_zhong: true },
      { gong_num: 6, xing: 8, xing_name: '八白左辅', wuxing: '土', ru_zhong: false },
      { gong_num: 7, xing: 9, xing_name: '九紫右弼', wuxing: '火', ru_zhong: false },
      { gong_num: 8, xing: 1, xing_name: '一白贪狼', wuxing: '水', ru_zhong: false },
      { gong_num: 9, xing: 2, xing_name: '二黑巨门', wuxing: '土', ru_zhong: false },
    ],
  },
}
const bond = {
  'day_master': {
    'a': {
      'gan': '甲',
      'gender': 'male',
      'wuxing': '木'
    },
    'b': {
      'gan': '辛',
      'gender': 'female',
      'wuxing': '金'
    },
    'gan_guan_xi': {
      'gan_a': '甲',
      'gan_b': '辛',
      'relation': '辛克甲',
      'type': '相克'
    }
  },
  'nayin_cross': {
    'pairs': [
      {
        'a_na_yin': '白蜡金',
        'a_zhu': 'nian',
        'b_na_yin': '白蜡金',
        'b_zhu': 'nian',
        'relation': '相同'
      },
      {
        'a_na_yin': '白蜡金',
        'a_zhu': 'nian',
        'b_na_yin': '泉中水',
        'b_zhu': 'yue',
        'relation': '相生'
      },
      {
        'a_na_yin': '白蜡金',
        'a_zhu': 'nian',
        'b_na_yin': '白蜡金',
        'b_zhu': 'ri',
        'relation': '相同'
      },
      {
        'a_na_yin': '白蜡金',
        'a_zhu': 'nian',
        'b_na_yin': '长流水',
        'b_zhu': 'shi',
        'relation': '相生'
      },
      {
        'a_na_yin': '杨柳木',
        'a_zhu': 'yue',
        'b_na_yin': '白蜡金',
        'b_zhu': 'nian',
        'relation': '相克'
      },
      {
        'a_na_yin': '杨柳木',
        'a_zhu': 'yue',
        'b_na_yin': '泉中水',
        'b_zhu': 'yue',
        'relation': '相生'
      },
      {
        'a_na_yin': '杨柳木',
        'a_zhu': 'yue',
        'b_na_yin': '白蜡金',
        'b_zhu': 'ri',
        'relation': '相克'
      },
      {
        'a_na_yin': '杨柳木',
        'a_zhu': 'yue',
        'b_na_yin': '长流水',
        'b_zhu': 'shi',
        'relation': '相生'
      },
      {
        'a_na_yin': '覆灯火',
        'a_zhu': 'ri',
        'b_na_yin': '白蜡金',
        'b_zhu': 'nian',
        'relation': '相克'
      },
      {
        'a_na_yin': '覆灯火',
        'a_zhu': 'ri',
        'b_na_yin': '泉中水',
        'b_zhu': 'yue',
        'relation': '相克'
      },
      {
        'a_na_yin': '覆灯火',
        'a_zhu': 'ri',
        'b_na_yin': '白蜡金',
        'b_zhu': 'ri',
        'relation': '相克'
      },
      {
        'a_na_yin': '覆灯火',
        'a_zhu': 'ri',
        'b_na_yin': '长流水',
        'b_zhu': 'shi',
        'relation': '相克'
      },
      {
        'a_na_yin': '路旁土',
        'a_zhu': 'shi',
        'b_na_yin': '白蜡金',
        'b_zhu': 'nian',
        'relation': '相生'
      },
      {
        'a_na_yin': '路旁土',
        'a_zhu': 'shi',
        'b_na_yin': '泉中水',
        'b_zhu': 'yue',
        'relation': '相克'
      },
      {
        'a_na_yin': '路旁土',
        'a_zhu': 'shi',
        'b_na_yin': '白蜡金',
        'b_zhu': 'ri',
        'relation': '相生'
      },
      {
        'a_na_yin': '路旁土',
        'a_zhu': 'shi',
        'b_na_yin': '长流水',
        'b_zhu': 'shi',
        'relation': '相克'
      }
    ],
    'wuxing_counts': {
      'a': {
        '土': 1,
        '木': 1,
        '火': 1,
        '金': 1
      },
      'b': {
        '水': 2,
        '金': 2
      }
    }
  },
  'shensha_cross': {
    'kong_wang': {
      'a_present_in_b': false,
      'b_present_in_a': false
    },
    'kui_gang': {
      'a_present_in_b': true,
      'b_present_in_a': true
    },
    'lu': {
      'a_present_in_b': false,
      'b_present_in_a': false
    },
    'ri_de': {
      'a_present_in_b': true,
      'b_present_in_a': true
    },
    'ri_gui': {
      'a_present_in_b': false,
      'b_present_in_a': false
    },
    'tao_hua': {
      'a_present_in_b': true,
      'b_present_in_a': true
    },
    'tian_yi': {
      'a_present_in_b': false,
      'b_present_in_a': true
    },
    'yi_ma': {
      'a_present_in_b': false,
      'b_present_in_a': false
    }
  },
  'shi_shen_cross': {
    'a_to_b': {
      'nian_gan': '七杀',
      'ri_gan': '正官',
      'shi_gan': '偏印',
      'yue_gan': '劫财'
    },
    'b_to_a': {
      'nian_gan': '劫财',
      'ri_gan': '正财',
      'shi_gan': '劫财',
      'yue_gan': '伤官'
    }
  },
  'spouse_palace': {
    'a_zhi': '辰',
    'b_zhi': '巳',
    'relation': {
      'detail': '无特殊关系',
      'type': '无',
      'zhi_a': '辰',
      'zhi_b': '巳'
    }
  },
  'spouse_star': {
    'a': {
      'gender': 'male',
      'primary': {
        'occurrences': [
          {
            'branch': '午',
            'is_void': false,
            'pillar': 'yue',
            'source': 'mid_qi',
            'stem': '己'
          },
          {
            'branch': '午',
            'is_void': false,
            'pillar': 'shi',
            'source': 'mid_qi',
            'stem': '己'
          }
        ],
        'ten_god': '正财'
      },
      'secondary': {
        'occurrences': [
          {
            'branch': '辰',
            'is_void': false,
            'pillar': 'nian',
            'source': 'main_qi',
            'stem': '戊'
          },
          {
            'branch': '辰',
            'is_void': false,
            'pillar': 'ri',
            'source': 'main_qi',
            'stem': '戊'
          }
        ],
        'ten_god': '偏财'
      }
    },
    'b': {
      'gender': 'female',
      'primary': {
        'occurrences': [
          {
            'branch': '巳',
            'is_void': false,
            'pillar': 'ri',
            'source': 'main_qi',
            'stem': '丙'
          }
        ],
        'ten_god': '正官'
      },
      'secondary': {
        'occurrences': [],
        'ten_god': '七杀'
      }
    }
  },
  'wuxing_cross': {
    'a': {
      '土': 4,
      '木': 3,
      '水': 3,
      '火': 2,
      '金': 2
    },
    'b': {
      '土': 3,
      '木': 3,
      '水': 3,
      '火': 1,
      '金': 4
    },
    'combined': {
      '土': 7,
      '木': 6,
      '水': 6,
      '火': 3,
      '金': 6
    },
    'fu_yi_in_other': {
      'a': {
        'ji': '金',
        'ji_in_other': 4,
        'model': 'support_control_with_day_master_strength',
        'xi': '木',
        'xi_in_other': 3,
        'yong': '水',
        'yong_in_other': 3
      },
      'b': {
        'ji': '土',
        'ji_in_other': 4,
        'model': 'support_control_with_day_master_strength',
        'xi': '木',
        'xi_in_other': 3,
        'yong': '火',
        'yong_in_other': 2
      }
    }
  },
  'zhu_cross': {
    'pairs': [
      {
        'a_gan': '庚',
        'a_zhi': '辰',
        'a_zhu': 'nian',
        'b_gan': '庚',
        'b_zhi': '辰',
        'b_zhu': 'nian',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '庚',
          'relation': '庚庚同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '辰辰自刑',
          'type': '相刑',
          'zhi_a': '辰',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '辰',
        'a_zhu': 'nian',
        'b_gan': '乙',
        'b_zhi': '酉',
        'b_zhu': 'yue',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '乙',
          'relation': '庚乙合化金',
          'type': '天干五合'
        },
        'zhi_guan_xi': {
          'detail': '辰酉合化金',
          'type': '六合',
          'zhi_a': '辰',
          'zhi_b': '酉'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '辰',
        'a_zhu': 'nian',
        'b_gan': '辛',
        'b_zhi': '巳',
        'b_zhu': 'ri',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '辛',
          'relation': '庚辛同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '辰',
          'zhi_b': '巳'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '辰',
        'a_zhu': 'nian',
        'b_gan': '壬',
        'b_zhi': '辰',
        'b_zhu': 'shi',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '壬',
          'relation': '庚生壬',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '辰辰自刑',
          'type': '相刑',
          'zhi_a': '辰',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '壬',
        'a_zhi': '午',
        'a_zhu': 'yue',
        'b_gan': '庚',
        'b_zhi': '辰',
        'b_zhu': 'nian',
        'gan_guan_xi': {
          'gan_a': '壬',
          'gan_b': '庚',
          'relation': '庚生壬',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '壬',
        'a_zhi': '午',
        'a_zhu': 'yue',
        'b_gan': '乙',
        'b_zhi': '酉',
        'b_zhu': 'yue',
        'gan_guan_xi': {
          'gan_a': '壬',
          'gan_b': '乙',
          'relation': '壬生乙',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '酉'
        }
      },
      {
        'a_gan': '壬',
        'a_zhi': '午',
        'a_zhu': 'yue',
        'b_gan': '辛',
        'b_zhi': '巳',
        'b_zhu': 'ri',
        'gan_guan_xi': {
          'gan_a': '壬',
          'gan_b': '辛',
          'relation': '辛生壬',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '巳'
        }
      },
      {
        'a_gan': '壬',
        'a_zhi': '午',
        'a_zhu': 'yue',
        'b_gan': '壬',
        'b_zhi': '辰',
        'b_zhu': 'shi',
        'gan_guan_xi': {
          'gan_a': '壬',
          'gan_b': '壬',
          'relation': '壬壬同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '甲',
        'a_zhi': '辰',
        'a_zhu': 'ri',
        'b_gan': '庚',
        'b_zhi': '辰',
        'b_zhu': 'nian',
        'gan_guan_xi': {
          'gan_a': '甲',
          'gan_b': '庚',
          'relation': '庚克甲',
          'type': '相克'
        },
        'zhi_guan_xi': {
          'detail': '辰辰自刑',
          'type': '相刑',
          'zhi_a': '辰',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '甲',
        'a_zhi': '辰',
        'a_zhu': 'ri',
        'b_gan': '乙',
        'b_zhi': '酉',
        'b_zhu': 'yue',
        'gan_guan_xi': {
          'gan_a': '甲',
          'gan_b': '乙',
          'relation': '甲乙同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '辰酉合化金',
          'type': '六合',
          'zhi_a': '辰',
          'zhi_b': '酉'
        }
      },
      {
        'a_gan': '甲',
        'a_zhi': '辰',
        'a_zhu': 'ri',
        'b_gan': '辛',
        'b_zhi': '巳',
        'b_zhu': 'ri',
        'gan_guan_xi': {
          'gan_a': '甲',
          'gan_b': '辛',
          'relation': '辛克甲',
          'type': '相克'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '辰',
          'zhi_b': '巳'
        }
      },
      {
        'a_gan': '甲',
        'a_zhi': '辰',
        'a_zhu': 'ri',
        'b_gan': '壬',
        'b_zhi': '辰',
        'b_zhu': 'shi',
        'gan_guan_xi': {
          'gan_a': '甲',
          'gan_b': '壬',
          'relation': '壬生甲',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '辰辰自刑',
          'type': '相刑',
          'zhi_a': '辰',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '午',
        'a_zhu': 'shi',
        'b_gan': '庚',
        'b_zhi': '辰',
        'b_zhu': 'nian',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '庚',
          'relation': '庚庚同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '辰'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '午',
        'a_zhu': 'shi',
        'b_gan': '乙',
        'b_zhi': '酉',
        'b_zhu': 'yue',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '乙',
          'relation': '庚乙合化金',
          'type': '天干五合'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '酉'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '午',
        'a_zhu': 'shi',
        'b_gan': '辛',
        'b_zhi': '巳',
        'b_zhu': 'ri',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '辛',
          'relation': '庚辛同气',
          'type': '相同'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '巳'
        }
      },
      {
        'a_gan': '庚',
        'a_zhi': '午',
        'a_zhu': 'shi',
        'b_gan': '壬',
        'b_zhi': '辰',
        'b_zhu': 'shi',
        'gan_guan_xi': {
          'gan_a': '庚',
          'gan_b': '壬',
          'relation': '庚生壬',
          'type': '相生'
        },
        'zhi_guan_xi': {
          'detail': '无特殊关系',
          'type': '无',
          'zhi_a': '午',
          'zhi_b': '辰'
        }
      }
    ]
  }
}

const liuyaoCasting = {
  mode: 'auto',
  order: 'bottom_up',
  yaos: [7, 8, 9, 7, 8, 7],
  dong_yao: [3],
  casting_id: 'c1',
}
const liuyaoSnapshot = {
  name: '水火既济',
  ben_gua: '水火既济',
  gong: '坎',
  gong_wuxing: '水',
  lines: Array.from({ length: 6 }, (_, index) => ({
    position: index + 1,
    type: liuyaoCasting.yaos[index],
    gan: '戊',
    zhi: '子',
    wuxing: '水',
    liu_qin: index === 2 ? '妻财' : '兄弟',
    shi_ying: index === 2 ? '世' : '',
    liu_shou: '青龙',
  })),
  bian_yao: Array.from({ length: 6 }, () => ({ position: 0, type: 0, gan: '', zhi: '', wuxing: '', liu_qin: '', shi_ying: '', liu_shou: '' })),
  ri_chen_gan: '戊',
  ri_chen_zhi: '子',
  yue_jian_gan: '甲',
  yue_jian_zhi: '寅',
  yong_shen: { name: '妻财', position: 3, wang_shuai: '相' },
  gua_ci: { name: '水火既济', gua_ci: '亨，小利贞，初吉终乱。', yao_ci: Array(6).fill('') },
  wang_shuai: Array(6).fill('相'),
  ri_chen_relations: Array.from({ length: 6 }, () => ({ relations: ['平'] })),
  xun_kong: ['午', '未'],
  timing_candidates: [{ id: 'timing_1', mechanism: '值期', window: '午月', confidence: '候选' }],
  patterns: [{ type: '空', sub_type: '真空', position: 0, is_true: true }],
  casting: liuyaoCasting,
}
const huangliDays = [
  {
    'date': '2026-10-06',
    'ri_zhu': {
      'gan': '癸',
      'zhi': '丑',
      'na_yin': '桑柘木'
    },
    'na_yin': '桑柘木',
    'wuxing': '土',
    'jian_chu': '定',
    'huangdao': {
      'index': 11,
      'name': '勾陈',
      'path': '黑道',
      'sequence': 11
    },
    'xi_shen': '东南',
    'cai_shen': '正南',
    'fu_shen': '正西',
    'gan_ji': '癸不词讼理弱敌强',
    'zhi_ji': '丑不冠带主不还乡',
    'mansion': {
      'index': 3,
      'name': '房日兔',
      'dong_wu': '兔',
      'wuxing': '日',
      'group': '东方青龙',
      'group_idx': 4
    },
    'jie_qi': '秋分',
    'jie_qi_days': 13,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '司命',
        'jian_chu': '定',
        'suitable': true
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '勾陈',
        'jian_chu': '执',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '青龙',
        'jian_chu': '破',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '明堂',
        'jian_chu': '危',
        'suitable': true
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '天刑',
        'jian_chu': '成',
        'suitable': false
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '朱雀',
        'jian_chu': '收',
        'suitable': false
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '金匮',
        'jian_chu': '开',
        'suitable': true
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '天德',
        'jian_chu': '闭',
        'suitable': true
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '白虎',
        'jian_chu': '建',
        'suitable': false
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '玉堂',
        'jian_chu': '除',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '天牢',
        'jian_chu': '满',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玄武',
        'jian_chu': '平',
        'suitable': false
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'possible',
    'reason': '建除「定」无明确适配或冲突。'
  },
  {
    'date': '2026-10-07',
    'ri_zhu': {
      'gan': '甲',
      'zhi': '寅',
      'na_yin': '大溪水'
    },
    'na_yin': '大溪水',
    'wuxing': '木',
    'jian_chu': '执',
    'huangdao': {
      'index': 0,
      'name': '青龙',
      'path': '黄道',
      'sequence': 0
    },
    'xi_shen': '东北',
    'cai_shen': '东北',
    'fu_shen': '东南',
    'gan_ji': '甲不开仓财物耗散',
    'zhi_ji': '寅不祭祀神鬼不尝',
    'mansion': {
      'index': 4,
      'name': '心月狐',
      'dong_wu': '狐',
      'wuxing': '月',
      'group': '东方青龙',
      'group_idx': 5
    },
    'jie_qi': '秋分',
    'jie_qi_days': 14,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '司命',
        'jian_chu': '执',
        'suitable': true
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '勾陈',
        'jian_chu': '破',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '青龙',
        'jian_chu': '危',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '明堂',
        'jian_chu': '成',
        'suitable': true
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '天刑',
        'jian_chu': '收',
        'suitable': false
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '朱雀',
        'jian_chu': '开',
        'suitable': false
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '金匮',
        'jian_chu': '闭',
        'suitable': true
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '天德',
        'jian_chu': '建',
        'suitable': true
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '白虎',
        'jian_chu': '除',
        'suitable': false
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '玉堂',
        'jian_chu': '满',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '天牢',
        'jian_chu': '平',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玄武',
        'jian_chu': '定',
        'suitable': false
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'possible',
    'reason': '建除「执」无明确适配或冲突。'
  },
  {
    'date': '2026-10-08',
    'ri_zhu': {
      'gan': '乙',
      'zhi': '卯',
      'na_yin': '大溪水'
    },
    'na_yin': '大溪水',
    'wuxing': '木',
    'jian_chu': '破',
    'huangdao': {
      'index': 1,
      'name': '明堂',
      'path': '黄道',
      'sequence': 1
    },
    'xi_shen': '西北',
    'cai_shen': '东北',
    'fu_shen': '东南',
    'gan_ji': '乙不栽植千株不长',
    'zhi_ji': '卯不穿井水泉不香',
    'mansion': {
      'index': 5,
      'name': '尾火虎',
      'dong_wu': '虎',
      'wuxing': '火',
      'group': '东方青龙',
      'group_idx': 6
    },
    'jie_qi': '寒露',
    'jie_qi_days': 0,
    'ren_yuan': '庚',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '司命',
        'jian_chu': '破',
        'suitable': true
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '勾陈',
        'jian_chu': '危',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '青龙',
        'jian_chu': '成',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '明堂',
        'jian_chu': '收',
        'suitable': true
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '天刑',
        'jian_chu': '开',
        'suitable': false
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '朱雀',
        'jian_chu': '闭',
        'suitable': false
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '金匮',
        'jian_chu': '建',
        'suitable': true
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '天德',
        'jian_chu': '除',
        'suitable': true
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '白虎',
        'jian_chu': '满',
        'suitable': false
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '玉堂',
        'jian_chu': '平',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '天牢',
        'jian_chu': '定',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玄武',
        'jian_chu': '执',
        'suitable': false
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'unsuitable',
    'reason': '建除「破」忌嫁娶。'
  },
  {
    'date': '2026-10-09',
    'ri_zhu': {
      'gan': '丙',
      'zhi': '辰',
      'na_yin': '沙中土'
    },
    'na_yin': '沙中土',
    'wuxing': '土',
    'jian_chu': '破',
    'huangdao': {
      'index': 0,
      'name': '青龙',
      'path': '黄道',
      'sequence': 0
    },
    'xi_shen': '西南',
    'cai_shen': '正西',
    'fu_shen': '西北',
    'gan_ji': '丙不修灶必见灾殃',
    'zhi_ji': '辰不哭泣必主重丧',
    'mansion': {
      'index': 6,
      'name': '箕水豹',
      'dong_wu': '豹',
      'wuxing': '水',
      'group': '东方青龙',
      'group_idx': 7
    },
    'jie_qi': '寒露',
    'jie_qi_days': 1,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '天牢',
        'jian_chu': '破',
        'suitable': false
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '玄武',
        'jian_chu': '危',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '司命',
        'jian_chu': '成',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '勾陈',
        'jian_chu': '收',
        'suitable': false
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '青龙',
        'jian_chu': '开',
        'suitable': true
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '明堂',
        'jian_chu': '闭',
        'suitable': true
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '天刑',
        'jian_chu': '建',
        'suitable': false
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '朱雀',
        'jian_chu': '除',
        'suitable': false
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '金匮',
        'jian_chu': '满',
        'suitable': true
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '天德',
        'jian_chu': '平',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '白虎',
        'jian_chu': '定',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玉堂',
        'jian_chu': '执',
        'suitable': true
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'unsuitable',
    'reason': '建除「破」忌嫁娶。'
  },
  {
    'date': '2026-10-10',
    'ri_zhu': {
      'gan': '丁',
      'zhi': '巳',
      'na_yin': '沙中土'
    },
    'na_yin': '沙中土',
    'wuxing': '火',
    'jian_chu': '危',
    'huangdao': {
      'index': 1,
      'name': '明堂',
      'path': '黄道',
      'sequence': 1
    },
    'xi_shen': '正南',
    'cai_shen': '正西',
    'fu_shen': '正东',
    'gan_ji': '丁不剃头头必生疮',
    'zhi_ji': '巳不远行财物伏藏',
    'mansion': {
      'index': 7,
      'name': '斗木獬',
      'dong_wu': '獬',
      'wuxing': '木',
      'group': '北方玄武',
      'group_idx': 1
    },
    'jie_qi': '寒露',
    'jie_qi_days': 2,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '天牢',
        'jian_chu': '危',
        'suitable': false
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '玄武',
        'jian_chu': '成',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '司命',
        'jian_chu': '收',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '勾陈',
        'jian_chu': '开',
        'suitable': false
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '青龙',
        'jian_chu': '闭',
        'suitable': true
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '明堂',
        'jian_chu': '建',
        'suitable': true
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '天刑',
        'jian_chu': '除',
        'suitable': false
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '朱雀',
        'jian_chu': '满',
        'suitable': false
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '金匮',
        'jian_chu': '平',
        'suitable': true
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '天德',
        'jian_chu': '定',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '白虎',
        'jian_chu': '执',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玉堂',
        'jian_chu': '破',
        'suitable': true
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'possible',
    'reason': '建除「危」无明确适配或冲突。'
  },
  {
    'date': '2026-10-11',
    'ri_zhu': {
      'gan': '戊',
      'zhi': '午',
      'na_yin': '天上火'
    },
    'na_yin': '天上火',
    'wuxing': '火',
    'jian_chu': '成',
    'huangdao': {
      'index': 2,
      'name': '天刑',
      'path': '黑道',
      'sequence': 2
    },
    'xi_shen': '东南',
    'cai_shen': '正北',
    'fu_shen': '正南',
    'gan_ji': '戊不受田田主不祥',
    'zhi_ji': '午不苫盖屋主更张',
    'mansion': {
      'index': 8,
      'name': '牛金牛',
      'dong_wu': '牛',
      'wuxing': '金',
      'group': '北方玄武',
      'group_idx': 2
    },
    'jie_qi': '寒露',
    'jie_qi_days': 3,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '天牢',
        'jian_chu': '成',
        'suitable': false
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '玄武',
        'jian_chu': '收',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '司命',
        'jian_chu': '开',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '勾陈',
        'jian_chu': '闭',
        'suitable': false
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '青龙',
        'jian_chu': '建',
        'suitable': true
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '明堂',
        'jian_chu': '除',
        'suitable': true
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '天刑',
        'jian_chu': '满',
        'suitable': false
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '朱雀',
        'jian_chu': '平',
        'suitable': false
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '金匮',
        'jian_chu': '定',
        'suitable': true
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '天德',
        'jian_chu': '执',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '白虎',
        'jian_chu': '破',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玉堂',
        'jian_chu': '危',
        'suitable': true
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'recommended',
    'reason': '建除「成」适合嫁娶。'
  },
  {
    'date': '2026-10-12',
    'ri_zhu': {
      'gan': '己',
      'zhi': '未',
      'na_yin': '天上火'
    },
    'na_yin': '天上火',
    'wuxing': '土',
    'jian_chu': '收',
    'huangdao': {
      'index': 3,
      'name': '朱雀',
      'path': '黑道',
      'sequence': 3
    },
    'xi_shen': '东北',
    'cai_shen': '正北',
    'fu_shen': '正南',
    'gan_ji': '己不破券二比并亡',
    'zhi_ji': '未不服药毒气入肠',
    'mansion': {
      'index': 9,
      'name': '女土蝠',
      'dong_wu': '蝠',
      'wuxing': '土',
      'group': '北方玄武',
      'group_idx': 3
    },
    'jie_qi': '寒露',
    'jie_qi_days': 4,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '天牢',
        'jian_chu': '收',
        'suitable': false
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '玄武',
        'jian_chu': '开',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '司命',
        'jian_chu': '闭',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '勾陈',
        'jian_chu': '建',
        'suitable': false
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '青龙',
        'jian_chu': '除',
        'suitable': true
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '明堂',
        'jian_chu': '满',
        'suitable': true
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '天刑',
        'jian_chu': '平',
        'suitable': false
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '朱雀',
        'jian_chu': '定',
        'suitable': false
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '金匮',
        'jian_chu': '执',
        'suitable': true
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '天德',
        'jian_chu': '破',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '白虎',
        'jian_chu': '危',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玉堂',
        'jian_chu': '成',
        'suitable': true
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'unsuitable',
    'reason': '建除「收」忌嫁娶。'
  },
  {
    'date': '2026-10-13',
    'ri_zhu': {
      'gan': '庚',
      'zhi': '申',
      'na_yin': '石榴木'
    },
    'na_yin': '石榴木',
    'wuxing': '金',
    'jian_chu': '开',
    'huangdao': {
      'index': 4,
      'name': '金匮',
      'path': '黄道',
      'sequence': 4
    },
    'xi_shen': '西北',
    'cai_shen': '正东',
    'fu_shen': '西南',
    'gan_ji': '庚不经络织机虚张',
    'zhi_ji': '申不安床鬼祟入房',
    'mansion': {
      'index': 10,
      'name': '虚日鼠',
      'dong_wu': '鼠',
      'wuxing': '日',
      'group': '北方玄武',
      'group_idx': 4
    },
    'jie_qi': '寒露',
    'jie_qi_days': 5,
    'ren_yuan': '辛',
    'shi_chen': [
      {
        'zhi': '子',
        'time': '23:00-01:00',
        'huangdao': '天牢',
        'jian_chu': '开',
        'suitable': false
      },
      {
        'zhi': '丑',
        'time': '01:00-03:00',
        'huangdao': '玄武',
        'jian_chu': '闭',
        'suitable': false
      },
      {
        'zhi': '寅',
        'time': '03:00-05:00',
        'huangdao': '司命',
        'jian_chu': '建',
        'suitable': true
      },
      {
        'zhi': '卯',
        'time': '05:00-07:00',
        'huangdao': '勾陈',
        'jian_chu': '除',
        'suitable': false
      },
      {
        'zhi': '辰',
        'time': '07:00-09:00',
        'huangdao': '青龙',
        'jian_chu': '满',
        'suitable': true
      },
      {
        'zhi': '巳',
        'time': '09:00-11:00',
        'huangdao': '明堂',
        'jian_chu': '平',
        'suitable': true
      },
      {
        'zhi': '午',
        'time': '11:00-13:00',
        'huangdao': '天刑',
        'jian_chu': '定',
        'suitable': false
      },
      {
        'zhi': '未',
        'time': '13:00-15:00',
        'huangdao': '朱雀',
        'jian_chu': '执',
        'suitable': false
      },
      {
        'zhi': '申',
        'time': '15:00-17:00',
        'huangdao': '金匮',
        'jian_chu': '破',
        'suitable': true
      },
      {
        'zhi': '酉',
        'time': '17:00-19:00',
        'huangdao': '天德',
        'jian_chu': '危',
        'suitable': true
      },
      {
        'zhi': '戌',
        'time': '19:00-21:00',
        'huangdao': '白虎',
        'jian_chu': '成',
        'suitable': false
      },
      {
        'zhi': '亥',
        'time': '21:00-23:00',
        'huangdao': '玉堂',
        'jian_chu': '收',
        'suitable': true
      }
    ],
    'event': 'wedding',
    'event_label': '嫁娶',
    'suitability': 'recommended',
    'reason': '建除「开」适合嫁娶。'
  }
]

// ── 流年盘（bazi_liunian 真实形状）──
const baziLiunian = {
  year: 2026,
  nian_name: '丙午',
  shi_shen: '正官',
  nian_gan: '丙',
  nian_zhi: '午',
  wuxing: '火',
  na_yin: '天河水',
  sheng: 0,
  ke: 1,
  shensha: [
    { name: '天乙贵人', category: '吉', description: '流运天乙贵人，有贵人相助' },
    { name: '吊客', category: '凶', description: '流运吊客临命，主吊丧/孝服' },
  ],
  fuyin_fanyin: [{ detail: '午地支伏吟', natal_index: 0, type: '伏吟' }],
  dayun_interactions: [{
    pillar_label: '偏财运(乙酉)',
    gan_rels: [
      { gan_a: '乙', gan_b: '庚', relation: '乙庚合化金', type: '天干五合' },
      { gan_a: '乙', gan_b: '丁', relation: '乙生丁', type: '相生' },
      { gan_a: '乙', gan_b: '壬', relation: '壬生乙', type: '相生' },
      { gan_a: '乙', gan_b: '癸', relation: '癸生乙', type: '相生' },
    ],
    zhi_rels: [
      { detail: '无特殊关系', type: '无', zhi_a: '酉', zhi_b: '戌' },
      { detail: '巳酉半合金局', type: '半合', zhi_a: '酉', zhi_b: '巳' },
    ],
  }],
  natal_interactions: [{
    pillar_label: '丙午',
    gan_rels: [
      { gan_a: '丙', gan_b: '庚', relation: '丙克庚', type: '相克' },
      { gan_a: '丙', gan_b: '丁', relation: '丁克丙', type: '相克' },
      { gan_a: '丙', gan_b: '壬', relation: '丙壬相冲', type: '相冲' },
      { gan_a: '丙', gan_b: '癸', relation: '癸克丙', type: '相克' },
    ],
    zhi_rels: [
      { detail: '午午自刑', type: '相刑', zhi_a: '午', zhi_b: '午' },
      { detail: '无特殊关系', type: '无', zhi_a: '午', zhi_b: '巳' },
    ],
  }],
}

// ── 玄空飞星宅盘（xuankong_chart 真实形状：九运 子山午向）──
const xuankongChart = {
  yun: { year: 2026, year_boundary: 'gregorian_calendar_year', yuan: '下元', yun_number: 9, yun_name: '九运', start_year: 2024, end_year: 2043 },
  gong_wei: [
    { gong_num: 1, yun_xing: { number: 5, color: '黄', name: '五黄廉贞', wuxing: '土' }, shan_xing: { number: 9, color: '紫', name: '九紫右弼', wuxing: '火' }, xiang_xing: { number: 9, color: '紫', name: '九紫右弼', wuxing: '火' } },
    { gong_num: 2, yun_xing: { number: 6, color: '白', name: '六白武曲', wuxing: '金' }, shan_xing: { number: 8, color: '白', name: '八白左辅', wuxing: '土' }, xiang_xing: { number: 1, color: '白', name: '一白贪狼', wuxing: '水' } },
    { gong_num: 3, yun_xing: { number: 7, color: '赤', name: '七赤破军', wuxing: '金' }, shan_xing: { number: 7, color: '赤', name: '七赤破军', wuxing: '金' }, xiang_xing: { number: 2, color: '黑', name: '二黑巨门', wuxing: '土' } },
    { gong_num: 4, yun_xing: { number: 8, color: '白', name: '八白左辅', wuxing: '土' }, shan_xing: { number: 6, color: '白', name: '六白武曲', wuxing: '金' }, xiang_xing: { number: 3, color: '碧', name: '三碧禄存', wuxing: '木' } },
    { gong_num: 5, yun_xing: { number: 9, color: '紫', name: '九紫右弼', wuxing: '火' }, shan_xing: { number: 5, color: '黄', name: '五黄廉贞', wuxing: '土' }, xiang_xing: { number: 4, color: '绿', name: '四绿文曲', wuxing: '木' } },
    { gong_num: 6, yun_xing: { number: 1, color: '白', name: '一白贪狼', wuxing: '水' }, shan_xing: { number: 4, color: '绿', name: '四绿文曲', wuxing: '木' }, xiang_xing: { number: 5, color: '黄', name: '五黄廉贞', wuxing: '土' } },
    { gong_num: 7, yun_xing: { number: 2, color: '黑', name: '二黑巨门', wuxing: '土' }, shan_xing: { number: 3, color: '碧', name: '三碧禄存', wuxing: '木' }, xiang_xing: { number: 6, color: '白', name: '六白武曲', wuxing: '金' } },
    { gong_num: 8, yun_xing: { number: 3, color: '碧', name: '三碧禄存', wuxing: '木' }, shan_xing: { number: 2, color: '黑', name: '二黑巨门', wuxing: '土' }, xiang_xing: { number: 7, color: '赤', name: '七赤破军', wuxing: '金' } },
    { gong_num: 9, yun_xing: { number: 4, color: '绿', name: '四绿文曲', wuxing: '木' }, shan_xing: { number: 1, color: '白', name: '一白贪狼', wuxing: '水' }, xiang_xing: { number: 8, color: '白', name: '八白左辅', wuxing: '土' } },
  ],
  four_situation: { name: '双星会坐', sit_mountain_timely: true, face_facing_timely: false, sit_facing_timely: true, face_mountain_timely: false },
  fu_yin_layers: [],
  fan_yin_layers: ['山星反吟'],
  xing_jia_hui: [
    { shan_num: 9, xiang_num: 9, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 8, xiang_num: 1, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 7, xiang_num: 2, name: '七二同道', meaning: '二七同宫，传统论火灾风险；成灾与否须核元运、生旺与峦头' },
    { shan_num: 6, xiang_num: 3, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 5, xiang_num: 4, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 4, xiang_num: 5, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 3, xiang_num: 6, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
    { shan_num: 2, xiang_num: 7, name: '二七同道', meaning: '二七同宫，传统论火灾风险；成灾与否须核元运、生旺与峦头' },
    { shan_num: 1, xiang_num: 8, name: '星曜加会', meaning: '该组合未列入固定通则，须参合星、宫、运与峦头判断' },
  ],
  chart_digest: 'sha256:50173497269a148a0ad0',
  zuo_shan: 0,
  zuo_shan_name: '子',
  xiang_shan: 12,
  xiang_shan_name: '午',
  shou_shan_chu_sha: { zheng_shen: 9, ling_shen: 1, shou_shan: true, chu_sha: false, assessment: '收山得宜（坐宫山星当令）；拨水入零堂未得，向首零神不见水' },
}

// ── 八宅布局（bazhai_layout 真实形状：艮命 门坤/主兑/灶乾）──
const bazhaiLayout = {
  group: '西四命',
  ming_gua: '艮',
  door: { direction: '西南', gua_name: '坤', wuxing: '土', youxing: '生气', rating: '大吉', group: '西四卦', match: '吉' },
  master: { direction: '西', gua_name: '兑', wuxing: '金', youxing: '延年', rating: '吉', group: '西四卦', match: '吉' },
  stove: { direction: '西北', gua_name: '乾', wuxing: '金', youxing: '天医', rating: '吉', group: '西四卦', match: '吉' },
}

function routeEngine(page: Page) {
  return page.route(/\/(engine|counsel)\/mcp$/, async route => {
    const headers = route.request().headers()
    const body = JSON.parse(route.request().postData() ?? '{}') as {
      params?: { name?: string; _meta?: Record<string, unknown> }
    }
    const tool = body.params?.name ?? ''
    // Browser-to-MCP contract: the edge requires the method/name headers before
    // it parses the JSON-RPC body. These assertions catch a client that still
    // sends only the protocol version.
    expect(headers['mcp-method']).toBe('tools/call')
    expect(headers['mcp-name']).toBe(tool)
    expect(headers['mcp-protocol-version']).toBe('2026-07-28')
    expect(headers['accept']).toContain('text/event-stream')
    expect(body.params?._meta?.['io.modelcontextprotocol/protocolVersion']).toBe('2026-07-28')
    let data: unknown
    switch (tool) {
      case ENGINE_TOOLS.CityCoords: data = { candidates: [{ name: '上海', region: '上海市', country: '中国', level: 'city', longitude: 121.47, latitude: 31.23, timezone: 'Asia/Shanghai' }] }; break
      case ENGINE_TOOLS.BaziChart: data = baziChart; break
      case ENGINE_TOOLS.BaziFullChart: data = baziFull; break
      case ENGINE_TOOLS.BaziLiuNian: data = baziLiunian; break
      case ENGINE_TOOLS.ZiweiChart: data = ziweiChart; break
      case ENGINE_TOOLS.ZiweiFullChart: data = ziweiFull; break
      case ENGINE_TOOLS.ZiweiDaXian: data = ziweiDaXian; break
      case ENGINE_TOOLS.ZiweiLiuNian: data = ziweiLiuNian; break
      case ENGINE_TOOLS.BazhaiChart: data = bazhai; break
      case ENGINE_TOOLS.BazhaiLayout: data = bazhaiLayout; break
      case ENGINE_TOOLS.XuankongChart: data = xuankongChart; break
      case ENGINE_TOOLS.BaziBond: data = bond; break
      case ENGINE_TOOLS.LiuyaoQigua: data = { casting: liuyaoCasting }; break
      case ENGINE_TOOLS.LiuyaoChart: data = liuyaoSnapshot; break
      case ENGINE_TOOLS.HuangliDays: data = huangliDays; break
      default: data = {}
    }
    return route.fulfill({ json: { jsonrpc: '2.0', id: 1, result: { content: [{ type: 'text', text: JSON.stringify(data) }] } } })
  })
}

async function routeApp(page: Page) {
  await mockWorkspace(page, { user, allowedSkills: ['liki'], profiles: [profile] })
  // 详情页按 id 取单个档案（覆盖壳里的列表响应）。
  await page.route('**/api/birth-profiles/p_1', r => r.fulfill(json(profile)))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)
}

test('免费排盘按真实引擎契约渲染（八字/紫微/八宅）', async ({ page }) => {
  await routeApp(page)

  await page.goto('/app/birth-profiles/p_1')
  await page.waitForSelector('.pillars', { timeout: 12000 }) // 自动排盘
  await expect(page.locator('.pillar .gl').first()).toContainText('戊戌')
  await expect(page.locator('.pillar .cang').first()).toContainText('戊 辛 丁') // 藏干
  await expect(page.locator('.pillar .sub').first()).toContainText('七杀') // 十神（shi_shens 数组首项）
  await expect(page.locator('.void-tag')).toHaveCount(1) // 旬空
  await expect(page.locator('.bazi-panel .shensha').first()).toContainText('天乙贵人') // 神煞
  await expect(page.locator('.bazi-panel .shensha').first()).toContainText('华盖')
  await expect(page.locator('.san-yuan .sy-body')).toContainText('胎元 壬申')

  // 流年盘（bazi_liunian）：五柱流年列 + 流年详览（神煞/伏吟/与命局/与大运关系）
  await expect(page.locator('.pillars.with-liunian .pillar.liunian')).toContainText('丙午')
  await expect(page.locator('.pillar.liunian')).toContainText('正官')
  await expect(page.locator('.pillar.liunian')).toContainText('生數 0')
  await expect(page.locator('.liunian-detail .chart-chip').first()).toContainText('天乙贵人')
  await expect(page.locator('.ln-block-title').first()).toContainText('與命局')
  await expect(page.locator('.ln-block-title').nth(1)).toContainText('與大運')
  await expect(page.locator('.liunian-detail')).toContainText('丙克庚')
  await expect(page.locator('.liunian-detail')).toContainText('午午自刑')
  await expect(page.locator('.liunian-detail')).not.toContainText('无特殊关系')

  // 空态：命盘无合冲刑害时不渲染「命盘关系」区块（mock 命盘无关系数据）
  await expect(page.locator('.bazi-panel')).not.toContainText('命盤關係')

  await page.locator('.el-tabs__item:has-text("紫微")').click()
  const natalGrid = page.locator('.ziwei-panel .zw-grid:not(.zw-grid--liunian)')
  await expect(natalGrid.locator('.zw-palace')).toHaveCount(12) // 环形十二宫
  await expect(natalGrid.locator('.zw-palace').first()).toContainText('紫微庙') // 庙旺平陷（命宫己丑）
  await expect(natalGrid.locator('.zw-palace').first()).toContainText('天德') // 杂曜
  await expect(natalGrid.locator('.zw-palace').first()).toContainText('長生') // 长生神
  await expect(natalGrid.locator('.zw-palace').first()).toHaveAttribute('title', /小限/) // 小限进 tooltip
  await expect(natalGrid.locator('.zw-palace.ming .zw-pname')).toHaveText('命宫')
  await expect(natalGrid.locator('.zw-palace.shen')).toHaveCount(1) // 身宫角标
  await expect(natalGrid.locator('.zw-center')).toContainText('火六局') // 五行局
  await expect(natalGrid.locator('.zw-center')).toContainText('巨门') // 命主
  await expect(natalGrid.locator('.zw-center')).toContainText('庚辰年') // 年干支
  await expect(natalGrid.locator('.zw-center .zw-sihua-chip')).toHaveCount(4) // 四化
  await expect(page.locator('.zw-sf')).toHaveCount(4) // 三方四正
  await expect(page.locator('.zw-sf:has-text("武曲")')).toHaveCount(1) // 财帛宫含武曲七杀
  await expect(page.locator('.zw-daxian .dy')).toHaveCount(12) // 大限十二限
  await expect(page.locator('.zw-daxian .dy').first()).toContainText('2005') // golden 首限起始年
  await expect(page.locator('.zw-daxian .dy').first()).toContainText('6–15') // 虚岁区间
  await expect(page.locator('.zw-daxian .dy.cur')).toHaveCount(1) // 当前大限高亮
  await expect(page.locator('.zw-liunian .el-input-number')).toBeVisible() // 流年年份控件
  await expect(page.locator('.zw-liunian .zw-sihua-chip')).toHaveCount(4) // 流年四化
  await expect(page.locator('.zw-grid--liunian .zw-palace')).toHaveCount(12) // 流年十二宫
  await expect(page.locator('.zw-grid--liunian .zw-palace.ming')).toContainText('命宫') // 流年命宫
  await expect(page.locator('.zw-grid--liunian .zw-center')).toContainText('仆役') // 流年命宫落本命宫（golden ming_gong_name）

  await page.locator('.el-tabs__item:has-text("八宅")').click()
  await expect(page.locator('.mingua')).toHaveText('艮')
  await expect(page.locator('.fly-cell')).toHaveCount(9) // 流年飞星九宫
  await expect(page.locator('.ln-row')).toContainText('三碧禄存')
})

test('合盘按真实引擎契约渲染（bazi_bond）', async ({ page }) => {
  await routeApp(page)

  await page.goto('/app/hepan')
  await page.waitForSelector('.b-form', { timeout: 10000 })
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("妈妈")').first().click()
  // 等下拉关闭再点击表单（固定等待在负载下会点穿浮层）。
  await expect(page.locator('.el-select-dropdown')).toBeHidden()
  await page.locator('.b-form .el-radio-button:has-text("男")').click()
  await page.getByRole('combobox', { name: '出生日期' }).click()
  await page.locator('.el-picker-panel .el-date-table td.available').first().click()
  await page.getByRole('combobox', { name: '出生時間' }).fill('08:00')
  await page.locator('input[name="birth-place"]').fill('上海')
  await page.locator('input[name="birth-place"]').blur()
  await page.locator('.region-manual-toggle').click()
  await expect(page.locator('input[name="birth-longitude"]')).toHaveValue('121.4726')
  await page.locator('.b-form button[type="submit"]').click()
  await page.waitForSelector('.rel-sec', { timeout: 12000 })
  const dayMaster = page.locator('.rel-sec', { hasText: '日主關係' })
  await expect(dayMaster.locator('.rel-text')).toContainText('甲辰') // 日主（golden a_zhu=b_zhu=ri）
  await expect(dayMaster.locator('.rel-text')).toContainText('辛克甲') // 天干关系
  await expect(page.locator('.rel-chip')).toContainText(['辰酉合化金', '辰辰自刑']) // 关键地支：按 type 过滤、detail 展示
  await expect(page.locator('.rel-chip.nayin')).toContainText(['白蜡金']) // 纳音互看
  await expect(page.locator('.rel-chip.shensha')).toContainText(['天乙貴人']) // 神煞标识符→locale 标签
  await expect(page.locator('.rel-chip')).toContainText(['妈妈年柱看對方：七杀']) // 十神 a_to_b
})

test('问卦按真实契约渲染（liuyao_snapshot）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/birth-profiles**', r => r.fulfill(json({ profiles: [profile], next_cursor: '' })))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/divination')
  await page.waitForSelector('input[placeholder*="項目"]', { timeout: 10000 })
  await page.locator('input[placeholder*="項目"]').fill('這個項目能成嗎')
  await page.locator('input[name="casting-place"]').fill('上海')
  await page.locator('input[name="casting-place"]').blur()
  await expect(page.locator('input[name="casting-longitude"]')).toHaveValue('121.4726')
  await page.click('button:has-text("搖卦")')
  await page.waitForSelector('.gua-name', { timeout: 12000 })
  await expect(page.locator('.gua-name')).toHaveText('水火既济') // board.name
  await expect(page.locator('.gua-info')).toContainText('坎') // board.gong
  await expect(page.locator('.rel-sec', { hasText: '用神' }).locator('.rel-text')).toContainText('妻财') // yong_shen
})

test('择日按真实契约渲染（huangli_days）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/birth-profiles**', r => r.fulfill(json({ profiles: [profile], next_cursor: '' })))
  await page.route('**/api/chat/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/huangli')
  await page.waitForSelector('.el-date-editor', { timeout: 10000 })
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("結婚")').first().click()
  await page.getByRole('combobox', { name: '起始日期' }).fill('2026-10-06')
  await page.keyboard.press('Enter')
  await page.getByRole('combobox', { name: '結束日期' }).fill('2026-10-30')
  await page.keyboard.press('Enter')
  await page.locator('button[type="submit"]').click()
  await page.waitForSelector('.day.good', { timeout: 12000 })
  await expect(page.locator('.day.good .d-date').first()).toHaveText('2026-10-11') // 引擎真数据首个 recommended
  await expect(page.locator('.day.possible')).toHaveCount(3)
  await expect(page.locator('.day.bad')).toHaveCount(3) // unsuitable
})

test('风水页按真实契约渲染（命卦/玄空飞星/八宅布局）', async ({ page }) => {
  await routeApp(page)
  await page.route('**/api/birth-profiles**', r => r.fulfill(json({ profiles: [profile], next_cursor: '' })))
  await page.route('**/api/me/sessions', r => r.fulfill(json([])))
  await page.route('**/api/me/auth/identities', r => r.fulfill(json([])))
  await routeEngine(page)

  await page.goto('/app/fengshui')
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("妈妈")').first().click()
  await page.waitForSelector('.mingua', { timeout: 12000 })

  // 命卦（八宅）
  await expect(page.locator('.mingua')).toHaveText('艮')

  // 玄空飞星盘（xuankong_chart）：标准九宫（山星左上/向星右上/运星下）+ 坐向角标
  await expect(page.locator('.chart-grid .xk-cell')).toHaveCount(9)
  await expect(page.locator('.xk-head')).toContainText('九运')
  await expect(page.locator('.xk-head')).toContainText('子')
  await expect(page.locator('.xk-head')).toContainText('午')
  await expect(page.locator('.xk-cell.sit')).toContainText('坐')
  await expect(page.locator('.xk-cell.face')).toContainText('向')
  await expect(page.locator('.xuankong-panel')).toContainText('双星会坐')
  await expect(page.locator('.xk-combo-name')).toContainText(['七二同道', '二七同道'])
  await expect(page.locator('.chart-chip', { hasText: '山星反吟' })).toHaveCount(1)

  // 八宅布局（bazhai_layout）：门/主/灶三要判断
  await page.locator('.fengshui-layout-fields .el-select').nth(1).click()
  await page.locator('.fx-popper-door .el-select-dropdown__item', { hasText: '坤' }).first().click()
  await page.locator('.fengshui-layout-fields .el-select').nth(2).click()
  await page.locator('.fx-popper-master .el-select-dropdown__item', { hasText: '兑' }).first().click()
  await page.locator('.fengshui-layout-fields .el-select').nth(3).click()
  await page.locator('.fx-popper-stove .el-select-dropdown__item', { hasText: '乾' }).first().click()
  await page.waitForSelector('.bz-verdict', { timeout: 12000 })
  await expect(page.locator('.bz-table tbody tr')).toHaveCount(3)
  await expect(page.locator('.bz-table')).toContainText('生气') // 引擎游年星值（简体内文，不随 locale 翻译）
  await expect(page.locator('.bz-table')).toContainText('大吉')
  await expect(page.locator('.bz-badge')).toContainText(['門', '主臥', '灶'])
  await expect(page.locator('.bz-cell.good')).toHaveCount(4) // 四吉方着色
})
