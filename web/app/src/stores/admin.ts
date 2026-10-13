import { defineStore } from 'pinia'
import { ApiError } from '@/api/http'
import { fetchAdminSession, type AdminCapability, type AdminSession } from '@/api/admin'

interface AdminState {
  status: 'unknown' | 'loading' | 'ready' | 'denied' | 'unreachable'
  email: string
  capabilities: AdminCapability[]
}

export const useAdminStore = defineStore('admin', {
  state: (): AdminState => ({
    status: 'unknown',
    email: '',
    capabilities: [],
  }),
  getters: {
    can: state => (capability: AdminCapability) => state.capabilities.includes(capability),
    canAny: state => (capabilities: AdminCapability[]) => capabilities.some(item => state.capabilities.includes(item)),
  },
  actions: {
    reset() {
      this.$patch({ status: 'unknown', email: '', capabilities: [] })
    },
    async loadSession(force = false) {
      if (this.status === 'ready' && !force) return true
      this.status = 'loading'
      try {
        const session: AdminSession = await fetchAdminSession()
        this.email = session.email
        this.capabilities = session.capabilities ?? []
        this.status = this.capabilities.length ? 'ready' : 'denied'
        return this.status === 'ready'
      } catch (error) {
        if (error instanceof ApiError && error.status === 403) {
          this.status = 'denied'
          return false
        }
        this.status = 'unreachable'
        return false
      }
    },
  },
})
