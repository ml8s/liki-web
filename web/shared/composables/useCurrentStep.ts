import { computed } from 'vue'

export interface StepLike {
  name?: string
  start_year?: number
  end_year?: number
}

/** Current step by year; never reads a saved index. */
export function useCurrentStep(steps: () => StepLike[] | undefined, currentYear = new Date().getFullYear()) {
  const index = computed(() => {
    const list = steps() ?? []
    return list.findIndex((step) => (step.start_year ?? 0) <= currentYear && currentYear <= (step.end_year ?? 0))
  })
  const step = computed(() => (steps() ?? [])[index.value])
  const nextInYears = computed(() => (step.value?.end_year ? step.value.end_year - currentYear : undefined))
  return { currentYear, index, step, nextInYears }
}
