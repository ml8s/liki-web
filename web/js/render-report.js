// render-report.js — 注册表驱动报告渲染器
// 不关心字段名，只按注册表规则 + 值类型递归渲染 JSON → HTML
(function () {
  'use strict';

  function esc(s) { return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }

  // marked 文本渲染
  function toHtml(s) {
    if (!s) return '';
    try {
      var html = window.marked.parse(s, { breaks: true });
      html = html.replace(/<p>([^]*?)<\/p>/g, function(m, c) {
        if (c.indexOf('<br>') === -1) return m;
        return c.split('<br>').map(function(l) { return '<p>' + l.trim() + '</p>'; }).join('\n') + '\n';
      });
      return html;
    } catch {
      var t = esc(s);
      t = t.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
      var paras = t.split(/\n{2,}/);
      if (paras.length > 1) {
        for (var i = 0; i < paras.length; i++) {
          paras[i] = '<p>' + paras[i].replace(/\n/g, '<br>') + '</p>';
        }
        return paras.join('');
      }
      return t.replace(/\n/g, '<br>');
    }
  }

  // ── 字段渲染注册表 ──
  // display: column 双列 | tag 标签行 | table 表格 | list 列表 | recursive 递归展开 | persons 个人信息+报告 | summary 合盘节
  // label: 字段中文化显示名（不设置则用字段原名）
  var FIELD = {
    // 顶层结构
    persons: { display: 'persons', nameColor: 'text-wood' },
    summary: { display: 'summary', titleColor: 'text-wood' },
    // 命理双列
    bazi:   { display: 'column', label: '八字' },
    ziwei:  { display: 'column', label: '紫微' },
    // 特殊标签行（颜色移入规则）
    cross:  { display: 'tag', prefix: '交叉验证：', color: 'text-stone-500' },
    advice: { display: 'tag', prefix: '💡 ', color: 'text-wood' },
    window: { display: 'tag', prefix: '🔑 换运窗口：', color: 'text-fire' },
    // 表格 / 列表
    phases:  { display: 'table', label: '大运', headers: { age: '年龄', pillar: '干支', focus: '重点', analysis: '分析', advice: '建议' }, headerColor: 'text-wood' },
    daxian:  { display: 'table', label: '紫微大限', headers: { age: '年龄', palace: '宫位', focus: '重点', analysis: '分析', cross_bazi: '八字对照' }, headerColor: 'text-wood' },
    liunian: { display: 'table', label: '流年', headers: { year: '年份', bazi: '八字', ziwei: '紫微', cross: '综合' }, headerColor: 'text-wood' },
    items:   { display: 'list' },
    // 独立渲染类型（替代 renderObject 中的硬编码）
    chart:    { display: 'chart', titleColor: 'text-wood' },
    sections: { display: 'sections' },
    // chart 字段中文化映射
    advantages: { display: 'list', itemColor: 'text-wood' },
    risks: { display: 'list', itemColor: 'text-fire' },
    sizhu: { label: '四柱', short: true }, rizhu: { label: '日主', short: true },
    shenqiang: { label: '身强', short: true }, geju: { label: '格局', short: true },
    yongshen: { label: '用神', short: true }, xishen: { label: '喜神', short: true },
    jishen: { label: '忌神', short: true }, mingong: { label: '命宫', short: true },
    shengong: { label: '身宫', short: true }, rating: { label: '评级' },
    patterns: { label: '星曜格局', short: true },
    level: { label: '综合评级', short: true, valueClassMap: { '良配': 'text-wood', '可配': 'text-wood', '慎配': 'text-fire', '不利': 'text-fire' } },
    // persons 中的冗余字段不重复渲染（由 personsHtml 处理）
    name: { display: 'skip' }, label: { display: 'skip' },
    birth: { display: 'skip' }
  };

  // ── 递归渲染引擎 ──

  function renderValue(key, value, level) {
    if (value == null || value === '') return '';
    var rule = FIELD[key];
    if (rule) {
      if (rule.display === 'skip') return '';
      if (rule.display === 'persons') return renderPersons(value);
      if (rule.display === 'summary') return renderSummary(value);
      if (rule.display === 'chart') return renderChart(value);
      if (rule.display === 'sections') return renderSections(value);
      if (rule.display === 'column') return renderColumn(key, value, rule);
      if (rule.display === 'tag') return renderTag(key, value, rule);
      if (rule.display === 'table' && Array.isArray(value)) return renderTable(value, key);
      if (rule.display === 'list' && Array.isArray(value)) return renderList(value, key);
    }
    return renderByType(key, value, level);
  }

  function renderByType(key, value, level) {
    var rule = FIELD[key];
    var label = rule && rule.label ? rule.label : key;
    if (typeof value === 'string') {
      // short 标记 → label: value 一行
      if (rule && rule.short) {
        var valClass = 'text-stone-700';
        if (rule.valueClassMap && rule.valueClassMap[value]) {
          valClass = rule.valueClassMap[value];
        }
        return '<div class="flex items-baseline gap-2 text-sm leading-loose px-1"><span class="text-xs text-stone-400 w-12 flex-shrink-0">' + esc(label) + '</span><span class="' + valClass + '">' + toHtml(value) + '</span></div>';
      }
      // 无映射的长文本 → 直接渲染
      return '<div class="text-sm text-stone-700 leading-loose">' + toHtml(value) + '</div>';
    }
    if (Array.isArray(value)) {
      if (typeof value[0] === 'string') return renderList(value, key);
      if (typeof value[0] === 'object') return renderTable(value);
      return '';
    }
    if (typeof value === 'object') return renderObject(key, value, level);
    return '';
  }

  // ── 渲染器 ──

  // 双列（八字 + 紫微并行）
  function renderColumn(key, value, rule) {
    return '<div class="flex-1 bg-stone-50 rounded-lg p-4"><div class="text-xs font-medium text-stone-600 mb-2">' + esc(rule.label) + '</div><div class="text-sm text-stone-700 leading-loose">' + toHtml(value) + '</div></div>';
  }

  // 标签行（交叉验证 / 💡 建议 / 🔑 窗口）
  function renderTag(key, value, rule) {
    var cls = rule.color || 'text-stone-500';
    return '<p class="' + cls + ' text-sm mt-2">' + esc(rule.prefix) + toHtml(value) + '</p>';
  }

  // 对象 → 卡片（不套 section，由调用者负责外层的 card）
  function renderObject(key, value, level) {
    if (typeof value !== 'object' || !value) return '';
    var keys = Object.keys(value);
    if (!keys.length) return '';
    var h = '';
    // 普通对象：递归渲染每个字段
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      var v = value[k];
      var child = renderValue(k, v, level + 1);
      if (child) h += child;
    }
    return h;
  }

  // 对象数组 → 表格
  function renderTable(items, fieldKey) {
    if (!items || !items.length) return '';
    var rule = FIELD[fieldKey];
    var headers = rule && rule.headers || {};
    var headerColor = (rule && rule.headerColor) || 'text-stone-600';
    var h = '';
    if (rule && rule.label) h += '<div class="mt-4"><div class="text-xs font-medium text-stone-600 mb-2 uppercase tracking-wider">' + esc(rule.label) + '</div>';
    var cols = Object.keys(items[0]);
    var thead = '<thead><tr class="border-b border-stone-200">';
    for (var j = 0; j < cols.length; j++) {
      thead += '<th class="py-1.5 pr-2 text-left font-medium ' + headerColor + ' text-xs">' + esc(headers[cols[j]] || cols[j]) + '</th>';
    }
    thead += '</tr></thead>';
    var rows = '';
    for (var i = 0; i < items.length; i++) {
      rows += '<tr class="border-b border-stone-50 hover:bg-stone-50">';
      for (j = 0; j < cols.length; j++) {
        rows += '<td class="py-1.5 pr-2 text-stone-500 text-xs">' + toHtml(String(items[i][cols[j]] || '')) + '</td>';
      }
      rows += '</tr>';
    }
    return h + '<div class="mt-3 overflow-x-auto"><table class="w-full text-xs striped">' + thead + rows + '</table></div>' + (rule && rule.label ? '</div>' : '');
  }

  // 字符串数组 → 列表
  function renderList(items, fieldKey) {
    if (!items || !items.length) return '';
    var itemColor = 'text-stone-600';
    if (fieldKey) {
      var rule = FIELD[fieldKey];
      if (rule && rule.itemColor) itemColor = rule.itemColor;
    }
    var h = '<ul class="list-disc pl-4 mt-2 space-y-1">';
    for (var i = 0; i < items.length; i++) {
      h += '<li class="' + itemColor + ' text-sm">' + toHtml(items[i]) + '</li>';
    }
    return h + '</ul>';
  }

  // 个人信息卡片（单人或多人）
  function personsHtml(persons) {
    if (!persons || !persons.length) return '';
    var h = '';
    for (var i = 0; i < persons.length; i++) {
      var p = persons[i];
      if (i === 0) h += '<div class="bg-white rounded-lg border border-stone-200 p-4 mb-4">';
      if (i > 0) h += '</div><div class="bg-white rounded-lg border border-stone-200 p-4 mb-4 mt-6">';
      // 如果多人，每人的卡片独立显示，卡片标题显示 label
      if (persons.length > 1 && p.label) h += '<div class="text-xs font-medium text-stone-400 mb-2 uppercase">—— ' + esc(p.label) + ' ——</div>';
      h += '<div class="flex flex-wrap items-baseline gap-x-2 gap-y-1 mb-2">';
      if (p.label && (persons.length > 1 || p.label !== '本人')) h += '<span class="text-xs font-medium text-stone-400 uppercase">' + esc(p.label) + '</span>';
      if (p.name) {
        var nameColor = (FIELD.persons && FIELD.persons.nameColor) || 'text-stone-800';
        h += '<span class="text-base font-bold ' + nameColor + '">' + esc(p.name) + '</span>';
      }
      if (p.birth) h += '<span class="text-xs text-stone-400">· ' + esc(p.birth) + '</span>';
      h += '</div>';
      var cells = [];
      if (p.sizhu) cells.push('<div><span class="text-xs text-stone-400">四柱</span><div class="text-sm text-stone-700 font-mono">' + esc(p.sizhu) + '</div></div>');
      if (p.rizhu) cells.push('<div><span class="text-xs text-stone-400">日主</span><div class="text-sm text-stone-700">' + esc(p.rizhu) + '</div></div>');
      if (p.shenqiang) cells.push('<div><span class="text-xs text-stone-400">身强</span><div class="text-sm text-stone-700">' + esc(p.shenqiang) + '</div></div>');
      if (p.yongshen) cells.push('<div><span class="text-xs text-stone-400">用神</span><div class="text-sm text-stone-700">' + esc(p.yongshen) + '</div></div>');
      h += '<div class="flex flex-wrap gap-x-6 gap-y-1">' + cells.join('') + '</div>';
      var cells2 = [];
      if (p.mingong) cells2.push('<div><span class="text-xs text-stone-400">命宫</span><div class="text-sm text-stone-700">' + esc(p.mingong) + '</div></div>');
      if (p.shengong) cells2.push('<div><span class="text-xs text-stone-400">身宫</span><div class="text-sm text-stone-700">' + esc(p.shengong) + '</div></div>');
      if (cells2.length) h += '<div class="flex flex-wrap gap-x-6 gap-y-1 mt-1">' + cells2.join('') + '</div>';
    }
    return h + '</div>';
  }

  // chart 速览表（key-value 两列）
  function renderChart(value) {
    if (typeof value !== 'object' || !value) return '';
    var keys = Object.keys(value);
    if (!keys.length) return '';
    var rows = '';
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      var v = value[k];
      if (v == null || v === '' || (Array.isArray(v) && v.length === 0)) continue;
      var label = (FIELD[k] && FIELD[k].label) || k;
      var valRule = FIELD[k];
      var valClass = 'text-stone-800';
      if (valRule && valRule.valueClassMap && valRule.valueClassMap[v]) {
        valClass = valRule.valueClassMap[v];
      }
      rows += '<tr class="border-b border-stone-100"><td class="py-1.5 pr-3 font-medium text-stone-600 text-xs">' + esc(label) + '</td><td class="' + valClass + ' text-xs">' + (typeof v === 'string' ? esc(v) : Array.isArray(v) ? esc(v.join(', ')) : '') + '</td></tr>';
    }
    if (!rows) return '';
    return '<div class="overflow-x-auto"><table class="w-full text-xs striped"><tbody>' + rows + '</tbody></table></div>';
  }

  // sections 子节列表（inline 风格，在 card 内部使用）
  function renderSections(value) {
    if (typeof value !== 'object' || !value) return '';
    var keys = Object.keys(value);
    if (!keys.length) return '';
    var h = '';
    for (var i = 0; i < keys.length; i++) {
      var secKey = keys[i];
      var secVal = value[secKey];
      if (typeof secVal === 'string') {
        h += '<div class="mt-4"><div class="text-xs font-medium text-stone-600 mb-2 uppercase tracking-wider">' + esc(secKey) + '</div><div class="text-sm text-stone-700 leading-loose">' + toHtml(secVal) + '</div></div>';
      }
    }
    return h;
  }

  // ── 顶层渲染器 ──

  // 个人信息 + bazi/ziwei 报告
  function renderPersons(persons) {
    if (!persons || !persons.length) return '';
    var h = '';
    var pkg = ['bazi', 'ziwei'];
    for (var pi = 0; pi < persons.length; pi++) {
      var person = persons[pi];
      h += personsHtml([person]);
      for (var pi2 = 0; pi2 < pkg.length; pi2++) {
        var key = pkg[pi2];
        var report = person[key];
        if (!report) continue;
        var reportLabel = (key === 'bazi' ? '八字' : '紫微斗数');
        if (report.chart) {
          var chartKeys = Object.keys(report.chart).filter(function(k) {
            var v = report.chart[k];
            return v != null && v !== '' && !(Array.isArray(v) && v.length === 0);
          });
          if (chartKeys.length) {
            h += '<section class="card p-4 mb-6">';
            var chartTitleColor = (FIELD.chart && FIELD.chart.titleColor) || 'text-stone-800';
            h += '<h2 class="text-lg font-bold ' + chartTitleColor + ' mb-3">' + reportLabel + '速览</h2>';
            h += renderChart(report.chart);
            h += '</section>';
          }
        }
        if (report.sections) {
          var secKeys = Object.keys(report.sections);
          for (var si = 0; si < secKeys.length; si++) {
            var secVal = report.sections[secKeys[si]];
            if (typeof secVal !== 'string' || !secVal) continue;
            h += '<section class="card p-4 mb-6">';
            h += '<h2 class="text-lg font-bold text-stone-800 mb-3">' + esc(secKeys[si]) + '</h2>';
            h += '<div class="text-sm text-stone-700 leading-loose">' + toHtml(secVal) + '</div>';
            h += '</section>';
          }
        }
      }
    }
    return h;
  }

  // 合盘分析节（summary）
  function renderSummary(summary) {
    if (!summary) return '';
    var sumKeys = Object.keys(summary);
    var h = '';
    for (var i = 0; i < sumKeys.length; i++) {
      var s = summary[sumKeys[i]];
      if (!s || typeof s !== 'object') continue;
      h += '<section class="card p-4 mb-6">';
      if (s.title) {
        var titleColor = (FIELD.summary && FIELD.summary.titleColor) || 'text-stone-800';
        h += '<h2 class="text-lg font-bold ' + titleColor + ' mb-3">' + esc(s.title) + '</h2>';
      }
      if (s.bazi && s.ziwei) {
        h += '<div class="flex flex-col md:flex-row gap-4 mb-2">';
        h += renderColumn('bazi', s.bazi, FIELD.bazi);
        h += renderColumn('ziwei', s.ziwei, FIELD.ziwei);
        h += '</div>';
        var subKeys = Object.keys(s);
        for (var j = 0; j < subKeys.length; j++) {
          if (subKeys[j] === 'title' || subKeys[j] === 'bazi' || subKeys[j] === 'ziwei') continue;
          var child = renderValue(subKeys[j], s[subKeys[j]], 1);
          if (child) h += child;
        }
      } else {
        subKeys = Object.keys(s);
        for (j = 0; j < subKeys.length; j++) {
          if (subKeys[j] === 'title') continue;
          child = renderValue(subKeys[j], s[subKeys[j]], 1);
          if (child) h += child;
        }
      }
      h += '</section>';
    }
    return h;
  }

  // ── 主入口 ──
  window.renderReport = function (data) {
    if (!data) return '<p class="py-12 text-center text-stone-400 text-sm">报告为空</p>';
    var r = data.data || data;
    if (!r) return '<p class="py-12 text-center text-stone-400 text-sm">报告为空</p>';

    if (!r.title && !r.persons && !r.summary) return '<p class="py-12 text-center text-stone-400 text-sm">报告内容为空</p>';

    var h = '<div class="report max-w-2xl mx-auto py-8 px-4">';
    if (r.title) h += '<h1 class="text-2xl font-bold text-stone-800 mb-6">' + esc(r.title) + '</h1>';

    // 遍历顶层 key，顺序由 JSON 数据的 key 顺序决定
    var topKeys = Object.keys(r);
    for (var i = 0; i < topKeys.length; i++) {
      var key = topKeys[i];
      if (key === 'title' || key === 'data') continue;
      var child = renderValue(key, r[key], 0);
      if (child) h += child;
    }

    return h + '</div>';
  };
})();
