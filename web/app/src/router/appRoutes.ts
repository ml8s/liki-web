import type { LocationQuery, RouteRecordRaw } from 'vue-router'
import AppLayout from '@/views/AppLayout.vue'
import MeView from '@/views/MeView.vue'
import { appFlowRedesignEnabled } from '@/utils/featureFlags'
import { readPreferredAppExperience, type AppExperience } from '@/router/experience'

function withExperience(
  route: RouteRecordRaw,
  experience: AppExperience,
): RouteRecordRaw {
  return {
    ...route,
    meta: {
      ...route.meta,
      experience,
    },
  }
}

function appLayoutRoute(path: string, children: RouteRecordRaw[]): RouteRecordRaw {
  return {
    path,
    component: AppLayout,
    meta: { requiresAuth: true },
    children,
  }
}

/**
 * Classic owns the original logged-in information architecture. Callers choose
 * whether it is mounted at the legacy root (rollback mode) or the explicit
 * `/classic` prefix (Consultation-default mode).
 */
export function createClassicAppRoutes(): RouteRecordRaw[] {
  return ([
    { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { titleKey: 'router.dashboard' } },
    { path: 'birth-profiles', name: 'birth-profiles', component: () => import('@/views/birth-profiles/BirthProfileListView.vue'), meta: { titleKey: 'router.profiles' } },
    { path: 'birth-profiles/new', name: 'birth-profile-new', component: () => import('@/views/birth-profiles/BirthProfileNewView.vue'), meta: { titleKey: 'router.profileNew' } },
    { path: 'birth-profiles/:id', name: 'birth-profile-detail', component: () => import('@/views/birth-profiles/BirthProfileDetailView.vue'), meta: { titleKey: 'router.profileDetail' } },
    { path: 'hepan', name: 'hepan', component: () => import('@/views/HepanView.vue'), meta: { titleKey: 'router.hepan' } },
    { path: 'divination', name: 'divination', component: () => import('@/views/DivinationView.vue'), meta: { titleKey: 'router.divination' } },
    { path: 'fengshui', name: 'fengshui', component: () => import('@/views/FengshuiView.vue'), meta: { titleKey: 'router.fengshui' } },
    { path: 'huangli', name: 'huangli', component: () => import('@/views/HuangliView.vue'), meta: { titleKey: 'router.huangli' } },
    { path: 'chat', name: 'chat', component: () => import('@/views/ChatListView.vue'), meta: { titleKey: 'router.chat' } },
    { path: 'chat/:sessionId', name: 'chat-session', component: () => import('@/views/ChatSessionView.vue'), meta: { titleKey: 'router.chatSession', viewportBound: true } },
    { path: 'orders', name: 'orders', component: () => import('@/views/OrdersView.vue'), meta: { titleKey: 'router.orders' } },
    { path: 'partners', name: 'partner-center', component: () => import('@/views/PartnerCenterView.vue'), meta: { titleKey: 'router.partnerCenter' } },
    { path: 'partners/kol', name: 'partner-kol', component: () => import('@/views/PartnerKOLView.vue'), meta: { titleKey: 'router.partnerKol' } },
    { path: 'partners/scout', name: 'partner-scout', component: () => import('@/views/PartnerScoutView.vue'), meta: { titleKey: 'router.partnerScout' } },
    { path: 'account', name: 'account', component: () => import('@/views/AccountView.vue'), meta: { titleKey: 'router.account' } },
  ] satisfies RouteRecordRaw[]).map(route => withExperience(route, 'classic'))
}

export interface ConsultationRoutesOptions {
  enabled?: boolean
}

/**
 * Consultation owns the four-domain experience. Its canonical public path is
 * `/app/consultation`; the bare `/app` root redirects to the user's preferred
 * template (see createAppExperienceRoutes).
 */
export function createConsultationAppRoutes({
  enabled = appFlowRedesignEnabled,
}: ConsultationRoutesOptions = {}): RouteRecordRaw[] {
  if (!enabled) return []

  return ([{
    path: 'consultation',
    name: 'app-consultation',
    component: () => import('@/views/DomainFlowView.vue'),
    meta: {
      titleKey: 'router.consultation',
      viewportBound: true,
    },
  }, {
    path: 'me',
    name: 'app-me',
    component: MeView,
    meta: {
      titleKey: 'router.me',
    },
  }] satisfies RouteRecordRaw[]).map(route => withExperience(route, 'consultation'))
}

