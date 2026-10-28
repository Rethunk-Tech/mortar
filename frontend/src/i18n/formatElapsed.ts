const SECOND = 1000
const MINUTE = 60
const HOUR = 3600

const pad = (n: number) => String(n).padStart(2, '0')

/** A duration as m:ss, or h:mm:ss from an hour on; negative durations read 0:00. */
export function formatElapsed(ms: number): string {
  const total = Math.max(0, Math.floor(ms / SECOND))
  const hours = Math.floor(total / HOUR)
  const minutes = Math.floor((total % HOUR) / MINUTE)
  const seconds = pad(total % MINUTE)
  return hours > 0 ? `${hours}:${pad(minutes)}:${seconds}` : `${minutes}:${seconds}`
}
