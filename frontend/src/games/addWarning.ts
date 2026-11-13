import { msg } from '@lingui/core/macro'
import { BEPINEX, GOG } from '../brand/productNames.ts'
import { i18n } from '../i18n/index.ts'

/** Guess.Warning for the GOG edition of Graveyard Keeper, whose 32-bit build Mortar's BepInEx pack does not cover. */
const GOG_32BIT_BEPINEX = 'gog-32bit'

const gogStores = new Set(['gog', 'gog-heroic', 'gog-minigalaxy'])

/** The add-game warning for a Guess.Warning code; empty when the files raise nothing. */
export function addGameWarning(code: string): string {
  if (code === GOG_32BIT_BEPINEX) {
    return i18n._(
      msg`The ${{ store: GOG }} edition is 32-bit. Mortar's ${{ loader: BEPINEX }} pack does not cover it, so plugins will not load.`,
    )
  }
  return ''
}

/** The add-game warning for a catalog game's selected store; empty when that store is fine. */
export function storeAddWarning(id: string, store: string): string {
  if (id === 'graveyard-keeper' && gogStores.has(store)) {
    return addGameWarning(GOG_32BIT_BEPINEX)
  }
  return ''
}
