import { onBeforeUnmount, type Ref } from 'vue'

/** Ask the browser to confirm unload while the ref marks unsaved work. */
export function useUnloadGuard(enabled: Ref<boolean>) {
  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (!enabled.value) return
    event.preventDefault()
    event.returnValue = ''
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('beforeunload', handleBeforeUnload)
  }

  onBeforeUnmount(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    }
  })
}
