function needle(s: string): string {
  return s.trim().toLowerCase()
}

const WORDS = /\s+/
const RANK_FUZZY = 1
const RANK_WORD = 2
const RANK_CONTIGUOUS = 3
const RANK_EXACT = 4
const MAX_RESULTS = 50

function fuzzyRank(query: string, text: string): number {
  const q = needle(query)
  const t = needle(text)
  if (q.length === 0) {
    return RANK_FUZZY
  }
  if (t.includes(q)) {
    return RANK_CONTIGUOUS
  }
  const words = t.split(WORDS)
  if (words.some((word) => word.startsWith(q))) {
    return RANK_WORD
  }
  let qi = 0
  for (const ch of t) {
    if (ch === q[qi]) {
      qi += 1
      if (qi === q.length) {
        return RANK_FUZZY
      }
    }
  }
  return 0
}

function rank(query: string, item: PaletteItem): number {
  const q = needle(query)
  if (q.length === 0) {
    return RANK_FUZZY
  }
  const fields = [item.label, item.hint ?? '', item.match ?? '']
  let best = 0
  for (const field of fields) {
    const t = needle(field)
    if (t === q) {
      best = Math.max(best, RANK_EXACT)
    } else if (t.startsWith(q)) {
      best = Math.max(best, RANK_CONTIGUOUS)
    } else {
      best = Math.max(best, fuzzyRank(query, field))
    }
  }
  return best
}

type PaletteKind = 'profile' | 'mod' | 'settings' | 'action' | 'shortcut'

export interface PaletteItem {
  id: string
  kind: PaletteKind
  label: string
  hint?: string
  match?: string
  shortcut?: string
  /** Why the action cannot run now; the row shows it in place of the hint and ignores picks. */
  disabled?: string
}

export function matchPaletteItems(items: readonly PaletteItem[], query: string): PaletteItem[] {
  const scored = items
    .map((item) => ({ item, score: rank(query, item) }))
    .filter((row) => row.score > 0)
  scored.sort((a, b) => {
    if (b.score !== a.score) {
      return b.score - a.score
    }
    return a.item.label.localeCompare(b.item.label)
  })
  return scored.slice(0, MAX_RESULTS).map((row) => row.item)
}
