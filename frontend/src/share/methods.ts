import type { ShareInclude } from './shareDefaults.ts'

const MAX_COLUMNS = 3
const SQUARE_TILES = 4
const DESTINATIONS = ['mortar', 'nexus', 'thunderstore', 'nearby', 'list'] as const

// Destinations that carry only mods the source sites know; a mod list, a .mortar file and a paired computer carry
// local archives too.
const SITE_ONLY: readonly Destination[] = ['nexus', 'thunderstore']

// Why a tile cannot be used: nothing at all to send, or only local archives, which a collection cannot carry.
type DisabledReason = 'empty' | 'local-only'

export type Destination = (typeof DESTINATIONS)[number]

export const isDestination = (v: unknown): v is Destination => DESTINATIONS.some((d) => d === v)

// What Mortar sends: a link, or a .mortar file with the mod settings.
export type MortarFormat = 'link' | 'file'

export interface DestinationEntry {
  id: Destination
  // The tile stays in place when disabled so the grid does not shift.
  disabled: boolean
  reason: DisabledReason | null
}

// The grid's tiles in order. Thunderstore's r2modman code and modpack only exist for games modded from Thunderstore.
// `count` is the mods a link carries; `leftOut` the enabled ones it cannot (local archives).
export function shareDestinations(opts: {
  thunderstore: boolean
  count: number
  leftOut: number
}): DestinationEntry[] {
  return DESTINATIONS.filter((id) => id !== 'thunderstore' || opts.thunderstore).map((id) => {
    const needsSite = SITE_ONLY.includes(id)
    let reason: DisabledReason | null = null
    if (opts.count + opts.leftOut === 0) {
      reason = 'empty'
    } else if (needsSite && opts.count === 0) {
      reason = 'local-only'
    }
    return { id, disabled: reason !== null, reason }
  })
}

// Columns that leave no lone tile on the last row: one row up to three tiles, 2x2 for four, 3 for five.
export const gridColumns = (tiles: number): number =>
  tiles === SQUARE_TILES ? 2 : Math.min(tiles, MAX_COLUMNS)

// The remembered destination when it still has a tile that can be used.
export function lastUsedDestination(
  remembered: Destination | null,
  entries: readonly DestinationEntry[],
): Destination | null {
  return entries.some((e) => e.id === remembered && !e.disabled) ? remembered : null
}

export const destinationStorageKey = (game: string) => `mortar.share.method.${game}`

// Where an Included setting goes: a link, a .mortar file, or a nearby computer, which is paired (gets whole files) or
// not (gets only what the list of mods and settings can carry).
export type IncludeTarget = 'link' | 'file' | 'paired' | 'unpaired'

// Why a setting cannot apply to a target.
export type Unavailable = 'file-only' | 'paired-only'

// What each Included setting cannot do for the target; null means it applies. A link has no room for config files or
// problem choices, and config files can hold keys, so only a paired computer gets them.
export function includeAvailability(
  target: IncludeTarget,
): Record<keyof ShareInclude, Unavailable | null> {
  const linkOnly = target === 'link' ? 'file-only' : null
  return {
    disabledMods: null,
    fomodChoices: null,
    notes: null,
    configFiles: target === 'unpaired' ? 'paired-only' : linkOnly,
    problemChoices: linkOnly,
  }
}

// Which of the "Included" settings a target honours, in the order the line lists them.
export function includedKeys(
  include: ShareInclude,
  opts: { target: IncludeTarget; fomod: boolean },
): (keyof ShareInclude)[] {
  const can = includeAvailability(opts.target)
  const keys: (keyof ShareInclude)[] = []
  for (const key of ['notes', 'fomodChoices', 'configFiles', 'problemChoices'] as const) {
    if (include[key] && can[key] === null && (key !== 'fomodChoices' || opts.fomod)) {
      keys.push(key)
    }
  }
  return keys
}

// Mods the share leaves out, split into the ones the user added from archives and the rest.
export function leftOutCounts(leftOut: readonly { reason: string }[]): {
  local: number
  other: number
} {
  const local = leftOut.filter((o) => o.reason === 'local').length
  return { local, other: leftOut.length - local }
}
