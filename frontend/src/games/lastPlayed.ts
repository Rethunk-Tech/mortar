const MS_PER_SEC = 1000
const JUST_NOW_SECS = 45
const SECS_PER_MIN = 60
const MINS_PER_HOUR = 60
const HOURS_PER_DAY = 24

export type RelativePlay =
  | { kind: 'now' }
  | { kind: 'unit'; n: number; unit: 'minute' | 'hour' | 'day' }

export function relativePlay(at: string, nowMs: number): RelativePlay | null {
  const then = Date.parse(at)
  if (!Number.isFinite(then)) {
    return null
  }
  const sec = Math.max(0, Math.floor((nowMs - then) / MS_PER_SEC))
  if (sec < JUST_NOW_SECS) {
    return { kind: 'now' }
  }
  const min = Math.floor(sec / SECS_PER_MIN)
  if (min < MINS_PER_HOUR) {
    return { kind: 'unit', n: Math.max(1, min), unit: 'minute' }
  }
  const hr = Math.floor(min / MINS_PER_HOUR)
  if (hr < HOURS_PER_DAY) {
    return { kind: 'unit', n: hr, unit: 'hour' }
  }
  return { kind: 'unit', n: Math.floor(hr / HOURS_PER_DAY), unit: 'day' }
}
