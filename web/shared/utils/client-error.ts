/**
 * PII-free client failure reporting.
 *
 * Errors were previously swallowed into generic UI copy, which made protocol
 * and serialization regressions invisible. This reporter sends only a stable
 * code, the current path, locale and technical detail. It never sends birth
 * data, chat content, email addresses or provider payloads.
 */
export interface ClientErrorEvent {
  code: string
  view?: string
  detail?: string
  transport?: string
  agent?: string
}

const reported = new Set<string>()

function mode(): string | undefined {
  return (import.meta as unknown as { env?: { MODE?: string } }).env?.MODE
}

function sanitizeDetail(value: string | undefined): string {
  if (!value) return ''
  const cleaned = Array.from(value, char => {
    const code = char.codePointAt(0) ?? 0
    return code < 0x20 || code === 0x7f ? ' ' : char
  }).join('')
  return cleaned.trim().slice(0, 300)
}

export function reportClientError(event: ClientErrorEvent): void {
  if (typeof window === 'undefined' || mode() === 'test') return
  const code = event.code || 'unknown'
  if (reported.has(code)) return
  reported.add(code)

  const payload = {
    code,
    view: event.view ?? window.location.pathname,
    locale: document.documentElement.lang || navigator.language,
    transport: event.transport,
    agent: event.agent,
    detail: sanitizeDetail(event.detail),
  }
  const body = JSON.stringify(payload)
  try {
    if (typeof navigator.sendBeacon === 'function') {
      const blob = new Blob([body], { type: 'application/json' })
      if (navigator.sendBeacon('/api/client-errors', blob)) return
    }
    void fetch('/api/client-errors', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
      credentials: 'same-origin',
      keepalive: true,
    }).catch(() => {})
  } catch {
    // Telemetry must never break the product.
  }
}

/** Test helper: allow a fresh page-load dedupe state. */
export function resetClientErrorReporting(): void {
  reported.clear()
}
