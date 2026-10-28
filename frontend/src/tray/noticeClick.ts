import { Events } from '@wailsio/runtime'
import type { NoticeClick } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useTab } from '../game/tab.ts'
import { useMods } from '../mods/store.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function initTrayNoticeClick() {
  Events.On('tray:notice', (event) => {
    showNotice(event.data as NoticeClick).catch(reportUnexpected)
  })
}

async function showNotice({ game, profile, tab }: NoticeClick) {
  if (!isGameId(game) || profile === '') {
    return
  }
  useNav.getState().openGame(game)
  if (useProfiles.getState().game?.id !== game) {
    await useProfiles.getState().load(game)
  }
  useProfiles.getState().open(profile)
  if (tab === 'updates') {
    useMods.getState().showUpdates()
    useTab.getState().setTab('mods')
    return
  }
  useTab.getState().setTab('console')
}
