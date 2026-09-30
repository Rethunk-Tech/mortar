import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import {
  type Crash,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  Status as LaunchStatus,
  Start,
  StartVanilla,
  Stop,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Broken } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { UpdateWarning } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function resetConsole(status: Status, prev: Status | null) {
  const same =
    prev?.state === State.Launching && prev.game === status.game && prev.profile === status.profile
  if (same) {
    return false
  }
  const keep =
    status.profile === '' && prev?.game === status.game
      ? (useConsole.getState().shown.profile ?? '')
      : status.profile
  useConsole.getState().reset(status.game, keep || status.profile)
  return true
}

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

export interface UpdateWarn {
  game: string
  profile: string
  direct: boolean
  recorded: string
  installed: string
  broken: Broken[]
}

export const useLaunch = create<{
  status: Status | null
  hidden: boolean
  failure: Failure | null
  askDirect: { game: string; profile: string } | null
  updateWarn: UpdateWarn | null
  stopping: boolean
  // Play was pressed and no launch:state has answered yet, which is when SMAPI installs first.
  starting: boolean
  crash: Crash | null
  // polled marks a status read by refresh() rather than announced by a launch:state event.
  apply: (status: Status, polled?: boolean) => void
  refresh: (game: string) => Promise<void>
  start: (game: string, profile: string, direct: boolean) => Promise<void>
  startVanilla: (game: string, direct: boolean) => Promise<void>
  hide: () => void
  dismissFailure: () => void
  dismissCrash: () => void
  setCrash: (crash: Crash) => void
  dismissUpdateWarn: () => void
  answerDirect: (agreed: boolean) => Promise<void>
  playAnyway: () => Promise<void>
  openProblems: () => void
  stop: (game: string) => Promise<void>
}>((set, get) => ({
  status: null,
  hidden: false,
  failure: null,
  askDirect: null,
  updateWarn: null,
  stopping: false,
  starting: false,
  crash: null,
  apply: (status, polled = false) => {
    // A poll that lands before the first launch:state still reports Idle; only an event ends preparation.
    if (!polled || status.state !== State.Idle) {
      set({ starting: false })
    }
    if (status.state === State.Launching) {
      if (resetConsole(status, get().status)) {
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
    try {
      const warning = await UpdateWarning(game, profile)
      if (warning.changed) {
        set({
          updateWarn: {
            game,
            profile,
            direct,
            recorded: warning.recorded,
            installed: warning.installed,
            broken: warning.broken ?? [],
          },
        })
        return
      }
    } catch (e) {
      reportError(i18n._(msg`Could not check the game version`))(e)
      return
    }
    set({ starting: true })
    try {
      await Start(game, profile, direct)
    } catch (e) {
      set({ starting: false })
      reportError(i18n._(msg`Could not launch the game`))(e)
    }
  },
  startVanilla: async (game, direct) => {
    set({ starting: true })
    try {
      await StartVanilla(game, direct)
    } catch (e) {
      set({ starting: false })
      reportError(i18n._(msg`Could not launch the game`))(e)
    }
  },
  hide: () => set({ hidden: true }),
  dismissFailure: () => set({ failure: null }),
  dismissCrash: () => set({ crash: null }),
  setCrash: (crash) => set({ crash }),
  dismissUpdateWarn: () => set({ updateWarn: null }),
  playAnyway: async () => {
    const warn = get().updateWarn
    set({ updateWarn: null })
    if (!warn) {
      return
    }
    set({ starting: true })
    try {
      await Start(warn.game, warn.profile, warn.direct)
    } catch (e) {
      set({ starting: false })
      reportError(i18n._(msg`Could not launch the game`))(e)
    }
  },
  openProblems: () => {
    const warn = get().updateWarn
    set({ updateWarn: null })
    if (!warn) {
      return
    }
    if (warn.game === 'stardew') {
      useNav.getState().openGame('stardew')
    }
    useProfiles.getState().open(warn.profile)
    useTab.getState().setTab('mods')
  },
  answerDirect: async (agreed) => {
    const { askDirect } = get()
    set({ askDirect: null })
    if (agreed && askDirect) {
      if (askDirect.profile === '') {
        await get().startVanilla(askDirect.game, true)
      } else {
        await get().start(askDirect.game, askDirect.profile, true)
      }
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
  Events.On('launch:crash', (event) => useLaunch.getState().setCrash(event.data))
}

export const overlayGame = (routeName: string, routeGame: string, statusGame: string) =>
  routeName === 'game' ? routeGame : statusGame
