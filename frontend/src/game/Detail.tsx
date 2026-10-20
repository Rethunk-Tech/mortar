import { useEffect, useMemo } from 'react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useMods } from '../mods/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSaves } from '../saves/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ProfilesEmpty, ProfilesFailed, ProfilesLoading } from './detailStates.tsx'
import { ProfileWorkspace } from './ProfileWorkspace.tsx'

const saveKeyOf = (entries: Entry[] | null | undefined) =>
  (entries ?? []).map((e) => `${e.key}:${(e.mods ?? []).map((mod) => mod.id).join(',')}`).join('|')

// Reloads the saves when the open profile's mods change. Its own component, so a mod switched on or off re-renders
// this and not the whole workspace.
function SavesFollower({ game }: { game: string }) {
  const loadSaves = useSaves((s) => s.load)
  const mods = useMods((s) => s.mods)
  const modState = useMemo(
    () => mods.map((mod) => `${mod.key}:${mod.id}:${mod.enabled}`).join('|'),
    [mods],
  )
  const id = useProfiles((s) => s.openId)
  // A string, so a save that changes no entry's mods leaves it equal.
  const saveEntries = useProfiles((s) => saveKeyOf(openProfileOf(s)?.entries))
  const saveKey = `${saveEntries}|${modState}`
  useEffect(() => {
    if (game) {
      loadSaves(game, id, saveKey).catch(reportUnexpected)
    }
  }, [game, id, saveKey, loadSaves])
  return null
}

export function Detail() {
  const loaded = useProfiles((s) => s.loaded)
  const failed = useProfiles((s) => s.failed)
  const game = useProfiles((s) => s.game?.id ?? '')
  const gameName = useProfiles((s) => s.game?.name ?? '')
  // Whether one is open, not the profile: a save of it re-renders only what reads the fields that changed.
  const open = useProfiles((s) => openProfileOf(s) !== undefined)
  if (!loaded) {
    return failed ? <ProfilesFailed /> : <ProfilesLoading />
  }
  if (!open) {
    return <ProfilesEmpty />
  }
  return (
    <>
      <SavesFollower game={game} />
      <ProfileWorkspace game={game} gameName={gameName} />
    </>
  )
}
