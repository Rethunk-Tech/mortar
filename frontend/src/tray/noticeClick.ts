import { Events } from '@wailsio/runtime'
import type { NoticeClick } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useTab } from '../game/tab.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'

export function initTrayNoticeClick() {
  Events.On('tray:notice', (event) => {
    const { game, profile } = event.data as NoticeClick
    if (!isGameId(game) || profile === '') {
      return
    }
    useNav.getState().openGame(game)
    useProfiles.getState().open(profile)
    useTab.getState().setTab('console')
  })
}
