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
  Runs,
  Start,
  StartVanilla,
  Stop,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Broken } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { UpdateWarning } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  type AutoUpdateRestorePoint,
  type AutoUpdateResult,
  isAutoUpdateError,
  rollbackAutoUpdate,
  updateBeforePlay,
} from './autoUpdate.ts'

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

const reportError = (title: string) => (e: unknown) => {
  useToasts.getState().push({ kind: 'error', title, body: errorMessage(e) })
}

function failureBody(status: Status): string {
  if (status.hint === Hint.HintSteam) {
    return i18n._(msg`Steam may not be running or signed in. Start Steam, sign in and try again.`)
  }
  if (status.hint === Hint.HintFlatpakFS) {
    return i18n._(
      msg`Flatpak Steam cannot read Mortar's mods folder. Grant the sandbox access (Settings › Stardew Valley) or SMAPI will not see this profile.`,
    )
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

interface UpdateWarn {
  game: string
  profile: string
  direct: boolean
  recorded: string
  installed: string
  broken: Broken[]
  update?: UpdateContext
}

// SaveWarn is the newest save when it uses mods the profile lacks or has switched off.
interface SaveWarn {
  game: string
  profile: string
  direct: boolean
  save: Fit
  update?: UpdateContext
}

interface UpdateContext {
  restorePoint: AutoUpdateRestorePoint
  previousRunId: string
  previousErrors: number | null
}

interface UpdateRollback {
  context: UpdateContext
  game: string
  profile: string
}

function updateContext(result: AutoUpdateResult): UpdateContext | undefined {
  return result.restorePoint ? { ...result, restorePoint: result.restorePoint } : undefined
}

function rollbackAction(point: AutoUpdateRestorePoint) {
  return () =>
    rollbackAutoUpdate(point).catch((error) => {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Could not roll back updates`),
        body: errorMessage(error),
      })
    })
}

function updateFailure(error: unknown) {
  const point = isAutoUpdateError(error) ? error.restorePoint : undefined
  useToasts.getState().push({
    kind: 'error',
    title: i18n._(msg`Could not update mods before Play`),
    body: errorMessage(error),
    ...(point && point.updates.length > 0
      ? { action: { label: i18n._(msg`Roll back`), run: rollbackAction(point) } }
      : {}),
  })
}

async function startProfile(opts: {
  set: (p: {
    starting?: boolean
    startingProfile?: string
    updateRollback?: UpdateRollback | null
  }) => void
  game: string
  profile: string
  direct: boolean
  update: UpdateContext | undefined
}) {
  if (opts.update) {
    opts.set({
      updateRollback: { context: opts.update, game: opts.game, profile: opts.profile },
    })
  }
  try {
    await Start(opts.game, opts.profile, opts.direct)
  } catch (error) {
    opts.set({ starting: false, startingProfile: '', updateRollback: null })
    reportError(i18n._(msg`Could not launch the game`))(error)
  }
}

async function startWithWarning(opts: {
  get: () => { starting: boolean }
  set: (p: {
    starting?: boolean
    startingProfile?: string
    updateWarn?: UpdateWarn | null
    saveWarn?: SaveWarn | null
    updateRollback?: UpdateRollback | null
  }) => void
  game: string
  profile: string
  direct: boolean
}) {
  if (opts.get().starting) {
    return
  }
  opts.set({ starting: true, startingProfile: opts.profile })
  let update: UpdateContext | undefined
  try {
    update = updateContext(await updateBeforePlay(opts.game, opts.profile))
  } catch (error) {
    opts.set({ starting: false, startingProfile: '' })
    updateFailure(error)
    return
  }
  try {
    const warning = await UpdateWarning(opts.game, opts.profile)
    if (warning.changed) {
      opts.set({
        starting: false,
        startingProfile: '',
        updateWarn: {
          game: opts.game,
          profile: opts.profile,
          direct: opts.direct,
          recorded: warning.recorded,
          installed: warning.installed,
          broken: warning.broken ?? [],
          ...(update ? { update } : {}),
        },
      })
      return
    }
  } catch (e) {
    opts.set({ starting: false, startingProfile: '' })
    reportError(i18n._(msg`Could not check the game version`))(e)
    return
  }
  try {
    const [save, gap] = await LastSaveGap(opts.game, opts.profile)
    if (gap) {
      opts.set({
        starting: false,
        startingProfile: '',
        saveWarn: {
          game: opts.game,
          profile: opts.profile,
          direct: opts.direct,
          save,
          ...(update ? { update } : {}),
        },
      })
      return
    }
  } catch {
    // An unreadable save never blocks Play; the Saves tab shows the same check.
  }
  await startProfile({
    set: opts.set,
    game: opts.game,
    profile: opts.profile,
    direct: opts.direct,
    update,
  })
}

async function startVanillaGame(opts: {
  get: () => { starting: boolean }
  set: (p: { starting: boolean; startingProfile?: string }) => void
  game: string
  direct: boolean
}) {
  if (opts.get().starting) {
    return
  }
  opts.set({ starting: true, startingProfile: '' })
  try {
    await StartVanilla(opts.game, opts.direct)
  } catch (e) {
    opts.set({ starting: false, startingProfile: '' })
    reportError(i18n._(msg`Could not launch the game`))(e)
  }
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
            title: i18n._(msg`Errors appeared after updating`),
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
  if (!polled || status.state !== State.Idle) {
    set({ starting: false, startingProfile: '' })
  }
  if (status.state === State.Launching) {
    if (resetConsole(status, previous)) {
      set({ hidden: false, failure: null })
    }
    set({ status })
    return
  }
  if (status.state === State.Failed) {
    set({ failure: { profile: status.profile, body: failureBody(status), hint: status.hint } })
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
      checkUpdatedRun(rollback, get, set).catch(() => undefined)
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
  updateRollback: UpdateRollback | null
  stopping: boolean
  // Play was pressed and no launch:state has answered yet, which is when SMAPI installs first.
  starting: boolean
  startingProfile: string
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
  dismissSaveWarn: () => void
  openSaves: () => void
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
  saveWarn: null,
  updateRollback: null,
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
  start: (game, profile, direct) => startWithWarning({ get, set, game, profile, direct }),
  startVanilla: (game, direct) => startVanillaGame({ get, set, game, direct }),
  hide: () => set({ hidden: true }),
  dismissFailure: () => set({ failure: null }),
  dismissCrash: () => set({ crash: null }),
  setCrash: (crash) => set({ crash }),
  dismissUpdateWarn: () => set({ updateWarn: null }),
  dismissSaveWarn: () => set({ saveWarn: null }),
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
  playAnyway: async () => {
    const warn = get().updateWarn ?? get().saveWarn
    set({ updateWarn: null, saveWarn: null })
    if (!warn) {
      return
    }
    set({ starting: true, startingProfile: warn.profile })
    await startProfile({
      set,
      game: warn.game,
      profile: warn.profile,
      direct: warn.direct,
      update: warn.update,
    })
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

export function initLaunch() {
  Events.On('launch:state', (event) => useLaunch.getState().apply(event.data))
  Events.On('launch:line', (event) => useConsole.getState().add(event.data))
  Events.On('launch:crash', (event) => useLaunch.getState().setCrash(event.data))
}

export const overlayGame = (routeName: string, routeGame: string, statusGame: string) =>
  routeName === 'game' ? routeGame : statusGame
