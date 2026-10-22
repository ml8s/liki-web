import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  baziChartFor,
  resolvePaipanContext,
  type BirthInfo,
} from '@/utils/paipan'

const toolOk = (data: unknown) =>
  new Response(
    JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [{ type: 'text', text: JSON.stringify(data) }] },
    }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  )

const shanghai = {
  name: '上海', region: '上海市', country: '中国', level: 'city',
  longitude: 121.4737, latitude: 31.2304, timezone: 'Asia/Shanghai',
}

const birth = (overrides: Partial<BirthInfo> = {}): BirthInfo => ({
  gender: 'female',
  birth_date: '1958-05-12',
  birth_time: '06:00',
  birth_place: '上海',
  ...overrides,
})

function mockEngineTools(handlers: Record<string, unknown>) {
  return vi.fn().mockImplementation((_url: string, init?: RequestInit) => {
    const body = JSON.parse(String(init?.body)) as {
      params?: { name?: string; arguments?: Record<string, unknown> }
    }
    const tool = body.params?.name ?? ''
    return Promise.resolve(toolOk(handlers[tool] ?? {}))
  })
}

function mcpCalls(mock: ReturnType<typeof mockEngineTools>) {
  return mock.mock.calls.map(([, init]) =>
    JSON.parse(String((init as RequestInit)?.body)) as {
      params?: { name?: string; arguments?: Record<string, unknown> }
    },
  )
}

describe('resolvePaipanContext', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('resolves a city into longitude and timezone, then converts wall time locally', async () => {
    const fetchMock = mockEngineTools({ city_coords: { candidates: [shanghai] } })
    vi.stubGlobal('fetch', fetchMock)
    const ctx = await resolvePaipanContext(birth({ birth_longitude: null, birth_timezone: undefined }))
    expect(ctx).toEqual({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.4737,
    })
    expect(mcpCalls(fetchMock).map(call => call.params?.name)).toEqual(['city_coords'])
  })

  it('uses an explicit longitude/timezone pair without city lookup', async () => {
    const fetchMock = mockEngineTools({})
    vi.stubGlobal('fetch', fetchMock)
    const ctx = await resolvePaipanContext(birth({
      birth_longitude: -0.1276,
      birth_timezone: 'Europe/London',
      birth_place: undefined,
    }))
    expect(ctx).toEqual({
      instant: '1958-05-12T05:00:00Z',
      timezone: 'Europe/London',
      longitude: -0.1276,
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rejects longitude without timezone instead of guessing', async () => {
    vi.stubGlobal('fetch', mockEngineTools({}))
    await expect(resolvePaipanContext(birth({ birth_longitude: 121, birth_timezone: undefined })))
      .rejects.toMatchObject({ code: 'birth_timezone_required' })
  })

  it('rejects an ambiguous or missing local time', async () => {
    vi.stubGlobal('fetch', mockEngineTools({}))
    await expect(resolvePaipanContext(birth({
      birth_date: '2024-11-03',
      birth_time: '01:30',
      birth_longitude: -74,
      birth_timezone: 'America/New_York',
      birth_place: undefined,
    }))).rejects.toMatchObject({ code: 'birth_time_ambiguous' })
  })

  it('requires a unique city candidate', async () => {
    vi.stubGlobal('fetch', mockEngineTools({
      city_coords: { candidates: [shanghai, { ...shanghai, region: 'Another' }] },
    }))
    await expect(resolvePaipanContext(birth({ birth_longitude: null, birth_timezone: undefined })))
      .rejects.toMatchObject({ code: 'city_ambiguous' })
  })
})

describe('baziChartFor', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('sends only the canonical chart context to bazi.chart', async () => {
    const fetchMock = mockEngineTools({
      bazi_chart: { birth_year: 1958, gender: 'female' },
    })
    vi.stubGlobal('fetch', fetchMock)
    const chart = await baziChartFor(birth({
      birth_longitude: 121.4737,
      birth_timezone: 'Asia/Shanghai',
      birth_place: undefined,
    }))
    expect(chart.birth_year).toBe(1958)
    const chartCall = mcpCalls(fetchMock).find(call => call.params?.name === 'bazi_chart')
    expect(chartCall?.params?.arguments).toEqual({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.4737,
      gender: 'female',
    })
  })
})
