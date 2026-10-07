import { useEffect, useMemo } from 'react'
import { useMods } from '../mods/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSaves } from '../saves/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ProfilesEmpty, ProfilesFailed, ProfilesLoading } from './detailStates.tsx'
import { ProfileWorkspace } from './ProfileWorkspace.tsx'

export function Detail() {
  const loaded = useProfiles((s) => s.loaded)
  const failed = useProfiles((s) => s.failed)
  const game = useProfiles((s) => s.game?.id ?? '')
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const profile = useProfiles(openProfileOf)
  const loadSaves = useSaves((s) => s.load)
  const mods = useMods((s) => s.mods)
  const modState = useMemo(
    () => mods.map((mod) => `${mod.key}:${mod.id}:${mod.enabled}`).join('|'),
    [mods],
  )
  const profileId = profile?.id
  const saveEntries = (profile?.entries ?? [])
    .map((entry) => `${entry.key}:${(entry.mods ?? []).map((mod) => mod.id).join(',')}`)
    .join('|')
  const saveKey = `${saveEntries}|${modState}`
  useEffect(() => {
    if (game && profileId) {
      loadSaves(game, profileId, saveKey).catch(reportUnexpected)
    }
  }, [game, profileId, saveKey, loadSaves])
  if (!loaded) {
    return failed ? <ProfilesFailed /> : <ProfilesLoading />
  }
  if (!profile) {
    return <ProfilesEmpty />
  }
  return <ProfileWorkspace profile={profile} game={game} gameName={gameName} />
}
