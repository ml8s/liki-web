import { expect, test } from '../fixtures'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'
import { mockBFFChatMessage } from '../helpers/chat'

const profile = flowProfile('person-1', '张三')

function createdSession() {
  return {
    id: 'session-offline',
    user_id: 'usr_flow',
    profile_id: 'person-1',
    skill: 'liki',
    expert: 'natal',
    title: '张三 · 命理',
    status: 'active',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_message_at: null,
  }
}

test.describe('consultation resilience', () => {
  test.describe(() => {
    test.use({ viewport: { width: 390, height: 844 } })

    test('keeps the draft and moves focus when session creation fails', async ({ page }) => {
      await mockFlowWorkspace(page, { profiles: [profile] })
      await page.route('**/api/chat/sessions', async route => {
        if (route.request().method() !== 'POST') return route.fallback()
        return route.fulfill({
          status: 403,
          contentType: 'application/json',
          body: JSON.stringify({ type: 'entitlement_required', title: 'No entitlement' }),
        })
      })

      await page.goto('/app/?locale=zh-Hans')
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
      const question = page.getByRole('textbox', { name: '直接提出你的问题' })
      await question.fill('今年事业如何？')
      const send = page.getByRole('button', { name: '发送' })
      await expect(send).toBeEnabled()
      await send.click()

      const alert = page.locator('.flow-error')
      await expect(alert).toBeVisible()
      await expect(alert).toContainText(/网络|请求失败|无权|创建会话失败/)
      await expect(page.locator('.flow-error-focus')).toBeFocused()
      await expect(question).toHaveValue('今年事业如何？')
    })

    test('restores the draft when streaming becomes unreachable', async ({ page }) => {
      await mockFlowWorkspace(page, { profiles: [profile] })
      await page.route('**/api/chat/sessions', async route => {
        if (route.request().method() !== 'POST') return route.fallback()
        return route.fulfill({ json: { data: createdSession() } })
      })
      await page.route('**/api/chat/sessions/session-offline', route => {
        return route.fulfill({ json: { data: createdSession() } })
      })
      await page.route('**/api/chat/sessions/session-offline/messages', route => {
        return route.fulfill({ json: { data: { messages: [], next_cursor: '' } } })
      })
      await page.route('**/api/birth-profiles/person-1', route => {
        return route.fulfill({ json: { data: profile } })
      })
      await mockBFFChatMessage(page, {
        session: createdSession(),
        answer: '不应到达的答案',
        permanentFailure: true,
      })

      await page.goto('/app/?locale=zh-Hans')
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
      const question = page.getByRole('textbox', { name: '直接提出你的问题' })
      await question.fill('今年事业如何？')
      await page.getByRole('button', { name: '发送' }).click()

      await expect(page.locator('.flow-error')).toBeVisible()
      await expect(page.locator('.flow-error')).toContainText(/网络|失败/)
      await expect(question).toHaveValue('今年事业如何？')
    })
  })
})
