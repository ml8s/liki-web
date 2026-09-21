// console-kol.js — KOL-only surface. It must not reference Scout concepts.
(function() {
  'use strict';
  const { $, escapeHTML: esc, money, show, request, tag, setWorkspace, initEmailAuth } = window.LikiConsole;

  initEmailAuth({ onAuthenticated: () => load() });

  $('copy').addEventListener('click', async () => {
    await navigator.clipboard.writeText($('referral-link').value);
    show('overview-status', '链接已复制。');
  });

  $('activate-form').addEventListener('submit', async event => {
    event.preventDefault();
    const button = $('activate');
    button.disabled = true;
    button.textContent = '开通中…';
    try {
      await request('/api/referral/register', { method: 'POST', body: '{}' });
      await load();
    } catch (error) {
      show('overview-status', error.message, true);
    } finally {
      button.disabled = false;
      button.textContent = '开通合作权限';
    }
  });

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
      await request('/api/kol/payouts', {
        method: 'POST',
        body: JSON.stringify({ method, account }),
      });
      show('payout-status', '提现申请已提交。');
      $('account').value = '';
      await loadPayouts();
    } catch (error) {
      show('payout-status', error.message, true);
    } finally {
      button.disabled = false;
      button.textContent = '申请提现';
    }
  });

  async function loadKolWorkspace() {
    const status = await request('/api/kol/status');
    if (!status.registered) {
      setWorkspace(true);
      $('metrics').innerHTML = '';
      $('referral-link').value = '';
      $('commissions').innerHTML = '<tr><td colspan="5" class="console-empty">暂无收益记录。</td></tr>';
      $('payouts').innerHTML = '<tr><td colspan="4" class="console-empty">暂无提现记录。</td></tr>';
      if (status.eligible) {
        $('activate-form').classList.remove('hidden');
        show('overview-status', '你已符合开通条件，可立即开通合作权限。');
      } else {
        $('activate-form').classList.add('hidden');
        show('overview-status', '请先购买并激活 Liki Chat，再开通合作权限。');
      }
      return;
    }
    if (status.status !== 'active') {
      setWorkspace(true);
      $('activate-form').classList.add('hidden');
      show('overview-status', '当前合作权限已冻结，请联系运营处理。', true);
      return;
    }

    const session = await request('/api/kol/session');
    const overview = session.overview;
    const origin = location.origin;
    $('referral-link').value = `${origin}/r/${session.partner.code}`;
    const balances = overview.balances || [];
    $('metrics').innerHTML = `
      <div class="console-metric"><span class="label">推荐注册</span><span class="value">${overview.attributed_users}</span></div>
      ${balances.map(item => `
        <div class="console-metric"><span class="label">累计收益 · ${esc(item.currency)}</span><span class="value">${money(item.total_minor, item.currency)}</span></div>
        <div class="console-metric"><span class="label">待结算 · ${esc(item.currency)}</span><span class="value">${money(item.pending_minor, item.currency)}</span></div>
        <div class="console-metric"><span class="label">已结算 · ${esc(item.currency)}</span><span class="value">${money(item.paid_minor, item.currency)}</span></div>
      `).join('')}`;
    await Promise.all([loadCommissions(), loadPayouts()]);
  }

  async function loadCommissions() {
    const result = await request('/api/kol/commissions?limit=100');
    const items = result.commissions || [];
    $('commissions').innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${esc(item.business_order_id)}</td>
        <td>${money(item.amount_minor, item.currency)}</td>
        <td>${money(item.referrer_commission_minor, item.currency)}</td>
        <td>${tag(item.status)}</td>
        <td>${esc(item.created_at)}</td>
      </tr>`).join('') : '<tr><td colspan="5" class="console-empty">暂无收益记录。</td></tr>';
  }

  async function loadPayouts() {
    const result = await request('/api/kol/payouts');
    const items = result.payouts || [];
    $('payouts').innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${money(item.amount_minor, item.currency)}</td>
        <td>${esc(item.method)}</td>
        <td>${tag(item.status)}</td>
        <td>${esc(item.created_at)}</td>
      </tr>`).join('') : '<tr><td colspan="4" class="console-empty">暂无提现记录。</td></tr>';
  }

  async function load() {
    try { await request('/api/me'); } catch {
      setWorkspace(false);
      $('auth-section').classList.remove('hidden');
      return;
    }
    $('auth-section').classList.add('hidden');
    setWorkspace(true);
    try {
      await loadKolWorkspace();
    } catch (error) {
      show('overview-status', error.message, true);
    }
  }

  load();
})();
