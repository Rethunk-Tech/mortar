import {
  HidePrompt,
  LeavePlayMode,
  PlayMode,
  ShowPrompt,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { ConfirmQuit } from '../../bindings/github.com/Rethunk-Tech/mortar/quitservice.ts'
import { useRoomy } from '../theme/roomy.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { gameBusy } from './busy.ts'
import { usePlayMode } from './playModeState.ts'
import { useLaunch } from './store.ts'

// The window stays hidden while the game starts and runs, a blocked Play shows only its dialog in a small window, and
// the Go side exits when the game closes. Cancel in that dialog quits; any other way out of it but playing (Open
// Problems, Open Saves, Switch profile) is a fix, which brings up the full window as a normal session.

type Launch = ReturnType<typeof useLaunch.getState>

const blocked = (s: Launch) =>
  s.playCheck !== null ||
  s.updateWarn !== null ||
  s.saveWarn !== null ||
  s.askDirect !== null ||
  s.failure !== null

function leave() {
  usePlayMode.setState({ solo: false })
  LeavePlayMode().catch(reportUnexpected)
}

// Runs after the action that closed the dialog has had its synchronous turn, by which Play anyway has set starting.
function afterDialog() {
  const s = useLaunch.getState()
  if (!usePlayMode.getState().solo || blocked(s)) {
    return
  }
  if (usePlayMode.getState().cancelled) {
    ConfirmQuit().catch(reportUnexpected)
  } else if (s.starting || gameBusy(s.status)) {
    HidePrompt().catch(reportUnexpected)
  } else {
    leave()
  }
}

async function initPlayMode() {
  if (!(await PlayMode())) {
    return
  }
  usePlayMode.setState({ solo: true })
  useRoomy.setState({ roomy: true })
  useLaunch.subscribe((now, before) => {
    if (!usePlayMode.getState().solo || blocked(now) === blocked(before)) {
      return
    }
    if (blocked(now)) {
      ShowPrompt().catch(reportUnexpected)
    } else {
      setTimeout(afterDialog, 0)
    }
  })
  // An error toast in a hidden window would go unseen, so it brings up the full window.
  useToasts.subscribe((now, before) => {
    if (
      usePlayMode.getState().solo &&
      now.toasts.some((t) => t.kind === 'error' && !before.toasts.includes(t))
    ) {
      leave()
    }
  })
}

export { initPlayMode }
