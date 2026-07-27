import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { readJS } from './helpers.js';

function el(tag) {
  return {
    _tag: tag,
    textContent: '',
    className: '',
    setAttribute: () => {},
    animate: () => ({ onfinish: null }),
    remove: () => {},
  };
}

describe('payment.js goCheckout', () => {
  let code;

  beforeAll(() => { code = readJS('payment'); });

  beforeEach(() => {
    globalThis.window = { Liki: { currency: 'CNY' }, location: {} };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ data: { checkout_url: 'https://pay.example.com/checkout' } })
    });
    globalThis.apiPost = vi.fn().mockResolvedValue({ checkout_url: 'https://pay.example.com/checkout' });
    globalThis.i18next = { isInitialized: true, t: (k) => k };
    globalThis.document = {
      querySelector: () => null,
      createElement: () => el('div'),
      body: { appendChild: () => {} },
    };
    eval(code);
  });

  afterEach(() => {
    delete globalThis.window;
    delete globalThis.fetch;
    delete globalThis.apiPost;
    delete globalThis.i18next;
    delete globalThis.document;
  });

  it('sets provider to xunhu when currency is CNY', async () => {
    globalThis.window.Liki.currency = 'CNY';
    await globalThis.window.Liki.goCheckout('naming', 'order-1');

    const body = globalThis.apiPost.mock.calls[0][1];
    expect(body.provider).toBe('xunhu');
  });

  it('sets provider to dodo when currency is USD', async () => {
    globalThis.window.Liki.currency = 'USD';
    await globalThis.window.Liki.goCheckout('naming', 'order-1');

    const body = globalThis.apiPost.mock.calls[0][1];
    expect(body.provider).toBe('dodo');
  });

  it('does not override provider if already set in extra', async () => {
    globalThis.window.Liki.currency = 'CNY';
    await globalThis.window.Liki.goCheckout('naming', 'order-1', { email: 'a@b.co', provider: 'dodo' });

    const body = globalThis.apiPost.mock.calls[0][1];
    expect(body.provider).toBe('dodo');
    expect(body.email).toBe('a@b.co');
  });

  it('falls back to dodo when currency is undefined', async () => {
    globalThis.window.Liki.currency = undefined;
    await globalThis.window.Liki.goCheckout('naming', 'order-1');

    const body = globalThis.apiPost.mock.calls[0][1];
    expect(body.provider).toBe('dodo');
  });
});
