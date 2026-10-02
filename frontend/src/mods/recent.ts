const DAYS_PER_WEEK = 7
const DAY_MS = 86_400_000

export const WEEK_MS = DAYS_PER_WEEK * DAY_MS

export const addedWithin = (added: string | undefined, span: number, now: number) => {
  const at = added ? Date.parse(added) : Number.NaN
  return Number.isFinite(at) && now - at <= span
}
