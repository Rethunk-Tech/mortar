import { msg } from '@lingui/core/macro'

/** The sentence for a control the running game holds; pass it to `t()` where the reason is conditional. */
export const lockedHint = msg`Stop the game to change mods.`

/** Names the profile the running game holds, for a control that can target more than one. */
export const lockedInHint = (name: string) =>
  msg`${{ name }} is in use by the running game. Close the game to change its mods.`
