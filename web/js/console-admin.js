// console-admin.js — internal back-office application.
(function() {
  'use strict';
  const { $, escapeHTML: h, setStatus: show, request, maskAccount, busyButton: busy, initEmailAuth } = window.LikiConsole;
  const API = '/api/admin';

  const setAuthed = authed => {
    $('auth-section').classList.toggle('hidden', authed);
    $('main-section').classList.toggle('hidden', !authed);
  };

  const actionDialog = options => new Promise(resolve => {
    const dialog = $('action-dialog');
    $('action-dialog-title').textContent = options.title;
    $('action-description').textContent = options.description || '';
    $('action-reason-field').classList.toggle('hidden', !options.reason);
    $('action-reference-field').classList.toggle('hidden', !options.reference);
    $('action-status-field').classList.toggle('hidden', !options.status);
    $('action-reason').required = Boolean(options.reason);
    $('action-reference').required = Boolean(options.reference);
    $('action-reason').value = '';
    $('action-reference').value = '';
    $('action-confirm').textContent = options.confirmText || '确认';
    dialog.addEventListener('close', () => {
      const confirmed = dialog.returnValue === 'confirm';
      resolve(confirmed ? {
        reason: $('action-reason').value.trim(),
        reference: $('action-reference').value.trim(),
        status: $('action-status').value
      } : null);
    }, { once: true });
    dialog.showModal();
  });

  initEmailAuth({
    onAuthenticated: () => checkAdminSession(),
    codeFormId: 'send-code-form',
    verifyFormId: 'verify-code-form',
    emailId: 'login-email',
    codeId: 'login-code',
    sendId: 'send-code-btn',
    verifyId: 'verify-code-btn',
    statusId: 'login-status',
    errorId: 'login-error',
    logoutId: null,
  });

async function checkAdminSession() {
  try {
    const session = await request(`${API}/session`);
    currentAdminEmail = session.email || '';
    $('admin-capability').innerHTML = Object.entries(adminCapabilityLabels)
      .map(([value, label]) => `<option value="${h(value)}">${h(label)}</option>`).join('');
    setAuthed(true);
    loadAdmins();
    loadVCList();
    loadReferralAdmin();
  } catch {
    setAuthed(false);
  }
}

$('create-order-form').addEventListener('submit', async event => {
  event.preventDefault();
  const email = $('co-email').value.trim();
  const expire = $('co-expire').value;
  show('co-status', '');
  if (!email) { show('co-status', '请输入邮箱', true); return; }
  let expiresAt = '2999-12-31 23:59:59';
  if (expire === '7d') expiresAt = new Date(Date.now() + 7 * 86400000).toISOString().slice(0, 19).replace('T', ' ');
  if (expire === '30d') expiresAt = new Date(Date.now() + 30 * 86400000).toISOString().slice(0, 19).replace('T', ' ');
  const button = $('co-btn');
  busy(button, true, '创建中…');
  try {
    const data = await request(`${API}/orders`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, expires_at: expiresAt }),
    });
    show('co-status', `已创建 ${data.order_id}`);
    $('search-email').value = email;
    doSearch();
  } catch (error) {
    show('co-status', error.message || '创建失败', true);
  } finally { busy(button, false); }
});

$('search-form').addEventListener('submit', event => { event.preventDefault(); doSearch(); });
async function doSearch() {
  const email = $('search-email').value.trim();
  if (!email) return;
  try {
    const data = await request(`${API}/orders?email=${encodeURIComponent(email)}`);
    const orders = data.orders || [];
    $('results-section').classList.remove('hidden');
    const container = $('results-container');
    if (!orders.length) {
      container.innerHTML = '<div class="empty-state">未找到订单。请确认邮箱后重试。</div>';
      return;
    }
    container.innerHTML = `
      <table>
        <thead><tr><th>订单 ID</th><th>状态</th><th>过期时间</th><th>创建时间</th><th>操作</th></tr></thead>
        <tbody>
          ${orders.map(o => `
            <tr>
              <td>${h(o.order_id)}</td><td>${statusTag(o.status)}</td>
              <td>${h(o.expires_at)}</td><td>${h(o.created_at)}</td>
              <td><button class="btn btn-secondary" data-expire="${h(o.order_id)}" data-current="${h(o.expires_at)}">改过期</button></td>
            </tr>`).join('')}
        </tbody>
      </table>`;
  } catch (error) {
    $('results-section').classList.remove('hidden');
    $('results-container').innerHTML = `<div class="empty-state status-error">${h(error.message)}</div>`;
  }
}

