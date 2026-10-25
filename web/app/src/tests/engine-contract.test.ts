// 引擎请求/错误契约锁：MCP 直连（/engine/mcp、/counsel/mcp）的 JSON-RPC 形状与
// 字段名须与引擎/counsel mcpcontract 一字不差，防前端臆造参数或误包协议。
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/http'
import { mcpTool } from '@shared/api/mcp/client'
import { engineBaziBond, engineBaziChart } from '@shared/api/engine/bazi'
import { engineHuangliDays } from '@shared/api/engine/huangli'
import { engineLiuyaoChart, engineLiuyaoQigua } from '@shared/api/engine/liuyao'
import { engineZiweiChart, engineZiweiDaXian, engineZiweiLiuNian } from '@shared/api/engine/ziwei'
import { NatalStageError, natalChartForChartInput } from '@shared/domain/natal'

const mcpOk = (data: unknown, status = 200): Response =>
  new Response(
    JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [{ type: 'text', text: JSON.stringify(data) }] },
    }),
    { status },
  )

const mcpIsError = (message: string): Response =>
  new Response(
    JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [{ text: `Error executing tool: ${message}`, type: 'text' }], isError: true },
    }),
    { status: 200 },
  )

function stubFetch(response: Response): void {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
}

function lastCall(): { url: string; body: { method?: string; params?: { name?: string; arguments?: Record<string, unknown> } }; headers: Record<string, string> } {
  const call = vi.mocked(fetch).mock.calls.at(-1)!
  return {
    url: String(call[0]),
    body: JSON.parse(String(call[1]?.body)),
    headers: (call[1]?.headers ?? {}) as Record<string, string>,
  }
}

