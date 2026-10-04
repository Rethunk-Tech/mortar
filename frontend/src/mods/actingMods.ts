import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { modId } from './lookup.ts'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'

// The mods an action on `mod` covers: the whole selection when `mod` is one of 2+ selected, else just `mod`.
export function actingMods(mod: Mod): Mod[] {
  const { ids } = useSelection.getState()
  if (ids.length < 2 || !ids.includes(modId(mod))) {
    return [mod]
  }
  return useMods.getState().mods.filter((m) => ids.includes(modId(m)))
}

export function toggleActing(mod: Mod): Promise<void> {
  const { setEnabled, setEnabledMany } = useMods.getState()
  const mods = actingMods(mod)
  return mods.length > 1 ? setEnabledMany(mods, !mod.enabled) : setEnabled(mod, !mod.enabled)
}
