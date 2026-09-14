// @ts-check
import { test, expect } from '@playwright/test';

const protectedPages = [
  { path: '/app/', heading: '工作台' },
  { path: '/app/profiles', heading: '档案' },
  { path: '/app/chat', heading: 'Liki Chat' },
  { path: '/app/reports', heading: '报告' },
  { path: '/app/orders', heading: '订单' },
  { path: '/app/account', heading: '账户' },
];

const json = data => ({
  status: 200,
  contentType: 'application/json',
  body: JSON.stringify({ data }),
});

test.describe('Authenticated app shell', () => {
  test('login page mounts without runtime errors', async ({ page }) => {
    const errors = [];
    page.on('pageerror', error => errors.push(error));
    await page.goto('/app/login');
    await expect(page).toHaveURL(/\/app\/login$/);
    await expect(page.locator('input[type="email"]')).toBeVisible();
    expect(errors).toEqual([]);
  });

  test('protected routes redirect unauthenticated users to login', async ({ page }) => {
    for (const { path } of protectedPages) {
      await page.goto(path);
      await expect(page).toHaveURL(/\/app\/login/);
    }
  });

  test('authenticated pages render their headings', async ({ page }) => {
    await page.route('**/api/me', async route => {
      await route.fulfill(json({
        id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant',
      }));
    });
    await page.route('**/api/me/entitlements', async route => {
      await route.fulfill(json({ entitlements: [], allowed_skills: ['liki-naming'] }));
    });
    await page.route('**/api/chat/sessions', async route => route.fulfill(json([])));
    await page.route('**/api/chat/products', async route => route.fulfill(json([])));
    await page.route('**/api/profiles**', async route => route.fulfill(json({ profiles: [], next_cursor: '' })));

    for (const { path, heading } of protectedPages) {
      await page.goto(path);
      await expect(page.getByRole('heading', { name: heading })).toBeVisible();
    }
  });

  test('chat direct link restores session and messages', async ({ page }) => {
    await page.route('**/api/me', async route => route.fulfill(json({
      id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant',
    })));
    await page.route('**/api/me/entitlements', async route => route.fulfill(json({
      entitlements: [], allowed_skills: ['liki-naming'],
    })));
    await page.route('**/api/chat/sessions/chat_1', async route => route.fulfill(json({
      id: 'chat_1', user_id: 'usr_test', profile_id: '', skill: 'liki-naming', title: '起名咨询',
      status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
      last_message_at: null,
    })));
    await page.route('**/api/chat/sessions/chat_1/messages', async route => route.fulfill(json([
      { id: 'msg_1', session_id: 'chat_1', user_id: 'usr_test', role: 'user', kind: 'text', content: '你好', created_at: '2026-01-01T00:01:00Z' },
      { id: 'msg_2', session_id: 'chat_1', user_id: 'usr_test', role: 'assistant', kind: 'text', content: '你好，请提供信息。', created_at: '2026-01-01T00:02:00Z' },
    ])));
    await page.route('**/api/chat/products', async route => route.fulfill(json([])));

    await page.goto('/app/chat/chat_1');
    await expect(page.getByRole('heading', { name: '起名咨询' })).toBeVisible();
    await expect(page.getByText('你好，请提供信息。')).toBeVisible();
  });
});

test('paid return confirms entitlement and jumps straight into a session', async ({ page }) => {
  const profile = {
    id: 'p_1', user_id: 'usr_test', version: 1, nickname: '妈妈', name: '', gender: 'female',
    relation: '家人', calendar: 'lunar', birth_date: '1958-闰5-12', birth_time: '06:00',
    birth_time_known: 'known', birth_place: '上海', note: '', third_party_consent: true,
    last_used_at: null, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  };

  await page.route('**/api/me', route => route.fulfill(json({
    id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant',
  })));

  let entitlementRequest = 0;
  await page.route('**/api/me/entitlements', route => {
    entitlementRequest += 1;
    const allowed = entitlementRequest > 1
      ? ['liki-naming', 'liki-bazi', 'liki-divination', 'liki-fengshui']
      : [];
    return route.fulfill(json({ entitlements: [], allowed_skills: allowed }));
  });

  let productRequest = 0;
  await page.route('**/api/chat/products', route => {
    productRequest += 1;
    return route.fulfill(json([{
      id: 'liki-chat',
      name: 'Liki Chat',
      description: 'All four skills.',
      skills: ['liki-naming', 'liki-bazi', 'liki-divination', 'liki-fengshui'],
      prices: { CNY: 2990, USD: 2990 },
      purchased: entitlementRequest > 1,
    }]));
  });

  await page.route('**/api/profiles**', route => route.fulfill(json({ profiles: [profile], next_cursor: '' })));
  await page.route('**/api/chat/sessions', route => {
    if (route.request().method() === 'POST') {
      return route.fulfill(json({
        id: 'chat_new', user_id: 'usr_test', profile_id: profile.id, skill: 'liki-bazi', title: '八字紫微咨询',
        status: 'active', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z', last_message_at: null,
      }));
    }
    return route.fulfill(json([]));
  });
  await page.route('**/api/chat/sessions/chat_new/messages', route => route.fulfill(json([])));

  await page.goto('/app/chat?order_id=order_1&status=paid&payment_intent_id=pi_1&status_token=token');
  // 权益确认后自动用默认 skill 建会话并直达。
  await expect(page).toHaveURL(/\/app\/chat\/chat_new$/);
  await expect(page.getByRole('heading', { name: '八字紫微咨询' })).toBeVisible();
});

test('profile create form mounts and validates', async ({ page }) => {
  await page.route('**/api/me', route => route.fulfill(json({
    id: 'usr_test', email: 'user@example.com', status: 'active', locale: 'zh-Hant',
  })));
  await page.route('**/api/me/entitlements', route => route.fulfill(json({ entitlements: [], allowed_skills: [] })));

  await page.goto('/app/profiles/new');
  await expect(page.getByRole('heading', { name: '新建档案' })).toBeVisible();
  await expect(page.getByPlaceholder('如：妈妈 / 我本人')).toBeVisible();
});