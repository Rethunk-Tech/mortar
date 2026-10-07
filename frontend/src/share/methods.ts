import type { ShareInclude } from './shareDefaults.ts'

const SHARE_METHODS = ['link', 'file', 'list', 'nexus', 'nearby', 'thunderstore'] as const

export type ShareMethod = (typeof SHARE_METHODS)[number]

export const isShareMethod = (v: unknown): v is ShareMethod => SHARE_METHODS.some((m) => m === v)

export interface MethodEntry {
  id: ShareMethod
  // Nothing to send yet: the method stays listed so the rail does not shift.
  disabled: boolean
}

// The rail's methods in order. Thunderstore's r2modman code and modpack only exist for games modded from Thunderstore.
export function shareMethods(opts: { thunderstore: boolean; count: number }): MethodEntry[] {
  return SHARE_METHODS.filter((id) => id !== 'thunderstore' || opts.thunderstore).map((id) => ({
    id,
    disabled: opts.count === 0,
  }))
}

// The remembered method when it is still usable, else the suggested one, else the link.
export function pickMethod(
  remembered: ShareMethod | null,
  suggested: ShareMethod,
  methods: readonly MethodEntry[],
): ShareMethod {
  const usable = (id: ShareMethod | null) =>
    id !== null && methods.some((m) => m.id === id && !m.disabled)
  if (usable(remembered)) {
    return remembered as ShareMethod
  }
  return usable(suggested) ? suggested : 'link'
}

export const methodStorageKey = (game: string) => `mortar.share.method.${game}`

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
