export function firstQueryValue(value: unknown): string {
  if (Array.isArray(value)) return firstQueryValue(value[0])
  return typeof value === 'string' ? value : ''
}

/** Reads a query enum without letting a hand-edited URL corrupt view state. */
export function queryChoice(
  query: Record<string, unknown>,
  key: string,
  allowed: readonly string[],
  fallback: string,
): string {
  const value = firstQueryValue(query[key])
  return allowed.includes(value) ? value : fallback
}

export function queryText(query: Record<string, unknown>, key: string): string {
  return firstQueryValue(query[key]).trim()
}