describe('engine request contract', () => {
  it('posts tools/call to /engine/mcp with the MCP-required Accept header', async () => {
    stubFetch(mcpOk([]))
    await engineHuangliDays('wedding', '2026-10-01', 30)
    const { url, body, headers } = lastCall()
    expect(url).toBe('/engine/mcp')
    expect(body.method).toBe('tools/call')
    expect(body.params?.name).toBe('huangli_days')
    const meta = (body.params as { _meta?: Record<string, unknown> })?._meta
    expect(meta?.['io.modelcontextprotocol/protocolVersion']).toBe('2026-07-28')
    // MCP Streamable HTTP 规范：客户端必须同时接受 JSON 与 SSE 形态。
    expect(headers.Accept).toBe('application/json, text/event-stream')
    // 2026-07-28 起 Method/Name 进入 HTTP header，服务端在解析 body 前校验。
    expect(headers['Mcp-Method']).toBe('tools/call')
    expect(headers['Mcp-Name']).toBe('huangli_days')
  })

  it('routes counsel tools to /counsel/mcp', async () => {
    stubFetch(mcpOk({ selected: {} }))
    await mcpTool('bazi_yongshen', { chart: {} })
    expect(lastCall().url).toBe('/counsel/mcp')
  })

  it('huangli sends {event, start_date, count} exactly (engine.huangli.HuangliDaysInput)', async () => {
    stubFetch(mcpOk([]))
    await engineHuangliDays('wedding', '2026-10-01', 30)
    expect(lastCall().body.params?.arguments).toEqual({ event: 'wedding', start_date: '2026-10-01', count: 30 })
  })

  it('bazi chart sends the unified Gregorian time contract exactly', async () => {
    stubFetch(mcpOk({}))
    await engineBaziChart({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
    expect(lastCall().body.params?.arguments).toEqual({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
  })

  it('ziwei chart sends the unified Gregorian time contract exactly', async () => {
    stubFetch(mcpOk({ gong_wei: [] }))
    await engineZiweiChart({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
    expect(lastCall().body.params?.arguments).toEqual({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
  })

  it('liuyao chart sends the unified Gregorian time contract exactly', async () => {
    stubFetch(mcpOk({}))
    await engineLiuyaoChart({
      instant: '2026-10-20T12:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 116.4,
      casting: {
        mode: 'auto',
        order: 'bottom_up',
        yaos: [7, 8, 8, 7, 8, 7],
        dong_yao: [],
        casting_id: 'a'.repeat(64),
      },
      yongShen: '世爻',
    })
    expect(lastCall().body.params?.arguments).toEqual({
      instant: '2026-10-20T12:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 116.4,
      casting: {
        mode: 'auto',
        order: 'bottom_up',
        yaos: [7, 8, 8, 7, 8, 7],
        dong_yao: [],
        casting_id: 'a'.repeat(64),
      },
      yong_shen: '世爻',
    })
  })

  it('liuyao qigua requires the frontend to materialize its mode', async () => {
    stubFetch(mcpOk({ casting: { mode: 'coins' } }))
    await engineLiuyaoQigua({ mode: 'auto' })
    expect(lastCall().body.params?.arguments).toEqual({ mode: 'auto' })
  })

  it('bazi bond sends both charts under {a, b} (engine.bazi.BaziBondInput)', async () => {
    stubFetch(mcpOk({}))
    await engineBaziBond({ nian: { gan: '甲' } }, { nian: { gan: '庚' } })
    expect(lastCall().body.params?.arguments).toEqual({
      a: { chart: { nian: { gan: '甲' } } },
      b: { chart: { nian: { gan: '庚' } } },
    })
  })

  it('ziwei daxian sends {chart} exactly (engine.ziwei.ZiweiDaXianInput)', async () => {
    stubFetch(mcpOk([]))
    await engineZiweiDaXian({ gong_wei: [] })
    expect(lastCall().body.params?.arguments).toEqual({ chart: { gong_wei: [] } })
  })

  it('ziwei liunian sends {chart, lunar_year} exactly (engine.ziwei.ZiweiLiuNianInput)', async () => {
    stubFetch(mcpOk({}))
    await engineZiweiLiuNian({ gong_wei: [] }, 2026)
    expect(lastCall().body.params?.arguments).toEqual({ chart: { gong_wei: [] }, lunar_year: 2026 })
  })

  it('maps MCP tool errors (isError) to engine_tool_failed with the tool message', async () => {
    stubFetch(mcpIsError('invalid_arguments: chart.gender: Field required'))
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).rejects.toMatchObject({
      code: 'engine_tool_failed',
      message: 'Error executing tool: invalid_arguments: chart.gender: Field required',
    } satisfies Partial<ApiError>)
  })

  it('maps non-2xx transport failures to engine_unavailable', async () => {
    stubFetch(new Response('bad gateway', { status: 502 }))
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).rejects.toMatchObject({
      status: 502,
      code: 'engine_unavailable',
    } satisfies Partial<ApiError>)
  })

  it('returns the parsed payload from the MCP content envelope', async () => {
    stubFetch(mcpOk([{ date: '2026-10-11', suitability: 'recommended' }]))
    await expect(engineHuangliDays('wedding', '2026-10-01', 1)).resolves.toEqual([
      { date: '2026-10-11', suitability: 'recommended' },
    ])
  })
})


describe('natal chart fan-out', () => {
  it('sends the base ziwei chart to ziwei_daxian, not the full chart', async () => {
    const base = { gong_wei: [{ zhi: '子' }], ming_gong: '子' }
    const full = { ...base, gong_wei: [{ zhi: '子', jiang_qian: 1, ages: [1] }] }
    const byTool: Record<string, unknown> = {
      bazi_chart: { nian: {}, yue: {}, ri: {}, shi: {}, birth_year: 1990, gender: 'female', zi_shi_rule: 'late' },
      ziwei_chart: base,
      bazi_fullchart: {},
      ziwei_fullchart: full,
      ziwei_daxian: [],
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation((_url: string, init: RequestInit) => {
      const name = JSON.parse(String(init.body)).params.name as string
      return Promise.resolve(mcpOk(byTool[name]))
    }))

    const natal = await natalChartForChartInput({
      instant: '1958-05-11T22:00:00Z',
      timezone: 'Asia/Shanghai',
      longitude: 121.47,
      gender: 'female',
    })
    expect(natal.ziweiBase).toEqual(base)

    const daxianCall = vi.mocked(fetch).mock.calls.find(call => {
      const body = JSON.parse(String(call[1]?.body)) as { params?: { name?: string } }
      return body.params?.name === 'ziwei_daxian'
    })
    expect(daxianCall).toBeTruthy()
    const daxianBody = JSON.parse(String(daxianCall![1]?.body)) as {
      params: { arguments: { chart: unknown } }
    }
    expect(daxianBody.params.arguments.chart).toEqual(base)
    expect(daxianBody.params.arguments.chart).not.toEqual(full)
  })

  it('attributes a failed natal call to its exact stage', async () => {
    const base = { gong_wei: [{ zhi: '子' }], ming_gong: '子' }
    const full = { ...base, gong_wei: [{ zhi: '子', jiang_qian: 1, ages: [1] }] }
    const byTool: Record<string, unknown> = {
      bazi_chart: { nian: {}, yue: {}, ri: {}, shi: {}, birth_year: 1990, gender: 'female', zi_shi_rule: 'late' },
      ziwei_chart: base,
      bazi_fullchart: {},
      ziwei_fullchart: full,
      ziwei_daxian: [],
    }
    const stages = ['bazi_chart', 'ziwei_chart', 'bazi_fullchart', 'ziwei_fullchart', 'ziwei_daxian'] as const
    for (const stage of stages) {
      vi.stubGlobal('fetch', vi.fn().mockImplementation((_url: string, init: RequestInit) => {
        const name = JSON.parse(String(init.body)).params.name as string
        return Promise.resolve(name === stage ? mcpIsError(`failed ${stage}`) : mcpOk(byTool[name]))
      }))
      const error = await natalChartForChartInput({
        instant: '1958-05-11T22:00:00Z',
        timezone: 'Asia/Shanghai',
        longitude: 121.47,
        gender: 'female',
      }).catch(cause => cause)
      expect(error).toBeInstanceOf(NatalStageError)
      expect(error).toMatchObject({ stage })
    }
  })
})
