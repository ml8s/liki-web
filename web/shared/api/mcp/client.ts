import { ApiError } from '../http'

/** counsel（裁决/问卦/起名）工具走 /counsel/mcp；排盘/历法工具走 /engine/mcp。 */
export const COUNSEL_TOOLS = new Set<string>([
  'bazi_yongshen', 'compute_factors', 'natal_query', 'period_query',
  'liuyao_query', 'liuyao_snapshot', 'qimen_query', 'qimen_snapshot',
  'naming_surname', 'naming_pick', 'naming_char', 'naming_compose', 'naming_check',
])

interface EngineToolEnvelope {
  result?: { content?: Array<{ type?: string; text?: string }>; isError?: boolean }
  error?: { code?: number; message?: string }
}

/** Single source for both the HTTP contract and the in-band metadata contract. */
export const MCP_PROTOCOL_VERSION = '2026-07-28'

function endpoint(tool: string): string {
  return COUNSEL_TOOLS.has(tool) ? '/counsel/mcp' : '/engine/mcp'
}

/** Streamable MCP tool call; JSON-response only is the supported browser profile. */
export async function mcpTool<T = unknown>(tool: string, params: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch(endpoint(tool), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json, text/event-stream',
        'Mcp-Protocol-Version': MCP_PROTOCOL_VERSION,
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: 1,
        method: 'tools/call',
        params: {
          name: tool,
          arguments: params,
          _meta: {
            'io.modelcontextprotocol/protocolVersion': MCP_PROTOCOL_VERSION,
            'io.modelcontextprotocol/clientCapabilities': {},
          },
        },
      }),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Network request failed')
  }
  if (!response.ok) {
    throw new ApiError(response.status, 'engine_unavailable', `Engine HTTP ${response.status}`)
  }
  let envelope: EngineToolEnvelope
  try {
    envelope = await response.json() as EngineToolEnvelope
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  if (envelope.error) {
    throw new ApiError(response.status, 'engine_tool_failed', envelope.error.message || 'Engine tool failed')
  }
  const content = envelope.result?.content?.[0]
  const text = content?.type === 'text' ? content.text : undefined
  if (envelope.result?.isError || text === undefined) {
    throw new ApiError(response.status, 'engine_tool_failed', text || 'Engine tool failed')
  }
  try {
    return JSON.parse(text) as T
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
}
