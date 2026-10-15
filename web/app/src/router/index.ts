import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAdminStore } from '@/stores/admin'
import { normalizeUnknownLocale, setAppLocale } from '@/i18n'
import { i18n } from '@/i18n'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  scrollBehavior(_to, _from, savedPosition) {
    return savedPosition ?? { top: 0 }
  },
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, titleKey: 'router.login' },
    },
    {
      path: '/admin',
      component: () => import('@/views/admin/AdminLayout.vue'),
      meta: { requiresAuth: true, requiresAdmin: true, titleKey: 'router.admin' },
      children: [
        { path: '', name: 'admin-overview', component: () => import('@/views/admin/AdminOverviewView.vue') },
        { path: 'entitlements', name: 'admin-entitlements', component: () => import('@/views/admin/AdminEntitlementsView.vue'), meta: { adminCapability: 'entitlement.read' } },
        { path: 'partners', name: 'admin-partners', component: () => import('@/views/admin/AdminPartnersView.vue'), meta: { adminCapability: 'partner.read' } },
        { path: 'payouts', name: 'admin-payouts', component: () => import('@/views/admin/AdminPayoutsView.vue'), meta: { adminCapability: 'payout.read' } },
        { path: 'finance', name: 'admin-finance', component: () => import('@/views/admin/AdminFinanceView.vue'), meta: { adminCapability: 'commission.read' } },
        { path: 'orders', name: 'admin-orders', component: () => import('@/views/admin/AdminOrdersView.vue'), meta: { adminCapability: 'sales.read' } },
        { path: 'subscriptions', name: 'admin-subscriptions', component: () => import('@/views/admin/AdminSubscriptionsView.vue'), meta: { adminCapability: 'sales.read' } },
        { path: 'audit', name: 'admin-audit', component: () => import('@/views/admin/AdminAuditView.vue'), meta: { adminCapability: 'audit.read' } },
        { path: 'administrators', name: 'admin-administrators', component: () => import('@/views/admin/AdminAdministratorsView.vue'), meta: { adminCapability: 'admin.user.manage' } },
      ],
    },
    {
      path: '/',
      component: () => import('@/views/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { titleKey: 'router.dashboard' } },
        { path: 'birth-profiles', name: 'birth-profiles', component: () => import('@/views/birth-profiles/BirthProfileListView.vue'), meta: { titleKey: 'router.profiles' } },
        { path: 'birth-profiles/new', name: 'birth-profile-new', component: () => import('@/views/birth-profiles/BirthProfileNewView.vue'), meta: { titleKey: 'router.profileNew' } },
        { path: 'birth-profiles/:id', name: 'birth-profile-detail', component: () => import('@/views/birth-profiles/BirthProfileDetailView.vue'), meta: { titleKey: 'router.profileDetail' } },
        { path: 'synastry', name: 'synastry', component: () => import('@/views/SynastryView.vue'), meta: { titleKey: 'router.synastry' } },
        { path: 'divination', name: 'divination', component: () => import('@/views/DivinationView.vue'), meta: { titleKey: 'router.divination' } },
        { path: 'fengshui', name: 'fengshui', component: () => import('@/views/FengshuiView.vue'), meta: { titleKey: 'router.fengshui' } },
        { path: 'huangli', name: 'huangli', component: () => import('@/views/HuangliView.vue'), meta: { titleKey: 'router.huangli' } },
        { path: 'chat', name: 'chat', component: () => import('@/views/ChatListView.vue'), meta: { titleKey: 'router.chat' } },
        { path: 'chat/:sessionId', name: 'chat-session', component: () => import('@/views/ChatSessionView.vue'), meta: { titleKey: 'router.chatSession' } },
        { path: 'orders', name: 'orders', component: () => import('@/views/OrdersView.vue'), meta: { titleKey: 'router.orders' } },
        { path: 'partners', name: 'partner-center', component: () => import('@/views/PartnerCenterView.vue'), meta: { titleKey: 'router.partnerCenter' } },
        { path: 'partners/kol', name: 'partner-kol', component: () => import('@/views/PartnerKOLView.vue'), meta: { titleKey: 'router.partnerKol' } },
        { path: 'partners/scout', name: 'partner-scout', component: () => import('@/views/PartnerScoutView.vue'), meta: { titleKey: 'router.partnerScout' } },
        { path: 'account', name: 'account', component: () => import('@/views/AccountView.vue'), meta: { titleKey: 'router.account' } },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { public: true, titleKey: 'router.notFound' },
    },
  ],
})

router.beforeEach(async to => {
  const auth = useAuthStore()
  const locale = normalizeUnknownLocale(to.query.locale)
  if (locale) await setAppLocale(locale)
  if (!to.meta.requiresAuth) return true
  try {
    const user = await auth.loadMe()
    if (!user) return {
      name: 'login',
      query: {
        redirect: to.fullPath,
        ...(auth.sessionError === 'unreachable' ? { reason: 'unreachable' } : {}),
        ...(locale ? { locale } : {}),
      },
    }
    if (to.meta.requiresAdmin) {
      await useAdminStore().loadSession(true)
    }
    return true
  } catch {
    return { name: 'login', query: { redirect: to.fullPath, reason: 'unreachable' } }
  }
})

router.afterEach(to => {
  const base = `Liki·${i18n.global.t('auth.brandTail')}`
  document.title = to.meta.titleKey ? `${i18n.global.t(String(to.meta.titleKey))} · ${base}` : base
})

export default router
