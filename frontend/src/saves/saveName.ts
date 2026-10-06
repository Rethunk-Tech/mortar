import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

const lcSlot = /^LCSaveFile(\d+)$/
const extension = /\.[^.]+$/

// A save's display name: its farm, else Lethal Company's slot (its saves are files named by slot), else its folder or
// file name without the extension (a Valheim world is worlds_local/<name>.fwl).
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
  return (save.folder.split('/').at(-1) ?? '').replace(extension, '')
}

// What a Valheim save is, from the folder it sits in; empty for any other game's save.
export function saveKind(folder: string): string {
  if (folder.startsWith('characters_local/')) {
    return i18n._(msg`Character`)
  }
  if (folder.startsWith('worlds_local/')) {
    return i18n._(msg`World`)
  }
  return ''
}
