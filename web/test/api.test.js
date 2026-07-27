import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// ── minimal DOM mock for Node ──

function encodeText(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function createMockDocument() {
  const allElements = [];
  const body = {
    _id: 0, _tag: 'body', _text: '', _html: '', _className: '', _children: [],
    appendChild(c) { this._children.push(c); },
  };
  allElements.push(body);

  const doc = {
    body,
    createElement(tag) {
      const el = {
        _id: allElements.length, _tag: tag,
        _text: '', _html: '', _className: '',
        _children: [],
        set textContent(v) { this._text = String(v); this._html = encodeText(v); },
        get textContent() { return this._text; },
        set innerHTML(v) { this._html = v; },
        get innerHTML() { return this._html; },
        set className(v) { this._className = v; },
        get className() { return this._className; },
        appendChild(c) { this._children.push(c); },
        remove() {
          for (const e of allElements) {
            const idx = (e._children || []).indexOf(this);
            if (idx !== -1) { e._children.splice(idx, 1); }
          }
          const ai = allElements.indexOf(this);
          if (ai !== -1) allElements.splice(ai, 1);
        },
      };
      allElements.push(el);
      return el;
    },
    querySelector(sel) {
      for (const e of allElements) {
        if (sel === '.toast' && e._className && e._className.includes('toast')) return e;
        if (e._className && e._className.split(' ').every(c => sel.includes(c))) return e;
      }
      // also check children
      for (const e of allElements) {
        for (const c of (e._children || [])) {
          if (sel === '.toast' && c._className && c._className.includes('toast')) return c;
          if (c._className && c._className.split(' ').every(cls => sel.includes(cls))) return c;
        }
      }
      return null;
    },
  };
  return doc;
}

// simplified AbortSignal.timeout mock
function fakeAbortSignal(ms) {
  const ctrl = new AbortController();
  setTimeout(() => ctrl.abort(new DOMException('Timeout', 'TimeoutError')), ms);
  return ctrl.signal;
}

let mockDoc;
beforeEach(() => {
  mockDoc = createMockDocument();
  vi.stubGlobal('document', mockDoc);
  vi.stubGlobal('i18next', { t: (k) => k });
  vi.stubGlobal('marked', { parse: (text) => `<h2>${text}</h2>` });
  vi.stubGlobal('DOMPurify', { sanitize: (html) => html });
  vi.stubGlobal('AbortSignal', { timeout: fakeAbortSignal });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

// ── functions under test (copied from api.js) ──

const API_BASE = '/api';
const DEFAULT_TIMEOUT = 30000;

const RETRY_DELAY = 1000;

async function handleResponse(resp) {
  if (!resp.ok) {
    let msg = i18next.t('error.requestFailed');
    try { const body = await resp.json(); msg = body.error?.message || msg; } catch (_) {}
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


function showQRModal(orderID, qrcodeUrl, fallbackUrl) {
  var existing = document.querySelector('.qr-modal');
  if (existing) existing.remove();

  var dialog = document.createElement('dialog');
  dialog.className = 'qr-modal';
  dialog.innerHTML =
    '<form method="dialog">' +
      '<button class="qr-modal-close" aria-label="' + escapeHTML(i18next.t('payment.qrClose')) + '">&times;</button>' +
    '</form>' +
    '<p class="qr-modal-title">' + escapeHTML(i18next.t('payment.scanQR')) + '</p>' +
    '<img class="qr-modal-img" alt="QR Code">' +
    '<p class="qr-modal-hint" data-qr-hint>' + escapeHTML(i18next.t('payment.qrHint')) + '</p>';

  var img = dialog.querySelector('.qr-modal-img');
  img.src = qrcodeUrl;
  if (fallbackUrl) {
    img.onerror = function() { this.onerror = null; location.href = fallbackUrl; };
  }
  document.body.appendChild(dialog);
  if (!dialog.showModal) dialog.showModal = function() { this.setAttribute('open', ''); };
  dialog.showModal();
  return dialog;
}

async function goCheckout(product, orderID, extra) {
  var payload = { order_id: orderID };
  if (extra) Object.assign(payload, extra);
  var data = await apiPost('/' + product + '/checkout', payload);
  if (!data) throw new Error(i18next.t('error.noCheckoutUrl'));

  if (data.qrcode_url) {
    showQRModal(orderID, data.qrcode_url, data.checkout_url);
    return { mode: 'qr' };
  }
  var url = data.checkout_url;
  if (!url) throw new Error(i18next.t('error.noCheckoutUrl'));
  window.location.href = url;
  return { mode: 'redirect' };
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
  } catch (_) { return escapeHTML(text); }
}

function showToast(msg, type = 'error') {
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 4000);
  return el;
}

// ── helpers ──

function mockFetch(status, data, ok = true) {
  return Promise.resolve({ ok, status, json: () => Promise.resolve(data) });
}

// ── tests ──

describe('handleResponse', () => {
  it('returns data.data on success', async () => {
    const resp = await mockFetch(200, { data: { hello: 'world' } });
    expect(await handleResponse(resp)).toEqual({ hello: 'world' });
  });

  it('throws with error message from response body on non-ok', async () => {
    const resp = await mockFetch(422, { error: { message: 'bad input' } }, false);
    await expect(handleResponse(resp)).rejects.toThrow('bad input');
  });

  it('throws with fallback when body has no error message', async () => {
    const resp = await mockFetch(500, {}, false);
    await expect(handleResponse(resp)).rejects.toThrow('error.requestFailed');
  });

  it('throws with fallback when body is not JSON', async () => {
    const resp = { ok: false, json: () => Promise.reject(new Error('parse')) };
    await expect(handleResponse(resp)).rejects.toThrow('error.requestFailed');
  });

  it('throws when response has no data envelope', async () => {
    const resp = await mockFetch(200, { not_data: true });
    await expect(handleResponse(resp)).rejects.toThrow('Unexpected API response');
  });

  it('throws when response body is null', async () => {
    const resp = await mockFetch(200, null);
    await expect(handleResponse(resp)).rejects.toThrow('Unexpected API response');
  });
});

describe('apiPost', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  it('sends POST with JSON body and Content-Type header', async () => {
    fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({ data: { id: 1 } }) });
    const result = await apiPost('/submit', { name: 'test' });
    const [url, init] = fetch.mock.calls[0];
    expect(url).toBe('/api/submit');
    expect(init.method).toBe('POST');
    expect(init.headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(init.body)).toEqual({ name: 'test' });
    expect(result).toEqual({ id: 1 });
  });

  it('propagates fetch errors', async () => {
    fetch.mockRejectedValue(new Error('network error'));
    await expect(apiPost('/submit', {})).rejects.toThrow('network error');
  });
});

