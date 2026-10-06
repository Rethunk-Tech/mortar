import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

const lcSlot = /^LCSaveFile(\d+)$/

// A save's display name: its farm, else Lethal Company's slot (its saves are files named by slot), else its folder.
export function saveName(save: { farm: string; folder: string }): string {
  if (save.farm) {
    return save.farm
  }
  const slot = lcSlot.exec(save.folder)?.[1]
  if (slot !== undefined) {
    return i18n._(msg`Save file ${slot}`)
  }
  if (save.folder === 'LCChallengeFile') {
    return i18n._(msg`Challenge moon`)
  }
  return save.folder
}
