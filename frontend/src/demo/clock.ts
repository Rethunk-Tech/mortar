// Recorded times moved to the visit, so the demo reads as happening now: what was installed minutes before the
// recording was made was installed minutes ago, and a run started on Play started now.
const ISO = /"(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d))"/g
// Milliseconds since the epoch, as some answers count them.
const EPOCH = /(?<=[:,[])(1\d{12})(?=[,\]}])/g
// Only a stamp near the recording is the recording's own; an older one (a mod's release) stays put.
const DAY_MS = 86_400_000
const NEAR_MS = 2 * DAY_MS

export function shifted(value: unknown, delta: number, around: number): unknown {
  if (delta === 0 || value === null || typeof value !== 'object') {
    return value
  }
  const near = (ms: number) => Math.abs(ms - around) < NEAR_MS
  const text = JSON.stringify(value)
    .replace(ISO, (whole, iso: string) => {
      const ms = Date.parse(iso)
      return Number.isNaN(ms) || !near(ms) ? whole : `"${new Date(ms + delta).toISOString()}"`
    })
    .replace(EPOCH, (n) => (near(Number(n)) ? String(Number(n) + delta) : n))
  return JSON.parse(text)
}
