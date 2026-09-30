function needle(s: string): string {
  return s.trim().toLowerCase()
}

function fuzzyHit(query: string, text: string): boolean {
  const q = needle(query)
  const t = needle(text)
  if (q.length === 0) {
    return true
  }
  if (t.includes(q)) {
    return true
  }
  let qi = 0
  for (const ch of t) {
    if (ch === q[qi]) {
      qi += 1
      if (qi === q.length) {
        return true
      }
    }
  }
  return false
}

const rankExact = 3
const rankPrefix = 2
const rankFuzzy = 1

function rank(query: string, item: PaletteItem): number {
  const q = needle(query)
  if (q.length === 0) {
    return rankFuzzy
  }
  const fields = [item.label, item.hint ?? '', item.id]
  let best = 0
  for (const field of fields) {
    const t = needle(field)
    if (t === q) {
      best = Math.max(best, rankExact)
    } else if (t.startsWith(q)) {
      best = Math.max(best, rankPrefix)
    } else if (fuzzyHit(query, field)) {
      best = Math.max(best, rankFuzzy)
    }
  }
  return best
}

export type PaletteKind = 'profile' | 'mod' | 'settings' | 'action' | 'shortcut'

export interface PaletteItem {
  id: string
  kind: PaletteKind
  label: string
  hint?: string
  shortcut?: string
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
  return scored.map((row) => row.item)
}