$('results-container').addEventListener('click', async event => {
  const button = event.target.closest('[data-expire]');
  if (!button) return;
  const value = window.prompt('修改过期时间（留空 = 永久）', button.dataset.current);
  if (value === null) return;
  await request(`${API}/orders/${encodeURIComponent(button.dataset.expire)}/expire`, {
    method: 'PATCH', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ expires_at: value.trim() || '2999-12-31 23:59:59' }),
  });
  doSearch();
});

const vcLimit = 20;
let vcOffset = 0;

$('vc-generate-form').addEventListener('submit', async event => {
  event.preventDefault();
  const count = Number($('vc-count').value) || 1;
  const button = $('vc-generate-btn');
  show('vc-status', '');
  busy(button, true, '生成中…');
  try {
    const data = await request(`${API}/verification-codes`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count }),
    });
    $('vc-new-codes').innerHTML = (data.codes || []).map(item =>
      `<button class="btn btn-secondary" data-code="${h(item.code)}">${h(formatCode(item.code))}</button>`).join('');
    $('vc-generated').classList.remove('hidden');
    show('vc-status', `已生成 ${data.codes?.length || 0} 个激活码`);
    loadVCList();
  } catch (error) {
    show('vc-status', error.message || '生成失败', true);
  } finally { busy(button, false); }
});

$('vc-new-codes').addEventListener('click', async event => {
  const button = event.target.closest('[data-code]');
  if (!button) return;
  await navigator.clipboard.writeText(button.dataset.code);
  show('vc-status', `已复制：${button.dataset.code}`);
});

$('vc-list-btn').addEventListener('click', () => { vcOffset = 0; loadVCList(); });
$('vc-prev-btn').addEventListener('click', () => { vcOffset = Math.max(0, vcOffset - vcLimit); loadVCList(); });
$('vc-next-btn').addEventListener('click', () => { vcOffset += vcLimit; loadVCList(); });

async function loadVCList() {
  try {
    const data = await request(`${API}/verification-codes?limit=${vcLimit}&offset=${vcOffset}`);
    const codes = data.codes || [];
    $('vc-list').innerHTML = codes.length ? `
      <table>
        <thead><tr><th>激活码</th><th>状态</th><th>邮箱</th><th>创建时间</th><th>使用时间</th></tr></thead>
        <tbody>${codes.map(c => `
          <tr><td>${h(formatCode(c.code))}</td><td>${statusTag(c.status)}</td>
          <td>${h(c.email || '-')}</td><td>${h(c.created_at)}</td><td>${h(c.used_at || '-')}</td></tr>`).join('')}
        </tbody>
      </table>` : '<div class="empty-state">暂无激活码。</div>';
    const total = Number(data.total || 0);
    $('vc-pager').classList.toggle('hidden', total <= vcLimit);
    $('vc-page-info').textContent = `${vcOffset + 1} – ${Math.min(vcOffset + vcLimit, total)} / ${total}`;
  } catch (error) {
    show('vc-status', error.message, true);
  }
}

