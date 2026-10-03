const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const DAYS_PER_WEEK = 7
const WEEK = DAYS_PER_WEEK * DAY
const FIRST_YEAR = 1970

export interface WhenOptions {
  withTime?: boolean
  now?: number
  locale?: string
  // The translated text for under a minute ago, which Intl words as "now".
  fewSeconds?: string
  absolute?: boolean
}

// relativeWhen reads a moment as "3 minutes ago" or "yesterday" within the last week and as a date before that,
// with the time too when withTime is set. A zero or unparseable time gives ''.
export function relativeWhen(
  value: string | number | Date,
  {
    withTime = false,
    now = Date.now(),
    locale = 'en',
    fewSeconds = 'a few seconds ago',
    absolute = false,
  }: WhenOptions = {},
): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime()) || d.getUTCFullYear() < FIRST_YEAR) {
    return ''
  }
  if (absolute) {
    return new Intl.DateTimeFormat(locale, {
      dateStyle: 'medium',
      ...(withTime ? { timeStyle: 'short' as const } : {}),
    }).format(d)
  }
  const elapsed = now - d.getTime()
  if (elapsed < MINUTE) {
    return fewSeconds
  }
  if (elapsed < WEEK) {
    const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
    if (elapsed < HOUR) {
      return rtf.format(-Math.floor(elapsed / MINUTE), 'minute')
    }
    if (elapsed < DAY) {
      return rtf.format(-Math.floor(elapsed / HOUR), 'hour')
    }
    return rtf.format(-Math.floor(elapsed / DAY), 'day')
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    ...(withTime ? { timeStyle: 'short' as const } : {}),
  }).format(d)
}
