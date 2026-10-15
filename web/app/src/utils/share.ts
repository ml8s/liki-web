export interface ReferralLinkOptions {
  target?: string
  origin?: string
}

const REFERRAL_TARGET_PATHS = new Set((() => {
  const paths = new Set(['/', '/app', '/app/'])
  for (const locale of ['zh-Hans', 'zh-Hant', 'en']) {
    paths.add(`/${locale}`)
    paths.add(`/${locale}/`)
    for (const slug of [
      'naming', 'mingshu', 'hepan', 'skills', 'partners',
      'about', 'contact', 'privacy', 'terms', 'disclaimer',
    ]) {
      paths.add(`/${locale}/${slug}`)
      paths.add(`/${locale}/${slug}/`)
    }
  }
  return paths
})())

function isPublicReferralTarget(target: string, origin: string): boolean {
  try {
    const url = new URL(target, origin)
    return url.origin === origin && REFERRAL_TARGET_PATHS.has(url.pathname)
  } catch {
    return false
  }
}

export function buildReferralLink(code: string, options: ReferralLinkOptions = {}): string {
  const origin = options.origin ?? window.location.origin
  const normalizedCode = code.trim()
  if (!normalizedCode) return ''

  const link = new URL(`/r/${encodeURIComponent(normalizedCode)}`, origin).toString()
  const target = options.target
  if (!target || !isPublicReferralTarget(target, origin)) return link

  const url = new URL(link)
  url.searchParams.set('to', target)
  return url.toString()
}
