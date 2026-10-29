import { ApiError } from '../http'
import { reportClientError } from '../../utils/client-error'

/** counsel（裁决/问卦/起名）工具走 /counsel/mcp；排盘/历法工具走 /engine/mcp。 */
export const COUNSEL_TOOLS = new Set<string>([
  'bazi_yongshen', 'compute_factors', 'natal_query', 'period_query',
  'liuyao_query', 'liuyao_snapshot', 'qimen_query', 'qimen_snapshot',
  'naming_surname', 'naming_pick', 'naming_char', 'naming_compose', 'naming_check',
])

interface EngineToolEnvelope {
  result?: { content?: Array<{ type?: string; text?: string }>; structuredContent?: unknown; isError?: boolean }
  error?: { code?: number; message?: string }
}

/** Single source for both the HTTP contract and the in-band metadata contract. */
export const MCP_PROTOCOL_VERSION = '2026-07-28'

/**
 * Streamable HTTP carries the JSON-RPC method and tool name in headers so the
 * server can route before parsing the body. Engine/counsel require them for
 * every tools/call request.
 */
export function mcpHeaders(tool: string): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    Accept: 'application/json, text/event-stream',
    'Mcp-Protocol-Version': MCP_PROTOCOL_VERSION,
    'Mcp-Method': 'tools/call',
    'Mcp-Name': tool,
  }
}

function endpoint(tool: string): string {
  return COUNSEL_TOOLS.has(tool) ? '/counsel/mcp' : '/engine/mcp'
}

/** Streamable MCP tool call; JSON-response only is the supported browser profile. */
export async function mcpTool<T = unknown>(tool: string, params: unknown): Promise<T> {
  let response: Response
  try {
    response = await fetch(endpoint(tool), {
      method: 'POST',
      headers: mcpHeaders(tool),
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
    let detail = `Engine HTTP ${response.status}`
    let code = 'engine_unavailable'
    try {
      const envelope = await response.json() as EngineToolEnvelope
      if (envelope.error?.message) detail = envelope.error.message
      if (envelope.error?.code === -32020) code = 'engine_protocol_mismatch'
    } catch {
      // Keep the HTTP status fallback.
    }
    reportClientError({ code, detail })
    throw new ApiError(response.status, code, detail)
  }
  let envelope: EngineToolEnvelope
  try {
    envelope = await response.json() as EngineToolEnvelope
  } catch {
    throw new ApiError(response.status, 'invalid_engine_response', 'Invalid engine response')
  }
  if (envelope.error) {
    const code = envelope.error.code === -32020 ? 'engine_protocol_mismatch' : 'engine_tool_failed'
    reportClientError({ code, detail: envelope.error.message })
    throw new ApiError(response.status, code, envelope.error.message || 'Engine tool failed')
  }
  if (envelope.result?.isError) {
    const detail = envelope.result.content?.find(c => c.type === 'text')?.text
    throw new ApiError(response.status, 'engine_tool_failed', detail || 'Engine tool failed')
  }
  // Structured-only wire: every tool publishes a typed outputSchema and the
  // duplicated text payload was removed; a missing structuredContent is a
  // contract violation, not something to paper over with a text fallback.
  const structured = envelope.result?.structuredContent
  if (structured === undefined) {
    throw new ApiError(response.status, 'invalid_engine_response', 'Engine tool result missing structuredContent')
  }
  return structured as T
}
