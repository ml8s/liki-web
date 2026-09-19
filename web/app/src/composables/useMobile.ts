import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * 响应式移动端检测：监听 matchMedia (max-width: 768px) 变化。
 * 组件卸载时自动清理监听器。
 */
export function useMobile(breakpoint = 768) {
  const isMobile = ref(false)

  onMounted(() => {
    const mq = window.matchMedia(`(max-width: ${breakpoint}px)`)
    isMobile.value = mq.matches
    const handler = (e: MediaQueryListEvent) => { isMobile.value = e.matches }
    mq.addEventListener('change', handler)
    onBeforeUnmount(() => mq.removeEventListener('change', handler))
  })

  return isMobile
}
