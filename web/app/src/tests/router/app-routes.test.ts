import { describe, expect, it } from 'vitest'
import {
  createAppExperienceRoutes,
  createClassicAppRoutes,
  createConsultationAppRoutes,
} from '@/router/appRoutes'
import { readPreferredAppExperience, writePreferredAppExperience } from '@/router/experience'

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

  it('mounts consultation at its stable canonical path', () => {
    const routes = createConsultationAppRoutes({ enabled: true })

    expect(routes).toHaveLength(2)
    expect(routes[0]?.path).toBe('consultation')
    expect(routes[0]?.name).toBe('app-consultation')
    expect(routes[0]?.meta?.experience).toBe('consultation')
    expect(routes[0]?.meta?.viewportBound).toBe(true)
  })

  it('registers the Me hub only inside the Consultation experience', () => {
    const routes = createConsultationAppRoutes({ enabled: true })
    const me = routes[1]

    expect(me?.path).toBe('me')
    expect(me?.name).toBe('app-me')
    expect(me?.meta?.experience).toBe('consultation')
    expect(me?.meta?.viewportBound).toBeUndefined()
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

  it('keeps stable template paths and redirects the bare root by preference', async () => {
    const routes = createAppExperienceRoutes({ consultationEnabled: true })
    const appRoot = routes[0]
    const classicRoot = routes[1]

    expect(appRoot?.path).toBe('/')
    expect(appRoot?.children?.[0]?.name).toBe('app-root')
    expect(appRoot?.children?.[0]?.path).toBe('')
    expect(appRoot?.children?.[1]?.name).toBe('app-consultation')
    expect(appRoot?.children?.[1]?.path).toBe('consultation')
    expect(appRoot?.children?.[2]?.name).toBe('app-me')
    expect(appRoot?.children?.[2]?.path).toBe('me')

    const redirects = appRoot?.children?.slice(3) ?? []
    expect(redirects.map(route => route.path)).toEqual([
      'hepan',
      'divination',
      'fengshui',
      'liunian',
      'naming',
      'birth-profiles',
      'birth-profiles/new',
      'birth-profiles/:id',
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

    const redirectResults = redirects.map(route => (route.redirect as unknown as (target: {
      path: string
      query: Record<string, string>
      hash: string
    }) => ({
      path: string
      query: Record<string, string>
      hash: string
      state: { likiAppFlowLegacyRedirect: true }
    }))({
      path: route.path,
      query: { locale: 'zh-Hans' },
      hash: '#save',
    }))
    expect(redirectResults.map(result => result.path)).toEqual([
      '/',
      '/',
      '/',
      '/',
      '/',
      '/classic/birth-profiles',
      '/classic/birth-profiles/new',
      '/classic/birth-profiles/:id',
      '/classic/huangli',
      '/classic/chat',
      '/classic/chat/:sessionId',
      '/classic/orders',
      '/classic/partners',
      '/classic/partners/kol',
      '/classic/partners/scout',
      '/classic/account',
    ])
    expect(redirectResults.map(result => result.query)).toEqual([
      { locale: 'zh-Hans', domain: 'natal', seed: 'hepan' },
      { locale: 'zh-Hans', domain: 'divination', seed: 'divination' },
      { locale: 'zh-Hans', domain: 'fengshui', seed: 'fengshui' },
      { locale: 'zh-Hans', domain: 'natal', seed: 'liunian' },
      { locale: 'zh-Hans', domain: 'naming', seed: 'naming' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
      { locale: 'zh-Hans' },
    ])
    expect(redirectResults.map(result => result.hash)).toHaveLength(16)
    expect(redirectResults.slice(0, 5).map(result => result.state)).toEqual(Array.from({ length: 5 }, () => ({
      likiAppFlowLegacyRedirect: true,
    })))
    expect(redirectResults.slice(5).map(result => result.state)).toEqual(Array.from({ length: 11 }, () => undefined))



    expect(classicRoot?.path).toBe('/classic')
    expect(classicRoot?.children).toHaveLength(15)
    expect(classicRoot?.children?.[0]?.name).toBe('dashboard')
    expect(classicRoot?.children?.every(route => route.meta?.experience === 'classic')).toBe(true)
  })

  it('redirects the bare root to the saved template preference', () => {
    const routes = createAppExperienceRoutes({ consultationEnabled: true })
    const rootRedirect = routes[0]?.children?.find(route => route.name === 'app-root')
    const resolve = rootRedirect?.redirect as () => { name: string }

    // The unset preference must fall back to the build default — whatever the
    // current build flags say — and saved preferences override both ways.
    localStorage.removeItem('liki.preferred-app-experience')
    const buildDefault = import.meta.env.VITE_FEATURE_APP_FLOW_REDESIGN === 'true'
      ? 'app-consultation'
      : 'dashboard'
    expect(resolve().name).toBe(buildDefault)

    writePreferredAppExperience('consultation')
    expect(resolve().name).toBe('app-consultation')

    writePreferredAppExperience('classic')
    expect(resolve().name).toBe('dashboard')
  })

  it('persists and restores the template preference, ignoring malformed values', () => {
    localStorage.removeItem('liki.preferred-app-experience')
    expect(readPreferredAppExperience()).toBeNull()

    writePreferredAppExperience('classic')
    expect(readPreferredAppExperience()).toBe('classic')

    localStorage.setItem('liki.preferred-app-experience', 'bogus')
    expect(readPreferredAppExperience()).toBeNull()
  })
})
