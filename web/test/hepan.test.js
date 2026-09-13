import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest';
import { el, readJS } from './helpers.js';

function makeIds(prefix) {
  const map = {};
  ['name', 'birth', 'city'].forEach(function(f) {
    map['hp-' + prefix + '-' + f] = el('input');
  });
  map['hp-' + prefix + '-male'] = el('input', { checked: true });
  map['hp-' + prefix + '-female'] = el('input');
  return map;
}

describe('hepan.js', () => {
  let els, code;

  beforeAll(() => { code = readJS('hepan'); });

  beforeEach(() => {
    globalThis.t = (k) => k;
    globalThis.Liki = { currency: 'CNY', t: globalThis.t, showErr: (el, msg) => { el.textContent = msg; el.classList.remove('hidden'); } };
    globalThis.window = { Liki: globalThis.Liki };
    globalThis.i18next = { isInitialized: true, language: 'zh-Hant', t: (k) => k };
    globalThis.i18n = { language: 'zh-Hant' };
    globalThis.createOrderAndCheckout = vi.fn().mockResolvedValue(undefined);

    els = {};
    els['hp-buy'] = el('button', { text: '获取报告' });
    els['form-error'] = el('p', { className: 'hidden' });
    els['hp-email'] = el('input');

    Object.assign(els, makeIds('a'), makeIds('b'));

    globalThis.document = {
      getElementById: (id) => els[id] || null,
      addEventListener: () => {},
    };
  });

  afterEach(() => {
    delete globalThis.document;
    delete globalThis.window;
    delete globalThis.Liki;
    delete globalThis.t;
    delete globalThis.i18next;
    delete globalThis.i18n;
    delete globalThis.createOrderAndCheckout;
  });

  function run() { eval(code); }

  function fill() {
    els['hp-a-name'].value = '张三';
    els['hp-a-birth'].value = '1990-05-20T14:30';
    els['hp-a-city'].value = '北京';
    els['hp-b-name'].value = '李四';
    els['hp-b-birth'].value = '1992-08-15T10:00';
    els['hp-b-city'].value = '上海';
    els['hp-email'].value = 'test@example.com';
  }

  it('passes { email } as 4th argument to createOrderAndCheckout', async () => {
    fill();
    run();
    await els['hp-buy']._lsclick({ preventDefault() {} });

    expect(globalThis.createOrderAndCheckout).toHaveBeenCalledTimes(1);
    const args = globalThis.createOrderAndCheckout.mock.calls[0];
    expect(args[2]).toBe('hepan');
    expect(args[3]).toEqual({ email: 'test@example.com' });
  });

  it('shows error when names are missing', async () => {
    fill();
    els['hp-a-name'].value = '';
    els['hp-b-name'].value = '';
    run();
    await els['hp-buy']._lsclick({ preventDefault() {} });

    expect(globalThis.createOrderAndCheckout).not.toHaveBeenCalled();
    expect(els['form-error'].classList._cs).not.toContain('hidden');
  });

  it('does not send client-controlled currency', async () => {
    globalThis.window.Liki.currency = 'USD';
    globalThis.Liki.currency = 'USD';
    fill();
    run();
    await els['hp-buy']._lsclick({ preventDefault() {} });

    const body = globalThis.createOrderAndCheckout.mock.calls[0][1];
    expect(body.currency).toBeUndefined();
  });
});
