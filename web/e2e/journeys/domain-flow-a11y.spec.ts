import { expect, test } from '../fixtures'
import { expectNoCriticalA11yViolations } from '../helpers/a11y'
import { flowProfile, mockFlowWorkspace } from '../helpers/domain-flow'
import { mockBFFChatMessage } from '../helpers/chat'

test.describe('consultation accessibility', () => {
  test.describe(() => {
    test.use({ viewport: { width: 390, height: 844 } })

    test('passes the accessibility contract and keyboard journey', async ({ page }) => {
      await mockFlowWorkspace(page, {
        profiles: [flowProfile('person-1', '张三')],
      })

      await page.goto('/app/')
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()
      await expectNoCriticalA11yViolations(page, '/app consultation')

      const tabs = page.getByRole('tab')
      await expect(tabs).toHaveText(['命理', '问卦', '风水', '起名'])
      await tabs.first().focus()
      await page.keyboard.press('ArrowRight')
      await expect(tabs.nth(1)).not.toHaveAttribute('aria-selected', 'true')
      await expect(tabs.nth(1)).toBeFocused()
      await tabs.nth(1).focus()
      await page.keyboard.press('ArrowLeft')
      await expect(tabs.first()).toBeFocused()
      await expect(tabs.first()).toHaveAttribute('aria-selected', 'true')
      await page.keyboard.press('Enter')
      await page.keyboard.press('Home')
      await expect(tabs.first()).toBeFocused()
      await page.keyboard.press('Enter')
      await expect(tabs.first()).toHaveAttribute('aria-selected', 'true')

      await page.locator('.flow-parameters').getByRole('button', { name: '张三' }).click()
      const sheet = page.locator('.flow-sheet__body')
      await expect(sheet).toBeVisible()
      await expectNoCriticalA11yViolations(page, '/app consultation person sheet')
      await page.keyboard.press('Escape')
      await expect(sheet).toBeHidden()
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeFocused()

      await page.goto('/app/me')
      await expect(page.getByRole('heading', { name: '我的' })).toBeVisible()
      await expect(page.locator('.entry-row')).toHaveCount(4)
      await expectNoCriticalA11yViolations(page, '/app/me')
    })
  })
})

test.describe('consultation keyboard reflow', () => {
  test.describe(() => {
    test.use({ viewport: { width: 320, height: 720 } })

    test('works at 320px without clipping primary controls', async ({ page }) => {
      await mockFlowWorkspace(page, {
        profiles: [flowProfile('person-1', '张三')],
      })

      const createdSessions: unknown[] = []
      const session = {
        id: 'chat_new',
        user_id: 'usr_flow',
        profile_id: 'person-1',
        skill: 'liki',
        expert: 'natal',
        title: '张三 · 命理',
        status: 'active',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-02T00:00:00Z',
        last_message_at: null,
      }
      await page.route('**/api/chat/sessions', async route => {
        if (route.request().method() !== 'POST') return route.fallback()
        createdSessions.push(route.request().postDataJSON())
        return route.fulfill({ json: { data: session } })
      })
      await mockBFFChatMessage(page, {
        session,
        answer: '先看你的八字，日主偏弱。',
      })
      await page.route('**/api/chat/sessions/chat_new/messages', route => route.fulfill({
        json: { data: { messages: [], next_cursor: '' } },
      }))
      await page.route('**/api/birth-profiles/person-1', route => route.fulfill({
        json: { data: flowProfile('person-1', '张三') },
      }))

      await page.goto('/app/')
      await expect(page.locator('.flow-parameters').getByRole('button', { name: '张三' })).toBeVisible()

      // Keyboard activation reaches the shell's skip target, then focus moves
      // into the consultation document without pointer interaction.
      await page.locator('.flow-skip-link').focus()
      await page.keyboard.press('Enter')
      await expect(page.locator('.flow-main')).toBeFocused()

      // 320px reflow: the strip intentionally scrolls with a peek while the
      // page itself never overflows.
      const strip = page.locator('.flow-domain-strip')
      await expect(strip).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320)
      expect(await strip.evaluate(node => node.scrollWidth - node.clientWidth)).toBeGreaterThan(1)

      const personTrigger = page.locator('.flow-parameters').getByRole('button', { name: '张三' })
      await personTrigger.focus()
      await page.keyboard.press('Enter')
      const sheet = page.locator('.flow-sheet__body')
      await expect(sheet).toBeVisible()
      await page.keyboard.press('Escape')
      await expect(sheet).toBeHidden()
      await expect(personTrigger).toBeFocused()

      // The composer remains available on a natal draft. Focused-form composer
      // hiding and radio commit are covered by focused domain-flow suites.
      const domainTrigger = page.locator('.flow-parameters').getByRole('button', { name: '命理' })
      await domainTrigger.focus()
      await page.keyboard.press('Enter')
      const domainSheet = page.getByRole('region', { name: '选择领域' })
      await expect(domainSheet).toBeVisible()
      const domainRadios = domainSheet.getByRole('radio', { name: /命理|问卦|风水|起名/ })
      await expect(domainRadios).toHaveCount(4)
      await expect(domainRadios.first()).toBeChecked()
      await page.keyboard.press('Escape')
      await expect(domainSheet).toBeHidden()
      await expect(domainTrigger).toBeFocused()


      const question = page.getByRole('textbox', { name: '直接提出你的问题' })
      await question.focus()
      await question.fill('今年事业如何？')
      await page.getByRole('button', { name: '发送' }).click()
      await expect.poll(() => createdSessions.length).toBe(1)

      await expect(page.getByRole('list', { name: '咨询记录' })).toContainText('日主偏弱')

      // History (after a real session exists): the title filter is reachable by
      // keyboard and filters without a pointer.
      const historyFilter = page.getByRole('searchbox', { name: '按标题筛选' })
      await historyFilter.focus()
      await historyFilter.fill('不存在的会话')
      await expect(page.locator('.flow-history__empty')).toBeVisible()
      await historyFilter.fill('')
      await expect(page.locator('.flow-history__list')).toBeVisible()
      expect(createdSessions).toHaveLength(1)

      const overflowTrigger = page.locator('.flow-overflow-trigger')
      await overflowTrigger.focus()
      await page.keyboard.press('Enter')
      const contextMenu = page.getByRole('dialog', { name: '更多入口' }).locator('.flow-sheet__body')
      await expect(contextMenu).toBeVisible()
      await contextMenu.getByRole('button', { name: '我的' }).click()
      await expect(page).toHaveURL(/\/app\/me$/)
      await expect(page.locator('.entry-row')).toHaveCount(4)

      await page.locator('.entry-row').first().focus()
      await page.keyboard.press('Enter')
      await expect(page).toHaveURL(/\/app\/classic\/birth-profiles$/)
    })
  })
})
