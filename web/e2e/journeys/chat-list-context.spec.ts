import { expect, test } from '../fixtures'
import type { Page } from '@playwright/test'
import { json, mockWorkspace, testUser } from '../mocks'

const matchingBirth = {
  gender: 'female',
  date: '1958-05-12',
  time: '06:00',
  location: { longitude: 121.4726, timezone: 'Asia/Shanghai' },
}

function profile(overrides: Record<string, unknown> = {}) {
  return {
    user_id: 'usr_context',
    version: 1,
    name: '',
    note: '',
    third_party_consent: true,
    last_used_at: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    birth: {
      gender: 'male',
      date: '1990-05-12',
      time: '08:30',
      instant: '1990-05-12T00:30:00Z',
      location: { place: '北京', longitude: 116.4074, timezone: 'Asia/Shanghai' },
    },
    ...overrides,
  }
}

async function seedHandoff(page: Page, birth: object) {
  await page.addInitScript(value => {
    sessionStorage.setItem('liki:form-drafts', JSON.stringify({
      'handoff:birth': { data: value, createdAt: Date.now() },
      'handoff:consultation': {
        data: { version: 1, intent: 'naming', birth: value },
        createdAt: Date.now(),
      },
    }))
  }, birth)
}

test('Chat List creates a Naming session with the selected profile birth facts', async ({ page }) => {
  const user = testUser({ id: 'usr_context', email: 'context@example.com' })
  const session = {
    id: 'chat_context',
    user_id: user.id,
    profile_id: 'p_1',
    skill: 'liki',
    intent: 'naming',
    title: '命名諮詢',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_message_at: null,
  }
  const created: Array<{ context?: { intent?: string; birth?: Record<string, unknown> } }> = []

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

  await page.goto('/app/classic/chat?intent=naming')
  await page.locator('#chat-intent .entry-row', { hasText: '起名' }).click()

  await expect.poll(() => created.length).toBe(1)
  expect(created[0]?.context).toEqual({
    version: 1,
    intent: 'naming',
      birth: {
        ...matchingBirth,
      },
  })
})

test('Chat List selects the profile that matches a public tool handoff', async ({ page }) => {
  const user = testUser({ id: 'usr_context', email: 'context@example.com' })
  const session = {
    id: 'chat_matched', user_id: user.id, profile_id: 'p_match', skill: 'liki', intent: 'naming',
    title: '命名諮詢', status: 'active', created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z', last_message_at: null,
  }
  const created: Array<{ profile_id?: string; context?: { birth?: object } }> = []
  await seedHandoff(page, matchingBirth)
  await mockWorkspace(page, {
    user,
    allowedSkills: ['liki'],
    profiles: [
      profile({ id: 'p_other', nickname: '爸爸' }),
      profile({ id: 'p_match', nickname: '媽媽', birth: { ...matchingBirth, instant: '1958-05-11T22:00:00Z', location: { place: '上海', ...matchingBirth.location } } }),
    ],
  })
  await page.route('**/api/chat/sessions', async route => {
    if (route.request().method() === 'POST') {
      created.push(route.request().postDataJSON())
      return route.fulfill(json(session))
    }
    return route.fulfill(json([]))
  })
  await page.route('**/api/chat/sessions/chat_matched', route => route.fulfill(json(session)))
  await page.route('**/api/chat/sessions/chat_matched/messages', route =>
    route.fulfill(json({ messages: [], next_cursor: '' })))

  await page.goto('/app/classic/chat?intent=naming')
  await page.locator('#chat-intent .entry-row', { hasText: '起名' }).click()

  await expect.poll(() => created.length).toBe(1)
  expect(created[0]?.profile_id).toBe('p_match')
  expect(created[0]?.context?.birth).toEqual(matchingBirth)
})

test('Chat List routes an unmatched public birth handoff to profile creation', async ({ page }) => {
  const user = testUser({ id: 'usr_context', email: 'context@example.com' })
  await seedHandoff(page, matchingBirth)
  await mockWorkspace(page, {
    user,
    allowedSkills: ['liki'],
    profiles: [profile({ id: 'p_other', nickname: '爸爸' })],
  })

  await page.goto('/app/classic/chat?intent=naming')
  await page.locator('#chat-intent .entry-row', { hasText: '起名' }).click()

  await expect(page).toHaveURL(/\/app\/classic\/birth-profiles\/new/)
  await expect(page.getByText('已帶入主頁的出生資訊，可繼續排盤')).toBeVisible()
})
