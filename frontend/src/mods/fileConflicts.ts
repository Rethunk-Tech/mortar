import { useEffect, useRef } from 'react'
import type { FileConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { FileConflicts } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
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

const NO_GROUPS: ConflictGroup[] = []

/** The open profile's package file conflicts, read again whenever the profile changes. */
export function useFileConflicts(): ConflictGroup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const openId = useProfiles((s) => openProfileOf(s)?.id ?? '')
  const updated = useProfiles((s) => openProfileOf(s)?.updated)
  const { data, reload } = useLoaded(
    game && openId
      ? () => FileConflicts(game, openId).then((got) => groupConflicts(got ?? []))
      : null,
    [game, openId],
    NO_GROUPS,
    reportUnexpected,
  )
  // An edit to the same profile reads again but keeps the list shown until the answer, where new keys would blank it.
  const seen = useRef({ openId, updated })
  useEffect(() => {
    const was = seen.current
    seen.current = { openId, updated }
    if (was.openId === openId && was.updated !== updated) {
      reload()
    }
  }, [openId, updated, reload])
  return data
}
