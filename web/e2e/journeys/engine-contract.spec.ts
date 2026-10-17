// 引擎契约回归：用「真实引擎输出形状」渲染免费排盘与合盘，防止字段漂移被假 mock 掩盖。
// 排盘/合盘 = 前端调用 web BFF，BFF 服务端调用 engine-mcp（只读）。
import { test, expect } from '../fixtures'
import type {Page} from '@playwright/test'
import { ENGINE_TOOLS } from '../../app/src/api/engine'
import { json, mockWorkspace, testUser } from '../mocks'

test.use({ locale: 'zh-Hant' })


const user = testUser()
const profile = {
  id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', gender: 'female', relation: '家人',
  calendar: 'solar', birth_date: '1958-05-12', birth_time: '06:00', birth_time_known: 'known',
  birth_place: '上海', note: '', third_party_consent: true, last_used_at: null,
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
    direction: '逆排', current_step_index: 5, next_step_in_years: 2,
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
  yong_shen: {
    fu_yi: { qiangruo: '身弱', yong: '金', xi: '金水', ji: '火土' },
    tiao_hou: { yong: '辛', xi: '壬', season: '夏' },
    ge_ju: { ge_ju: '正财格', yong_fa: '财旺身弱宜比劫帮身', yong: '辛', xi: '金水', ji: '火土' },
  },
}
const ziweiChart = { school: 'x', gong_wei: [], ming_gong: 0, shen_gong: 5, ju_shu: 0, ju_shu_name: '水二局', ziwei_pos: 0, si_hua: {}, palace_facts: [], nian_gan: '戊', shi_zhi: '卯', birth_year: 1958, lunar_month: 4 }
const ziweiFull = {
  gong_wei: [
    { index: 0, name: '命宫', gan: '癸', zhi: '亥', is_shen_gong: false, xing_yao: [{ xing: 0, name: '紫微', is_zhu_xing: true, liang_du: '庙' }, { xing: 1, name: '贪狼', is_zhu_xing: false, liang_du: '平' }] },
    { index: 11, name: '父母', gan: '甲', zhi: '寅', is_shen_gong: true, xing_yao: [{ xing: 7, name: '破军', is_zhu_xing: true, liang_du: '陷' }] },
    { index: 1, name: '兄弟', gan: '壬', zhi: '子', is_shen_gong: false, xing_yao: [] },
    { index: 2, name: '夫妻', gan: '壬', zhi: '丑', is_shen_gong: false, xing_yao: [] },
    { index: 3, name: '子女', gan: '辛', zhi: '寅', is_shen_gong: false, xing_yao: [] },
    { index: 4, name: '财帛', gan: '辛', zhi: '卯', is_shen_gong: false, xing_yao: [] },
    { index: 5, name: '疾厄', gan: '庚', zhi: '辰', is_shen_gong: false, xing_yao: [] },
    { index: 6, name: '迁移', gan: '庚', zhi: '巳', is_shen_gong: false, xing_yao: [] },
    { index: 7, name: '仆役', gan: '己', zhi: '午', is_shen_gong: false, xing_yao: [] },
    { index: 8, name: '官禄', gan: '己', zhi: '未', is_shen_gong: false, xing_yao: [] },
    { index: 9, name: '田宅', gan: '戊', zhi: '申', is_shen_gong: false, xing_yao: [] },
    { index: 10, name: '福德', gan: '戊', zhi: '酉', is_shen_gong: false, xing_yao: [] },
  ],
  ming_gong: 0, shen_gong: 11, ju_shu: 0, ju_shu_name: '水二局', ziwei_pos: 0, si_hua: {}, palace_facts: [],
}
const bazhai = {
  ming_gua: { gua: { index: 7, name: '艮', wuxing: '土', yin_yang: '阳' }, group: '西四命', year_boundary: 'gregorian_calendar_year' },
  ba_zhai_dirs: {
    sheng_qi: ['西南'], tian_yi: ['正西'], yan_nian: ['西北'], fu_wei: ['正东'],
    huo_hai: ['正北'], wu_gui: ['东北'], liu_sha: ['正南'], jue_ming: ['东南'],
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

const liuyaoSnapshot = {
  question: '这个项目能成吗',
  casting: { mode: 'auto', rounds: 3, yaos: [[1, 2, 3], [4, 5, 6], [7, 8, 9]], dong_yao: [3], casting_id: 'c1', order: 0 },
  board: { name: '水火既济', gong: '坎', gong_wuxing: '水', xun_kong: ['午', '未'], lines: [], bian_lines: [], ri_chen_gan: '戊', ri_chen_zhi: '子', yue_jian_gan: '甲', yue_jian_zhi: '寅' },
  focus: { yong_shen: '妻财', yong_line: '三爻', world: '三爻', other: '六爻' },
  timing_candidates: [{ note: '午月' }],
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

function routeEngine(page: Page) {
  return page.route('**/api/engine/tools/*', async route => {
    const tool = new URL(route.request().url()).pathname.split('/').pop()
    let data: unknown
    switch (tool) {
      case ENGINE_TOOLS.CityCoords: data = { longitude: 121.47, latitude: 31.23 }; break
      case ENGINE_TOOLS.TianwenTime: data = { solar: '1958-05-12T06:00:00+08:00', lunar: { year: 1958, month: 4, day: 24 } }; break
      case ENGINE_TOOLS.BaziChart: data = baziChart; break
      case ENGINE_TOOLS.BaziFullChart: data = baziFull; break
      case ENGINE_TOOLS.ZiweiChart: data = ziweiChart; break
      case ENGINE_TOOLS.ZiweiFullChart: data = ziweiFull; break
      case ENGINE_TOOLS.BazhaiChart: data = bazhai; break
      case ENGINE_TOOLS.BaziBond: data = bond; break
      case ENGINE_TOOLS.LiuyaoQigua: data = { casting: liuyaoSnapshot.casting }; break
      case ENGINE_TOOLS.LiuyaoChart: data = liuyaoSnapshot; break
      case ENGINE_TOOLS.HuangliDays: data = huangliDays; break
      default: data = {}
    }
    return route.fulfill(json(data))
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
  await expect(page.locator('.ss-chip')).toContainText(['天乙贵人', '华盖']) // 神煞
  await expect(page.locator('.san-yuan .sy-body')).toContainText('胎元 壬申')

  await page.locator('.el-tabs__item:has-text("紫微")').click()
  await expect(page.locator('.palace .pstar').first()).toContainText('紫微庙') // 庙旺平陷
  await expect(page.locator('.palace.shen .pname')).toHaveText('父母') // 身宫

  await page.locator('.el-tabs__item:has-text("八宅")').click()
  await expect(page.locator('.mingua')).toHaveText('艮')
  await expect(page.locator('.fly-cell')).toHaveCount(9) // 流年飞星九宫
  await expect(page.locator('.ln-row')).toContainText('三碧禄存')
})

test('合盘按真实引擎契约渲染（bazi_bond）', async ({ page }) => {
  await routeApp(page)

  await page.goto('/app/synastry')
  await page.waitForSelector('.b-form', { timeout: 10000 })
  await page.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown__item:has-text("妈妈")').first().click()
  await page.waitForTimeout(300)
  await page.locator('.b-form .el-radio-button:has-text("男")').click()
  await page.getByRole('combobox', { name: '出生日期' }).click()
  await page.locator('.el-picker-panel .el-date-table td.available').first().click()
  await page.getByRole('combobox', { name: '出生時間' }).fill('08:00')
  await page.locator('.b-form button[type="submit"]').click()
  await page.waitForSelector('.rel-sec', { timeout: 12000 })
  await expect(page.locator('.rel-text')).toContainText('甲辰') // 日主（golden a_zhu=b_zhu=ri）
  await expect(page.locator('.rel-text')).toContainText('辛克甲') // 天干关系
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
  await page.click('button:has-text("搖卦")')
  await page.waitForSelector('.gua-name', { timeout: 12000 })
  await expect(page.locator('.gua-name')).toHaveText('水火既济') // board.name
  await expect(page.locator('.gua-info')).toContainText('坎') // board.gong
  await expect(page.locator('.rel-text')).toContainText('妻财') // focus.yong_shen
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