$('vc-export-btn').addEventListener('click', async () => {
  const data = await request(`${API}/verification-codes?limit=1000&offset=0`);
  const codes = data.codes || [];
  if (!codes.length) { show('vc-status', '暂无数据可导出', true); return; }
  const csv = ['code,status,email,created_at,used_at'].concat(codes.map(c =>
    [c.code, c.status, c.email || '', c.created_at || '', c.used_at || ''].join(','))).join('\n');
  const url = URL.createObjectURL(new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' }));
  const link = document.createElement('a');
  link.href = url; link.download = 'verification-codes.csv'; link.click();
  URL.revokeObjectURL(url);
});

const referralApplicationsBody = $('referral-applications-body');
const referralPartnersBody = $('referral-partners-body');
const referralPayoutsBody = $('referral-payouts-body');
const referralActionsBody = $('referral-actions-body');
const financeTasksBody = $('finance-tasks-body');

const adminCapabilityLabels = {
  'partner.read': '合作伙伴查看',
  'partner.approve': '合作伙伴审批',
  'partner.freeze': '合作伙伴冻结',
  'partner.unfreeze': '合作伙伴解冻',
  'commission.read': '佣金查看',
  'commission.void': '佣金作废',
  'payout.read': '提现查看',
  'payout.approve': '提现批准',
  'payout.reject': '提现拒绝',
  'payout.mark_paid': '提现打款',
  'audit.read': '审计查看',
  'admin.user.manage': '管理员管理',
  'admin.super': '超级管理员',
};
const financeTaskTypeLabels = {
  commission_refund_clawback: '佣金退款追回',
  user_erasure_review: '注销后待结算',
};
let currentAdminEmail = '';

async function loadAdmins() {
  const statusEl = $('admin-status');
  statusEl.textContent = '载入中…';
  statusEl.classList.remove('status-error');
  try {
    const data = await request(`${API}/admins`);
    const items = data.admins || [];
    $('admins-container').innerHTML = items.length ? `
      <table>
        <thead><tr><th>邮箱</th><th>状态</th><th>能力</th><th>操作</th></tr></thead>
        <tbody>
          ${items.map(item => `
            <tr>
              <td>${h(item.email)}${item.email === currentAdminEmail ? '<div class="muted">当前账号</div>' : ''}</td>
              <td>${statusTag(item.status)}</td>
              <td>
                ${(item.capabilities || []).filter(cap => cap.status === 'active').map(cap => `
                  <span class="console-tag">
                    ${h(adminCapabilityLabels[cap.capability] || cap.capability)}
                    ${item.email === currentAdminEmail ? '' : `<button class="btn btn-secondary" data-admin-revoke="${h(item.email)}" data-capability="${h(cap.capability)}" title="撤销">${h('×')}</button>`}
                  </span>`).join(' ') || '<span class="muted">无</span>'}
              </td>
              <td>${item.email === currentAdminEmail ? '<span class="muted">不能修改自己</span>' : item.status === 'active'
                ? '<button class="btn btn-danger" data-admin-disable="${h(item.email)}">禁用</button>'
                : '<button class="btn btn-secondary" data-admin-enable="${h(item.email)}">启用</button>'}</td>
            </tr>`).join('')}
        </tbody>
      </table>` : '<div class="empty-state">暂无管理员。</div>';
    statusEl.textContent = `共 ${items.length} 位`;
  } catch (error) {
    statusEl.textContent = error.message;
    statusEl.classList.add('status-error');
  }
}

$('admin-create-form').addEventListener('submit', async event => {
  event.preventDefault();
  const email = $('admin-email').value.trim();
  const button = event.target.querySelector('button');
  show('admin-status', '');
  busy(button, true, '新增中…');
  try {
    await request(`${API}/admins`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    });
    show('admin-status', '管理员已新增。该邮箱登录成功后会绑定账号。');
    $('admin-email').value = '';
    loadAdmins();
    loadReferralActions();
  } catch (error) {
    show('admin-status', error.message, true);
  } finally { busy(button, false); }
});

$('admin-capability-form').addEventListener('submit', async event => {
  event.preventDefault();
  const email = $('admin-target-email').value.trim();
  const capability = $('admin-capability').value;
  const action = await actionDialog({
    title: '授予权限',
    description: '授权会写入管理审计记录。',
    reason: true,
    confirmText: '授权'
  });
  if (!action?.reason) return;
  try {
    await request(`${API}/admins/${encodeURIComponent(email)}/capabilities`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ capability, reason: action.reason }),
    });
    show('admin-status', '权限已授予。');
    loadAdmins();
    loadReferralActions();
  } catch (error) {
    show('admin-status', error.message, true);
  }
});

