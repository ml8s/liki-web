const API_BASE = '/api';
const DEFAULT_TIMEOUT = 30000;
const TOAST_DURATION = 4000;
const RETRY_DELAY = 1000;

let offlineToast = null;

window.addEventListener('online', () => {
  if (offlineToast) { offlineToast.remove(); offlineToast = null; }
  showToast(i18next.t('error.backOnline'), 'info');
});

window.addEventListener('offline', () => {
  offlineToast = showToast(i18next.t('error.offline'), 'error');
});

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

async function apiPost(path, body, opts = {}) {
  const timeout = opts.timeout ?? DEFAULT_TIMEOUT;
  const retries = opts.retries ?? 0;
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

const t = (key) => i18next.t(key);
window.Liki = window.Liki || {};
Object.assign(window.Liki, { apiPost, escapeHTML, renderMD, t,
  showErr: function(el, msg) { el.textContent = msg; el.classList.remove('hidden'); }
});
