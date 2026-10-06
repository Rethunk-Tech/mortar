import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import type { ModUpdateDigestNotice } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/models.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

function digestTitle(notice: ModUpdateDigestNotice): string {
  return i18n._(
    msg`${plural(notice.totalUpdates, {
      one: `# update for ${plural(notice.profilesWith, { one: '# profile', other: '# profiles' })}`,
      other: `# updates for ${plural(notice.profilesWith, { one: '# profile', other: '# profiles' })}`,
    })}`,
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
  Events.On('updates:digest', (event) => showDigest(event.data as ModUpdateDigestNotice))
}

// The digest setting decides only how often a notice comes; the In Mortar switch decides whether it toasts.
function showDigest(notice: ModUpdateDigestNotice | null) {
  if (!notice || notice.totalUpdates <= 0 || !useSettings.getState().notifyModUpdates) {
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
}

export { initUpdateDigestToast, showDigest }
