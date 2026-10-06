import { i18n } from '@lingui/core'

const minute = 60_000
const hour = 60 * minute

export const hoursPlayed = (ms: number) => Math.floor(ms / hour)

// Before Lingui activates a locale, i18n.locale is "", which toLocaleString rejects.
export const goldText = (money: number) => `${money.toLocaleString(i18n.locale || undefined)}g`