describe('goCheckout', () => {
  let qrUrls = [];
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    vi.stubGlobal('window', { location: { href: '' } });
    // Default: desktop (non-mobile) for QR tests
    vi.stubGlobal('navigator', { userAgent: '', maxTouchPoints: 0 });
    vi.stubGlobal('innerWidth', 1920);
    qrUrls = [];
  });

  it('throws when checkout_url is missing from response', async () => {
    fetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({ data: {} }) });
    await expect(goCheckout('naming', 'order-1')).rejects.toThrow('error.noCheckoutUrl');
  });

  it('redirects to checkout_url on success', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { checkout_url: 'https://pay.example.com/checkout/xyz' } }),
    });
    await goCheckout('naming', 'order-1');
    expect(window.location.href).toBe('https://pay.example.com/checkout/xyz');
  });

  it('shows QR modal on desktop when qrcode_url is present', async () => {
    var calledUrl = null, calledFallback = null;
    var orig = showQRModal;
    showQRModal = function(oid, url, fb) { calledUrl = url; calledFallback = fb; };
    try {
      fetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: { checkout_url: 'https://pay.example.com/checkout/xyz', qrcode_url: 'https://qr.example.com/qr.png' } }),
      });
      await goCheckout('naming', 'order-1');
      expect(window.location.href).toBe('');
      expect(calledUrl).toBe('https://qr.example.com/qr.png');
      expect(calledFallback).toBe('https://pay.example.com/checkout/xyz');
    } finally {
      showQRModal = orig;
    }
  });


  it('redirects on desktop when qrcode_url is absent (omitempty)', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { checkout_url: 'https://pay.example.com/checkout/xyz' } }),
    });
    await goCheckout('naming', 'order-1');
    expect(window.location.href).toBe('https://pay.example.com/checkout/xyz');
  });

  it('redirects on desktop when qrcode_url is empty string', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { checkout_url: 'https://pay.example.com/checkout/xyz', qrcode_url: '' } }),
    });
    await goCheckout('naming', 'order-1');
    expect(window.location.href).toBe('https://pay.example.com/checkout/xyz');
  });

  it('propagates network error from apiPost', async () => {
    fetch.mockRejectedValue(new TypeError('Failed to fetch'));
    await expect(goCheckout('naming', 'order-1')).rejects.toThrow('Failed to fetch');
  });
});

