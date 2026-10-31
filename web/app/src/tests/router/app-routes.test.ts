import { describe, expect, it } from 'vitest'
import type { RouteRecordRaw } from 'vue-router'
import { createAppRoutes } from '@/router/appRoutes'

type AppRouteRoot = Extract<RouteRecordRaw, { children: RouteRecordRaw[] }>

function appRouteChildren(): RouteRecordRaw[] {
  const [root] = createAppRoutes() as [AppRouteRoot]
  return root.children ?? []
}

describe('app routes', () => {
  it('keeps Classic as the only authenticated information architecture', () => {
    const [root] = createAppRoutes() as [AppRouteRoot]
    const children = root.children ?? []
    const paths = children
      .map(route => route.path)
      .filter(path => !String(path).startsWith('classic') && !['consultation', 'me'].includes(String(path)))

    expect(root.path).toBe('/')
    expect(paths).toEqual([
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
  })

  it('does not expose retired consultation routes', () => {
    const children = appRouteChildren()
    const retired = children.filter(route => ['consultation', 'me'].includes(String(route.path)))

    expect(retired.map(route => route.path)).toEqual(['consultation', 'me'])
    expect(retired.every(route => typeof route.redirect === 'function')).toBe(true)
  })

  it('redirects the retired /classic prefix without dropping location context', () => {
    const children = appRouteChildren()
    const deepClassic = children.find(route => route.path === 'classic/:pathMatch(.*)*')
    const redirect = deepClassic?.redirect as (target: {
      path: string
      query: Record<string, string>
      hash: string
    }) => { path: string, query: Record<string, string>, hash: string }

    expect(redirect({
      path: '/classic/account',
      query: { locale: 'en' },
      hash: '#sessions',
    })).toEqual({
      path: '/account',
      query: { locale: 'en' },
      hash: '#sessions',
    })
  })
})
