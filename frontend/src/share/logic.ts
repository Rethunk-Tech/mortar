import type {
  Group,
  Info,
  Mod,
  Preview,
  Problem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'

const WARN_AT = 0.8
const SOURCE_SITE_NEXUS = 'nexus'
const KB = 1024

const downloads = (state: ModState) =>
  state === 'download' || state === 'dependency' || state === 'later'

// The bindings type a Go slice as nullable; the dialogs work with lists.
export type ShownGroup = Omit<Group, 'mods'> & { mods: string[] }
export type ShownInfo = Omit<Info, 'groups' | 'leftOut'> & {
  groups: ShownGroup[]
  leftOut: NonNullable<Info['leftOut']>
}
export type ShownPreview = Omit<Preview, 'mods' | 'problems'> & { mods: Mod[]; problems: Problem[] }

export const shownInfo = (i: Info): ShownInfo => ({
  ...i,
  groups: (i.groups ?? []).map((g) => ({ ...g, mods: g.mods ?? [] })),
  leftOut: i.leftOut ?? [],
})

export const shownPreview = (p: Preview): ShownPreview => ({
  ...p,
  mods: p.mods ?? [],
  problems: p.problems ?? [],
})

// About how many mods fit a link under Discord's limit (measured: 498 characters for 50 mods, 1,630 for 200).
export const SUGGEST_FILE_AT = 240

export type MeterLevel = 'ok' | 'warn' | 'over'

// How full the link is against the message limit; the bar stops at full while the level says it went over.
export function meter(length: number, limit: number): { ratio: number; level: MeterLevel } {
  const ratio = limit > 0 ? length / limit : 0
  let level: MeterLevel = 'ok'
  if (ratio > 1) {
    level = 'over'
  } else if (ratio > WARN_AT) {
    level = 'warn'
  }
  return { ratio: Math.min(1, ratio), level }
}

export const suggestFile = (count: number, tooLarge: boolean): boolean =>
  tooLarge || count > SUGGEST_FILE_AT

export const MOD_STATES = ['installed', 'download', 'dependency', 'later', 'unavailable'] as const

export type ModState = (typeof MOD_STATES)[number]

export const isModState = (s: string): s is ModState => MOD_STATES.some((x) => x === s)

export interface Summary {
  counts: Record<ModState, number>
  leftOut: number
  // Mods the import would queue.
  toImport: number
  // Of those, the ones that come from Nexus, which need a signed-in account.
  fromNexus: number
  sizeKb: number
}

// Counts what the import would do, leaving out the mods the user unticked.
export function summarize(mods: readonly Mod[], excluded: ReadonlySet<string>): Summary {
  const s: Summary = {
    counts: { installed: 0, download: 0, dependency: 0, later: 0, unavailable: 0 },
    leftOut: 0,
    toImport: 0,
    fromNexus: 0,
    sizeKb: 0,
  }
  for (const m of mods) {
    if (isModState(m.state)) {
      if (excluded.has(m.key) && m.state !== 'unavailable') {
        s.leftOut += 1
      } else {
        s.counts[m.state] += 1
        if (downloads(m.state)) {
          s.toImport += 1
          s.sizeKb += m.sizeKb
          s.fromNexus += m.site === SOURCE_SITE_NEXUS ? 1 : 0
        }
      }
    }
  }
  return s
}

// "12 MB" or "1.4 GB": the size is approximate, so one decimal only where it matters.
export function formatSize(kb: number): string {
  if (kb < KB) {
    return `${Math.max(0, Math.round(kb))} KB`
  }
  const mb = kb / KB
  if (mb < KB) {
    return `${mb < 10 ? mb.toFixed(1) : Math.round(mb)} MB`
  }
  return `${(mb / KB).toFixed(1)} GB`
}