describe('apiPost retry', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  it('does not retry by default', async () => {
    fetch.mockRejectedValue(new TypeError('Failed to fetch'));
    await expect(apiPost('/submit', {})).rejects.toThrow('Failed to fetch');
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it('accepts retries option', async () => {
    fetch.mockRejectedValue(new TypeError('Failed'));
    await expect(apiPost('/submit', {}, { retries: 1 })).rejects.toThrow('Failed');
    expect(fetch).toHaveBeenCalledTimes(2); // initial + 1 retry
  });
});

describe('isRetryable', () => {
  it('returns true for TypeError', () => {
    expect(isRetryable(new TypeError('Failed to fetch'))).toBe(true);
  });

  it('returns true for AbortError (not TimeoutError)', () => {
    const e = new DOMException('The operation was aborted', 'AbortError');
    expect(isRetryable(e)).toBe(true);
  });

  it('returns false for AbortError with TimeoutError message', () => {
    const e = new DOMException('Timeout', 'TimeoutError');
    expect(isRetryable(e)).toBe(false);
  });

  it('returns false for TypeError with TimeoutError message', () => {
    const e = new TypeError('TimeoutError');
    expect(isRetryable(e)).toBe(false);
  });

  it('returns false for regular Error', () => {
    expect(isRetryable(new Error('something'))).toBe(false);
  });

  it('returns false for SyntaxError', () => {
    expect(isRetryable(new SyntaxError('bad JSON'))).toBe(false);
  });
});

describe('escapeHTML', () => {
  it('escapes angle brackets and ampersand', () => {
    expect(escapeHTML('<script>alert("xss")</script>'))
      .toBe('&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;');
  });

  it('returns plain text unchanged', () => {
    expect(escapeHTML('hello')).toBe('hello');
  });

  it('returns empty string for empty input', () => {
    expect(escapeHTML('')).toBe('');
  });
});

describe('renderMD', () => {
  it('returns sanitized HTML from marked.parse', () => {
    expect(renderMD('# Hello')).toBe('<h2># Hello</h2>');
  });

  it('falls back to escaped HTML when marked.parse throws', () => {
    vi.stubGlobal('marked', { parse: () => { throw new Error('parse error'); } });
    expect(renderMD('<b>bold</b>')).toBe('&lt;b&gt;bold&lt;/b&gt;');
  });
});

describe('showToast', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('creates toast element with correct class and appends to body', () => {
    showToast('error message', 'error');
    const toast = document.querySelector('.toast.toast-error');
    expect(toast).not.toBeNull();
    expect(toast.textContent).toBe('error message');
  });

  it('defaults type to "error"', () => {
    showToast('msg');
    expect(document.querySelector('.toast.toast-error')).not.toBeNull();
  });

  it('returns the toast element', () => {
    const el = showToast('msg');
    expect(el).not.toBeNull();
    expect(el._tag).toBe('div');
    expect(el._className).toBe('toast toast-error');
  });

  it('removes toast after 4000ms', () => {
    showToast('msg');
    expect(document.querySelector('.toast')).not.toBeNull();
    vi.advanceTimersByTime(4000);
    expect(document.querySelector('.toast')).toBeNull();
  });
});
