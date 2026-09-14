import { defineStore } from 'pinia'
import {
  createProfile,
  deleteProfile,
  fetchProfile,
  fetchProfiles,
  updateProfile,
  type Profile,
  type ProfileInput,
} from '@/api/profiles'

interface ProfilesState {
  list: Profile[]
  detail: Profile | null
  loading: boolean
  detailLoading: boolean
  error: string
}

export const useProfilesStore = defineStore('profiles', {
  state: (): ProfilesState => ({
    list: [],
    detail: null,
    loading: false,
    detailLoading: false,
    error: '',
  }),
  getters: {
    byId: state => (id: string) => state.list.find(item => item.id === id) ?? state.detail,
    defaultProfile: state => state.list.find(item => item.id) ?? null,
  },
  actions: {
    async loadList(force = false) {
      if (this.loading || (this.list.length && !force)) return
      this.loading = true
      this.error = ''
      try {
        const result = await fetchProfiles()
        this.list = result.profiles
      } catch (error) {
        this.error = errorMessage(error)
      } finally {
        this.loading = false
      }
    },
    async loadDetail(id: string) {
      this.detailLoading = true
      this.error = ''
      try {
        this.detail = await fetchProfile(id)
      } catch (error) {
        this.detail = null
        this.error = errorMessage(error)
      } finally {
        this.detailLoading = false
      }
    },
    async create(input: ProfileInput): Promise<Profile> {
      const profile = await createProfile(input)
      this.list.unshift(profile)
      return profile
    },
    async update(id: string, expectedVersion: number, input: ProfileInput): Promise<Profile> {
      const profile = await updateProfile(id, expectedVersion, input)
      const index = this.list.findIndex(item => item.id === id)
      if (index >= 0) this.list[index] = profile
      if (this.detail?.id === id) this.detail = profile
      return profile
    },
    async remove(id: string) {
      await deleteProfile(id)
      this.list = this.list.filter(item => item.id !== id)
      if (this.detail?.id === id) this.detail = null
    },
    reset() {
      this.list = []
      this.detail = null
      this.loading = false
      this.detailLoading = false
      this.error = ''
    },
  },
})

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '请求失败'
}