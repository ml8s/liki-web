import { describe, expect, it, vi, beforeEach } from 'vitest'
import { baziChartFor, resolvePaipanContext, type BirthInfo } from '@/utils/paipan'

const rpcOk = (data: unknown) =>
  new Response(JSON.stringify({ result: { data } }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })

const birth = (overrides: Partial<BirthInfo> = {}): BirthInfo => ({
  gender: 'female',
  birth_date: '1958-05-12',
  birth_time: '06:00',
  birth_time_known: 'known',
  birth_place: '上海',
  ...overrides,
})

function mockRpc(handlers: Record<string, unknown>) {
  return vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
    const body = JSON.parse((init?.body as string) ?? '{}')
    const data = handlers[body.method] ?? {}
    return Promise.resolve(rpcOk(data))
  })
}

describe('resolvePaipanContext', () => {
  beforeEach(() => { vi.restoreAllMocks() })

  it('applies true solar time correction when place is given', async () => {
    vi.stubGlobal('fetch', mockRpc({
      'city.coords': { longitude: 121.47, latitude: 31.23 },
      'tianwen.time': { solar: '1958-05-12T05:53:20+08:00', lunar: '1958-闰4-24' },
    }))
    const ctx = await resolvePaipanContext(birth())
    expect(ctx.corrected).toBe(true)
    expect(ctx.longitude).toBe(121.47)
    expect(ctx.solar).toBe('1958-05-12T05:53:20+08:00')
    expect(ctx.lunar).toBe('1958-闰4-24')
  })

  it('skips correction when no birth place', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      rpcOk({ solar: '1958-05-12T06:00:00+08:00', lunar: '1958-闰4-24' }),
    ))
    const ctx = await resolvePaipanContext(birth({ birth_place: undefined }))
    expect(ctx.corrected).toBe(false)
    expect(ctx.solar).toBe('1958-05-12T06:00:00+08:00')
  })

  it('skips correction when time is unknown', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      rpcOk({ solar: '1958-05-12T06:00:00+08:00', lunar: '1958-闰4-24' }),
    ))
    const ctx = await resolvePaipanContext(birth({ birth_time_known: 'unknown' }))
    expect(ctx.corrected).toBe(false)
  })

  it('throws descriptive error when city lookup fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('network error')))
    await expect(resolvePaipanContext(birth())).rejects.toMatchObject({ code: 'city_not_found' })
  })
})

describe('baziChartFor', () => {
  beforeEach(() => { vi.restoreAllMocks() })

  it('throws for lunar calendar', async () => {
    await expect(baziChartFor(birth({ calendar: 'lunar' }))).rejects.toMatchObject({ code: 'lunar_preview_unsupported' })
  })

  it('throws when time is unknown', async () => {
    await expect(baziChartFor(birth({ birth_time_known: 'unknown' }))).rejects.toMatchObject({ code: 'birth_time_required' })
  })

  it('calls engine with corrected solar time', async () => {
    vi.stubGlobal('fetch', mockRpc({
      'city.coords': { longitude: 121.47, latitude: 31.23 },
      'tianwen.time': { solar: '1958-05-12T05:53:20+08:00', lunar: '1958-闰4-24' },
      'bazi.chart': { birth_year: 1958, gender: 'female' },
    }))
    const chart = await baziChartFor(birth())
    expect(chart.birth_year).toBe(1958)
  })
})
