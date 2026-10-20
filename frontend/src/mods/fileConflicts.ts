import { useEffect, useState } from 'react'
import type { FileConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { FileConflicts } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

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

/** The open profile's package file conflicts, read again whenever the profile changes. */
export function useFileConflicts(): ConflictGroup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const open = useProfiles(openProfileOf)
  const [groups, setGroups] = useState<ConflictGroup[]>([])
  useEffect(() => {
    if (!(game && open)) {
      setGroups([])
      return
    }
    FileConflicts(game, open.id)
      .then((got) => setGroups(groupConflicts(got ?? [])))
      .catch(reportUnexpected)
  }, [game, open])
  return groups
}