document.addEventListener('click', async event => {
  const revoke = event.target.closest('[data-admin-revoke]');
  const disable = event.target.closest('[data-admin-disable]');
  const enable = event.target.closest('[data-admin-enable]');
  if (!revoke && !disable && !enable) return;

  const email = revoke?.dataset.adminRevoke || disable?.dataset.adminDisable || enable?.dataset.adminEnable;
  let body;
  let path;
  if (revoke) {
    const action = await actionDialog({
      title: `撤销 ${adminCapabilityLabels[revoke.dataset.capability] || revoke.dataset.capability}`,
      description: '撤销会立即生效并写入审计。',
      reason: true,
      confirmText: '撤销'
    });
    if (!action?.reason) return;
    path = 'capabilities/revoke';
    body = { capability: revoke.dataset.capability, reason: action.reason };
  } else if (disable) {
    const action = await actionDialog({ title: '禁用管理员', reason: true, confirmText: '禁用' });
    if (!action?.reason) return;
    path = 'disable';
    body = { status: 'disabled', reason: action.reason };
  } else {
    const action = await actionDialog({ title: '启用管理员', reason: true, confirmText: '启用' });
    if (!action?.reason) return;
    path = 'enable';
    body = { status: 'active', reason: action.reason };
  }

  try {
    await request(`${API}/admins/${encodeURIComponent(email)}/${path}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    loadAdmins();
    loadReferralActions();
  } catch (error) {
    show('admin-status', error.message, true);
  }
});

async function loadReferralAdmin() {
  await Promise.all([
    loadReferralApplications(), loadReferralPartners(), loadReferralPayouts(),
    loadReferralActions(), loadFinanceTasks()
  ]);
}

async function loadReferralApplications(status = 'pending') {
  show('referral-application-status', '载入中…');
  try {
    const data = await request(`${API}/referral/applications?status=${encodeURIComponent(status)}&limit=100`);
    const items = data.applications || [];
    referralApplicationsBody.innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${h(item.code || item.user_id)}</td><td>${h(item.note || '-')}</td>
        <td>${statusTag(item.status)}</td><td>${h(item.created_at)}</td>
        <td>${item.status === 'pending' ? `
          <div class="actions">
            <button class="btn btn-primary" data-application-approve="${h(item.id)}">通过</button>
            <button class="btn btn-secondary" data-application-reject="${h(item.id)}">拒绝</button>
          </div>` : '<span class="muted">已处理</span>'}</td>
      </tr>`).join('') : '<tr><td colspan="5"><div class="empty-state">当前没有申请。<button class="btn btn-secondary" data-referral-refresh>刷新</button></div></td></tr>';
    show('referral-application-status', items.length ? `共 ${items.length} 条` : '');
  } catch (error) { show('referral-application-status', error.message, true); }
}

async function loadReferralPartners() {
  show('referral-partner-status', '载入中…');
  const params = new URLSearchParams({ limit: '100' });
  if ($('referral-partner-role').value) params.set('role', $('referral-partner-role').value);
  if ($('referral-partner-status-filter').value) params.set('status', $('referral-partner-status-filter').value);
  if ($('referral-partner-query').value.trim()) params.set('q', $('referral-partner-query').value.trim());
  try {
    const data = await request(`${API}/referral/partners?${params}`);
    const items = data.partners || [];
    referralPartnersBody.innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${h(item.code)}</td><td>${h(item.user_id)}</td>
        <td>${statusTag(item.role)}</td><td>${statusTag(item.status)}</td><td>${h(item.created_at)}</td>
        <td><div class="actions">
          ${item.status === 'active'
            ? `<button class="btn btn-danger" data-partner-freeze="${h(item.user_id)}">冻结</button>`
            : `<button class="btn btn-secondary" data-partner-unfreeze="${h(item.user_id)}">解冻</button>`}
        </div></td>
      </tr>`).join('') : '<tr><td colspan="6"><div class="empty-state">没有匹配的合作伙伴。<button class="btn btn-secondary" data-referral-refresh>刷新</button></div></td></tr>';
    show('referral-partner-status', items.length ? `共 ${items.length} 位` : '');
  } catch (error) { show('referral-partner-status', error.message, true); }
}

async function loadReferralActions() {
  try {
    const data = await request(`${API}/referral/actions?limit=100`);
    const items = data.actions || [];
    referralActionsBody.innerHTML = items.length ? items.map(item => `
      <tr><td>${h(item.created_at)}</td><td>${h(item.actor)}</td><td>${h(item.action)}</td>
      <td>${h(item.target_id)}</td><td>${h(item.reason || '-')}</td></tr>`).join('')
      : '<tr><td colspan="5"><span class="muted">暂无管理记录</span></td></tr>';
  } catch (error) { show('referral-action-status', error.message, true); }
}

$('referral-application-form').addEventListener('submit', event => {
  event.preventDefault();
  loadReferralApplications($('referral-application-status-filter').value);
});
$('referral-partner-form').addEventListener('submit', event => {
  event.preventDefault();
  loadReferralPartners();
});

referralApplicationsBody.addEventListener('click', async event => {
  const approve = event.target.closest('[data-application-approve]');
  const reject = event.target.closest('[data-application-reject]');
  if (!approve && !reject) return;
  const id = approve?.dataset.applicationApprove || reject?.dataset.applicationReject;
  const action = await actionDialog({
    title: approve ? '通过申请' : '拒绝申请',
    description: '该操作将写入管理记录。',
    reason: true,
    confirmText: approve ? '通过' : '拒绝'
  });
  if (!action?.reason) return;
  try {
    await request(`${API}/referral/applications/${encodeURIComponent(id)}/decision`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approve: Boolean(approve), reason: action.reason }),
    });
    loadReferralApplications($('referral-application-status-filter').value);
    loadReferralPartners();
    loadReferralActions();
  } catch (error) { show('referral-application-status', error.message, true); }
});

referralPartnersBody.addEventListener('click', async event => {
  const promote = event.target.closest('[data-partner-promote]');
  if (promote) {
    const id = promote.dataset.partnerPromote;
    const action = await actionDialog({ title: '升级为 Scout', reason: true, confirmText: '升级' });
    if (!action?.reason) return;
    try {
      await request(`${API}/referral/partners/${encodeURIComponent(id)}/promote`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: action.reason }),
      });
      show('referral-partner-status', '已升级为 Scout');
      loadReferralPartners();
      loadReferralActions();
    } catch (error) {
      show('referral-partner-status', error.message, true);
    }
    return;
  }

  const freeze = event.target.closest('[data-partner-freeze]');
  const unfreeze = event.target.closest('[data-partner-unfreeze]');
  if (!freeze && !unfreeze) return;
  const id = freeze?.dataset.partnerFreeze || unfreeze?.dataset.partnerUnfreeze;
  const action = await actionDialog({
    title: freeze ? '冻结合作伙伴' : '解冻合作伙伴',
    description: '该操作将写入管理记录。',
    reason: true,
    confirmText: freeze ? '冻结' : '解冻'
  });
  if (!action?.reason) return;
  try {
    await request(`${API}/referral/partners/${encodeURIComponent(id)}/${freeze ? 'freeze' : 'unfreeze'}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: action.reason }),
    });
    loadReferralPartners();
    loadReferralActions();
  } catch (error) { show('referral-partner-status', error.message, true); }
});

