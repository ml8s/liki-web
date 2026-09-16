import { describe, it, expect, beforeAll } from 'vitest';
import { readJS } from './helpers.js';

let code;

beforeAll(() => {
  // stub marked — 与 render-report.js 中 toHtml 的调用方式一致
  //   window.marked.parse(esc(s), { breaks: true })
  globalThis.marked = {
    parse: (s, opts) => {
      // s 已经是 esc() 后的文本
      var t = String(s || '');
      // **粗体** → <strong>
      t = t.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
      // \n\n 分段落
      var paras = t.split(/\n{2,}/);
      if (paras.length > 1) {
        for (var i = 0; i < paras.length; i++) {
          // breaks:true — 段内 \n 转 <br>
          paras[i] = '<p>' + paras[i].replace(/\n/g, '<br>\n') + '</p>';
        }
        return paras.join('\n') + '\n';
      }
      if (t.indexOf('\n') > -1) {
        return '<p>' + t.replace(/\n/g, '<br>\n') + '</p>\n';
      }
      return '<p>' + t + '</p>\n';
    }
  };
  code = readJS('render-report');
  globalThis.window = { marked: globalThis.marked };
  globalThis.document = {};
  eval(code);
});

describe('toHtml (via renderReport internals)', () => {
  it('escapes HTML special characters', () => {
    const r = globalThis.window.renderReport({ title: '<script>alert(1)</script>' });
    expect(r).not.toContain('<script>');
    expect(r).toContain('&lt;script&gt;');
  });

  it('replaces \\n with <br>', () => {
    const r = globalThis.window.renderReport({
      summary: { personality: { title: '性格画像', bazi: '第一行\n第二行', cross: '', advice: '' } }
    });
    // <br> 被拆成独立 <p>，不再是内联 <br>
    expect(r).toContain('第一行');
    expect(r).toContain('第二行');
    expect(r).toContain('</p>\n<p>');
  });

  it('splits \\n\\n into <p> paragraphs', () => {
    const r = globalThis.window.renderReport({
      summary: { personality: { title: '性格画像', bazi: '段落一\n\n段落二', cross: '', advice: '' } }
    });
    // marked 把 \n\n 分成两个 <p>
    expect(r).toContain('</p>\n<p>');
    expect(r).toContain('段落一');
    expect(r).toContain('段落二');
  });

  it('converts **bold** to <strong>', () => {
    const r = globalThis.window.renderReport({
      summary: { personality: { title: '性格画像', bazi: '**四柱排列表**：甲子', cross: '', advice: '' } }
    });
    expect(r).toContain('<strong>四柱排列表</strong>');
  });
});

describe('chartTable', () => {
  it('renders key-value table from bazi chart', () => {
    const r = globalThis.window.renderReport({
      persons: [{
        bazi: {
          chart: { sizhu: '甲子', rizhu: '己土' },
          sections: {}
        }
      }]
    });
    expect(r).toContain('四柱');
    expect(r).toContain('甲子');
    expect(r).toContain('日主');
    expect(r).toContain('己土');
  });

  it('returns empty string for null chart', () => {
    const r = globalThis.window.renderReport({
      persons: [{ bazi: { sections: {} } }]
    });
    expect(r).not.toContain('四柱');
  });
});

describe('phaseTable', () => {
  it('renders decade fortune rows', () => {
    const r = globalThis.window.renderReport({
      summary: {
        fortune: {
          title: '大运与流年',
          phases: [{ age: '35-44', pillar: '壬戌', focus: '正财运', analysis: '事业上升', advice: '把握机会' }]
        }
      }
    });
    expect(r).toContain('35-44');
    expect(r).toContain('壬戌');
    expect(r).toContain('正财运');
    expect(r).toContain('把握机会');
  });
});

