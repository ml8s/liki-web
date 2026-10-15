export function buildReferralLink(code: string, origin: string = window.location.origin): string {
  const normalizedCode = code.trim()
  if (!normalizedCode) return ''
  return new URL(`/r/${encodeURIComponent(normalizedCode)}`, origin).toString()
}
