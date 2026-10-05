import { i18n } from '@lingui/core'

const minute = 60_000
const hour = 60 * minute

// Game1.whichFarm / Farm layout ids (1.6).
const farmTypes = [
  'Standard',
  'Riverland',
  'Forest',
  'Hill-top',
  'Wilderness',
  'Four Corners',
  'Beach',
  'Meadowlands',
] as const

export const farmTypeName = (whichFarm: number) => farmTypes[whichFarm] ?? ''

export const hoursPlayed = (ms: number) => Math.floor(ms / hour)

// Before Lingui activates a locale, i18n.locale is "", which toLocaleString rejects.
export const goldText = (money: number) => `${money.toLocaleString(i18n.locale || undefined)}g`
