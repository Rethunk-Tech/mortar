/** A recorded answer to one Wails runtime call: the HTTP status and the body, parsed when it was JSON. */
export type Reply = { status: number; json: unknown } | { status: number; text: string }

export type Recording = Record<string, Reply>

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null

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
