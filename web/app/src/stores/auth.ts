import { defineStore } from 'pinia'
import { fetchMe } from '@/api/me'
import * as authApi from '@/api/auth'
import { useAdminStore } from '@/stores/admin'
import { useChatStore } from '@/stores/chat'
import { useSalesStore } from '@/stores/sales-store'
import { useEntitlementsStore } from '@/stores/entitlements'
import { useBirthProfilesStore } from '@/stores/birth-profiles-store'
import { ApiError } from '@/api/http'
import type { Me } from '@/api/me'
import { normalizeUnknownLocale, requestedLocale, savedLocalePreference, setAppLocale } from '@/i18n'

type AuthStatus = 'unknown' | 'authenticated' | 'unauthenticated'

interface AuthState {
  status: AuthStatus
  user: Me | null
  loading: boolean
  sessionError: '' | 'expired' | 'unreachable'
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ status: 'unknown', user: null, loading: false, sessionError: '' }),
  getters: {
    isAuthenticated: state => state.status === 'authenticated',
    displayName: state => state.user?.email ?? '',
  },
  actions: {
    async loadMe(force = false) {
      if (this.status === 'authenticated' && !force) return this.user
      this.loading = true
      try {
        this.user = await fetchMe()
        // 显式 locale 交接（requestedLocale）优先，本标签页沿用；否则同步用户偏好。
        if (!requestedLocale) {
          const savedLocale = savedLocalePreference()
          const userLocale = savedLocale ?? normalizeUnknownLocale(this.user.locale)
          if (userLocale) await setAppLocale(userLocale)
        }
        this.status = 'authenticated'
        return this.user
      } catch (error) {
        // 与其他 store 统一：内部 catch，不 re-throw。
        // 路由守卫通过 user=null → 重定向 login；AppLayout 无需额外 catch。
        this.status = 'unauthenticated'
        this.user = null
        // 401=会话过期（登录页正常流程）；网络/5xx=暂时不可达（登录页提示重试）。
        this.sessionError = error instanceof ApiError && error.status === 401 ? 'expired' : 'unreachable'
        return null
      } finally {
        this.loading = false
      }
    },
    async requestEmailCode(email: string, locale: string) {
      return authApi.requestEmailCode(email, locale)
    },
    async verifyEmailCode(email: string, code: string, locale: string) {
      const result = await authApi.verifyEmailCode(email, code, locale)
      this.user = result.user
      this.status = 'authenticated'
      return result.user
    },
    async logout() {
    try {
      await authApi.logout()
    } finally {
      this.user = null
      this.status = 'unauthenticated'
      // 登出必须清理整个用户工作区：换号登录不得见到前一用户的缓存。
      // 管理能力缓存同样要清：它是授权结果，不能跨账号复用。
      useEntitlementsStore().reset()
      useChatStore().reset()
      useBirthProfilesStore().reset()
      useSalesStore().reset()
      useAdminStore().reset()
      sessionStorage.removeItem('likiChatDraft')
    }
  },
  },
})
