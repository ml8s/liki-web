import type { RouteRecordRaw } from 'vue-router'
import AppLayout from '@/views/AppLayout.vue'

function appLayoutRoute(path: string, children: RouteRecordRaw[]): RouteRecordRaw {
  return {
    path,
    component: AppLayout,
    meta: { requiresAuth: true },
    children,
  }
}

const classicRoutes: RouteRecordRaw[] = [
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
]

/** Classic is the only authenticated App IA. */
export function createAppRoutes(): RouteRecordRaw[] {
  return [appLayoutRoute('/', [
    ...classicRoutes,
    { path: 'consultation', redirect: to => ({ path: '/', query: to.query, hash: to.hash }) },
    { path: 'me', redirect: () => ({ path: '/account' }) },
    // Temporary bookmark compatibility for the retired dual-template rollout.
    { path: 'classic', redirect: to => ({ path: '/', query: to.query, hash: to.hash }) },
    {
      path: 'classic/:pathMatch(.*)*',
      redirect: to => ({ path: to.path.replace(/^\/classic/, ''), query: to.query, hash: to.hash }),
    },
  ])]
}
