import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { setOpenOverride } from '../profiles/openOverrides.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSettingsScope } from './settingsScope.ts'

/** A game setting a profile can override, as a hand-made row on the game's settings page shows it: on the open
 * profile's scope its own value and a save to it, else the game's. own is the profile's value, when it has one. */
export function useScopedKey(key: string, game: string, gameValue: string) {
  const profileScope = useSettingsScope((s) => s.profile)
  const own = useProfiles((s) => (profileScope ? openProfileOf(s)?.overrides?.[key] : undefined))
  return {
    profileScope,
    own,
    gameValue,
    value: own ?? gameValue,
    save: (value: string) =>
      profileScope ? setOpenOverride(game, key, value) : SetByKey(key, value, game),
    reset: () => setOpenOverride(game, key, undefined),
  }
}
