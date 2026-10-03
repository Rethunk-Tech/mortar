import { foldedOverrides, resolveOverride } from '../profiles/overrideValue.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'

export function playDirect(): boolean {
  const prefs = gamePrefs(useSettings.getState())
  const { openId, profiles } = useProfiles.getState()
  const profile = profiles.find((candidate) => candidate.id === openId)
  const method = resolveOverride(
    'defaultLaunchMethod',
    prefs.defaultLaunchMethod,
    profile ? foldedOverrides(profile) : undefined,
  )
  return method === 'direct'
}