describe('yearTable', () => {
  it('renders daxian rows with analysis column', () => {
    const r = globalThis.window.renderReport({
      summary: {
        fortune: {
          title: '大运与流年',
          daxian: [{ age: '35-44', palace: '财帛宫', focus: '财运', analysis: '求财欲望强', cross_bazi: '同步' }]
        }
      }
    });
    expect(r).toContain('财帛宫');
    expect(r).toContain('求财欲望强');
    expect(r).toContain('同步');
    // 表头中文
    expect(r).toContain('>年龄<');
    expect(r).toContain('>宫位<');
    expect(r).toContain('>八字对照<');
  });

  it('renders liunian rows', () => {
    const r = globalThis.window.renderReport({
      summary: {
        fortune: {
          title: '大运与流年',
          liunian: [{ year: '2025乙巳', bazi: '七杀透干', ziwei: '入夫妻宫', cross: '有突破' }]
        }
      }
    });
    expect(r).toContain('2025乙巳');
    expect(r).toContain('七杀透干');
  });
});

describe('itemList', () => {
  it('renders milestone items as bullet list', () => {
    const r = globalThis.window.renderReport({
      summary: {
        milestones: {
          title: '关键提醒',
          items: ['注意健康', '把握机遇'],
          advice: '提前规划'
        }
      }
    });
    expect(r).toContain('<ul');
    expect(r).toContain('注意健康');
    expect(r).toContain('把握机遇');
    expect(r).toContain('💡');
    expect(r).toContain('提前规划');
  });
});

describe('personsHtml', () => {
  it('renders person info header', () => {
    const r = globalThis.window.renderReport({
      title: 'Liki 命书',
      persons: [{ name: '张明', label: '本人', birth: '1984年', sizhu: '甲子 丙寅', rizhu: '己土', shenqiang: '身弱', mingong: '天同太阴', shengong: '天机禄存', yongshen: '火' }],
      summary: {}
    });
    // 单人报告不显示"本人"标签
    expect(r).not.toContain('本人');
    expect(r).toContain('张明');
    expect(r).toContain('1984年');
    expect(r).toContain('四柱');
    expect(r).toContain('甲子 丙寅');
    expect(r).toContain('日主');
    expect(r).toContain('己土');
    expect(r).toContain('命宫');
    expect(r).toContain('天同太阴');
    expect(r).toContain('身宫');
    expect(r).toContain('天机禄存');
    expect(r).toContain('用神');
    expect(r).toContain('火');
    expect(r).toContain('身强');
    expect(r).toContain('身弱');
  });

  it('renders multiple persons for hepan', () => {
    const r = globalThis.window.renderReport({
      title: 'Liki 合盘',
      persons: [
        { name: '张明', label: '甲方', birth: '1984年', sizhu: '甲子', rizhu: '己土', mingong: '天同', yongshen: '火' },
        { name: '李芳', label: '乙方', birth: '1985年', sizhu: '乙丑', rizhu: '辛金', mingong: '太阳', yongshen: '水' }
      ],
      summary: {}
    });
    expect(r).toContain('甲方');
    expect(r).toContain('乙方');
    expect(r).toContain('甲子');
    expect(r).toContain('乙丑');
  });
});

describe('summarySection', () => {
  it('renders bazi+ziwei+cross+advice in section card', () => {
    const r = globalThis.window.renderReport({
      summary: {
        personality: {
          title: '性格画像',
          bazi: '日主己土',
          ziwei: '天同太阴',
          cross: '两系一致',
          advice: '发挥学习能力'
        }
      }
    });
    expect(r).toContain('性格画像');
    expect(r).toContain('日主己土');
    expect(r).toContain('天同太阴');
    expect(r).toContain('交叉验证：');
    expect(r).toContain('两系一致');
    expect(r).toContain('💡');
    expect(r).toContain('发挥学习能力');
  });

  it('renders window field for career section', () => {
    const r = globalThis.window.renderReport({
      summary: {
        career: {
          title: '事业财运',
          bazi: '正印格',
          ziwei: '武曲天府',
          cross: '靠专业积累',
          window: '35岁前后换运',
          advice: '深耕专业'
        }
      }
    });
    expect(r).toContain('🔑 换运窗口：');
    expect(r).toContain('35岁前后换运');
  });
});

