import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { reportError } from '../toasts/report.ts'

export function persist(run: () => Promise<void>, title: string) {
  run().catch(reportError(title))
}

// Asks for a folder and stores the choice under a setting key; cancelling changes nothing.
export function pickFolderSetting(prompt: string, key: string, game: string, title: string) {
  persist(async () => {
    const dir = await PickFolder(prompt)
    if (dir) {
      await SetByKey(key, dir, game)
    }
  }, title)
}
