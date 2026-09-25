import { defineStore } from 'pinia'
import { fetchMe } from '@/api/me'
import * as authApi from '@/api/auth'
import type { Me } from '@/api/me'
import { normalizeUnknownLocale, requestedLocale, savedLocalePreference, setAppLocale } from '@/i18n'

type AuthStatus = 'unknown' | 'authenticated' | 'unauthenticated'

interface AuthState {
  status: AuthStatus
  user: Me | null
  loading: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({ status: 'unknown', user: null, loading: false }),
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
        if (requestedLocale) {
          // An explicit locale handoff wins for this browser tab.
        } else {
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
        return null
      } finally {
        this.loading = false
      }
    },
    async requestEmailCode(email: string) {
      return authApi.requestEmailCode(email)
    },
    async verifyEmailCode(email: string, code: string) {
      const result = await authApi.verifyEmailCode(email, code)
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
    }
  },
  },
})
