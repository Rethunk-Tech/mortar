import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { cmpText } from '../cmpText.ts'

const FIRST = ['nexus', 'github', 'thunderstore']

const rank = (name: string) => {
  const i = FIRST.findIndex((f) => name.toLowerCase().startsWith(f))
  return i === -1 ? FIRST.length : i
}

/** Rows grouped by the source that reported the update: Nexus, GitHub, Thunderstore, then the rest alphabetically. */
export function groupBySource(list: readonly Update[]): { name: string; list: Update[] }[] {
  const by = Map.groupBy(list, (u) => u.source)
  return [...by.entries()]
    .map(([name, rows]) => ({ name, list: rows }))
    .sort((a, b) => rank(a.name) - rank(b.name) || cmpText(a.name, b.name))
}
