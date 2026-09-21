// @ts-check
import { test, expect } from '@playwright/test';

const json = data => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ data }),
});

function trackProblems(page) {
  /** @type {string[]} */
  const problems = [];
  page.on('console', message => {
    if (message.type() === 'error' || message.type() === 'warning') {
      problems.push(`[console:${message.type()}] ${message.text()}`);
    }
  });
  page.on('pageerror', error => problems.push(`[pageerror] ${error.message}`));
  page.on('requestfailed', request => {
    // Playwright aborts our own mocked requests only if a test mishandles them.
    if (request.failure()?.errorText !== 'net::ERR_ABORTED') {
      problems.push(`[requestfailed] ${request.url()} ${request.failure()?.errorText || ''}`);
    }
  });
  return problems;
}

// These tests can run without the full stack: fulfill the isolated static shell
// and its modules, then mock only the APIs each partner surface owns.
async function serveStaticConsole(page, name) {
  await page.route(new RegExp(`(?:^|/)${name}$`), route => route.fulfill({
    path: `${name}.html`, contentType: 'text/html; charset=utf-8',
  }));
  for (const file of ['console-core.js', `console-${name}.js`]) {
    await page.route(`**/js/${file}`, route => route.fulfill({
      path: `js/${file}`, contentType: 'text/javascript; charset=utf-8',
    }));
  }
}