/**
 * Preserve bookmarks and marketing handoffs when Consultation takes over `/app`.
 * Redirects are explicit at the boundary; Classic internals use named routes.
 */
interface LegacyRedirectTarget {
  path: string
  query: LocationQuery
  hash: string
}

interface LegacyRedirectResult extends LegacyRedirectTarget {
  state: { likiAppFlowLegacyRedirect: true }
}

function createLegacyAppRedirects(): RouteRecordRaw[] {
  const consultationOwnedLegacyPaths = new Set([
    'hepan',
    'divination',
    'fengshui',
  ])
  const classicRedirects = createClassicAppRoutes()
    .filter(route => route.path !== '' && !consultationOwnedLegacyPaths.has(route.path))
    .map(route => ({
      path: route.path,
      redirect: (to: LegacyRedirectTarget): LegacyRedirectTarget => ({
        path: `/classic${to.path.startsWith('/') ? '' : '/'}${to.path}`,
        query: to.query,
        hash: to.hash,
      }),
    }))

  // Bridge-ready legacy tools enter the four-domain flow. Query is preserved so
  // profile handoffs, locale, and campaign context survive the redirect.
  const consultationRedirects: RouteRecordRaw[] = [
    {
      path: 'hepan',
      redirect: (to: LegacyRedirectTarget): LegacyRedirectResult => ({
        path: '/',
        query: { ...to.query, domain: 'natal', seed: 'hepan' },
        hash: to.hash,
        state: { likiAppFlowLegacyRedirect: true },
      }),
    },
    {
      path: 'divination',
      redirect: (to: LegacyRedirectTarget): LegacyRedirectResult => ({
        path: '/',
        query: { ...to.query, domain: 'divination', seed: 'divination' },
        hash: to.hash,
        state: { likiAppFlowLegacyRedirect: true },
      }),
    },
    {
      path: 'fengshui',
      redirect: (to: LegacyRedirectTarget): LegacyRedirectResult => ({
        path: '/',
        query: { ...to.query, domain: 'fengshui', seed: 'fengshui' },
        hash: to.hash,
        state: { likiAppFlowLegacyRedirect: true },
      }),
    },
    {
      path: 'liunian',
      redirect: (to: LegacyRedirectTarget): LegacyRedirectResult => ({
        path: '/',
        query: { ...to.query, domain: 'natal', seed: 'liunian' },
        hash: to.hash,
        state: { likiAppFlowLegacyRedirect: true },
      }),
    },
    {
      path: 'naming',
      redirect: (to: LegacyRedirectTarget): LegacyRedirectResult => ({
        path: '/',
        query: { ...to.query, domain: 'naming', seed: 'naming' },
        hash: to.hash,
        state: { likiAppFlowLegacyRedirect: true },
      }),
    },
  ]

  // huangli/date selection intentionally remains a compatibility-owned Classic
  // route until the natal expert owns it on the backend.

  return [...consultationRedirects, ...classicRedirects]
}

export interface AppExperienceRoutesOptions {
  consultationEnabled?: boolean
}

/**
 * Standard dual-template routing: both templates ship together, each owns a
 * stable canonical subtree (consultation at `/app/consultation`, Classic at
 * `/classic`), and the bare `/app` root redirects to the user's preferred
 * template. Preference changes the redirect target only — URLs stay stable
 * and switching is a plain navigation. Rollback mode (build flag off) keeps
 * only the original Classic App.
 */
export function createAppExperienceRoutes({
  consultationEnabled = appFlowRedesignEnabled,
}: AppExperienceRoutesOptions = {}): RouteRecordRaw[] {
  if (!consultationEnabled) {
    return [appLayoutRoute('/', createClassicAppRoutes())]
  }

  return [
    appLayoutRoute('/', [
      createAppRootRedirectRoute(),
      ...createConsultationAppRoutes({ enabled: true }),
      ...createLegacyAppRedirects(),
    ]),
    appLayoutRoute('/classic', createClassicAppRoutes()),
  ]
}

function createAppRootRedirectRoute(): RouteRecordRaw {
  return {
    path: '',
    name: 'app-root',
    redirect: () => ({
      name: resolvePreferredAppExperience() === 'classic' ? 'dashboard' : 'app-consultation',
    }),
  }
}

function resolvePreferredAppExperience(): AppExperience {
  return readPreferredAppExperience() ?? (appFlowRedesignEnabled ? 'consultation' : 'classic')
}
