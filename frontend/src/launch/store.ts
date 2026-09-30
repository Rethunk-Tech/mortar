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
import { useLoader } from '../loader/store.ts'
import { useToasts } from '../toasts/store.ts'

const reportError = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: String(e) })
}

function failureBody(status: Status): string {
  if (status.hint === Hint.HintSteam) {
    return i18n._(msg`Steam may not be running or signed in. Start Steam, sign in and try again.`)
  }
  if (status.hint === Hint.HintLaunchOptions) {
    return i18n._(
      msg`Steam's launch options for Stardew Valley lack the SMAPI line. Set them to "<game folder>\\StardewModdingAPI.exe" %command%.`,
    )
  }
  return status.error
}

function notify(status: Status) {
  const { push } = useToasts.getState()
  if (status.state === State.NeedsLoader) {
    push({
      kind: 'warning',
      title: i18n._(msg`SMAPI is missing or was replaced by a game update`),
      action: {
        label: i18n._(msg`Reinstall`),
        run: () => {
          useLoader
            .getState()
            .install(status.game)
            .catch(reportError(i18n._(msg`Could not install SMAPI`)))
        },
      },
    })
  }
}

interface Failure {
  profile: string
  body: string
}

export const useLaunch = create<{
  status: Status | null
  hidden: boolean
  failure: Failure | null
  askDirect: { game: string; profile: string } | null
  stopping: boolean
  apply: (status: Status) => void
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
  apply: (status) => {
    if (status.state === State.Launching) {
      useConsole.getState().reset()
      set({ status, hidden: false, failure: null })
      return
    }
    if (status.state === State.Failed) {
      set({ failure: { profile: status.profile, body: failureBody(status) } })
    }
    if (status.state === State.NoSteam) {
      set({ askDirect: { game: status.game, profile: status.profile } })
    }
    notify(status)
    // Failed, NoSteam and NeedsLoader are one-off announcements; the game itself is idle.
    if (status.state === State.Running || status.state === State.Idle) {
      if (status.state === State.Running && get().status?.state === State.Launching) {
        useTab.getState().setTab('console')
      }
      set({ status })
    } else {
      set({ status: { ...status, state: State.Idle } })
    }
  },
  refresh: async (game) => {
    try {
      get().apply(await LaunchStatus(game))
    } catch (e) {
      reportError(i18n._(msg`Could not check whether the game is running`))(e)
    }
  },
  start: async (game, profile, direct) => {
    try {
      await Start(game, profile, direct)
    } catch (e) {
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
  Events.On('launch:line', (event) => useConsole.getState().add(event.data.entries ?? []))
}
