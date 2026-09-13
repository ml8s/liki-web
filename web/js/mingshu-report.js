// mingshu-report.js — Report page: loads report via order_id and renders it.
(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var oid = params.get('order_id');
  var accessToken = params.get('access_token');
  var paymentIntentID = params.get('payment_intent_id');
  var statusToken = params.get('status_token');

  var container = document.getElementById('report-container');
  if (!container) return;

  if (!oid || oid === 'demo') {
    renderDemo(container);
    return;
  }

  var controller = new AbortController();
  var timeout = setTimeout(function () { controller.abort(); }, 30000);

  container.innerHTML = '<div class="py-12 text-center"><span class="inline-block w-4 h-4 rounded-full bg-stone-300 mr-2 align-middle" style="animation:spin 1s linear infinite"></span><span class="text-stone-400 text-sm">加载中…</span></div>';

  function loadReport() {
    return fetch('/api/mingshu/orders/' + encodeURIComponent(oid) + '/report?access_token=' + encodeURIComponent(accessToken || ''), { signal: controller.signal });
  }

  function waitFulfillment() {
    if (!paymentIntentID || !statusToken || !window.LikiPayment) return Promise.resolve();
    clearTimeout(timeout);
    container.innerHTML = '<div class="py-12 text-center text-stone-500 text-sm">报告生成中…</div>';
    return window.LikiPayment.createFulfillmentStatusPoller(paymentIntentID, { token: statusToken }).start().then(function (status) {
      timeout = setTimeout(function () { controller.abort(); }, 30000);
      if (status.fulfillment_status === 'failed') throw new Error('报告生成失败');
    }, function (err) {
      timeout = setTimeout(function () { controller.abort(); }, 30000);
      throw err;
    });
  }

  waitFulfillment().then(loadReport).then(function (r) {
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
      try { container.innerHTML = window.renderReport(data); }
      catch (_) { /* eslint-disable-line no-unused-vars */ container.innerHTML = '<p class="py-12 text-center text-stone-400 text-sm">渲染异常</p>'; }
    })
    .catch(function (e) {
      var msg = e && e.name === 'AbortError' ? '加载超时，请重试'
               : e && e.message || '加载失败';
      container.innerHTML = '<div class="py-12 text-center"><p class="text-stone-400 text-sm mb-3">' + escapeHtml(msg) + '</p><button onclick="window.location.reload()" class="px-4 py-2 border border-stone-300 rounded text-sm text-stone-600 hover:bg-stone-50">重试</button></div>';
    });

  function escapeHtml(s) { return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

  function renderDemo(container) {
    fetch('/js/mingshu-report-demo.json')
      .then(function (r) { return r.json(); })
      .then(function (data) {
        try { container.innerHTML = window.renderReport(data); }
        catch { container.innerHTML = '<p class="py-12 text-center text-stone-400 text-sm">演示数据渲染异常</p>'; }
      })
      .catch(function () {
        container.innerHTML = '<p class="py-12 text-center text-stone-400 text-sm">加载演示数据失败</p>';
      });
  }
})();
