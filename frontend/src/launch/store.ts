import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  type Crash,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  LastRunIssues,
  Status as LaunchStatus,
  Runs,
  Stop,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { gameName } from '../games/info.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportError, reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type AutoUpdateRestorePoint, rollbackAutoUpdate } from './autoUpdate.ts'
import { gameBusy } from './busy.ts'
import { applyOnPlayWindow } from './onPlay.ts'
import {
  openProblems,
  type PlayCheck,
  playAnyway,
  type SaveWarn,
  setPendingPreset,
  startProfile,
  startWithWarning,
  type UpdateContext,
  type UpdateRollback,
  type UpdateWarn,
  updateAndPlay,
} from './playStart.ts'
import { startVanillaGame } from './vanillaStart.ts'

const RUN_POLL_ATTEMPTS = 20
const RUN_POLL_MS = 250
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
function failureBody(status: Status): string {
  const name = gameName(status.game)
  if (status.hint === Hint.HintSteam) {
    return i18n._(msg`Steam may not be running or signed in. Start Steam, sign in and try again.`)
  }
  if (status.hint === Hint.HintFlatpakFS) {
    return i18n._(
      msg`Flatpak Steam cannot read Mortar's mods folder. Grant the sandbox access (Settings › ${name}) or SMAPI will not see this profile.`,
    )
  }
  if (status.hint === Hint.HintLaunchOptions) {
    return i18n._(
      msg`Steam's launch options for ${name} lack the SMAPI line. In Steam, right-click the game, choose Properties, and paste this line into Launch Options.`,
    )
  }
  return status.error
}
interface Failure {
  profile: string
  body: string
  hint: Hint
  cause: Status['cause']
}

function rollbackAction(point: AutoUpdateRestorePoint) {
  return () =>
    rollbackAutoUpdate(point).catch((error) => {
      toastError(i18n._(msg`Could not roll back updates`), error)
    })
}

function profileName(rollback: UpdateRollback): string {
  return (
    useProfiles.getState().profiles.find((p) => p.id === rollback.profile)?.name ?? rollback.profile
  )
}

async function erroredMods(game: string, profile: string): Promise<string> {
  const issues = await LastRunIssues(game, profile).catch(() => null)
  const names = (issues?.mods ?? []).filter((m) => m.errors > 0).map((m) => m.name)
  return listNames(names)
}

async function checkUpdatedRun(
  rollback: UpdateRollback,
  get: () => { updateRollback: UpdateRollback | null },
  set: (p: { updateRollback: UpdateRollback | null }) => void,
) {
  const { previousRunId, previousErrors } = rollback.context
  if (previousErrors === null) {
    set({ updateRollback: null })
    return
  }
  for (let attempt = 0; attempt < RUN_POLL_ATTEMPTS; attempt += 1) {
    try {
      const latest = (await Runs(rollback.game, rollback.profile))?.[0]
      if (latest && latest.id !== previousRunId) {
        if (latest.errors > previousErrors) {
          useToasts.getState().push({
            kind: 'warning',
            title: i18n._(msg`New errors after updating ${profileName(rollback)}`),
            body: await erroredMods(rollback.game, rollback.profile),
            action: {
              label: i18n._(msg`Roll back`),
              run: rollbackAction(rollback.context.restorePoint),
            },
          })
          return
        }
        if (get().updateRollback === rollback) {
          set({ updateRollback: null })
        }
        return
      }
    } catch {
      return
    }
    await new Promise<void>((resolve) => globalThis.setTimeout(resolve, RUN_POLL_MS))
  }
}

interface DirectAsk {
  game: string
  profile: string
  update?: UpdateContext
}

function applyStatus(
  status: Status,
  polled: boolean,
  get: () => {
    status: Status | null
    hidden: boolean
    updateRollback: UpdateRollback | null
  },
  set: (p: {
    starting?: boolean
    startingProfile?: string
    hidden?: boolean
    failure?: Failure | null
    askDirect?: DirectAsk | null
    status?: Status
    updateRollback?: UpdateRollback | null
  }) => void,
) {
  const previous = get().status
  applyOnPlayWindow(status, previous)
  if (!polled || status.state !== State.Idle) {
    set({ starting: false, startingProfile: '' })
    if (status.state !== State.Launching && status.state !== State.NoSteam) {
      setPendingPreset('')
    }
  }
  if (status.state === State.Launching) {
    if (resetConsole(status, previous)) {
      set({ hidden: false, failure: null })
    }
    set({ status })
    return
  }
  if (status.state === State.Failed) {
    set({
      failure: {
        profile: status.profile,
        body: failureBody(status),
        hint: status.hint,
        cause: status.cause,
      },
    })
    set({ updateRollback: null })
  }
  if (status.state === State.NoSteam) {
    const rollback = get().updateRollback
    set({
      askDirect: {
        game: status.game,
        profile: status.profile,
        ...(rollback ? { update: rollback.context } : {}),
      },
    })
  }
  if (status.state === State.Running || status.state === State.Idle) {
    if (status.state === State.Running && previous?.state === State.Launching && !get().hidden) {
      useTab.getState().setTab('console')
    }
    set({ status })
    const rollback = get().updateRollback
    if (
      status.state === State.Idle &&
      previous?.state === State.Running &&
      rollback &&
      previous.game === rollback.game &&
      previous.profile === rollback.profile
    ) {
      checkUpdatedRun(rollback, get, set).catch(reportUnexpected)
    }
  } else {
    set({ status: { ...status, state: State.Idle } })
  }
}

