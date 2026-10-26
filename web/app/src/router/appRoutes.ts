import type { RouteRecordRaw } from 'vue-router'
import AppLayout from '@/views/AppLayout.vue'
import { appFlowRedesignEnabled } from '@/utils/featureFlags'
import type { AppExperience } from './experience'

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
  /** Preview mode keeps `/consultation` as an alias of the `/app` root. */
  includePreviewAlias?: boolean
}

/**
 * Consultation owns the four-domain experience. Its canonical public path is
 * `/app`; `/app/consultation` is a transition alias, not the implementation
 * term "flow".
 */
export function createConsultationAppRoutes({
  enabled = appFlowRedesignEnabled,
  includePreviewAlias = true,
}: ConsultationRoutesOptions = {}): RouteRecordRaw[] {
  if (!enabled) return []

  return [{
    path: '',
    alias: includePreviewAlias ? 'consultation' : undefined,
    name: 'app-consultation',
    component: () => import('@/views/DomainFlowView.vue'),
    meta: {
      titleKey: 'router.consultation',
      viewportBound: true,
    },
  } satisfies RouteRecordRaw].map(route => withExperience(route, 'consultation'))
}

/**
 * Preserve bookmarks and marketing handoffs when Consultation takes over `/app`.
 * Redirects are explicit at the boundary; Classic internals use named routes.
 */
export function createLegacyAppRedirects(): RouteRecordRaw[] {
  return createClassicAppRoutes()
    .filter(route => route.path !== '')
    .map(route => ({
      path: route.path,
      redirect: to => ({
        path: `/classic${to.path}`,
        query: to.query,
        hash: to.hash,
      }),
    }))
}

export interface AppExperienceRoutesOptions {
  consultationEnabled?: boolean
}

/**
 * Rollback mode keeps the original App at `/app`. Preview mode makes
 * Consultation the default, moves Classic to `/classic`, and redirects old
 * logged-in URLs across the boundary without changing public marketing URLs.
 */
export function createAppExperienceRoutes({
  consultationEnabled = appFlowRedesignEnabled,
}: AppExperienceRoutesOptions = {}): RouteRecordRaw[] {
  if (!consultationEnabled) {
    return [appLayoutRoute('/', createClassicAppRoutes())]
  }

  return [
    appLayoutRoute('/', [
      ...createConsultationAppRoutes({ enabled: true }),
      ...createLegacyAppRedirects(),
    ]),
    appLayoutRoute('/classic', createClassicAppRoutes()),
  ]
}
