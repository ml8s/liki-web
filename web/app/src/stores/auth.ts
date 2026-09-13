import { defineStore } from 'pinia'
import { fetchMe } from '@/api/me'
import * as authApi from '@/api/auth'
import type { Me } from '@/types'

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
        this.status = 'authenticated'
        return this.user
      } catch (error) {
        if (error instanceof Error && 'status' in error && (error as any).status === 401) {
          this.status = 'unauthenticated'
          this.user = null
          return null
        }
        throw error
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
      await authApi.logout()
      this.user = null
      this.status = 'unauthenticated'
    },
  },
})
