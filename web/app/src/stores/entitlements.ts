import { defineStore } from 'pinia'
import { fetchEntitlements, type Entitlement } from '@/api/me'

interface EntitlementState {
  items: Entitlement[]
  allowedSkills: string[]
  loaded: boolean
  loading: boolean
}

export const useEntitlementsStore = defineStore('entitlements', {
  state: (): EntitlementState => ({
    items: [],
    allowedSkills: [],
    loaded: false,
    loading: false,
  }),
  getters: {
    hasSkill: state => (skill: string) => state.allowedSkills.includes(skill),
  },
  actions: {
    async load(force = false) {
      if (this.loading || (this.loaded && !force)) return
      this.loading = true
      try {
        const result = await fetchEntitlements()
        this.items = result.entitlements
        this.allowedSkills = result.allowed_skills
        this.loaded = true
      } finally {
        this.loading = false
      }
    },
    reset() {
      this.items = []
      this.allowedSkills = []
      this.loaded = false
      this.loading = false
    },
  },
})
