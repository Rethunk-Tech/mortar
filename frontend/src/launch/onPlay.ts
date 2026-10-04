import { Window } from '@wailsio/runtime'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useSettings } from '../settings/store.ts'
import { gameBusy } from './busy.ts'

let tucked = false

export function applyOnPlayWindow(next: Status, prev: Status | null) {
  const onPlay = useSettings.getState().onPlay || 'stay'
  const playing = gameBusy(next)
  if (playing) {
    if (tucked || onPlay === 'stay') {
      return
    }
    tucked = true
    if (onPlay === 'minimise') {
      Window.Minimise().catch(() => undefined)
    } else if (onPlay === 'hide') {
      Window.Hide().catch(() => undefined)
    }
    return
  }
  if (tucked && next.state === State.Idle && prev && gameBusy(prev)) {
    tucked = false
    Window.Show().catch(() => undefined)
    Window.Restore().catch(() => undefined)
  }
}
