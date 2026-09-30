import { Events } from '@wailsio/runtime'
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import { Inbox } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { openImport } from './store.ts'

const LINK = 'link'

// A link or .mortar file that reached the app from outside only fills the import dialog: the user sees the preview
// and decides. From a game screen, the open profile is offered as the place to add it.
export async function initShare(): Promise<void> {
  const seen = new Set<number>()
  const arrive = (a: Arrival) => {
    if (seen.has(a.id)) {
      return
    }
    seen.add(a.id)
    const inGame = useNav.getState().route.name === 'game'
    const profileId = inGame ? useProfiles.getState().openId : ''
    openImport(a.kind === LINK ? { profileId, link: a.value } : { profileId, file: a.value })
  }
  Events.On('share:arrived', (e) => arrive(e.data))
  for (const a of (await Inbox()) ?? []) {
    arrive(a)
  }
}
