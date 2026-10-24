#!/usr/bin/env node
/**
 * Production smoke / synthetic checks for liki.hk.
 *
 * One suite, two modes:
 *   npm run smoke       full post-deploy smoke
 *   npm run synthetic   fast scheduled subset
 *
 * Configuration is environment-only. Never hardcode the test account or the
 * admin notification address in this file.
 *
 *   SMOKE_BASE_URL=https://liki.hk
 *   SMOKE_EMAIL=<test-account@example.com>        # optional identity assertion
 *   SMOKE_SESSION_COOKIE=liki_session=...   # optional authenticated checks
 *   SMOKE_DOMAIN=liki.hk
 *   SMOKE_SENDING_DOMAIN=notify.liki.hk
 *   SMOKE_REQUIRE_ENTITLEMENT=1
 */
import { promises as dns } from 'node:dns'
import tls from 'node:tls'
import { randomUUID } from 'node:crypto'
import { chromium } from 'playwright'

const args = parseArgs(process.argv.slice(2))
if (args.help) {
  console.log(`Usage: node scripts/smoke.mjs [--mode=smoke|synthetic] [--base-url=URL] [--json] [--strict] [--no-browser] [--no-dns] [--experts=all|natal]`)
  process.exit(0)
}

const mode = String(args.mode || 'smoke')
const base = String(args['base-url'] || process.env.SMOKE_BASE_URL || 'https://liki.hk').replace(/\/+$/, '')
const baseOrigin = new URL(base).origin
const email = (process.env.SMOKE_EMAIL || '').trim()
const cookie = normalizeCookie(process.env.SMOKE_SESSION_COOKIE || '')
const domain = (process.env.SMOKE_DOMAIN || new URL(base).hostname).trim()
const sendingDomain = (process.env.SMOKE_SENDING_DOMAIN || `notify.${domain}`).trim()
const requireEntitlement = (process.env.SMOKE_REQUIRE_ENTITLEMENT || '1') !== '0'
const strict = Boolean(args.strict) || process.env.SMOKE_STRICT === '1'
const browserEnabled = !args['no-browser'] && process.env.SMOKE_NO_BROWSER !== '1'
const dnsEnabled = !args['no-dns'] && process.env.SMOKE_NO_DNS !== '1'
const experts = String(args.experts || (mode === 'synthetic' ? 'natal' : 'all'))
const results = []

const MARKETING = ['', 'natal', 'liunian', 'hepan', 'divination', 'huangli', 'fengshui', 'naming']
const LOCALES = ['zh-Hant', 'zh-Hans', 'en']
const EXPECTED_404 = {
  '/en/smoke-not-found': { lang: 'en', heading: 'Page not found' },
  '/zh-Hans/smoke-not-found': { lang: 'zh-Hans', heading: '页面未找到' },
  '/smoke-not-found': { lang: 'zh-Hant', heading: '頁面未找到' },
}

function parseArgs(argv) {
  const out = {}
  for (const item of argv) {
    if (item === '--json') { out.json = true; continue }
    if (item === '--strict') { out.strict = true; continue }
    if (item === '--no-browser') { out['no-browser'] = true; continue }
    if (item === '--no-dns') { out['no-dns'] = true; continue }
    if (item === '--help' || item === '-h') { out.help = true; continue }
    const m = item.match(/^--([^=]+)=(.*)$/)
    if (m) out[m[1]] = m[2]
  }
  return out
}

function normalizeCookie(value) {
  const v = value.trim()
  if (!v) return ''
  return v.startsWith('liki_session=') ? v : `liki_session=${v}`
}

function authHeaders() {
  return cookie ? { Cookie: cookie } : {}
}

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

async function check(name, fn, { warn = false, skip = false } = {}) {
  if (skip) {
    results.push({ name, status: 'skip', ms: 0, detail: 'not configured' })
    return
  }
  const started = Date.now()
  try {
    const detail = await fn()
    results.push({ name, status: 'pass', ms: Date.now() - started, detail: detail ? String(detail).slice(0, 240) : '' })
  } catch (error) {
    results.push({
      name,
      status: warn ? 'warn' : 'fail',
      ms: Date.now() - started,
      detail: String(error instanceof Error ? error.message : error).slice(0, 320),
    })
  }
}

