import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { NewDownloads } from '../../bindings/github.com/Rethunk-AI/mortar/internal/archivesvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { useInstall } from '../install/store.ts'
import { gameBusy } from '../launch/busy.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

async function offer(game: string) {
  const launch = useLaunch.getState()
  if (launch.starting || gameBusy(launch.status, game)) {
    return
  }
  const found = (await NewDownloads(game)) ?? []
  const [first] = found
  if (!first) {
    return
  }
  const { profiles, openId } = useProfiles.getState()
  const profile = profiles.find((p) => p.id === openId)
  const { push } = useToasts.getState()
  if (found.length === 1 && profile) {
    push({
      kind: 'info',
      title: i18n._(msg`Add ${first.name} to ${profile.name}?`),
      action: {
        label: i18n._(msg`Add`),
        run: () =>
          useInstall
            .getState()
            .install([first.path])
            .catch(reportError(i18n._(msg`Could not add the archive`))),
      },
    })
    return
  }
  push({
    kind: 'info',
    title: i18n._(
      msg`${plural(found.length, { one: '# new archive', other: '# new archives' })} in your downloads folder`,
    ),
    action: { label: i18n._(msg`Review`), run: openDownloadsDialog },
  })
}

let listening = false

/** Offers archives that land in the download folder while Mortar runs, one toast per batch. */
export function initNewDownloads() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('library:downloads', (event) => {
    offer(String(event.data)).catch(reportUnexpected)
  })
}
