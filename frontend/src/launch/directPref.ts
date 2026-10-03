import { useSettings } from '../settings/store.ts'

export function playDirect(): boolean {
  return useSettings.getState().defaultLaunchMethod === 'direct'
}
