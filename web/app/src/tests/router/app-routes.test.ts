import { describe, expect, it } from 'vitest'
import { createClassicAppRoutes, createConsultationAppRoutes } from '@/router/appRoutes'

describe('app experience routes', () => {
  it('keeps the original logged-in routes as the maintenance Classic experience', () => {
    const routes = createClassicAppRoutes()

    expect(routes.map(route => route.path)).toEqual([
      '',
      'birth-profiles',
      'birth-profiles/new',
      'birth-profiles/:id',
      'hepan',
      'divination',
      'fengshui',
      'huangli',
      'chat',
      'chat/:sessionId',
      'orders',
      'partners',
      'partners/kol',
      'partners/scout',
      'account',
    ])
    expect(routes).toHaveLength(15)
    expect(routes.every(route => route.meta?.experience === 'classic')).toBe(true)
  })

  it('registers one canonical Consultation route and no implementation-only flow route', () => {
    const routes = createConsultationAppRoutes({ enabled: true })

    expect(routes).toHaveLength(1)
    expect(routes[0]?.path).toBe('consultation')
    expect(routes[0]?.name).toBe('app-consultation')
    expect(routes[0]?.meta?.experience).toBe('consultation')
    expect(routes[0]?.meta?.viewportBound).toBe(true)
  })

  it('omits the consultation experience when disabled', () => {
    expect(createConsultationAppRoutes({ enabled: false })).toEqual([])
  })
})
