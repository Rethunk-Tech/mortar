const kindRe = /^\[([a-z_]+)] ([\s\S]*)$/

const knownKinds = [
  'not_found',
  'busy',
  'network',
  'permission',
  'disk_full',
  'damaged',
  'invalid',
  'other_game',
  'outdated',
  'external',
] as const

const kinds = new Set<string>(knownKinds)

type ErrorKind = (typeof knownKinds)[number] | 'unknown'

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

// A bound call's error that Go did not tag still carries its classified kind in the Wails cause.
function causeKind(e: unknown): string | undefined {
  const cause: unknown = e instanceof Error ? e.cause : undefined
  if (typeof cause === 'object' && cause !== null && 'kind' in cause) {
    return typeof cause.kind === 'string' ? cause.kind : undefined
  }
  return undefined
}

export function errorKind(e: unknown): ErrorKind {
  const kind = kindRe.exec(asText(e) ?? '')?.[1] ?? causeKind(e)
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
