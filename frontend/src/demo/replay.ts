/** A recorded answer to one Wails runtime call: the HTTP status and the body, parsed when it was JSON. */
export type Reply = { status: number; json: unknown } | { status: number; text: string }

export type Calls = Record<string, Reply>

/** A backend event, as the server's event socket sends it, `at` milliseconds into a recorded run. */
export interface TimedEvent {
  at: number
  name: string
  data: unknown
}

export interface TimedCall {
  at: number
  key: string
  reply: Reply
}

/**
 * What the demo replays: the answers its views open on; the same views in other mod states, keyed by `modState`; a new
 * empty profile's answers; one recorded play of the demo profile, timed from its start call; and the answers once
 * that run has ended.
 */
export interface Recording {
  calls: Calls
  states: Record<string, Calls>
  blank: { id: string; profile: unknown; calls: Calls }
  run: {
    /** When the recorded run started, in milliseconds since the epoch. */
    wall: number
    start: string
    stop: string
    stopAt: number
    endAt: number
    events: TimedEvent[]
    calls: TimedCall[]
  }
  after: Calls
}

export const isRecord = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null

/**
 * Names a runtime call by what it asks for. The runtime gives each binding call a fresh call-id, which the key leaves
 * out so a replayed call matches the recorded one.
 */
export function callKey(body: string): string {
  const parsed: unknown = JSON.parse(body)
  if (isRecord(parsed) && isRecord(parsed.args)) {
    const { 'call-id': _id, ...args } = parsed.args
    return JSON.stringify({ ...parsed, args })
  }
  return JSON.stringify(parsed)
}

/** The bound method a runtime call names, and its arguments. */
export function callOf(key: string): { method: number; args: unknown[] } {
  const parsed: unknown = JSON.parse(key)
  const call = isRecord(parsed) && isRecord(parsed.args) ? parsed.args : {}
  return {
    method: typeof call.methodID === 'number' ? call.methodID : 0,
    args: Array.isArray(call.args) ? call.args : [],
  }
}

/** A runtime call's key for a method and its arguments. */
export const keyOf = (method: number, args: unknown[]) =>
  JSON.stringify({ object: 0, method: 0, args: { methodID: method, args } })

/** A profile record as the backend sends it; an empty one leaves out its entries. */
export const isProfile = (v: unknown): v is Record<string, unknown> =>
  isRecord(v) && typeof v.id === 'string' && typeof v.name === 'string' && 'created' in v

/** Which of a profile's mods are switched off, the part of its state the recorded views differ by. */
export function modState(profile: unknown): string {
  if (!(isRecord(profile) && Array.isArray(profile.entries))) {
    return ''
  }
  const off = profile.entries
    .filter(isRecord)
    .flatMap((e) => (Array.isArray(e.disabled) ? e.disabled.map(String) : []))
    .sort()
  return `${String(profile.id)}:${off.join(',')}`
}
