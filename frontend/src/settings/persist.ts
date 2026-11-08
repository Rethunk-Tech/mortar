import { msg } from '@lingui/core/macro'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportError } from '../toasts/report.ts'

export function persist(run: () => Promise<void>, title: string) {
  run().catch(reportError(title))
}

// The one reporter for a failed settings write, read per failure so it follows a language change.
export function settingSaveFailed(e: unknown) {
  reportError(i18n._(msg`Could not save that setting`))(e)
}

export function saveSetting(run: () => Promise<void>) {
  run().catch(settingSaveFailed)
}

// Asks for a path, a folder or a program, and stores it under a setting key; cancelling changes nothing.
export function pickSetting(pick: () => Promise<string>, key: string, game: string) {
  saveSetting(async () => {
    const path = await pick()
    if (path) {
      await SetByKey(key, path, game)
    }
  })
}

export function pickFolderSetting(prompt: string, key: string, game: string) {
  pickSetting(() => PickFolder(prompt), key, game)
}
