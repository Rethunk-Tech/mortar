import type {
  Group,
  Info,
  Mod,
  Preview,
  Problem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'

const WARN_AT = 0.8
const SOURCE_SITE_NEXUS = 'nexus'

const downloads = (state: ModState) =>
  state === 'download' || state === 'dependency' || state === 'later'

// The bindings type a Go slice as nullable; the dialogs work with lists.
type ShownGroup = Omit<Group, 'mods'> & { mods: string[] }
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

export function missingModName(
  id: string,
  mods: readonly { name: string; ids?: string[] | null }[],
): string | undefined {
  return mods.find((m) => (m.ids ?? []).includes(id))?.name
}

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

// Mods the share carries, counted the way the profile's own mod count is (mods, not install entries).
export const sharedMods = (info: ShownInfo): number =>
  info.groups.reduce((n, g) => n + g.mods.length, 0)

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

export type Carried = 'notes' | 'configFiles' | 'problemChoices'

// What a received share carries besides its mods, in the order the preview lists it.
export function carriedBy(p: { notes: string; settings: number; choices: number }): Carried[] {
  const out: Carried[] = []
  if (p.notes.trim() !== '') {
    out.push('notes')
  }
  if (p.settings > 0) {
    out.push('configFiles')
  }
  if (p.choices > 0) {
    out.push('problemChoices')
  }
  return out
}

// A mod that travelled as an itch.io page alone: the receiver opens the page and saves the file there.
export function needsItchFile(mod: { site: string; reason?: string }): boolean {
  return mod.site === 'itch' && mod.reason === 'itch'
}

// A CurseForge file the receiver cannot download until they have a CurseForge key: its page is the way to fetch it.
export function needsCurseForgeKey(mod: { site: string; reason?: string }): boolean {
  return mod.site === 'curseforge' && mod.reason === 'curseforge-key'
}

// A mod that travelled as a Patreon post alone: the receiver opens the post and saves the file there.
export function needsPatreonFile(mod: { site: string; reason?: string }): boolean {
  return mod.site === 'patreon' && mod.reason === 'patreon'
}