async function loadReferralPayouts() {
  const status = document.getElementById('referral-payout-status-filter').value;
  const statusEl = document.getElementById('referral-payout-status');
  statusEl.textContent = '载入中…';
  statusEl.classList.remove('status-error');
  try {
    const params = new URLSearchParams({ limit: '100' });
    if (status) params.set('status', status);
    const data = await request(`${API}/referral/payouts?${params}`);
    const items = data.payouts || [];
    if (!items.length) {
      referralPayoutsBody.innerHTML = '<tr><td colspan="6"><div class="empty-state">当前没有提现申请。<button class="btn btn-secondary" data-referral-refresh>刷新</button></div></td></tr>';
      statusEl.textContent = '';
      return;
    }
    referralPayoutsBody.innerHTML = items.map(item => `
      <tr>
        <td>${h(formatMinor(item.amount_minor, item.currency))}${item.requested_amount_minor !== item.amount_minor ? `<div class="muted">申请 ${h(formatMinor(item.requested_amount_minor, item.currency))}</div>` : ''}</td>
        <td>${h(item.referrer_id)}</td>
        <td>${h(item.method)} / ${h(maskAccount(item.account))}${item.status === 'approved' ? ` <button class="btn btn-secondary" data-payout-reveal="${h(item.id)}">显示</button>` : ''}</td>
        <td>${statusTag(item.status)}</td>
        <td>${h(item.created_at)}</td>
        <td>
          ${item.status === 'requested' ? `
            <div class="actions">
              <button class="btn btn-primary" data-payout-approve="${h(item.id)}">批准</button>
              <button class="btn btn-secondary" data-payout-reject="${h(item.id)}">拒绝</button>
            </div>` : item.status === 'approved' ? `
            <button class="btn btn-primary" data-payout-paid="${h(item.id)}">标记已打款</button>` : `
            <span class="muted">已处理</span>`}
        </td>
      </tr>`).join('');
    statusEl.textContent = `共 ${items.length} 笔`;
  } catch (error) {
    statusEl.textContent = error.message;
    statusEl.classList.add('status-error');
  }
}

$('finance-task-form').addEventListener('submit', event => {
  event.preventDefault();
  loadFinanceTasks($('finance-task-status-filter').value);
});

financeTasksBody.addEventListener('click', async event => {
  const button = event.target.closest('[data-finance-resolve]');
  if (!button) return;
  const id = button.dataset.financeResolve;
  const action = await actionDialog({
    title: '处理财务追回',
    description: 'resolved 表示已完成追回；rejected 表示经审核豁免。',
    reason: true,
    status: true,
    confirmText: '提交'
  });
  if (!action?.reason) return;
  const statusEl = document.getElementById('finance-task-status');
  try {
    await request(`${API}/referral/finance-tasks/${encodeURIComponent(id)}/resolve`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: action.status, note: action.reason }),
    });
    loadFinanceTasks($('finance-task-status-filter').value);
    loadReferralActions();
  } catch (error) {
    statusEl.textContent = error.message;
    statusEl.classList.add('status-error');
  }
});

