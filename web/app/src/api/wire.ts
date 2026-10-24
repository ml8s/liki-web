/**
 * Wire boundary helper.
 *
 * Third-party protocol clients (AG-UI, MCP) must only receive plain JSON
 * values. Vue reactive proxies are not structured-cloneable: passing them to
 * libraries that clone their input (AG-UI does) fails before any request is
 * sent. Serializing at this boundary keeps UI state out of the wire contract.
 */
export function toWire<T>(value: T): T {
  if (value === undefined) return value
  return JSON.parse(JSON.stringify(value)) as T
}
