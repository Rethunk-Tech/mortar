const kindRe = /^\[([a-z_]+)] ([\s\S]*)$/

const kinds = new Set([
  'not_found',
  'busy',
  'network',
  'permission',
  'disk_full',
  'damaged',
  'invalid',
])

type ErrorKind =
  | 'not_found'
  | 'busy'
  | 'network'
  | 'permission'
  | 'disk_full'
  | 'damaged'
  | 'invalid'
  | 'unknown'

function asText(e: unknown): string {
  if (e instanceof Error) {
    return e.message
  }
  if (typeof e === 'string') {
    return e
  }
  if (e === undefined || e === null) {
    return ''
  }
  return String(e)
}

export function errorText(e: unknown): string | undefined {
  if (e instanceof Error) {
    return e.message
  }
  return typeof e === 'string' ? e : undefined
}

export function errorKind(e: unknown): ErrorKind {
  const kind = kindRe.exec(asText(e) ?? '')?.[1]
  if (kind !== undefined && kinds.has(kind)) {
    return kind as ErrorKind
  }
  return 'unknown'
}

/** Cause text for a Details tooltip; strips the `[kind] ` wire prefix when present. */
export function errorDetails(e: unknown): string {
  const text = asText(e) ?? ''
  const m = kindRe.exec(text)
  const kind = m?.[1]
  if (kind !== undefined && kinds.has(kind)) {
    return m?.[2] ?? text
  }
  return text
}
