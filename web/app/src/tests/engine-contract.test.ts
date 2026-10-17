// 引擎请求/错误契约锁：字段名与引擎 mcpcontract DTO 一字不差，防前端臆造参数。
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/http'
import { engineBaziBond, engineHuangliDays } from '@/api/engine'

function stubFetch(payload: unknown, status = 200): void {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status })))
}

function lastRequestBody(): Record<string, unknown> {
  const call = vi.mocked(fetch).mock.calls.at(-1)!
  return JSON.parse(String(call[1]?.body)) as Record<string, unknown>
}

describe('engine request contract', () => {
  it('huangli sends {event, start_date, count} exactly (engine.huangli.HuangliDaysInput)', async () => {
    stubFetch({ data: [] })
    await engineHuangliDays('wedding', '2026-10-01', 30)
    expect(lastRequestBody()).toEqual({ event: 'wedding', start_date: '2026-10-01', count: 30 })
  })

  it('bazi bond sends both charts under {a, b} (engine.bazi.BaziBondInput)', async () => {
    stubFetch({ data: {} })
    await engineBaziBond({ nian: { gan: '甲' } }, { nian: { gan: '庚' } })
    expect(lastRequestBody()).toEqual({
      a: { chart: { nian: { gan: '甲' } } },
      b: { chart: { nian: { gan: '庚' } } },
    })
  })

  it('maps BFF problem+json type to the ApiError code verbatim', async () => {
    stubFetch(
      { type: 'engine_tool_failed', title: 'engine_tool_failed', status: 422, detail: 'huangli.days: bad input' },
      422,
    )
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).rejects.toMatchObject({
      status: 422,
      code: 'engine_tool_failed',
      message: 'huangli.days: bad input',
    } satisfies Partial<ApiError>)
  })

  it('keeps engine_tool_not_found distinguishable from tool failures', async () => {
    stubFetch(
      { type: 'engine_tool_not_found', title: 'engine_tool_not_found', status: 404, detail: 'Engine tool not found' },
      404,
    )
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).rejects.toMatchObject({
      code: 'engine_tool_not_found',
    } satisfies Partial<ApiError>)
  })

  it('returns the raw data payload (array wrapper stripped)', async () => {
    stubFetch({ data: [{ date: '2026-10-11', suitability: 'recommended' }] })
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).resolves.toEqual([
      { date: '2026-10-11', suitability: 'recommended' },
    ])
  })
})
