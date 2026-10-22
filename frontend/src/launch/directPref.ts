import { openOverride, useOpenOverride } from '../profiles/openOverrides.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'

export function playDirect(): boolean {
  return (
    openOverride('defaultLaunchMethod', gamePrefs(useSettings.getState()).defaultLaunchMethod) ===
    'direct'
  )
}

// playDirect for a component: it follows the game's launch method and the open profile's override as they change.
export function usePlayDirect(): boolean {
  const method = useSettings((s) => gamePrefs(s).defaultLaunchMethod)
  return useOpenOverride('defaultLaunchMethod', method) === 'direct'
}
