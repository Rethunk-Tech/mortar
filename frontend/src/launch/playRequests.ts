import { Events } from '@wailsio/runtime'
import type { Request } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/models.ts'
import { Take } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { playDirect } from './directPref.ts'
import { useLaunch } from './store.ts'

// A desktop shortcut asks to play a profile: open it and play through the usual Play path, warnings included.
function play(r: Request) {
  if (!isGameId(r.game)) {
    return
  }
  useNav.getState().openGame(r.game)
  useProfiles.getState().open(r.profile)
  useLaunch.getState().start(r.game, r.profile, playDirect()).catch(reportUnexpected)
}

export function initPlayRequests() {
  if (typeof Events.On !== 'function') {
    return
  }
  const seen = new Set<number>()
  // A request that waits for the window also arrives as an event, and stays waiting until taken: taking it after
  // each event keeps a reload from playing it again. Requests without an id (the tray's) never wait.
  const arrive = (r: Request) => {
    if (r.id) {
      if (seen.has(r.id)) {
        return
      }
      seen.add(r.id)
    }
    play(r)
  }
  const drain = () =>
    Take()
      .then((r) => {
        if (r) {
          arrive(r)
        }
      })
      .catch(reportUnexpected)
  Events.On('play:requested', (event) => {
    arrive(event.data)
    drain()
  })
  drain()
}
