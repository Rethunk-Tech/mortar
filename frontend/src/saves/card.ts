const minute = 60_000
const hour = 60 * minute

// Stardew Game1.whichFarm / Farm layout ids (1.6).
export const farmTypes = [
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

export const goldText = (money: number) => `${money.toLocaleString('en-US')}g`
