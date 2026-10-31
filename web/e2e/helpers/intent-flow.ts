import type { Page } from '@playwright/test'
import { mockWorkspace } from '../mocks'
import type { JsonRecord } from './types'


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
    intent: 'natal',
    title: '张三 · 命理',
    status: 'active',
    created_at: now,
    updated_at: now,
    last_message_at: null,
    context: { intent: 'natal' },
    ...overrides,
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
