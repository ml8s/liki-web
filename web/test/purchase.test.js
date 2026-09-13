import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { el, readJS } from './helpers.js';

describe('purchase.js', () => {
  let els, code;

  beforeAll(() => { code = readJS('purchase'); });

  beforeEach(() => {
    globalThis.Liki = { currency: 'CNY', t: (k) => k, showErr: (el, msg) => { el.textContent = msg; el.classList.remove('hidden'); } };
    globalThis.window = { Liki: globalThis.Liki };
    globalThis.i18next = { isInitialized: true, language: 'zh-Hant', t: (k) => k };
    globalThis.createOrderAndCheckout = vi.fn().mockResolvedValue(undefined);
    els = {
      'purchase-form': el('form'),
      'purchase-btn': el('button'),
      'purchase-error': el('p', { className: 'hidden' }),
      'purchase-email': el('input'),
    };
    globalThis.document = {
      getElementById: (id) => els[id] || null,
      addEventListener: () => {},
    };
  });

  afterEach(() => {
    delete globalThis.document;
    delete globalThis.window;
    delete globalThis.Liki;
    delete globalThis.i18next;
    delete globalThis.createOrderAndCheckout;
  });

  function run() { eval(code); }

  it('passes { email } as 4th argument to createOrderAndCheckout', async () => {
    els['purchase-email'].value = 'test@example.com';
    run();
    await els['purchase-form']._lssubmit({ preventDefault() {} });

    expect(globalThis.createOrderAndCheckout).toHaveBeenCalledTimes(1);
    const args = globalThis.createOrderAndCheckout.mock.calls[0];
    expect(args[0]).toBe('/api/naming/orders');
    expect(args[2]).toBe('naming');
    expect(args[3]).toEqual({ email: 'test@example.com' });
  });

  it('shows error when email is empty', async () => {
    els['purchase-email'].value = '';
    run();
    await els['purchase-form']._lssubmit({ preventDefault() {} });

    expect(globalThis.createOrderAndCheckout).not.toHaveBeenCalled();
    expect(els['purchase-error'].classList._cs).not.toContain('hidden');
  });

  it('does not send client-controlled currency', async () => {
    globalThis.window.Liki.currency = 'USD';
    globalThis.Liki.currency = 'USD';
    els['purchase-email'].value = 'test@example.com';
    run();
    await els['purchase-form']._lssubmit({ preventDefault() {} });

    const body = globalThis.createOrderAndCheckout.mock.calls[0][1];
    expect(body.currency).toBeUndefined();
  });
});
