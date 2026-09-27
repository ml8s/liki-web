export function useClipboard() {
  const copy = async (text: string) => {
    const done = () => {
      setTimeout(() => { /* copied state updated by caller */ }, 1200)
    }

    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(text)
        done()
        return
      } catch {
        // Clipboard API 被拒绝（权限/非安全上下文）→ 落到 execCommand 兜底。
      }
    }

    // fallback: execCommand
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()

    try {
      document.execCommand('copy')
      done()
    } catch {
      // fallback to string copy prompt
    } finally {
      document.body.removeChild(ta)
    }
  }

  return { copy }
}
