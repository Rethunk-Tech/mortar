import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'

export function playDirect(): boolean {
  return gamePrefs(useSettings.getState()).defaultLaunchMethod === 'direct'
}
