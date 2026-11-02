import { formatNumber } from '../i18n/numberFormat.ts'

const minute = 60_000
const hour = 60 * minute

export const hoursPlayed = (ms: number) => Math.floor(ms / hour)

// Before Lingui activates a locale, i18n.locale is "", which toLocaleString rejects.
export const goldText = (money: number) => `${formatNumber(money)}g`