export const useLaunch = create<{
  status: Status | null
  hidden: boolean
  failure: Failure | null
  askDirect: DirectAsk | null
  updateWarn: UpdateWarn | null
  saveWarn: SaveWarn | null
  playCheck: PlayCheck | null
  updateRollback: UpdateRollback | null
  updating: number
  stopping: boolean
  starting: boolean
  startingProfile: string
  crash: Crash | null
  apply: (status: Status, polled?: boolean) => void
  refresh: (game: string) => Promise<void>
  /** preset names the launch preset for this Play only; omitted keeps the one a resumed launch already carries. */
  start: (game: string, profile: string, direct: boolean, preset?: string) => Promise<void>
  startVanilla: (game: string, direct: boolean) => Promise<void>
  hide: () => void
  dismissFailure: () => void
  dismissCrash: () => void
  setCrash: (crash: Crash) => void
  dismissUpdateWarn: () => void
  dismissSaveWarn: () => void
  dismissPlayCheck: () => void
  openSaves: () => void
  answerDirect: (agreed: boolean) => Promise<void>
  playAnyway: () => Promise<void>
  updateAndPlay: () => Promise<void>
  openProblems: () => void
  stop: (game: string) => Promise<void>
}>((set, get) => ({
  status: null,
  hidden: false,
  failure: null,
  askDirect: null,
  updateWarn: null,
  saveWarn: null,
  playCheck: null,
  updateRollback: null,
  updating: 0,
  stopping: false,
  starting: false,
  startingProfile: '',
  crash: null,
  apply: (status, polled = false) => applyStatus(status, polled, get, set),
  refresh: async (game) => {
    try {
      get().apply(await LaunchStatus(game), true)
    } catch (e) {
      reportError(i18n._(msg`Could not check whether the game is running`))(e)
    }
  },
  start: (game, profile, direct, preset) => {
    if (preset !== undefined) {
      setPendingPreset(preset)
    }
    return startWithWarning({ get, set, game, profile, direct })
  },
  startVanilla: (game, direct) => startVanillaGame({ get, set, game, direct }),
  hide: () => set({ hidden: true }),
  dismissFailure: () => set({ failure: null }),
  dismissCrash: () => set({ crash: null }),
  setCrash: (crash) => set({ crash }),
  dismissUpdateWarn: () => {
    setPendingPreset('')
    set({ updateWarn: null })
  },
  dismissSaveWarn: () => {
    setPendingPreset('')
    set({ saveWarn: null })
  },
  dismissPlayCheck: () => {
    setPendingPreset('')
    set({ playCheck: null })
  },
  openSaves: () => {
    const warn = get().saveWarn
    set({ saveWarn: null })
    if (!warn) {
      return
    }
    if (isGameId(warn.game)) {
      useNav.getState().openGame(warn.game)
    }
    useProfiles.getState().open(warn.profile)
    useTab.getState().setTab('saves')
  },
  playAnyway: () => playAnyway(get, set),
  updateAndPlay: () => updateAndPlay(get, set),
  openProblems: () => openProblems(get, set),
  answerDirect: async (agreed) => {
    const { askDirect } = get()
    set({ askDirect: null })
    if (!agreed) {
      setPendingPreset('')
      set({ updateRollback: null })
      return
    }
    if (askDirect) {
      if (askDirect.profile === '') {
        await get().startVanilla(askDirect.game, true)
      } else if (askDirect.update) {
        set({ starting: true, startingProfile: askDirect.profile })
        await startProfile({
          set,
          game: askDirect.game,
          profile: askDirect.profile,
          direct: true,
          update: askDirect.update,
        })
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

// Also true between the Play click and the first status, which gameBusy cannot see.
export const useGameBusy = (game?: string) =>
  useLaunch((s) => s.starting || gameBusy(s.status, game))

export const overlayGame = (routeName: string, routeGame: string, statusGame: string) =>
  routeName === 'game' ? routeGame : statusGame
