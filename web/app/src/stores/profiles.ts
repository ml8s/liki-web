import { defineStore } from 'pinia'
import { errorMessage } from '@/utils/errors'
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
  _lastRefresh: number
  next_cursor: string
}

export const useProfilesStore = defineStore('profiles', {
  state: (): ProfilesState => ({
    list: [],
    detail: null,
    loading: false,
    detailLoading: false,
    error: '',
    _lastRefresh: 0,
    next_cursor: '',
  }),
  getters: {},
  actions: {
    async loadList(cursor?: string, force = false) {
      const effectiveCursor = cursor || ''
      const lastRefresh = this?._lastRefresh ?? 0
      const now = Date.now()
      const STALE_AFTER = 30_000
      if (!force && this.loading) return
      if (!force && this.list.length && now - lastRefresh < STALE_AFTER) return
      this.loading = true
      this._lastRefresh = now
      this.error = ''
      try {
        const result = await fetchProfiles(effectiveCursor)
        this.list = result.profiles
        // 下一页 cursor（若后端返回非空，则下次传入）
        if (result.next_cursor) this.next_cursor = result.next_cursor
      } catch (error) {
        this.error = errorMessage(error)
        this.list = []
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