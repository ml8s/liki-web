/**
 * Shared fire-and-forget transport for first-party telemetry.
 *
 * Telemetry is deliberately separate from feature APIs: a rejected or absent
 * endpoint can never make a product action fail. Callers own privacy controls
 * and must send only bounded, non-PII values.
 */
export function telemetryDisabled(): boolean {
  if (typeof window === 'undefined') return true
  return (import.meta as unknown as { env?: { MODE?: string } }).env?.MODE === 'test'
}

export function postTelemetryJSON(endpoint: string, payload: object): void {
  const body = JSON.stringify(payload)
  try {
    if (typeof navigator.sendBeacon === 'function') {
      const blob = new Blob([body], { type: 'application/json' })
      if (navigator.sendBeacon(endpoint, blob)) return
    }
    void fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
      credentials: 'same-origin',
      keepalive: true,
    }).catch(() => {})
  } catch {
    // Private browsing or a disabled API must not affect the product.
  }
}
