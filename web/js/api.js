/* global goCheckout */
const API_BASE = '/api';
const DEFAULT_TIMEOUT = 30000;
const TOAST_DURATION = 4000;
const RETRY_DELAY = 1000;

// ── connectivity ──

let offlineToast = null;

window.addEventListener('online', () => {
  if (offlineToast) { offlineToast.remove(); offlineToast = null; }
  showToast(i18next.t('error.backOnline'), 'info');
});

window.addEventListener('offline', () => {
  offlineToast = showToast(i18next.t('error.offline'), 'error');
});

// ── response helpers ──

async function handleResponse(resp) {
  if (!resp.ok) {
    let msg = i18next.t('error.requestFailed');
    try { const body = await resp.json(); msg = body.error?.message || msg; } catch {}
    throw new Error(msg);
  }
  const data = await resp.json();
  if (!data || !('data' in data)) throw new Error('Unexpected API response');
  return data.data;
}

function isRetryable(err) {
  return (err.name === 'TypeError' || err.name === 'AbortError') && err.message !== 'TimeoutError';
}

// ── public API ──

async function apiPost(path, body, opts = {}) {
  const timeout = opts.timeout ?? DEFAULT_TIMEOUT;
  const retries = opts.retries ?? 0; // POST not idempotent by default
  let lastErr;
  for (let i = 0; i <= retries; i++) {
    try {
      const init = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      };
      if (timeout > 0) init.signal = AbortSignal.timeout(timeout);
      return handleResponse(await fetch(API_BASE + path, init));
    } catch (err) {
      lastErr = err;
      if (!isRetryable(err) || i >= retries) throw err;
      await new Promise(r => setTimeout(r, RETRY_DELAY * (i + 1)));
    }
  }
  throw lastErr;
}


function escapeHTML(text) {
  const d = document.createElement('div');
  d.textContent = text;
  return d.innerHTML;
}

function renderMD(text) {
  try {
    const raw = marked.parse(text);
    return DOMPurify.sanitize(raw, {
      ALLOWED_TAGS: ['h2', 'h3', 'h4', 'p', 'br', 'strong', 'em', 'ul', 'ol', 'li', 'code', 'pre', 'blockquote', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'hr'],
      ALLOWED_ATTR: [],
    });
  } catch { return escapeHTML(text); }
}

function showToast(msg, type = 'error') {
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.setAttribute('role', 'status');
  el.setAttribute('aria-live', 'polite');
  el.textContent = msg;
  document.body.appendChild(el);
  el.animate([
    { opacity: 0, transform: 'translateY(-8px)' },
    { opacity: 1, transform: 'translateY(0)' }
  ], { duration: 250, easing: 'cubic-bezier(0, 0, 0, 1)', fill: 'both' });
  setTimeout(() => {
    el.animate([
      { opacity: 1 },
      { opacity: 0 }
    ], { duration: 200, easing: 'cubic-bezier(0.2, 0, 0.4, 1)', fill: 'both' }).onfinish = () => el.remove();
  }, TOAST_DURATION);
  return el;
}

// ── shared order+checkout flow ──

async function createOrderAndCheckout(apiPath, body, product, extra) {
  const resp = await fetch(apiPath, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  const data = await resp.json();
  if (!resp.ok) throw new Error(data.error?.message || 'Request failed');
  const oid = data.data?.order_id;
  if (!oid) throw new Error('Order creation failed');
  await goCheckout(product, oid, extra || {});
}

// ── currency detection ──
(function() {
  var PRICE = {
    naming:   { CNY: '¥199', USD: '$29.90' },
    hepan:    { CNY: '¥199', USD: '$29.90' },
    mingshu: { CNY: '¥129', USD: '$18' }
  };
  window.Liki = window.Liki || {};
  window.Liki.currency = 'CNY';

  function update() {
    var cur = window.Liki.currency;
    document.querySelectorAll('[data-price]').forEach(function(el) {
      var p = PRICE[el.getAttribute('data-price')];
      if (p) el.textContent = p[cur];
    });
  }

  update();

  fetch('/api/location').then(function(r) { return r.json(); }).then(function(d) {
    if (d.data && d.data.currency === 'USD') {
      window.Liki.currency = 'USD';
      update();
    }
  }).catch(function() {});
})();

// ── namespace ──
const t = (key) => i18next.t(key);
window.Liki = window.Liki || {};
Object.assign(window.Liki, { apiPost, createOrderAndCheckout, escapeHTML, renderMD, t,
  showErr: function(el, msg) { el.textContent = msg; el.classList.remove('hidden'); }
});
