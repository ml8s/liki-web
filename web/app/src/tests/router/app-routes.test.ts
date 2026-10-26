import { describe, expect, it } from 'vitest'
import {
  createAppExperienceRoutes,
  createClassicAppRoutes,
  createConsultationAppRoutes,
} from '@/router/appRoutes'

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

  it('mounts consultation at the app root with a preview alias', () => {
    const routes = createConsultationAppRoutes({ enabled: true })

    expect(routes).toHaveLength(1)
    expect(routes[0]?.path).toBe('')
    expect(routes[0]?.alias).toBe('consultation')
    expect(routes[0]?.name).toBe('app-consultation')
    expect(routes[0]?.meta?.experience).toBe('consultation')
    expect(routes[0]?.meta?.viewportBound).toBe(true)
  })

  it('omits the consultation experience when disabled', () => {
    expect(createConsultationAppRoutes({ enabled: false })).toEqual([])
  })

  it('keeps rollback mode on the original information architecture', () => {
    const routes = createAppExperienceRoutes({ consultationEnabled: false })

    expect(routes).toHaveLength(1)
    expect(routes[0]?.path).toBe('/')
    expect(routes[0]?.children).toHaveLength(15)
    expect(routes[0]?.children?.every(route => route.meta?.experience === 'classic')).toBe(true)
  })

  it('makes consultation the default and preserves classic behind a prefixed shell', () => {
    const routes = createAppExperienceRoutes({ consultationEnabled: true })
    const appRoot = routes[0]
    const classicRoot = routes[1]

    expect(appRoot?.path).toBe('/')
    expect(appRoot?.children?.[0]?.name).toBe('app-consultation')
    expect(appRoot?.children?.[0]?.path).toBe('')

    const redirects = appRoot?.children?.slice(1) ?? []
    expect(redirects.map(route => route.path)).toEqual([
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
    expect(redirects.every(route => typeof route.redirect === 'function')).toBe(true)

    expect(classicRoot?.path).toBe('/classic')
    expect(classicRoot?.children).toHaveLength(15)
    expect(classicRoot?.children?.[0]?.name).toBe('dashboard')
    expect(classicRoot?.children?.every(route => route.meta?.experience === 'classic')).toBe(true)
  })
})
