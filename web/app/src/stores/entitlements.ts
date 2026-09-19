import { defineStore } from 'pinia'
import { fetchEntitlements, type Entitlement } from '@/api/me'
import { errorMessage } from '@/utils/errors'

interface EntitlementState {
  items: Entitlement[]
  allowedSkills: string[]
  loaded: boolean
  loading: boolean
  error: string
}

export const useEntitlementsStore = defineStore('entitlements', {
  state: (): EntitlementState => ({
    items: [],
    allowedSkills: [],
    loaded: false,
    loading: false,
    error: '',
  }),
  getters: {
    hasSkill: state => (skill: string) => state.allowedSkills.includes(skill),
    purchased: state => state.allowedSkills.length > 0,
  },
  actions: {
    async load(force = false) {
      if (this.loading || (this.loaded && !force)) return
      this.loading = true
      this.error = ''
      try {
        const result = await fetchEntitlements()
        this.items = result.entitlements
        this.allowedSkills = result.allowed_skills
        this.loaded = true
      } catch (error) {
        this.error = errorMessage(error)
      } finally {
        this.loading = false
      }
    },
    reset() {
      this.items = []
      this.allowedSkills = []
      this.loaded = false
      this.loading = false
      this.error = ''
    },
  },
})
