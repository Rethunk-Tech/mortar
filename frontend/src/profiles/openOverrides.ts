import { foldedOverrides, resolveOverride } from './overrideValue.ts'
import { useProfiles } from './store.ts'

type ProfilesState = ReturnType<typeof useProfiles.getState>

const overridesOf = (s: ProfilesState) => {
  const open = s.profiles.find((p) => p.id === s.openId)
  return open ? foldedOverrides(open) : undefined
}

/** A per-game setting as the open profile sees it: its own override when it has one, else the game's value. */
export function openOverride(key: string, gameValue: string): string {
  return resolveOverride(key, gameValue, overridesOf(useProfiles.getState()))
}

/** openOverride for a component, re-read when the open profile or its overrides change. */
export function useOpenOverride(key: string, gameValue: string): string {
  const own = useProfiles((s) => overridesOf(s)?.[key])
  return own ?? gameValue
}
