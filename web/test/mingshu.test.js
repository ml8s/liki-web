import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { el, readJS } from './helpers.js';

// ── Helper: create DOM matching mingshu.html ──
function createPage() {
  const gMale = el('input', { value: 'male' }); gMale.checked = true;
  const gFemale = el('input', { value: 'female' }); gFemale.checked = false;
  const form = el('form');
  form.querySelector = (sel) => sel === 'input[name="lb-gender"]:checked' ? gMale : null;

  const map = {
    'mingshu-form': form,
    'form-submit': el('button', { text: '¥129' }),
    'form-error': el('p', { className: 'hidden' }),
    'lb-name': el('input'),
    'lb-birth': el('input'),
    'lb-time': el('input', { value: '12:00' }),
    'lb-city': el('input'),
    'lb-email': el('input'),
    'lb-male': gMale,
    'lb-female': gFemale,
  };

  globalThis.document = {
    readyState: 'complete',
    documentElement: el('html'),
    addEventListener: (t, f) => { if (t === 'DOMContentLoaded') f(); },
    getElementById: (id) => map[id] || null,
    createElement: () => el('div'),
    body: el('body'),
  };

  return map;
}

describe('mingshu.js', () => {
  let els, code;

  beforeAll(() => { code = readJS('mingshu'); });
  beforeEach(() => {
    globalThis.Liki = { currency: 'CNY', t: (k) => k, showErr: (el, msg) => { el.textContent = msg; el.classList.remove('hidden'); } };
    globalThis.window = { Liki: globalThis.Liki };
    globalThis.i18next = { isInitialized: true, language: 'zh-Hans', t: (k) => k };
    globalThis.fetch = vi.fn();
    globalThis.goCheckout = vi.fn();
    globalThis.createOrderAndCheckout = async (apiPath, body, product, extra) => {
      const resp = await globalThis.fetch(apiPath, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error?.message || 'Request failed');
      const oid = data.data?.order_id;
      if (!oid) throw new Error('Order creation failed');
      await globalThis.goCheckout(product, oid, extra || {});
    };
    els = createPage();
  });
  afterEach(() => {
    delete globalThis.document;
    delete globalThis.window;
    delete globalThis.Liki;
    delete globalThis.i18next;
    delete globalThis.fetch;
    delete globalThis.goCheckout;
    delete globalThis.createOrderAndCheckout;
  });

  function run() { eval(code); }

  it('exits silently when DOM elements missing', () => {
    globalThis.document = { getElementById: () => null };
    expect(run).not.toThrow();
  });

  it('shows error when required fields empty', async () => {
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    expect(els['form-error'].classList._cs).not.toContain('hidden');
  });

  function fill(d) {
    if (d.name !== undefined) els['lb-name'].value = d.name;
    els['lb-birth'].value = d.birth ?? '';
    if (d.city !== undefined) els['lb-city'].value = d.city;
    if (d.email !== undefined) els['lb-email'].value = d.email;
  }

  it('submits correct API payload', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: { order_id: 'lb-x' } }) });
    fill({ name: '张三', birth: '1990-05-20T14:30', city: '北京', email: 'a@b.co' });
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/mingshu/orders', expect.objectContaining({ method: 'POST' }));
    const b = JSON.parse(globalThis.fetch.mock.calls[0][1].body);
    expect(b.birth.year).toBe(1990);
    expect(b.birth.month).toBe(5);
    expect(b.birth.day).toBe(20);
    expect(b.birth.hour).toBe(14);
    expect(b.birth.minute).toBe(30);
    expect(b.birth.gender).toBe('male');
    expect(b.email).toBe('a@b.co');
  });

  it('defaults time to 12:00', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: { order_id: 'lb-x' } }) });
    fill({ name: 'x', birth: '2024-01-01', city: 'x', email: 'x@x.co' });
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    const b = JSON.parse(globalThis.fetch.mock.calls[0][1].body);
    expect(b.birth.hour).toBe(12);
    expect(b.birth.minute).toBe(0);
  });

  it('calls goCheckout after order success', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: { order_id: 'lb-789' } }) });
    fill({ name: 'x', birth: '2024-06-15', city: 'x', email: 'x@x.co' });
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    expect(globalThis.goCheckout).toHaveBeenCalledWith('mingshu', 'lb-789', { email: 'x@x.co' });
  });

  it('shows error on API failure', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false, json: async () => ({ error: { message: 'err' } }) });
    fill({ name: 'x', birth: '2024-06-15', city: 'x', email: 'x@x.co' });
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    expect(els['form-error'].classList._cs).not.toContain('hidden');
    expect(els['form-submit'].disabled).toBe(false);
  });

  it('shows error when order_id missing', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: {} }) });
    fill({ name: 'x', birth: '2024-06-15', city: 'x', email: 'x@x.co' });
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    expect(els['form-error'].classList._cs).not.toContain('hidden');
    expect(globalThis.goCheckout).not.toHaveBeenCalled();
  });

  it('disables button during loading', async () => {
    globalThis.fetch = vi.fn().mockReturnValue(new Promise(() => {})); // never resolves
    fill({ name: 'x', birth: '2024-06-15', city: 'x', email: 'x@x.co' });
    run();
    const promise = els['mingshu-form']._lssubmit({ preventDefault() {} });
    expect(els['form-submit'].disabled).toBe(true);
    // Clean up by resolving
    globalThis.fetch.mock.calls[0][1]?.abort?.();
  });

  it('uses male gender as fallback when no radio checked', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: { order_id: 'lb-f' } }) });
    fill({ name: 'x', birth: '2024-06-15', city: 'x', email: 'x@x.co' });
    els['mingshu-form'].querySelector = () => null;
    run();
    await els['mingshu-form']._lssubmit({ preventDefault() {} });
    const b = JSON.parse(globalThis.fetch.mock.calls[0][1].body);
    expect(b.birth.gender).toBe('male');
  });
});
