export interface SSEError extends Error {
  status?: number
  code?: string
}

export async function streamRequest(
  path: string,
  body: unknown,
  onEvent: (event: { type: string; content?: string; data?: unknown }) => void,
  signal?: AbortSignal,
): Promise<void> {
  let response: Response
  try {
    response = await fetch(path, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    })
  } catch (cause) {
    const error = new Error('网络连接失败，请检查网络后重试') as SSEError
    error.status = 0
    error.code = 'network_error'
    throw error
  }

  if (!response.ok) {
    if (response.status === 401) {
      // 会话过期 → 回登录页（借命名 chat 实践）。
      const { redirectToLogin } = await import('./http')
      redirectToLogin()
    }
    const envelope = await response.json().catch(() => null)
    const error = new Error(envelope?.error?.message ?? `HTTP ${response.status}`) as SSEError
    error.status = response.status
    error.code = envelope?.error?.code
    throw error
  }
  if (!response.body) {
    throw new Error('Empty response body')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  const emitFrame = (frame: string) => {
    for (const line of frame.split('\n')) {
      if (!line.startsWith('data: ')) continue
      const raw = line.slice(6).trim()
      if (!raw || raw === '[DONE]') continue
      try {
        onEvent(JSON.parse(raw))
      } catch {
        // Ignore malformed keepalive frames; the server protocol is JSON-only.
      }
    }
  }

  for (;;) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value, { stream: true })
    if (done) break
    const frames = buffer.split('\n\n')
    buffer = frames.pop() ?? ''
    for (const frame of frames) emitFrame(frame)
  }
  buffer += decoder.decode()
  if (buffer) emitFrame(buffer)

}
