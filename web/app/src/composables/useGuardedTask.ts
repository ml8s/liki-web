import { ref } from 'vue'
import { toUserMessage } from '@/utils/errors'

export interface GuardedTaskOptions {
  /** Receives the formatted failure message; the caller decides placement
   *  (a local error ref, a view-level alert, etc.). */
  report: (message: string) => void
}

/**
 * Canonical async-task lifecycle shared by every tool computation across both
 * layouts: single-flight guard, loading flag, and one formatting rule for
 * thrown errors. Input gathering and result refs stay with each caller —
 * only the lifecycle is common.
 */
export function useGuardedTask(options: GuardedTaskOptions) {
  const loading = ref(false)

  /**
   * Runs `task` under the shared lifecycle. The task may report domain
   * failures directly (set an error ref / call an alert) and return early;
   * thrown errors are formatted with `fallbackMessage` and reported.
   */
  async function run(task: () => Promise<void>, fallbackMessage: string) {
    if (loading.value) return
    loading.value = true
    try {
      await task()
    } catch (error) {
      options.report(toUserMessage(error, fallbackMessage))
    } finally {
      loading.value = false
    }
  }

  return { loading, run }
}
