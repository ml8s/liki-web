import { beforeEach, describe, expect, it, vi } from 'vitest'
import { baziChartFor } from '@shared/domain/natal'
import { chartInputFromBirth, type BirthInput } from '@shared/domain/birth'

const toolOk = (data: unknown) =>
  new Response(
    JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [{ type: 'text', text: JSON.stringify(data) }] },
    }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  )

const birth = (overrides: Partial<BirthInput> = {}): BirthInput => ({
  gender: 'female',
  date: '1958-05-12',
  time: '06:00',
  location: {
    place: '上海',
    longitude: 121.4737,
    timezone: 'Asia/Shanghai',
    source: 'city',
  },
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

describe('chartInputFromBirth', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('converts exact wall time and location without a city lookup', async () => {
    const fetchMock = mockEngineTools({})
    vi.stubGlobal('fetch', fetchMock)
    await expect(chartInputFromBirth(birth())).resolves.toEqual({
      standard_time: '1958-05-12T06:00:00+08:00',
      longitude: 121.4737,
    })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('rejects an ambiguous local time', async () => {
    vi.stubGlobal('fetch', mockEngineTools({}))
    await expect(chartInputFromBirth(birth({
      date: '2024-11-03',
      time: '01:30',
      location: {
        place: 'New York',
        longitude: -74,
        timezone: 'America/New_York',
        source: 'city',
      },
    }))).rejects.toMatchObject({ code: 'birth_time_ambiguous' })
  })
})

describe('baziChartFor', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('sends only the canonical chart input to bazi_chart', async () => {
    const fetchMock = mockEngineTools({
      bazi_chart: { birth_year: 1958, gender: 'female' },
    })
    vi.stubGlobal('fetch', fetchMock)
    const chart = await baziChartFor(birth())
    expect(chart.birth_year).toBe(1958)
    const chartCall = mcpCalls(fetchMock).find(call => call.params?.name === 'bazi_chart')
    expect(chartCall?.params?.arguments).toEqual({
      standard_time: '1958-05-12T06:00:00+08:00',
      longitude: 121.4737,
      gender: 'female',
    })
  })
})
