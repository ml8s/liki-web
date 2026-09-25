// Shared Playwright API mock helpers.
// Use page.route() to intercept and fulfill backend requests.
// JWT is HttpOnly — can't be set from tests, so all APIs are mocked.

export function sseLine(obj) {
  return `data: ${JSON.stringify(obj)}\n\n`;
}

export async function mockLogin(page, orderID = 'e2e-order-123') {
  await page.route('**/api/naming/auth/login', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { order_id: orderID } }),
    });
  });
}

export async function mockOrderStatus(page, orderID = 'e2e-order-123') {
  await page.route(`**/api/naming/orders/${orderID}/status`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: { order_id: orderID, status: 'paid', product: 'naming', chat_expires_at: '2027-06-26 00:00:00', amount: 2990, currency: 'USD' },
      }),
    });
  });
}

export async function mockChatHistory(page, orderID = 'e2e-order-123', messages = []) {
  await page.route(`**/api/naming/orders/${orderID}/messages`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: messages }),
    });
  });
}

export async function setupAuthenticatedMocks(page, orderID = 'e2e-order-123') {
  await mockLogin(page, orderID);
  await mockOrderStatus(page, orderID);
  await mockChatHistory(page, orderID);
}

export async function mockNamingSSE(page, events) {
  await page.route('**/api/naming/agent/chat', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' },
      body: events,
    });
  });
}