async function request(path, init = {}) {
  const method = String(init.method || 'GET').toUpperCase()
  let lastError
  for (let attempt = 0; attempt < 2; attempt += 1) {
    try {
      return await fetch(`${base}${path}`, {
        redirect: 'manual',
        signal: AbortSignal.timeout(30_000),
        ...init,
        headers: { ...(init.headers || {}) },
      })
    } catch (error) {
      lastError = error
      if (method !== 'GET') break
      await new Promise(resolve => setTimeout(resolve, 400))
    }
  }
  throw lastError
}

async function json(path, init) {
  const response = await request(path, init)
  const text = await response.text()
  assert(response.ok, `${path} HTTP ${response.status}: ${text.slice(0, 200)}`)
  try { return JSON.parse(text) } catch { throw new Error(`${path} returned non-JSON`) }
}

async function mcpCall(endpoint, tool, toolArgs) {
  const response = await request(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json, text/event-stream',
      'Mcp-Protocol-Version': '2026-07-28',
      'Mcp-Method': 'tools/call',
      'Mcp-Name': tool,
    },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'tools/call',
      params: {
        name: tool,
        arguments: toolArgs,
        _meta: {
          'io.modelcontextprotocol/protocolVersion': '2026-07-28',
          'io.modelcontextprotocol/clientCapabilities': {},
        },
      },
    }),
  })
  const text = await response.text()
  assert(response.ok, `${tool} HTTP ${response.status}: ${text.slice(0, 240)}`)
  let envelope
  try { envelope = JSON.parse(text) } catch { throw new Error(`${tool} returned non-JSON`) }
  if (envelope.error) throw new Error(`${tool} JSON-RPC ${envelope.error.code}: ${envelope.error.message}`)
  assert(!envelope.result?.isError, `${tool} tool error: ${envelope.result?.content?.[0]?.text || 'unknown'}`)
  return envelope.result?.structuredContent ?? JSON.parse(envelope.result?.content?.[0]?.text || '{}')
}

