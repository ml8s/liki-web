import { ref, type Ref } from 'vue'
import { toUserMessage } from '@/utils/errors'
import type { AdminPageMeta, AdminPagedResult } from '@/api/admin'

export interface AdminResource<T> {
  data: Ref<T[]>
  loading: Ref<boolean>
  error: Ref<string>
  load: () => Promise<void>
}

/**
 * Owns one admin table's data lifecycle. Query parameters are read inside the
 * loader so changing a filter does not require rebuilding the resource.
 */
export function useAdminResource<T>(loader: () => Promise<T[]>): AdminResource<T> {
  const data = ref<T[]>([]) as Ref<T[]>
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      data.value = await loader()
    } catch (requestError) {
      error.value = toUserMessage(requestError, '查询失败')
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, load }
}

export interface AdminPagedResource<T> extends Omit<AdminResource<T>, 'load'> {
  meta: Ref<AdminPageMeta | null>
  load: (page?: number) => Promise<void>
}

export function useAdminPagedResource<T>(
  loader: (page: number) => Promise<AdminPagedResult<T>>,
): AdminPagedResource<T> {
  const data = ref<T[]>([]) as Ref<T[]>
  const meta = ref<AdminPageMeta | null>(null)
  const loading = ref(false)
  const error = ref('')

  async function load(page = meta.value?.page ?? 1) {
    loading.value = true
    error.value = ''
    try {
      const result = await loader(page)
      data.value = result.items
      meta.value = result.meta
    } catch (requestError) {
      error.value = toUserMessage(requestError, '查询失败')
    } finally {
      loading.value = false
    }
  }

  return { data, meta, loading, error, load }
}
