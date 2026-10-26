import { onBeforeUnmount, onMounted, ref } from 'vue'

type ScrollElementGetter = () => HTMLElement | null | undefined

/**
 * Tracks one scroll element without claiming ownership of it. Callers keep the
 * domain behavior (for example, showing a follow button); this composable only
 * measures the shared "near bottom" state.
 */
export function useNearBottom(getElement: ScrollElementGetter, threshold = 80) {
  const nearBottom = ref(true)

  function measure() {
    const element = getElement()
    if (!element) {
      nearBottom.value = true
      return
    }

    const distance = element.scrollHeight - element.scrollTop - element.clientHeight
    nearBottom.value = distance < threshold
  }

  onMounted(() => {
    getElement()?.addEventListener('scroll', measure, { passive: true })
    measure()
  })

  onBeforeUnmount(() => {
    getElement()?.removeEventListener('scroll', measure)
  })

  return {
    nearBottom,
    measure,
  }
}
