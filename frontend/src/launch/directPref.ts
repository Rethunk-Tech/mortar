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

// playDirect for a component: it follows the game's launch method and the open profile's override as they change.
export function usePlayDirect(): boolean {
  const method = useSettings((s) => gamePrefs(s).defaultLaunchMethod)
  const overrides = useProfiles((s) => s.profiles.find((p) => p.id === s.openId)?.overrides)
  return (
    resolveOverride(
      'defaultLaunchMethod',
      method,
      overrides ? foldedOverrides({ overrides }) : undefined,
    ) === 'direct'
  )
}
