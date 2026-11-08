import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useMods } from '../mods/store.ts'
import { useToasts } from '../toasts/store.ts'

/** Switches a mod off and offers an Undo that restores it with everything the switch took with it. */
export async function disableWithUndo(i18n: I18n, mod: Mod): Promise<void> {
  const off = await useMods.getState().setEnabled(mod, false)
  if (off.length === 0) {
    return
  }
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Disabled ${{ names: mod.name }}`),
    action: {
      label: i18n._(msg`Undo`),
      run: () => useMods.getState().setEnabledMany(off, true),
    },
  })
}
