import { defineStore } from 'pinia'
import { errorMessage } from '@/utils/errors'
import {
  createBirthProfile,
  deleteBirthProfile,
  fetchBirthProfile,
  fetchBirthProfiles,
  updateBirthProfile,
  type BirthProfile,
  type BirthProfileInput,
} from '@/api/birth-profiles'

interface BirthProfilesState {
  list: BirthProfile[]
  detail: BirthProfile | null
  loading: boolean
  detailLoading: boolean
  error: string
  _lastRefresh: number
  next_cursor: string
}

export const useBirthProfilesStore = defineStore('birth-profiles', {
  state: (): BirthProfilesState => ({
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
        const result = await fetchBirthProfiles(effectiveCursor)
        this.list = result.profiles
        // 下一页 cursor（若后端返回非空，则下次传入）
        // 每次以响应为准：空 cursor 表示已到末页，必须重置。
        this.next_cursor = result.next_cursor ?? ''
      } catch (error) {
        this.error = errorMessage(error)
        // 保留 stale 列表供继续查看，只展示错误与重试入口。
        if (!this.list.length) this.list = []
      } finally {
        this.loading = false
      }
    },
    async loadMore() {
      // 追加下一页：>50 条档案的可达性（后端分页契约的 UI 侧闭环）。
      if (!this.next_cursor || this.loading) return
      this.loading = true
      this.error = ''
      try {
        const result = await fetchBirthProfiles(this.next_cursor)
        this.list = [...this.list, ...result.profiles]
        this.next_cursor = result.next_cursor ?? ''
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
        this.detail = await fetchBirthProfile(id)
      } catch (error) {
        this.detail = null
        this.error = errorMessage(error)
      } finally {
        this.detailLoading = false
      }
    },
    async create(input: BirthProfileInput): Promise<BirthProfile> {
      const profile = await createBirthProfile(input)
      this.list.unshift(profile)
      return profile
    },
    async update(id: string, expectedVersion: number, input: BirthProfileInput): Promise<BirthProfile> {
      const profile = await updateBirthProfile(id, expectedVersion, input)
      const index = this.list.findIndex(item => item.id === id)
      if (index >= 0) this.list[index] = profile
      if (this.detail?.id === id) this.detail = profile
      return profile
    },
    async remove(id: string) {
      await deleteBirthProfile(id)
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
