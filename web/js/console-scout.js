// console-scout.js — Scout-only surface. It must not reference KOL concepts.
(function() {
  'use strict';
  const { $, escapeHTML: esc, money, show, request, tag, setWorkspace, initEmailAuth } = window.LikiConsole;

  initEmailAuth({ onAuthenticated: () => load() });

  $('payout-form').addEventListener('submit', async event => {
    event.preventDefault();
    const method = $('method').value;
    const account = $('account').value.trim();
    show('payout-status', '');
    if (!account) { show('payout-status', '请输入收款账号', true); return; }
    const button = $('payout');
    button.disabled = true;
    button.textContent = '提交中…';
    try {
      await request('/api/scout/payouts', {
        method: 'POST',
        body: JSON.stringify({ method, account }),
      });
      show('payout-status', '提现申请已提交。');
      $('account').value = '';
      await loadPayouts();
      await load();
    } catch (error) {
      show('payout-status', error.message, true);
    } finally {
      button.disabled = false;
      button.textContent = '申请提现';
    }
  });

  async function load() {
    try { await request('/api/me'); } catch {
      setWorkspace(false);
      $('auth-section').classList.remove('hidden');
      return;
    }
    try {
      const session = await request('/api/scout/session');
      setWorkspace(true);
      const overview = session.overview;
      $('metrics').innerHTML = `
        <div class="console-metric"><span class="label">合作创作者</span><span class="value">${overview.recruit_count}</span></div>
        <div class="console-metric"><span class="label">渠道订单</span><span class="value">${overview.order_count}</span></div>
        ${(overview.balances || []).map(item => `
          <div class="console-metric"><span class="label">销售金额 · ${esc(item.currency)}</span><span class="value">${money(item.sales_minor, item.currency)}</span></div>
          <div class="console-metric"><span class="label">待结算 · ${esc(item.currency)}</span><span class="value">${money(item.pending_minor, item.currency)}</span></div>
          <div class="console-metric"><span class="label">已结算 · ${esc(item.currency)}</span><span class="value">${money(item.paid_minor, item.currency)}</span></div>
        `).join('')}`;
      const recruits = overview.recruits || [];
      $('recruits').innerHTML = recruits.length ? recruits.map(item => `
        <tr>
          <td>${esc(item.code)}</td>
          <td>${esc(item.created_at)}</td>
          <td>${item.order_count}</td>
          <td>${(item.commission_balances || []).map(balance => money(balance.amount_minor, balance.currency)).join('<br>')}</td>
        </tr>`).join('') : '<tr><td colspan="4" class="console-empty">暂无合作创作者。</td></tr>';
      await Promise.all([loadOrders(), loadPayouts()]);
    } catch {
      setWorkspace(false);
      show('auth-error', '当前账号没有访问权限。', true);
    }
  }

  async function loadOrders() {
    const result = await request('/api/scout/commissions?limit=100');
    const items = result.sales || [];
    $('orders').innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${esc(item.order_ref)}</td>
        <td>${money(item.amount_minor, item.currency)}</td>
        <td>${money(item.scout_commission_minor, item.currency)}</td>
        <td>${tag(item.status)}</td>
        <td>${esc(item.created_at)}</td>
      </tr>`).join('') : '<tr><td colspan="5" class="console-empty">暂无渠道订单。</td></tr>';
  }

  async function loadPayouts() {
    const result = await request('/api/scout/payouts');
    const items = result.payouts || [];
    $('payouts').innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${money(item.amount_minor, item.currency)}</td>
        <td>${esc(item.method)}</td>
        <td>${tag(item.status)}</td>
        <td>${esc(item.created_at)}</td>
      </tr>`).join('') : '<tr><td colspan="4" class="console-empty">暂无提现记录。</td></tr>';
  }

  load();
})();