test.describe('Partner consoles', () => {
  test('KOL renders multi-currency balances and submits a payout', async ({ page }) => {
    const problems = trackProblems(page);
    let payoutBody = null;
    await serveStaticConsole(page, 'kol');

    await page.route('**/api/me', route => route.fulfill(json({
      id: 'user_kol', email: 'kol@example.com', status: 'active', locale: 'zh-Hant',
    })));
    await page.route('**/api/kol/status', route => route.fulfill(json({
      registered: true, eligible: true, status: 'active',
    })));
    await page.route('**/api/kol/session', route => route.fulfill(json({
      email: 'kol@example.com',
      partner: { user_id: 'user_kol', code: 'KOLCODE123', role: 'kol', status: 'active' },
      overview: {
        referrer: { user_id: 'user_kol', code: 'KOLCODE123', role: 'kol', status: 'active' },
        attributed_users: 7,
        balances: [
          { currency: 'CNY', total_minor: 123456, pending_minor: 50000, paid_minor: 73456 },
          { currency: 'USD', total_minor: 1000, pending_minor: 1000, paid_minor: 0 },
        ],
      },
    })));
    await page.route('**/api/kol/commissions*', route => route.fulfill(json({
      commissions: [{
        id: 'c_1', business_order_id: 'order_1', amount_minor: 8800, currency: 'CNY',
        referrer_commission_minor: 4400, status: 'pending', created_at: '2026-01-01T00:00:00Z',
      }],
      balances: [{ currency: 'CNY', total_minor: 4400, pending_minor: 4400, paid_minor: 0 }],
    })));
    await page.route('**/api/kol/payouts', async route => {
      if (route.request().method() === 'POST') {
        payoutBody = route.request().postDataJSON();
        return route.fulfill(json({ payout: { id: 'po_1', amount_minor: 5000, currency: 'CNY', status: 'requested' } }));
      }
      return route.fulfill(json({ payouts: [{
        id: 'po_old', amount_minor: 73456, currency: 'CNY', method: 'alipay',
        account: 'paid@example.com', status: 'paid', created_at: '2026-01-02T00:00:00Z',
      }] }));
    });

    await page.goto('/kol');
    await expect(page.getByRole('heading', { name: '合作创作者中心' })).toBeVisible();
    await expect(page.locator('#referral-link')).toHaveValue(/\/r\/KOLCODE123$/);
    await expect(page.locator('#metrics')).toContainText('累计收益 · CNY');
    await expect(page.locator('#metrics')).toContainText('CNY 1234.56');
    await expect(page.locator('#metrics')).toContainText('USD 10.00');
    await expect(page.locator('#commissions tbody tr')).toHaveCount(1);

    await page.locator('#method').selectOption('alipay');
    await page.locator('#account').fill('payout@example.com');
    await page.locator('#payout-form').getByRole('button', { name: '申请提现' }).click();

    await expect(page.locator('#payout-status')).toContainText('提现申请已提交');
    expect(payoutBody).toEqual({ method: 'alipay', account: 'payout@example.com' });
    await expect(page.locator('#payouts tbody tr')).toHaveCount(1);
    expect(problems).toEqual([]);
    const kolText = (await page.locator('body').innerText()).toLowerCase();
    expect(kolText).not.toContain('scout');
    expect(kolText).not.toContain('channel');
  });

  test('Scout renders channel balances and submits a payout without KOL exposure', async ({ page }) => {
    const problems = trackProblems(page);
    let payoutBody = null;
    await serveStaticConsole(page, 'scout');

    await page.route('**/api/me', route => route.fulfill(json({
      id: 'user_scout', email: 'scout@example.com', status: 'active', locale: 'zh-Hant',
    })));
    await page.route('**/api/scout/session', route => route.fulfill(json({
      email: 'scout@example.com',
      overview: {
        partner: 'scout', recruit_count: 2, order_count: 9,
        balances: [{ currency: 'CNY', sales_minor: 500000, commission_minor: 100000, pending_minor: 100000, paid_minor: 0 }],
        recruits: [{
          user_id: 'kol_1', code: 'KOLCODE123', created_at: '2026-01-01T00:00:00Z',
          order_count: 9, commission_balances: [{ currency: 'CNY', amount_minor: 100000 }],
        }],
      },
    })));
    await page.route('**/api/scout/commissions*', route => route.fulfill(json({
      sales: [{
        order_ref: 'order_9', amount_minor: 50000, currency: 'CNY',
        scout_commission_minor: 10000, status: 'pending', refunded_minor: 0,
        created_at: '2026-01-03T00:00:00Z',
      }],
      balances: [{ currency: 'CNY', amount_minor: 100000 }],
    })));
    await page.route('**/api/scout/payouts', async route => {
      if (route.request().method() === 'POST') {
        payoutBody = route.request().postDataJSON();
        return route.fulfill(json({ payout: { id: 'po_scout', amount_minor: 50000, status: 'requested' } }));
      }
      return route.fulfill(json({ payouts: [] }));
    });

    await page.goto('/scout');
    await expect(page.getByRole('heading', { name: '渠道合作中心' })).toBeVisible();
    await expect(page.locator('#metrics')).toContainText('销售金额 · CNY');
    await expect(page.locator('#metrics')).toContainText('CNY 5000.00');
    await expect(page.locator('#recruits tbody tr')).toHaveCount(1);

    await page.locator('#method').selectOption('paypal');
    await page.locator('#account').fill('scout@example.com');
    await page.locator('#payout-form').getByRole('button', { name: '申请提现' }).click();

    await expect(page.locator('#payout-status')).toContainText('提现申请已提交');
    expect(payoutBody).toEqual({ method: 'paypal', account: 'scout@example.com' });
    await expect(page.locator('#payouts tbody tr')).toHaveCount(1);
    expect(problems).toEqual([]);
    const scoutText = (await page.locator('body').innerText()).toLowerCase();
    expect(scoutText).not.toContain('kol');
  });

  test('unauthenticated KOL visitor sees only login', async ({ page }) => {
    const problems = trackProblems(page);
    await serveStaticConsole(page, 'kol');
    // Browsers record the expected 401 as a console resource error; the UI
    // handles it and displays the isolated login surface.
    await page.route('**/api/me', route => route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({
      error: { code: 'unauthorized', message: 'unauthorized' },
    }) }));

    await page.goto('/kol');
    await expect(page.getByRole('heading', { name: '合作创作者中心' })).toBeVisible();
    await expect(page.locator('#auth-section')).toBeVisible();
    await expect(page.locator('#email')).toBeVisible();
    expect(problems.filter(problem => !problem.includes('status of 401'))).toEqual([]);
  });
});
