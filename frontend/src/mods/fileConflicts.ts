import type { FileConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

/** The files one winner takes from the same set of mods, so a hundred shared files read as one row. */
export interface ConflictGroup {
  winner: string
  beaten: string[]
  paths: string[]
}

export function groupConflicts(conflicts: readonly FileConflict[]): ConflictGroup[] {
  const groups = new Map<string, ConflictGroup>()
  for (const c of conflicts) {
    const beaten = c.beaten ?? []
    const key = `${c.winner}\0${beaten.join('\0')}`
    const group = groups.get(key)
    if (group) {
      group.paths.push(c.path)
    } else {
      groups.set(key, { winner: c.winner, beaten: [...beaten], paths: [c.path] })
    }
  }
  return [...groups.values()].toSorted((a, b) => b.paths.length - a.paths.length)
}
