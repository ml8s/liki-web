import { del, get, patch, post } from './http'

/** 工具记录类型：与后端 tool_records.kind CHECK 一致。 */
export type RecordKind = 'liuyao' | 'huangli' | 'bond' | 'bazhai'

export interface ToolRecord {
  id: string
  user_id: string
  kind: RecordKind
  title: string
  input: Record<string, unknown>
  output?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export async function fetchRecords(kind: RecordKind): Promise<ToolRecord[]> {
  const payload = await get<{ records: ToolRecord[] }>(`/api/records?kind=${encodeURIComponent(kind)}`)
  return payload.records ?? []
}

export function createRecord(kind: RecordKind, title: string, input: unknown, output?: unknown): Promise<ToolRecord> {
  return post<ToolRecord>('/api/records', { kind, title, input, output })
}

export function renameRecord(id: string, title: string): Promise<ToolRecord> {
  return patch<ToolRecord>(`/api/records/${id}`, { title })
}

export function deleteRecord(id: string): Promise<{ status: string }> {
  return del<{ status: string }>(`/api/records/${id}`)
}
