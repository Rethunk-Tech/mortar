import { Window } from '@wailsio/runtime'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useSettings } from '../settings/store.ts'

let tucked = false

export function applyOnPlayWindow(next: Status, prev: Status | null) {
  const onPlay = useSettings.getState().onPlay || 'stay'
  const playing = next.state === State.Launching || next.state === State.Running
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
  if (
    tucked &&
    next.state === State.Idle &&
    prev &&
    (prev.state === State.Running || prev.state === State.Launching)
  ) {
    tucked = false
    Window.Show().catch(() => undefined)
    Window.Restore().catch(() => undefined)
  }
}
