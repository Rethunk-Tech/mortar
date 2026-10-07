import { readStored, writeStored } from '../shell/useStoredState.ts'
import { matchPaletteItems, type PaletteItem } from './match.ts'

// The order the sections appear in; the last is the one a long result list cuts.
const SECTION_ORDER: readonly ('recent' | 'goto' | 'actions' | 'settings' | 'mods')[] = [
  'recent',
  'goto',
  'actions',
  'settings',
  'mods',
]

const MAX_RESULTS = 50
const MAX_RECENT = 5
const RECENT_KEY = 'mortar.palette.recent'

const isIds = (v: unknown): v is string[] =>
  Array.isArray(v) && v.every((x) => typeof x === 'string')

function sectionOf(item: PaletteItem): Exclude<PaletteSection, 'recent'> {
  if (item.kind === 'profile' || item.id.startsWith('tab:')) {
    return 'goto'
  }
  if (item.kind === 'settings') {
    return 'settings'
  }
  if (
    item.kind === 'mod' ||
    item.id.startsWith('toggle-mod:') ||
    item.id.startsWith('configure-mod:')
  ) {
    return 'mods'
  }
  return 'actions'
}

export type PaletteSection = 'recent' | 'goto' | 'actions' | 'settings' | 'mods'

export const readRecent = (): string[] => readStored(RECENT_KEY, [], isIds)

export function rememberPicked(id: string): void {
  writeStored(RECENT_KEY, [id, ...readRecent().filter((x) => x !== id)].slice(0, MAX_RECENT))
}

export interface PaletteRow {
  item: PaletteItem
  section: PaletteSection
}

// The matches grouped under their section headers. With nothing typed, the entries picked lately lead in a Recent
// section; once there is a query, best matches lead within each section.
export function arrangePalette(
  items: readonly PaletteItem[],
  query: string,
  recent: readonly string[],
): PaletteRow[] {
  const matched = matchPaletteItems(items, query)
  const lead =
    query.trim() === '' ? recent.flatMap((id) => matched.filter((item) => item.id === id)) : []
  const led = new Set(lead.map((item) => item.id))
  const rows: PaletteRow[] = lead.map((item) => ({ item, section: 'recent' }))
  for (const section of SECTION_ORDER) {
    for (const item of matched) {
      if (!led.has(item.id) && sectionOf(item) === section) {
        rows.push({ item, section })
      }
    }
  }
  return rows.slice(0, MAX_RESULTS)
}