describe('renderReport — integration', () => {
  it('renders empty state for null data', () => {
    const r = globalThis.window.renderReport(null);
    expect(r).toContain('报告为空');
  });

  it('renders empty prompt for bare object', () => {
    const r = globalThis.window.renderReport({});
    expect(r).toContain('报告内容为空');
  });

  it('renders full mingshu report with all sections', () => {
    const data = {
      title: 'Liki 命书',
      persons: [{ name: '测试', label: '本人', birth: '测试数据', sizhu: '甲子', rizhu: '己土', mingong: '天同', shengong: '天机', yongshen: '火',
        bazi: {
          chart: { sizhu: '甲子', rizhu: '己土', geju: '正印格' },
          sections: { '命盘总览': '正印格顺用', '用神喜忌': '用火喜土' }
        },
        ziwei: {
          chart: { mingong: '天同太阴', rating: '上等', patterns: ['机月同梁'] },
          sections: { '命盘总览': '双水入命', '四化分布': '天机化禄' }
        }
      }],
      summary: {
        personality: { title: '性格画像', bazi: '己土', ziwei: '太阴', cross: '一致', advice: '多学习' },
        career: { title: '事业财运', bazi: '正印格', ziwei: '武曲', cross: '一致', advice: '深耕' },
        love: { title: '情感婚姻', bazi: '七杀', ziwei: '天相', cross: '一致', advice: '包容' },
        health: { title: '健康提示', bazi: '脾胃弱', ziwei: '注意心脏', cross: '一致', advice: '定期体检。不做医学诊断。' },
        fortune: {
          title: '大运与流年',
          phases: [{ age: '35-44', pillar: '壬戌', focus: '正财运', analysis: '上升', advice: '把握' }],
          daxian: [{ age: '35-44', palace: '财帛宫', focus: '财运', analysis: '求财旺', cross_bazi: '同步' }],
          liunian: [{ year: '2025', bazi: '七杀', ziwei: '迁移', cross: '突破' }],
          advice: '中年是黄金期'
        },
        milestones: { title: '关键提醒', items: ['注意健康'], advice: '提前规划' }
      },
    };
    const html = globalThis.window.renderReport(data);

    // 顶层结构
    expect(html).toContain('Liki 命书');
    expect(html).toContain('测试');
    expect(html).not.toContain('本人');

    // summary 所有节
    expect(html).toContain('性格画像');
    expect(html).toContain('事业财运');
    expect(html).toContain('情感婚姻');
    expect(html).toContain('健康提示');
    expect(html).toContain('大运与流年');
    expect(html).toContain('关键提醒');

    // bazi/ziwei 子报告
    expect(html).toContain('八字速览');
    expect(html).toContain('紫微斗数速览');
    expect(html).toContain('命盘总览');
    expect(html).toContain('用神喜忌');
    expect(html).toContain('四化分布');
  });

  it('handles data.data envelope', () => {
    const r = globalThis.window.renderReport({
      data: { title: 'Liki 命书', summary: {} }
    });
    expect(r).toContain('Liki 命书');
  });
});

describe('toHtml security（LLM 输出不可信——esc→marked→DOMPurify 链）', () => {
  it('raw HTML in long-text fields cannot survive as native tags', () => {
    const evil = '<img src=x onerror=alert(1)>**加粗**';
    const r = globalThis.window.renderReport({
      summary: { personality: { title: '性格', bazi: evil, cross: '', advice: '' } }
    });
    expect(r).not.toMatch(/<img/);
    expect(r).toContain('&lt;img');
    expect(r).toContain('<strong>加粗</strong>');
  });

  it('fails safe when marked is missing (catch → esc)', () => {
    const saved = globalThis.window.marked;
    globalThis.window.marked = undefined;
    try {
      const r = globalThis.window.renderReport({
        summary: { personality: { title: 'x', bazi: '<b>hi</b>', cross: '', advice: '' } }
      });
      expect(r).toContain('&lt;b&gt;');
      expect(r).not.toContain('<b>hi</b>');
    } finally {
      globalThis.window.marked = saved;
    }
  });
});
