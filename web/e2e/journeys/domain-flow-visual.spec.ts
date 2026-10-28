import { expect, test } from '../fixtures'
import { flowProfile, flowSession, mockFlowWorkspace } from '../helpers/domain-flow'

const profiles = [flowProfile('person-1', '张三')]
const sessions = [flowSession('session-1')]

async function stabilize(page: Parameters<typeof mockFlowWorkspace>[0]) {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.addStyleTag({
    content: '*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}',
  })
}

test.describe('consultation visual contracts', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('mobile consultation shell', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles, sessions })
    await page.route('**/api/chat/sessions/session-1', route =>
      route.fulfill({ json: { data: sessions[0] } }))
    await page.route('**/api/chat/sessions/session-1/messages', route =>
      route.fulfill({ json: { data: { messages: [], next_cursor: '' } } }))
    await page.goto('/app/')
    await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
    await stabilize(page)

    await expect(page.locator('.app-flow-shell'))
      .toHaveScreenshot('consultation-mobile.png', {
        animations: 'disabled',
        mask: [page.locator('.flow-history__meta')],
        maxDiffPixelRatio: 0.02,
      })
  })

  test('mobile me hub', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles, sessions })
    await page.route('**/api/chat/sessions/session-1', route =>
      route.fulfill({ json: { data: sessions[0] } }))
    await page.route('**/api/chat/sessions/session-1/messages', route =>
      route.fulfill({ json: { data: { messages: [], next_cursor: '' } } }))
    await page.goto('/app/me')
    await expect(page.locator('.entry-row')).toHaveCount(4)
    await stabilize(page)

    await expect(page.locator('.page-stack'))
      .toHaveScreenshot('me-mobile.png', {
        animations: 'disabled',
        maxDiffPixelRatio: 0.02,
      })
  })
})

test.describe('consultation desktop visual contract', () => {
  test.use({ viewport: { width: 1280, height: 900 } })

  test('desktop consultation shell', async ({ page }) => {
    await mockFlowWorkspace(page, { profiles, sessions })
    await page.route('**/api/chat/sessions/session-1', route =>
      route.fulfill({ json: { data: sessions[0] } }))
    await page.route('**/api/chat/sessions/session-1/messages', route =>
      route.fulfill({ json: { data: { messages: [], next_cursor: '' } } }))
    await page.goto('/app/')
    await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
    await stabilize(page)

    await expect(page.locator('.app-flow-shell'))
      .toHaveScreenshot('consultation-desktop.png', {
        animations: 'disabled',
        mask: [page.locator('.flow-history__meta')],
        maxDiffPixelRatio: 0.02,
      })
  })
})
