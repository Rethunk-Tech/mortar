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
  const text = asText(e)
  const m = kindRe.exec(text)
  if (m && kinds.has(m[1])) {
    return m[1] as ErrorKind
  }
  return 'unknown'
}

/** Cause text for a Details tooltip; strips the `[kind] ` wire prefix when present. */
export function errorDetails(e: unknown): string {
  const text = asText(e)
  const m = kindRe.exec(text)
  if (m && kinds.has(m[1])) {
    return m[2]
  }
  return text
}
