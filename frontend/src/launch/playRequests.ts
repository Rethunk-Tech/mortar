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
  Events.On('play:requested', (event) => play(event.data))
  Take()
    .then((r) => {
      if (r) {
        play(r)
      }
    })
    .catch(reportUnexpected)
}
