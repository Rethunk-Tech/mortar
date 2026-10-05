const MINUTE = 60_000
const HOUR = 60 * MINUTE
const WHOLE_HOURS = 10

// "45 min" under an hour, "2.5 hr" under ten, then whole hours; '' when nothing is recorded.
export function formatPlaytime(ms: number, locale = 'en'): string {
  if (!(ms >= MINUTE)) {
    return ''
  }
  const fmt = (unit: 'minute' | 'hour', digits: number, n: number) =>
    new Intl.NumberFormat(locale, {
      style: 'unit',
      unit,
      unitDisplay: 'short',
      maximumFractionDigits: digits,
    }).format(n)
  if (ms < HOUR) {
    return fmt('minute', 0, Math.floor(ms / MINUTE))
  }
  const hours = ms / HOUR
  return fmt('hour', hours < WHOLE_HOURS ? 1 : 0, hours)
}
