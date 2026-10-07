import { msg, plural } from '@lingui/core/macro'
import { RestoreDialog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useProfiles } from './store.ts'

// Makes a new profile from a backup file or an older profile zip; mods the file lacks download again.
export async function restoreBackup(): Promise<void> {
  const { game } = useProfiles.getState()
  if (!game) {
    return
  }
  try {
    const r = await RestoreDialog(game.id)
    if (!r.profile) {
      return
    }
    await useProfiles.getState().refresh()
    const missing = r.unavailable ?? []
    let body = i18n._(msg`Every mod is in place.`)
    if (missing.length > 0) {
      body = i18n._(msg`Install these again by hand: ${listNames(missing)}`)
    } else if (r.queued > 0) {
      body = plural(r.queued, { one: '# mod is downloading.', other: '# mods are downloading.' })
    }
    useToasts.getState().push({
      kind: missing.length > 0 ? 'warning' : 'success',
      title: i18n._(msg`Restored as ${r.name}`),
      body,
    })
  } catch (e) {
    reportError(i18n._(msg`Could not restore the backup`))(e)
  }
}
