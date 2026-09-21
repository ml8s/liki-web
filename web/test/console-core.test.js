import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';

beforeEach(async () => {
  globalThis.window = {};
  await import('../js/console-core.js');
});

afterEach(() => {
  vi.unstubAllGlobals();
  delete globalThis.window;
});


describe('console core', () => {
  it('escapes every HTML semantic character', async () => {
    const core = globalThis.LikiConsole;
    expect(core.escapeHTML('<a href="x" \'y\'>&')).toBe('&lt;a href=&quot;x&quot; &#39;y&#39;&gt;&amp;');
  });

  it('formats minor units with the returned currency', async () => {
    const core = globalThis.LikiConsole;
    expect(core.money(123456, 'CNY')).toBe('CNY 1234.56');
    expect(core.money(0, '')).toBe('0.00');
  });

  it('masks email and short accounts without retaining the full value', async () => {
    const core = globalThis.LikiConsole;
    expect(core.maskAccount('payout@example.com')).toBe('p***@example.com');
    expect(core.maskAccount('13800001234')).toBe('13****1234');
    expect(core.maskAccount('12')).toBe('****');
  });

  it('unwraps API data and converts API errors to Error messages', async () => {
    const core = globalThis.LikiConsole;
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ data: { value: 7 } }), { status: 200 },
    )));
    await expect(core.request('/example')).resolves.toEqual({ value: 7 });

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: { message: 'denied' } }), { status: 403 },
    )));
    await expect(core.request('/example')).rejects.toThrow('denied');
  });
});
