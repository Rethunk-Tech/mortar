import type { ListRow } from './listColumns.ts'

/** Each entry key's place in a profile's saved manual order. */
export function manualRank(
  order: readonly string[] | null | undefined,
): ReadonlyMap<string, number> {
  return new Map((order ?? []).map((key, i) => [key, i]))
}

/** Listed entries first in their saved order; the rest tie, so the caller's name order places them. */
export function compareManual(a: ListRow, b: ListRow, rank: ReadonlyMap<string, number>): number {
  const ai = rank.get(a.mod.key)
  const bi = rank.get(b.mod.key)
  if (ai === undefined && bi === undefined) {
    return 0
  }
  if (ai === undefined) {
    return 1
  }
  return bi === undefined ? -1 : ai - bi
}

/** keys with `from` moved to where `to` is, as a drop of one row onto another does. */
export function moveKey(keys: readonly string[], from: string, to: string): string[] {
  const i = keys.indexOf(from)
  const j = keys.indexOf(to)
  if (i < 0 || j < 0 || i === j) {
    return [...keys]
  }
  const next = keys.filter((k) => k !== from)
  next.splice(j, 0, from)
  return next
}

/** The keys of the entries in the order the list shows them, each once. */
export function displayedKeys(groups: readonly { items: readonly ListRow[] }[]): string[] {
  return [...new Set(groups.flatMap((g) => g.items.map((r) => r.mod.key)))]
}

/**
 * The order to save after dropping the entry `from` on the entry `to`: the displayed keys with that move made, then
 * the saved keys the list does not show (hidden implicit mods) in their saved order.
 */
export function droppedOrder(
  displayed: readonly string[],
  saved: readonly string[] | null | undefined,
  from: string,
  to: string,
): string[] {
  const moved = moveKey(displayed, from, to)
  const shown = new Set(moved)
  return [...moved, ...(saved ?? []).filter((k) => !shown.has(k))]
}
