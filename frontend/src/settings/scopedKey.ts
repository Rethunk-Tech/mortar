import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { setOpenOverride } from '../profiles/openOverrides.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useProfileScope } from './useProfileScope.ts'

/** A game setting a profile can override, as a hand-made row on the game's settings page shows it: on the open
 * profile's scope its own value and a save to it, else the game's. own is the profile's value, when it has one.
 * overridable is the registry's ProfileOverridable for the key: one it does not mark has no profile copy, so the
 * game's value is what shows and what is saved. This is the one place that picks the override or the game's value;
 * PrefRow's useScopedPref passes the spec's mark and adds the spec's scope on top. */
export function useScopedKey(key: string, game: string, gameValue: string, overridable = true) {
  const profileScope = useProfileScope() && overridable
  const own = useOpenOverride(key, profileScope)
  return {
    profileScope,
    own,
    gameValue,
    value: own ?? gameValue,
    save: (value: string) =>
      profileScope && game !== '' ? setOpenOverride(game, key, value) : SetByKey(key, value, game),
    // A profile's copy lives under a game's profiles, so without one there is none to drop.
    reset: () => (game === '' ? Promise.resolve() : setOpenOverride(game, key, undefined)),
  }
}

/** The open profile's own value for key, read only while on. */
function useOpenOverride(key: string, on: boolean) {
  return useProfiles((s) => (on ? openProfileOf(s)?.overrides?.[key] : undefined))
}