referralPayoutsBody.addEventListener('click', async event => {
  const approve = event.target.closest('[data-payout-approve]');
  const reject = event.target.closest('[data-payout-reject]');
  const paid = event.target.closest('[data-payout-paid]');
  if (!approve && !reject && !paid) return;
  const id = approve?.dataset.payoutApprove || reject?.dataset.payoutReject || paid?.dataset.payoutPaid;
  let path;
  let body;
  if (approve) {
    const action = await actionDialog({
      title: '批准提现',
      description: '批准后仍需人工打款并标记凭证。',
      reason: true,
      confirmText: '批准'
    });
    if (!action?.reason) return;
    path = 'approve';
    body = { reason: action.reason };
  } else if (reject) {
    const action = await actionDialog({ title: '拒绝提现', reason: true, confirmText: '拒绝' });
    if (!action?.reason) return;
    path = 'reject';
    body = { reason: action.reason };
  } else {
    const action = await actionDialog({ title: '标记已打款', reference: true, confirmText: '确认打款' });
    if (!action?.reference) return;
    path = 'mark-paid';
    body = { reason: 'manual payout', payment_reference: action.reference };
  }
  const statusEl = document.getElementById('referral-payout-status');
  try {
    await request(`${API}/referral/payouts/${encodeURIComponent(id)}/${path}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    loadReferralPayouts();
    loadReferralActions();
  } catch (error) {
    statusEl.textContent = error.message;
    statusEl.classList.add('status-error');
  }
});

document.addEventListener('click', event => {
  if (event.target.closest('[data-referral-refresh]')) loadReferralAdmin();
});

document.addEventListener('click', async event => {
  const button = event.target.closest('[data-payout-reveal]');
  if (!button) return;
  const id = button.dataset.payoutReveal;
  const action = await actionDialog({
    title: '显示收款账号',
    description: '该操作会写入管理记录，仅用于人工打款。',
    reason: true,
    confirmText: '显示'
  });
  if (!action?.reason) return;
  try {
    const data = await request(`${API}/referral/payouts/${encodeURIComponent(id)}/reveal-account`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: action.reason }),
    });
    button.closest('td').textContent = data.account;
  } catch (error) {
    show('referral-payout-status', error.message, true);
  }
});

async function loadFinanceTasks(status = 'pending') {
  const statusEl = $('finance-task-status');
  statusEl.textContent = '载入中…';
  statusEl.classList.remove('status-error');
  try {
    const params = new URLSearchParams({ limit: '100' });
    if (status) params.set('status', status);
    const data = await request(`${API}/referral/finance-tasks?${params}`);
    const items = data.tasks || [];
    financeTasksBody.innerHTML = items.length ? items.map(item => `
      <tr>
        <td>${h(formatMinor(item.amount_minor, item.currency))}</td>
        <td>${h(item.beneficiary_id)}</td>
        <td>${h(financeTaskTypeLabels[item.type] || item.type)}<div class="muted">${h(item.provider_refund_id)}</div></td>
        <td>${statusTag(item.status)}</td>
        <td>${h(item.created_at)}</td>
        <td>${item.status === 'pending' ? `<button class="btn btn-primary" data-finance-resolve="${h(item.id)}">处理</button>` : '<span class="muted">已处理</span>'}</td>
      </tr>`).join('') : '<tr><td colspan="6"><div class="empty-state">当前没有财务任务。</div></td></tr>';
    statusEl.textContent = `共 ${items.length} 项`;
  } catch (error) {
    statusEl.textContent = error.message;
    statusEl.classList.add('status-error');
  }
}

function formatMinor(amount, currency) {
  return `${currency || ''} ${(Number(amount || 0) / 100).toFixed(2)}`.trim();
}
function statusTag(status) {
  const value = (status || '-').toLowerCase();
  const positive = ['active', 'approved', 'paid', 'succeeded'];
  const negative = ['frozen', 'rejected', 'failed'];
  const cls = positive.includes(value) ? 'tag' : negative.includes(value) ? 'tag tag-danger' : 'tag tag-muted';
  return `<span class="${cls}">${h(status || '-')}</span>`;
}
function formatCode(code) { return code.slice(0, 4) + '-' + code.slice(4); }

checkAdminSession();

})();
