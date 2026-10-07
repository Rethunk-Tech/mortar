import type { ShareInclude } from './shareDefaults.ts'

const DESTINATIONS = ['mortar', 'nexus', 'thunderstore', 'nearby', 'list'] as const

export type Destination = (typeof DESTINATIONS)[number]

export const isDestination = (v: unknown): v is Destination => DESTINATIONS.some((d) => d === v)

// What Mortar sends: a link, or a .mortar file with the mod settings.
export type MortarFormat = 'link' | 'file'

export interface DestinationEntry {
  id: Destination
  // Nothing to send yet: the tile stays in place so the grid does not shift.
  disabled: boolean
}

// The grid's tiles in order. Thunderstore's r2modman code and modpack only exist for games modded from Thunderstore.
export function shareDestinations(opts: {
  thunderstore: boolean
  count: number
}): DestinationEntry[] {
  return DESTINATIONS.filter((id) => id !== 'thunderstore' || opts.thunderstore).map((id) => ({
    id,
    disabled: opts.count === 0,
  }))
}

// The remembered destination when it still has a tile that can be used.
export function lastUsedDestination(
  remembered: Destination | null,
  entries: readonly DestinationEntry[],
): Destination | null {
  return entries.some((e) => e.id === remembered && !e.disabled) ? remembered : null
}

export const destinationStorageKey = (game: string) => `mortar.share.method.${game}`

// Which of the "Included" settings a method honours, in the order the line lists them.
export function includedKeys(
  include: ShareInclude,
  opts: { file: boolean; fomod: boolean },
): (keyof ShareInclude)[] {
  const keys: (keyof ShareInclude)[] = []
  if (include.notes) {
    keys.push('notes')
  }
  if (opts.fomod && include.fomodChoices) {
    keys.push('fomodChoices')
  }
  if (opts.file && include.configFiles) {
    keys.push('configFiles')
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
