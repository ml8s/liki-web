import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      component: () => import('@/views/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { title: '工作台' } },
        { path: 'reports', name: 'reports', component: () => import('@/views/ReportsView.vue'), meta: { title: '报告' } },
        { path: 'orders', name: 'orders', component: () => import('@/views/OrdersView.vue'), meta: { title: '订单' } },
        { path: 'account', name: 'account', component: () => import('@/views/AccountView.vue'), meta: { title: '账户' } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async to => {
  const auth = useAuthStore()
  if (!to.meta.requiresAuth) return true
  try {
    const user = await auth.loadMe()
    return user ? true : { name: 'login', query: { redirect: to.fullPath } }
  } catch {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
})

router.afterEach(to => {
  const base = 'Liki App'
  document.title = to.meta.title ? `${String(to.meta.title)} · ${base}` : base
})

export default router
