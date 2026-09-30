interface Ago {
  unit: 'now' | 'minute' | 'hour' | 'day'
  n: number
}

const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

function ago(elapsedMs: number): Ago {
  if (elapsedMs < MINUTE) {
    return { unit: 'now', n: 0 }
  }
  if (elapsedMs < HOUR) {
    return { unit: 'minute', n: Math.floor(elapsedMs / MINUTE) }
  }
  if (elapsedMs < DAY) {
    return { unit: 'hour', n: Math.floor(elapsedMs / HOUR) }
  }
  return { unit: 'day', n: Math.floor(elapsedMs / DAY) }
}

export { ago }
