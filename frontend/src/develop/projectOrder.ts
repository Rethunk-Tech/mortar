import { arrayMove } from '@dnd-kit/sortable'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { cmpText } from '../mods/cmpText.ts'

export type ProjectSort = 'manual' | 'name'

export const isProjectSort = (v: unknown): v is ProjectSort => v === 'manual' || v === 'name'

/** The stored order, or by name for the app's language; the sort is stable, so equal names keep the stored order. */
export function sortProjects(projects: DevProject[], sort: ProjectSort): DevProject[] {
  return sort === 'name' ? projects.toSorted((a, b) => cmpText(a.name, b.name)) : projects
}

/** The dirs after moving `from` onto the place of `to`; either missing leaves the order as it was. */
export function movedDirs(dirs: string[], from: string, to: string): string[] {
  const a = dirs.indexOf(from)
  const b = dirs.indexOf(to)
  return a < 0 || b < 0 || a === b ? dirs : arrayMove(dirs, a, b)
}

/** `all` with the listed projects in the given order, in the slots they already occupy (SetDevProjectOrder's rule). */
export function placeInSlots(all: DevProject[], dirs: string[]): DevProject[] {
  const byDir = new Map(all.map((p) => [p.dir, p]))
  const ordered = dirs.flatMap((d) => byDir.get(d) ?? [])
  const listed = new Set(dirs)
  let next = 0
  return all.map((p) => {
    const slot = listed.has(p.dir) ? ordered[next] : undefined
    if (slot === undefined) {
      return p
    }
    next += 1
    return slot
  })
}
