import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  Status as LaunchStatus,
  Start,
  Stop,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const reportError = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })
}

function failureBody(status: Status): string {
  if (status.hint === Hint.HintSteam) {
    return i18n._(msg`Steam may not be running or signed in. Start Steam, sign in and try again.`)
  }
  if (status.hint === Hint.HintLaunchOptions) {
    return i18n._(
      msg`Steam's launch options for Stardew Valley lack the SMAPI line. In Steam, right-click the game, choose Properties, and paste this line into Launch Options.`,
    )
  }
  return status.error
}

interface Failure {
  profile: string
  body: string
  hint: Hint
}

export const useLaunch = create<{
  status: Status | null
  hidden: boolean
  failure: Failure | null
  askDirect: { game: string; profile: string } | null
  stopping: boolean
  // Play was pressed and no launch:state has answered yet, which is when SMAPI installs first.
  starting: boolean
  // polled marks a status read by refresh() rather than announced by a launch:state event.
  apply: (status: Status, polled?: boolean) => void
  refresh: (game: string) => Promise<void>
  start: (game: string, profile: string, direct: boolean) => Promise<void>
  hide: () => void
  dismissFailure: () => void
  answerDirect: (agreed: boolean) => Promise<void>
  stop: (game: string) => Promise<void>
}>((set, get) => ({
  status: null,
  hidden: false,
  failure: null,
  askDirect: null,
  stopping: false,
  starting: false,
  apply: (status, polled = false) => {
    // A poll that lands before the first launch:state still reports Idle; only an event ends preparation.
    if (!polled || status.state !== State.Idle) {
      set({ starting: false })
    }
    if (status.state === State.Launching) {
      // A status refresh during the same launch must keep its log and a hidden overlay hidden.
      const prev = get().status
      const same =
        prev?.state === State.Launching &&
        prev.game === status.game &&
        prev.profile === status.profile
      if (!same) {
        useConsole.getState().reset(status.game, status.profile)
        set({ hidden: false, failure: null })
      }
      set({ status })
      return
    }
    if (status.state === State.Failed) {
      set({ failure: { profile: status.profile, body: failureBody(status), hint: status.hint } })
    }
    if (status.state === State.NoSteam) {
      set({ askDirect: { game: status.game, profile: status.profile } })
    }
    // Failed and NoSteam are one-off announcements; the game itself is idle.
    if (status.state === State.Running || status.state === State.Idle) {
      if (
        status.state === State.Running &&
        get().status?.state === State.Launching &&
        !get().hidden
      ) {
        useTab.getState().setTab('console')
      }
      set({ status })
    } else {
      set({ status: { ...status, state: State.Idle } })
    }
  },
  refresh: async (game) => {
    try {
      get().apply(await LaunchStatus(game), true)
    } catch (e) {
      reportError(i18n._(msg`Could not check whether the game is running`))(e)
    }
  },
  start: async (game, profile, direct) => {
    set({ starting: true })
    try {
      await Start(game, profile, direct)
    } catch (e) {
      set({ starting: false })
      reportError(i18n._(msg`Could not launch the game`))(e)
    }
  },
  hide: () => set({ hidden: true }),
  dismissFailure: () => set({ failure: null }),
  answerDirect: async (agreed) => {
    const { askDirect } = get()
    set({ askDirect: null })
    if (agreed && askDirect) {
      await get().start(askDirect.game, askDirect.profile, true)
    }
  },
  stop: async (game) => {
    set({ stopping: true })
    try {
      await Stop(game)
    } catch (e) {
      reportError(i18n._(msg`Could not stop the game`))(e)
    } finally {
      set({ stopping: false })
    }
  },
}))

export function initLaunch() {
  Events.On('launch:state', (event) => useLaunch.getState().apply(event.data))
  Events.On('launch:line', (event) => useConsole.getState().add(event.data))
}
