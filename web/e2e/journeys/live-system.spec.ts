import { test, expect } from '../fixtures'

const answer = process.env.LIKI_E2E_ANSWER || 'E2E_FULL_CAPABILITY_ANSWER'
const publicUrl = new URL(process.env.BASE_URL || 'http://localhost:8080').origin

test('full system chat journey through Caddy and AG-UI @system', async ({ page }) => {
  test.skip(!process.env.LIVE_SYSTEM_E2E, 'Set LIVE_SYSTEM_E2E=1 for the deployed system journey')

  const sessionValue = process.env.LIKI_E2E_SESSION_VALUE ?? ''
  test.skip(!sessionValue, 'A seeded liki_session cookie is required')

  await page.context().addCookies([{
    name: 'liki_session',
    value: sessionValue,
    url: page.url().startsWith('http') ? new URL(page.url()).origin : process.env.BASE_URL,
    httpOnly: true,
    secure: true,
    sameSite: 'Lax',
  }])

  const profileResponse = await page.request.post('/api/birth-profiles', {
    headers: { Origin: publicUrl, Referer: `${publicUrl}/` },
    data: {
      nickname: 'System E2E',
      gender: 'male',
      calendar: 'solar',
      birth_date: '1990-05-20',
      birth_time: '12:00',
      birth_time_known: 'known',
      birth_place: 'Beijing',
      note: '',
      third_party_consent: false,
    },
  })
  expect(profileResponse.ok(), await profileResponse.text()).toBe(true)
  const profile = (await profileResponse.json()).data

  const sessionResponse = await page.request.post('/api/chat/sessions', {
    headers: { Origin: publicUrl, Referer: `${publicUrl}/` },
    data: {
      profile_id: profile.id,
      skill: 'liki',
      title: 'Full system E2E',
    },
  })
  expect(sessionResponse.ok(), await sessionResponse.text()).toBe(true)
  const session = (await sessionResponse.json()).data

  await page.goto(`/app/chat/${session.id}`)
  // .thread is shared by the loading wrapper and the message log; target the
  // accessible log explicitly (aria-label is localized).
  await expect(page.getByRole('log')).toBeVisible()

  const composer = page.locator('.composer textarea')
  await composer.fill('请基于我的出生档案做完整系统 E2E 分析')
  await page.locator('.composer button[type="submit"]').click()

  const assistant = page.locator('article.message.assistant .content')
  await expect(assistant.first()).toContainText(answer, { timeout: 90_000 })

  await page.reload()
  await expect(page.locator('article.message.assistant .content').first()).toContainText(answer)

  const messagesResponse = await page.request.get(`/api/chat/sessions/${session.id}/messages`)
  expect(messagesResponse.ok()).toBe(true)
  const messages: { role: string; content: string }[] = (await messagesResponse.json()).data
  const persisted = messages.filter(item => item.role === 'assistant').some(item => item.content === answer)
  expect(persisted).toBe(true)
})
