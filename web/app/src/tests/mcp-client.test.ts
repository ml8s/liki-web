import { afterEach, describe, expect, it, vi } from 'vitest'
import { mcpTool, MCP_PROTOCOL_VERSION } from '@shared/api/mcp/client'

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('mcpTool', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the protocol version in both HTTP and MCP metadata', async () => {
    const fetchMock = vi.fn(async (_url: string | URL | Request, _init?: RequestInit) => jsonResponse({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [], structuredContent: 'ok' },
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(mcpTool('bazi_chart', { gender: 'female' })).resolves.toBe('ok')

    expect(fetchMock).toHaveBeenCalledWith('/engine/mcp', expect.objectContaining({
      method: 'POST',
      headers: expect.objectContaining({
        'Content-Type': 'application/json',
        Accept: 'application/json, text/event-stream',
        'Mcp-Protocol-Version': MCP_PROTOCOL_VERSION,
        'Mcp-Method': 'tools/call',
        'Mcp-Name': 'bazi_chart',
      }),
    }))
    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))
    expect(body.params._meta['io.modelcontextprotocol/protocolVersion']).toBe(MCP_PROTOCOL_VERSION)
  })

  it('routes counsel tools to the counsel endpoint', async () => {
    const fetchMock = vi.fn(async () => jsonResponse({
      jsonrpc: '2.0',
      id: 1,
      result: { content: [], structuredContent: { answer: true } },
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(mcpTool('bazi_yongshen', {})).resolves.toEqual({ answer: true })
    expect(fetchMock).toHaveBeenCalledWith('/counsel/mcp', expect.anything())
  })

  it('classifies a missing MCP method header as a protocol mismatch', async () => {
    const fetchMock = vi.fn(async () => new Response(
      JSON.stringify({ jsonrpc: '2.0', id: 1, error: { code: -32020, message: 'missing required Mcp-Method header' } }),
      { status: 400, headers: { 'Content-Type': 'application/json' } },
    ))
    vi.stubGlobal('fetch', fetchMock)

    await expect(mcpTool('bazi_chart', {})).rejects.toMatchObject({
      code: 'engine_protocol_mismatch',
      message: 'missing required Mcp-Method header',
    })
  })

})
