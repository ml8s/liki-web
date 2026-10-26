import { expect, type Page } from '@playwright/test'
import { mockWorkspace } from '../mocks'

export type JsonRecord = Record<string, unknown>

const now = '2026-01-01T00:00:00Z'

export function flowProfile(id: string, nickname: string): JsonRecord {
  return {
    id,
    user_id: 'usr_flow',
    version: 1,
    nickname,
    name: '',
    birth: {
      gender: 'female',
      date: '1990-06-15',
      time: '08:30',
      instant: '1990-06-14T23:30:00Z',
      location: {
        place: '上海',
        place_id: 'CN-310000',
        location_version: 2,
        longitude: 121.4726,
        timezone: 'Asia/Shanghai',
      },
    },
    note: '',
    third_party_consent: true,
    last_used_at: null,
    created_at: now,
    updated_at: now,
  }
}

export function flowSession(id: string, overrides: JsonRecord = {}): JsonRecord {
  return {
    id,
    user_id: 'usr_flow',
    profile_id: 'person-1',
    skill: 'liki',
    expert: 'natal',
    title: '张三 · 命理',
    status: 'active',
    created_at: now,
    updated_at: now,
    last_message_at: null,
    context: { scenario: 'natal' },
    ...overrides,
  }
}

export function flowMessage(
  id: string,
  sessionId: string,
  role: 'user' | 'assistant',
  content: string,
): JsonRecord {
  return {
    id,
    session_id: sessionId,
    user_id: 'usr_flow',
    role,
    kind: 'text',
    content,
    created_at: '2026-01-01T00:01:00Z',
  }
}

/** 用简体中文固定界面文案，避免宿主机 Accept-Language 影响断言。 */
export async function mockFlowWorkspace(
  page: Page,
  options: {
    profiles?: JsonRecord[]
    sessions?: JsonRecord[]
    productEvents?: JsonRecord[]
  } = {},
) {
  await mockWorkspace(page, {
    user: {
      id: 'usr_flow',
      email: 'flow@example.com',
      status: 'active',
      locale: 'zh-Hans',
      partner_roles: [],
    },
    allowedSkills: ['liki'],
    profiles: options.profiles ?? [],
    sessions: options.sessions ?? [],
  })
  await page.route('**/api/client-errors', route => route.fulfill({ status: 204 }))
  await page.route('**/api/analytics/product-event', async route => {
    options.productEvents?.push(await route.request().postDataJSON() as JsonRecord)
    await route.fulfill({ status: 204 })
  })
}

export async function openFlowDomainSheet(page: Page) {
  await page.locator('.flow-parameters').getByRole('button', { name: '命理' }).click()
  const sheet = page.locator('.flow-sheet__body')
  await expectFlowSheet(page, sheet, '选择领域')
  return sheet
}

export async function openFlowPersonSheet(page: Page, personLabel: string) {
  await page.locator('.flow-parameters').getByRole('button', { name: personLabel }).click()
  const sheet = page.locator('.flow-sheet__body')
  await expectFlowSheet(page, sheet, '选择档案')
  return sheet
}

async function expectFlowSheet(page: Page, sheet: ReturnType<Page['locator']>, title: string) {
  await expect(page.locator('.flow-sheet__header h2')).toHaveText(title)
  await expect(sheet).toBeVisible()
}
