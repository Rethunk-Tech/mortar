import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import type { ModUpdateDigestNotice } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/models.ts'
import { useTab } from '../game/tab.ts'
import { updatesLabel } from '../i18n/counts.ts'
import { i18n } from '../i18n/index.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

function digestTitle(notice: ModUpdateDigestNotice): string {
  return i18n._(
    msg`${updatesLabel(notice.totalUpdates)} for ${plural(notice.profilesWith, { one: '# profile', other: '# profiles' })}`,
  )
}

function openDigestReview(notice: ModUpdateDigestNotice) {
  const { game } = notice
  if (!isGameId(game)) {
    return
  }
  if (notice.openProfiles) {
    useNav.getState().openGame(game)
    useNav.getState().openProfiles()
    return
  }
  if (notice.profile === '') {
    return
  }
  useNav.getState().openGame(game)
  useProfiles.getState().open(notice.profile)
  useMods.getState().showUpdates()
  useTab.getState().setTab('mods')
  useUpdates.getState().setReviewing(true)
}

let listening = false

function initUpdateDigestToast() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('updates:digest', (event) => {
    const notice = event.data as ModUpdateDigestNotice
    if (!notice || notice.totalUpdates <= 0) {
      return
    }
    useToasts.getState().push({
      kind: 'info',
      title: digestTitle(notice),
      action: {
        label: i18n._(msg`Review`),
        run: () => openDigestReview(notice),
      },
    })
  })
}

export { initUpdateDigestToast }
