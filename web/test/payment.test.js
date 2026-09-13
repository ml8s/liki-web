import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { readJS } from './helpers.js';

function el(tag) {
  return {
    _tag: tag,
    textContent: '',
    className: '',
    setAttribute: () => {},
    showModal: () => {},
    close: () => { this.closed = true; },
    appendChild: () => {},
    remove: () => {},
  };
}

let currentPaidHandler = () => {};

describe('payment.js shared intent QR flow', () => {
  let code;

  let pollerCode;
  beforeAll(() => { pollerCode = readJS('payment-poller'); code = readJS('payment'); });

  beforeEach(() => {
    vi.useFakeTimers();
    globalThis.window = { location: {} };
    globalThis.apiPost = vi.fn();
    globalThis.escapeHTML = (value) => String(value);;
    globalThis.fetch = vi.fn();
    globalThis.i18next = { isInitialized: true, t: (k) => k };
    const modal = {
      className: '', innerHTML: '', textContent: '',
      showModal: vi.fn(), close: vi.fn(), addEventListener: vi.fn(), appendChild: () => {},
      querySelector: () => ({ setAttribute: () => {} }),
    };
    const img = {};
    globalThis.CustomEvent = class CustomEvent {
      constructor(type, options) { this.type = type; this.detail = options?.detail; }
    };
    globalThis.document = {
      querySelector: () => null,
      dispatchEvent: (event) => { currentPaidHandler(event); },
      createElement: (tag) => tag === 'dialog' ? modal : img,
      body: { appendChild: () => {} },
    };
    eval(pollerCode);
    eval(code);
  });

  afterEach(() => {
    vi.useRealTimers();
    delete globalThis.window;
    delete globalThis.fetch;
    delete globalThis.apiPost;
    delete globalThis.i18next;
    delete globalThis.document;
  });

  it('polls intent status and dispatches paid with business order id', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: 'pi-1', business_order_id: 'biz-1', status: 'paid' } }),
    });

    window.Liki.showIntentQRModal('pi-1', 'https://qr.example/qr', 'https://pay.example/checkout');
    await vi.advanceTimersByTimeAsync(2000);
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(700);

    expect(fetch).toHaveBeenCalledWith('/api/payments/intents/pi-1/status');
    expect(document.dispatchEvent).toBeDefined();
    expect(window.location.href).toBeUndefined();
  });

  it('redirects to the product success URL after a QR payment is paid', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { id: 'pi-1', business_order_id: 'biz-1', status: 'paid' } }),
    });

    window.Liki.startIntentCheckout(
      'pi-1',
      { qr_url: 'https://qr.example/qr', checkout_url: 'https://pay.example/checkout' },
      '/zh-Hant/naming/chat?order_id=biz-1',
    );
    await vi.advanceTimersByTimeAsync(2000);
    await vi.advanceTimersByTimeAsync(700);

    expect(window.location.href).toBe('/zh-Hant/naming/chat?order_id=biz-1');
  });
});
