import { SetOverrides } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
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

/** Saves a per-game setting where the open profile reads it: its own override when it has one, else the game's. */
export async function saveOpenPref(game: string, key: string, value: string): Promise<void> {
  const s = useProfiles.getState()
  if (overridesOf(s)?.[key] === undefined) {
    await SetByKey(key, value, game)
    return
  }
  const open = s.profiles.find((p) => p.id === s.openId)
  if (open) {
    s.replace(await SetOverrides(game, open.id, { ...foldedOverrides(open), [key]: value }))
  }
}
