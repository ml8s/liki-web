// hepan-report.js — Report page: loads report via order_id and renders it.
(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var oid = params.get('order_id');

  var container = document.getElementById('report-container');
  if (!container) return;

  if (!oid || oid === 'demo') {
    renderDemo(container);
    return;
  }

  var controller = new AbortController();
  var timeout = setTimeout(function () { controller.abort(); }, 30000);

  container.innerHTML = '<div class="py-12 text-center"><span class="inline-block w-4 h-4 rounded-full bg-stone-300 mr-2 align-middle" style="animation:spin 1s linear infinite"></span><span class="text-stone-400 text-sm">加载中…</span></div>';

  fetch('/api/hepan/orders/' + encodeURIComponent(oid) + '/report', { signal: controller.signal })
    .then(function (r) {
      clearTimeout(timeout);
      if (!r.ok) { throw new Error(r.status === 404 ? '报告未找到' : (r.status >= 500 ? '服务器错误' : '请求失败')); }
      return r.json();
    })
    .then(function (data) {
      var d = data.data || data;
      if (!d || !d.summary || Object.keys(d.summary).length === 0) {
        container.innerHTML = '<p class="py-12 text-center text-stone-400 text-sm">报告内容为空</p>';
        return;
      }
      var html = window.renderReport(data);
      container.innerHTML = typeof html === 'string' ? html : '<p class="py-12 text-center text-stone-400 text-sm">渲染异常</p>';
    })
    .catch(function (e) {
      var msg = e && e.name === 'AbortError' ? '加载超时，请重试'
               : e && e.message || '请求失败';
      container.innerHTML = '<div class="py-12 text-center"><p class="text-stone-400 text-sm mb-3">' + escapeHtml(msg) + '</p><button onclick="window.location.reload()" class="px-4 py-2 border border-stone-300 rounded text-sm text-stone-600 hover:bg-stone-50">重试</button></div>';
    });

  function escapeHtml(s) { return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

  function renderDemo(container) {
    fetch('/js/hepan-report-demo.json')
      .then(function (r) { return r.json(); })
      .then(function (data) {
        var html = window.renderReport(data);
        container.innerHTML = typeof html === 'string' ? html : '<p class="py-12 text-center text-stone-400 text-sm">渲染异常</p>';
      })
      .catch(function () {
        container.innerHTML = '<p class="py-12 text-center text-stone-400 text-sm">加载演示数据失败</p>';
      });
  }
})();
