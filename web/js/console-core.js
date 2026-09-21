// console-core.js — shared runtime for KOL, Scout, and Admin consoles.
// The surfaces stay independent; only stateless browser plumbing is shared.
(function() {
  'use strict';

  const $ = id => document.getElementById(id);

  const escapeHTML = value => String(value == null ? '' : value)
    .replace(/[&<>'"]/g, char => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;'
    }[char]));

  const money = (amount, currency) => (
    `${currency || ''} ${(Number(amount || 0) / 100).toFixed(2)}`
  ).trim();

  const maskAccount = value => {
    const account = String(value || '');
    const at = account.indexOf('@');
    if (at > 0) return account.slice(0, 1) + '***' + account.slice(at);
    if (account.length > 7) return account.slice(0, 2) + '****' + account.slice(-4);
    return '****';
  };

  const setStatus = (id, text, error = false) => {
    const element = $(id);
    element.textContent = text;
    element.classList.toggle('console-status-error', error);
    element.classList.toggle('status-error', error);
  };

  const request = async (path, options = {}) => {
    const response = await fetch(path, {
      credentials: 'same-origin',
      headers: options.body ? { 'Content-Type': 'application/json' } : undefined,
      ...options,
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(body.error?.message || `HTTP ${response.status}`);
    }
    return body.data || {};
  };

  const tag = value => `<span class="mono">${escapeHTML(value || '-')}</span>`;

  const setWorkspace = visible => {
    $('workspace').classList.toggle('hidden', !visible);
    $('logout').classList.toggle('hidden', !visible);
  };

  const busyButton = (button, loading, activeText) => {
    if (!button) return;
    if (loading) {
      button.dataset.label = button.textContent;
      button.textContent = activeText;
      button.disabled = true;
      return;
    }
    if (button.dataset.label) button.textContent = button.dataset.label;
    button.disabled = false;
  };

  // Shared email-code login for console pages. Page-specific IDs and redirect
  // behavior are injected so KOL and Scout never learn about each other.
  const initEmailAuth = ({
    onAuthenticated,
    codeFormId = 'code-form',
    verifyFormId = 'verify-form',
    emailId = 'email',
    codeId = 'code',
    sendId = 'send',
    verifyId = 'verify',
    statusId = 'auth-status',
    errorId = 'auth-error',
    logoutId = 'logout',
  } = {}) => {
    $(codeFormId).addEventListener('submit', async event => {
      event.preventDefault();
      setStatus(statusId, '');
      setStatus(errorId, '');
      const email = $(emailId).value.trim();
      if (!email) {
        setStatus(errorId, '请输入邮箱', true);
        return;
      }
      const button = $(sendId);
      busyButton(button, true, '发送中…');
      try {
        await request('/api/auth/email/code', {
          method: 'POST', body: JSON.stringify({ email }),
        });
        $(verifyFormId).classList.remove('hidden');
        setStatus(statusId, '验证码已发送，请查收邮箱。');
      } catch (error) {
        setStatus(errorId, error.message, true);
      } finally {
        busyButton(button, false);
      }
    });

    $(verifyFormId).addEventListener('submit', async event => {
      event.preventDefault();
      setStatus(statusId, '');
      setStatus(errorId, '');
      const email = $(emailId).value.trim();
      const code = $(codeId).value.trim();
      if (!email || !code) {
        setStatus(errorId, '请输入邮箱和验证码', true);
        return;
      }
      const button = $(verifyId);
      busyButton(button, true, '验证中…');
      try {
        await request('/api/auth/email/verify', {
          method: 'POST', body: JSON.stringify({ email, code }),
        });
        await onAuthenticated();
      } catch (error) {
        setStatus(errorId, error.message, true);
      } finally {
        busyButton(button, false);
      }
    });

    const logout = logoutId ? $(logoutId) : null;
    if (!logout) return;
    logout.addEventListener('click', async () => {
      try { await request('/api/auth/logout', { method: 'POST' }); } catch {}
      location.reload();
    });
  };

  globalThis.LikiConsole = {
    $, escapeHTML, money, maskAccount, setStatus, request, tag,
    setWorkspace, busyButton, initEmailAuth,
  };
})();
