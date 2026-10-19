import { idKey, requiredIdsOf } from './dependents.ts'
import type { ListRow } from './listColumns.ts'

/**
 * Each group's implicit mods placed beneath the first mod in the group that requires them, deeper for what those
 * require in turn. An implicit mod nothing in its group requires keeps its place, with its own requirements beneath it.
 */
export function nestImplicit<G extends { items: readonly ListRow[] }>(groups: readonly G[]): G[] {
  return groups.map((group) => {
    const byId = new Map(group.items.map((r) => [idKey(r.mod.id), r]))
    const children = new Map<ListRow, ListRow[]>()
    const claimed = new Set<ListRow>()
    const claim = (parent: ListRow) => {
      for (const id of requiredIdsOf(parent.mod)) {
        const child = byId.get(idKey(id))
        if (child && child !== parent && child.implicit && !claimed.has(child)) {
          claimed.add(child)
          children.set(parent, [...(children.get(parent) ?? []), child])
          claim(child)
        }
      }
    }
    for (const r of group.items) {
      if (!r.implicit) {
        claim(r)
      }
    }
    for (const r of group.items) {
      if (r.implicit && !claimed.has(r)) {
        claim(r)
      }
    }
    const items: ListRow[] = []
    const emit = (r: ListRow, depth: number) => {
      items.push(depth > 0 ? { ...r, depth } : r)
      for (const child of children.get(r) ?? []) {
        emit(child, depth + 1)
      }
    }
    for (const r of group.items) {
      if (!claimed.has(r)) {
        emit(r, 0)
      }
    }
    return { ...group, items }
  })
}

/**
 * The nested groups for the card view: each nested implicit mod loses its place and rides on the nearest earlier mod
 * above it, in that mod's `libraries`.
 */
export function foldLibraries<G extends { items: readonly ListRow[] }>(groups: readonly G[]): G[] {
  return groups.map((group) => {
    const items: ListRow[] = []
    const libraries = new Map<ListRow, ListRow[]>()
    let owner: ListRow | undefined
    for (const r of group.items) {
      if (r.depth && owner) {
        libraries.set(owner, [...(libraries.get(owner) ?? []), r])
      } else {
        owner = r
        items.push(r)
      }
    }
    return {
      ...group,
      items: items.map((r) => (libraries.has(r) ? { ...r, libraries: libraries.get(r) } : r)),
    }
  })
}
