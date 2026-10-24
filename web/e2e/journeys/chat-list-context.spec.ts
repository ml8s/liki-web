import { expect, test } from '../fixtures'
import { json, mockWorkspace, testUser } from '../mocks'

test('Chat List creates a Naming session with the selected profile birth facts', async ({ page }) => {
  const user = testUser({ id: 'usr_context', email: 'context@example.com' })
  const session = {
    id: 'chat_context',
    user_id: user.id,
    profile_id: 'p_1',
    skill: 'liki',
    expert: 'naming',
    title: '命名諮詢',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_message_at: null,
  }
  const created: Array<{ context?: { scenario?: string; birth?: Record<string, unknown> } }> = []

  await mockWorkspace(page, {
    user,
    allowedSkills: ['liki'],
    profiles: [{
      id: 'p_1',
      user_id: user.id,
      version: 1,
      nickname: '媽媽',
      name: '',
      birth: {
        gender: 'female',
        date: '1958-05-12',
        time: '06:00',
        instant: '1958-05-11T22:00:00Z',
        location: { place: '上海', longitude: 121.4726, timezone: 'Asia/Shanghai' },
      },
      note: '',
      third_party_consent: true,
      last_used_at: null,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }],
  })
  await page.route('**/api/chat/sessions', async route => {
    if (route.request().method() === 'POST') {
      created.push(route.request().postDataJSON())
      return route.fulfill(json(session))
    }
    return route.fulfill(json([]))
  })
  await page.route('**/api/chat/sessions/chat_context', route => route.fulfill(json(session)))
  await page.route('**/api/chat/sessions/chat_context/messages', route =>
    route.fulfill(json({ messages: [], next_cursor: '' })))

  await page.goto('/app/chat?service=naming')
  await page.locator('#chat-domain .entry-row', { hasText: '起名' }).click()

  await expect.poll(() => created.length).toBe(1)
  expect(created[0]?.context).toEqual({
    version: 1,
    scenario: 'naming',
    birth: {
      gender: 'female',
      date: '1958-05-12',
      time: '06:00',
      location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
    },
  })
})
