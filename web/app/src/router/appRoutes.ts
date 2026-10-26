import type { RouteRecordRaw } from 'vue-router'
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

/**
 * Classic owns the original logged-in information architecture. It remains a
 * maintenance experience while Consultation reaches parity; no route is removed
 * here until its replacement and redirect contract pass.
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
 * Consultation owns the four-domain experience. Its canonical public path never
 * uses the internal implementation term "flow".
 */
export function createConsultationAppRoutes(
  { enabled = appFlowRedesignEnabled }: ConsultationRoutesOptions = {},
): RouteRecordRaw[] {
  if (!enabled) return []

  return [{
    path: 'consultation',
    name: 'app-consultation',
    component: () => import('@/views/DomainFlowView.vue'),
    meta: {
      titleKey: 'router.consultation',
      viewportBound: true,
    },
  } satisfies RouteRecordRaw].map(route => withExperience(route, 'consultation'))
}