async function expertIdentity(expert) {
  const response = await request(`/rt/${expert}/ag-ui`, {
    method: 'POST',
    headers: {
      ...authHeaders(),
      Origin: baseOrigin,
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify({
      threadId: `smoke-${randomUUID()}`,
      runId: `smoke-${randomUUID()}`,
      messages: [{ id: `smoke-${randomUUID()}`, role: 'user', content: '请只回答一句：你负责哪个领域？' }],
      tools: [],
      context: [],
      state: {},
      forwardedProps: { product: 'liki', locale: 'zh-Hant' },
    }),
  })
  assert(response.ok, `${expert} route HTTP ${response.status}`)
  const text = await response.text()
  const finished = text.split('\n')
    .filter(line => line.startsWith('data: '))
    .map(line => { try { return JSON.parse(line.slice(6)) } catch { return null } })
    .find(event => event?.type === 'RUN_FINISHED')
  assert(finished, `${expert} route ended without RUN_FINISHED`)
  assert(finished.result?.definition?.name === expert, `${expert} route served ${finished.result?.definition?.name || 'unknown'}`)
  return finished.result?.definition?.name
}

function expectedExperts() {
  if (experts === 'all') return ['natal', 'divination', 'fengshui', 'naming']
  return experts.split(',').map(item => item.trim()).filter(Boolean)
}

async function main() {
  await check('liveness', async () => {
    const payload = await json('/api/health')
    assert(payload?.data?.status === 'ok', 'liveness status is not ok')
    return 'ok'
  })

  await check('readiness', async () => {
    const payload = await json('/api/ready')
    assert(payload?.data?.status === 'ready', 'readiness status is not ready')
    return 'ready'
  })

  await check('version', async () => {
    const response = await request('/api/version')
    assert(response.ok, `version HTTP ${response.status}`)
    return response.status
  })

  await check('security headers', async () => {
    const response = await request('/zh-Hant/')
    assert(response.ok, `home HTTP ${response.status}`)
    for (const header of ['content-security-policy', 'strict-transport-security', 'x-content-type-options', 'x-frame-options']) {
      assert(response.headers.get(header), `missing ${header}`)
    }
    return 'CSP/HSTS/XCTO/XFO present'
  })

  await check('marketing pages', async () => {
    let count = 0
    for (const locale of LOCALES) {
      for (const slug of MARKETING) {
        const path = `/${locale}/${slug ? `${slug}/` : ''}`
        const response = await request(path)
        assert(response.status === 200, `${path} HTTP ${response.status}`)
        if (slug === 'natal') {
          const body = await response.text()
          assert(!body.includes('scenario.natalReading.q1'), `${path} leaks raw i18n key`)
        }
        count += 1
      }
    }
    return `${count} pages`
  })

  await check('robots and sitemap', async () => {
    const robots = await request('/robots.txt')
    assert(robots.ok, `robots HTTP ${robots.status}`)
    const robotsBody = await robots.text()
    assert(robotsBody.includes('Disallow: /app'), 'robots does not disallow /app')
    const sitemap = await request('/sitemap.xml')
    assert(sitemap.ok, `sitemap HTTP ${sitemap.status}`)
    const sitemapBody = await sitemap.text()
    assert((sitemapBody.match(/<loc>/g) || []).length > 0, 'sitemap has no URLs')
    return 'ok'
  })

  await check('engine MCP', async () => {
    const chartArgs = { instant: '1990-01-01T04:00:00Z', timezone: 'Asia/Hong_Kong', longitude: 114.169, gender: 'female' }
    const calls = [
      ['bazi_chart', chartArgs],
      ['ziwei_chart', chartArgs],
      ['liuyao_qigua', { mode: 'auto' }],
      ['huangli_days', { event: 'wedding', start_date: '2026-11-01', count: 10 }],
      ['bazhai_chart', { birth_year: 1990, gender: 'female' }],
      ['city_coords', { city: '香港' }],
    ]
    for (const [tool, toolArgs] of calls) {
      await mcpCall('/engine/mcp', tool, toolArgs)
    }
    return `${calls.length} tools`
  })

  await check('counsel MCP', async () => {
    await mcpCall('/counsel/mcp', 'naming_surname', { source_surname: 'Li' })
    return 'naming_surname'
  })

  await check('authenticated identity', async () => {
    const me = await json('/api/me', { headers: authHeaders() })
    if (email) assert(me?.data?.email === email, `signed-in email ${me?.data?.email || 'unknown'} != configured test account`)
    return me?.data?.email || 'signed in'
  }, { skip: !cookie })

  await check('entitlement', async () => {
    const payload = await json('/api/me/entitlements', { headers: authHeaders() })
    const skills = payload?.data?.allowed_skills || []
    assert(skills.includes('liki'), 'test account has no active liki entitlement')
    return 'liki'
  }, { skip: !cookie || !requireEntitlement })

  await check('expert identity', async () => {
    const names = []
    for (const expert of expectedExperts()) names.push(await expertIdentity(expert))
    return names.join(', ')
  }, { skip: !cookie })

  await check('payment products', async () => {
    const payload = await json('/api/sales/products', { headers: authHeaders() })
    const product = payload?.data?.products?.[0]
    assert(product?.id === 'liki-chat', 'liki-chat product missing')
    assert(['CNY', 'USD'].includes(product.display_currency), `unexpected currency ${product.display_currency}`)
    assert(product.purchase_options?.length > 0, 'no purchasable options')
    return `${product.display_currency}/${product.purchase_options.length} options`
  }, { skip: !cookie })

  await check('context session lifecycle', async () => {
    const payload = await json('/api/chat/sessions', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({
        skill: 'liki',
        scenario: 'divination',
        title: 'smoke-context',
        context: {
          version: 1,
          scenario: 'divination',
          question: 'smoke',
          casting: { mode: 'yaos', order: 'bottom_up', casting_id: 'smoke-cast', yaos: [7, 8, 8, 7, 8, 9], dong_yao: [6] },
          casting_instant: new Date().toISOString(),
          location: { longitude: 114.169, timezone: 'Asia/Hong_Kong' },
        },
      }),
    })
    const session = payload?.data
    assert(session?.expert === 'divination', `context session expert=${session?.expert || 'missing'}`)
    try {
      const deleted = await request(`/api/chat/sessions/${session.id}`, { method: 'DELETE', headers: authHeaders() })
      assert(deleted.ok, `session cleanup HTTP ${deleted.status}`)
    } catch (error) {
      throw new Error(`context session created but cleanup failed: ${error.message}`, { cause: error })
    }
    return 'created and deleted'
  }, { skip: !cookie || mode === 'synthetic' })

  if (browserEnabled) {
    const browser = await chromium.launch({ headless: true })
    try {
      await check('404 localization', async () => {
        const context = await browser.newContext({ locale: 'en-US' })
        const page = await context.newPage()
        try {
          for (const [path, expected] of Object.entries(EXPECTED_404)) {
            const response = await page.goto(`${base}${path}`, { waitUntil: 'domcontentloaded', timeout: 30_000 })
            assert(response?.status() === 404, `${path} status ${response?.status()}`)
            await page.waitForTimeout(300)
            await page.waitForSelector('[data-404-heading]', { timeout: 5_000 }).catch(() => {
              throw new Error(`${path} is missing 404 localization markers (deployed build predates the fix)`)
            })
            const lang = await page.getAttribute('html', 'lang')
            const heading = await page.locator('[data-404-heading]').innerText({ timeout: 5_000 })
            assert(lang === expected.lang, `${path} lang=${lang}`)
            assert(heading === expected.heading, `${path} heading=${heading}`)
          }
        } finally {
          await context.close()
        }
        return '3 locales'
      })

      await check('CSP console', async () => {
        const context = await browser.newContext()
        const page = await context.newPage()
        const errors = []
        page.on('console', message => {
          if (message.type() === 'error' && /Content Security Policy/i.test(message.text())) errors.push(message.text())
        })
        try {
          await page.goto(`${base}/zh-Hant/`, { waitUntil: 'networkidle', timeout: 45_000 })
        } finally {
          await context.close()
        }
        assert(errors.length === 0, `${errors.length} CSP console error(s): ${errors[0]?.slice(0, 160) || ''}`)
        return 'clean'
      }, { warn: process.env.SMOKE_STRICT_CSP !== '1' })
    } finally {
      await browser.close()
    }
  }

  if (dnsEnabled) {
    await check('mail DNS', async () => {
      const notes = []
      const mx = await dns.resolveMx(domain).catch(() => [])
      if (!mx.length) notes.push(`${domain} has no MX`)
      const rootTxt = (await dns.resolveTxt(domain).catch(() => [])).flat().join(' ')
      if (!/v=spf1/i.test(rootTxt)) notes.push(`${domain} has no SPF`)
      const dmarc = (await dns.resolveTxt(`_dmarc.${domain}`).catch(() => [])).flat().join(' ')
      if (!/v=DMARC1/i.test(dmarc)) notes.push(`${domain} has no DMARC`)
      const sendingTxt = (await dns.resolveTxt(sendingDomain).catch(() => [])).flat().join(' ')
      if (!/v=spf1/i.test(sendingTxt)) notes.push(`${sendingDomain} has no SPF`)
      assert(notes.length === 0, notes.join('; '))
      return 'MX/SPF/DMARC present'
    }, { warn: true })

    await check('TLS certificate', async () => {
      const cert = await new Promise((resolve, reject) => {
        const socket = tls.connect({ host: domain, servername: domain, port: 443, timeout: 8_000 }, () => {
          const peer = socket.getPeerCertificate()
          socket.end()
          resolve(peer)
        })
        socket.on('error', reject)
        socket.on('timeout', () => { socket.destroy(); reject(new Error('TLS timeout')) })
      })
      const expires = new Date(cert.valid_to)
      const days = Math.floor((expires.getTime() - Date.now()) / 86_400_000)
      assert(Number.isFinite(days) && days > 0, 'certificate is expired or invalid')
      if (days < 14) throw new Error(`certificate expires in ${days} day(s)`)
      return `${days} days`
    })
  }

  const failures = results.filter(item => item.status === 'fail')
  const warnings = results.filter(item => item.status === 'warn')
  const summary = {
    mode,
    base,
    checked_at: new Date().toISOString(),
    passed: results.filter(item => item.status === 'pass').length,
    warned: warnings.length,
    failed: failures.length,
    skipped: results.filter(item => item.status === 'skip').length,
    results,
  }

  if (args.json) {
    console.log(JSON.stringify(summary, null, 2))
  } else {
    for (const item of results) {
      const label = item.status.toUpperCase().padEnd(6)
      console.log(`${label} ${item.name} (${item.ms}ms)${item.detail ? ` — ${item.detail}` : ''}`)
    }
    console.log(`\n${summary.passed} passed, ${summary.warned} warned, ${summary.failed} failed, ${summary.skipped} skipped`)
  }

  if (failures.length || (strict && warnings.length)) process.exitCode = 1
}

main().catch(error => {
  console.error(error)
  process.exitCode = 1
})
