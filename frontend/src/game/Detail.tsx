import { useEffect, useMemo } from 'react'
import { problemCount } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSaves } from '../saves/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ProfilesEmpty, ProfilesFailed, ProfilesHidden, ProfilesLoading } from './detailStates.tsx'
import { ProfileWorkspace } from './ProfileWorkspace.tsx'

export function Detail() {
  const profiles = useProfiles((s) => s.profiles)
  const loaded = useProfiles((s) => s.loaded)
  const failed = useProfiles((s) => s.failed)
  const problemsResult = useMods((s) => s.problems)
  const problemsTabCount = problemsResult === null ? null : problemCount(problemsResult)
  const game = useProfiles((s) => s.game?.id ?? '')
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const profile = useProfiles(openProfileOf)
  const loadSaves = useSaves((s) => s.load)
  const mods = useMods((s) => s.mods)
  const modState = useMemo(
    () => mods.map((mod) => `${mod.key}:${mod.uniqueId}:${mod.enabled}`).join('|'),
    [mods],
  )
  const profileId = profile?.id
  const saveEntries = (profile?.entries ?? [])
    .map((entry) => `${entry.key}:${(entry.mods ?? []).map((mod) => mod.uniqueId).join(',')}`)
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
  if (!profile && profiles.length > 0) {
    return <ProfilesHidden />
  }
  if (!profile) {
    return <ProfilesEmpty />
  }
  return (
    <ProfileWorkspace
      profile={profile}
      game={game}
      gameName={gameName}
      problemsTabCount={problemsTabCount}
    />
  )
}
