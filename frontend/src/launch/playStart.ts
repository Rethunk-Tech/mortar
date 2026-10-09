import { msg } from '@lingui/core/macro'
import {
  GraphicsAsk,
  StartPreset,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { Broken } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { UpdateWarning } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { SetOverrides } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { foldedOverrides, resolveOverride } from '../profiles/overrideValue.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportError, toastError } from '../toasts/report.ts'
import {
  type AutoUpdateRestorePoint,
  type AutoUpdateResult,
  isAutoUpdateError,
  updateBeforePlay,
} from './autoUpdate.ts'
import { answerTargets } from './graphicsAsk.ts'
import { gatherPlayIssues, type PlayIssueGroup } from './playIssues.ts'

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

interface UpdateWarn {
  game: string
  profile: string
  direct: boolean
  recorded: string
  installed: string
  broken: Broken[]
  update?: UpdateContext
}

interface SaveWarn {
  game: string
  profile: string
  direct: boolean
  save: Fit
  update?: UpdateContext
}

interface PlayCheck {
  game: string
  profile: string
  direct: boolean
  groups: PlayIssueGroup[]
  skipPlayCheck: boolean
}

interface GraphicsPrompt {
  game: string
  profile: string
  direct: boolean
}

type LaunchSet = (p: {
  graphicsAsk?: GraphicsPrompt | null
  starting?: boolean
  startingProfile?: string
  updating?: number
  updateWarn?: UpdateWarn | null
  saveWarn?: SaveWarn | null
  playCheck?: PlayCheck | null
  updateRollback?: UpdateRollback | null
}) => void

type LaunchGet = () => {
  starting: boolean
  playCheck: PlayCheck | null
  updateWarn: UpdateWarn | null
  saveWarn: SaveWarn | null
  graphicsAsk: GraphicsPrompt | null
}

function updateContext(result: AutoUpdateResult): UpdateContext | undefined {
  return result.restorePoint ? { ...result, restorePoint: result.restorePoint } : undefined
}

function updateFailure(error: unknown, resume: () => Promise<void>) {
  const point = isAutoUpdateError(error) ? error.restorePoint : undefined
  toastError(i18n._(msg`Could not update mods before Play`), error, {
    ...(point && point.updates.length > 0
      ? { detail: i18n._(msg`You can roll back the completed updates from the profile history.`) }
      : {}),
    action: { label: i18n._(msg`Play anyway`), run: resume },
  })
}

// The preset one Play was started with, kept until that launch ends so the warning dialogs a launch can pass through
// resume with it.
let pendingPreset = ''

function setPendingPreset(name: string) {
  pendingPreset = name
}

async function startProfile(opts: {
  set: LaunchSet
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
    await StartPreset(opts.game, opts.profile, '', pendingPreset, opts.direct)
  } catch (error) {
    pendingPreset = ''
    opts.set({ starting: false, startingProfile: '', updateRollback: null })
    reportError(i18n._(msg`Could not launch the game`))(error)
  }
}

// True when Play must stop: the dialog is up, or the check failed and said so.
async function graphicsPending(opts: {
  set: LaunchSet
  game: string
  profile: string
  direct: boolean
}): Promise<boolean> {
  opts.set({ starting: true, startingProfile: opts.profile })
  try {
    if (!(await GraphicsAsk(opts.game, opts.profile)).ask) {
      return false
    }
    opts.set({
      starting: false,
      startingProfile: '',
      graphicsAsk: { game: opts.game, profile: opts.profile, direct: opts.direct },
    })
  } catch (e) {
    opts.set({ starting: false, startingProfile: '' })
    reportError(i18n._(msg`Could not check the graphics setting`))(e)
  }
  return true
}

async function startWithWarning(opts: {
  get: LaunchGet
  set: LaunchSet
  game: string
  profile: string
  direct: boolean
  skipPrePlay?: boolean
  forceUpdate?: boolean
}) {
  if (opts.get().starting) {
    return
  }
  if (await graphicsPending(opts)) {
    return
  }
  if (!opts.skipPrePlay) {
    const listed = useProfiles.getState().profiles.find((p) => p.id === opts.profile)
    const skipCheck =
      resolveOverride(
        'skipPlayCheck',
        String(
          (useSettings.getState().games?.[opts.game] as { skipPlayCheck?: boolean } | undefined)
            ?.skipPlayCheck ?? false,
        ),
        listed ? foldedOverrides(listed) : undefined,
      ) === 'true'
    if (!skipCheck) {
      opts.set({ starting: true, startingProfile: opts.profile })
      try {
        const groups = await gatherPlayIssues(opts.game, opts.profile)
        if (groups.length > 0) {
          opts.set({
            starting: false,
            startingProfile: '',
            playCheck: {
              game: opts.game,
              profile: opts.profile,
              direct: opts.direct,
              groups,
              skipPlayCheck: skipCheck,
            },
          })
          return
        }
      } catch (e) {
        opts.set({ starting: false, startingProfile: '' })
        reportError(i18n._(msg`Could not check the profile`))(e)
        return
      }
    }
  }
  opts.set({ starting: true, startingProfile: opts.profile })
  let update: UpdateContext | undefined
  try {
    update = updateContext(
      await updateBeforePlay(
        opts.game,
        opts.profile,
        (count) => opts.set({ updating: count }),
        opts.forceUpdate === true,
      ),
    )
  } catch (error) {
    opts.set({ starting: false, startingProfile: '', updating: 0 })
    updateFailure(error, async () => {
      opts.set({ starting: true, startingProfile: opts.profile })
      await startProfile({
        set: opts.set,
        game: opts.game,
        profile: opts.profile,
        direct: opts.direct,
        update: undefined,
      })
    })
    return
  }
  opts.set({ updating: 0 })
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
    // Save check unavailable; continue launching.
  }
  await startProfile({
    set: opts.set,
    game: opts.game,
    profile: opts.profile,
    direct: opts.direct,
    update,
  })
}

async function answerGraphics(get: LaunchGet, set: LaunchSet, choice: string) {
  const ask = get().graphicsAsk
  set({ graphicsAsk: null })
  if (!ask) {
    return
  }
  try {
    await SetByKey('graphicsApi', choice, ask.game)
    const listed = useProfiles.getState().profiles.find((p) => p.id === ask.profile)
    const overrides = listed ? foldedOverrides(listed) : undefined
    if (answerTargets(overrides).profileOverride) {
      useProfiles
        .getState()
        .replace(await SetOverrides(ask.game, ask.profile, { ...overrides, graphicsApi: choice }))
    }
  } catch (e) {
    reportError(i18n._(msg`Could not save the graphics choice`))(e)
    return
  }
  await startWithWarning({ get, set, game: ask.game, profile: ask.profile, direct: ask.direct })
}

async function playAnyway(get: LaunchGet, set: LaunchSet) {
  const check = get().playCheck
  if (check) {
    set({ playCheck: null })
    await startWithWarning({
      get,
      set,
      game: check.game,
      profile: check.profile,
      direct: check.direct,
      skipPrePlay: true,
    })
    return
  }
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
}

async function updateAndPlay(get: LaunchGet, set: LaunchSet) {
  const check = get().playCheck
  set({ playCheck: null })
  if (!check) {
    return
  }
  await startWithWarning({
    get,
    set,
    game: check.game,
    profile: check.profile,
    direct: check.direct,
    skipPrePlay: true,
    forceUpdate: true,
  })
}

function openProblems(get: LaunchGet, set: LaunchSet) {
  const check = get().playCheck
  if (check) {
    set({ playCheck: null })
    if (isGameId(check.game)) {
      useNav.getState().openGame(check.game)
    }
    useProfiles.getState().open(check.profile)
    useTab.getState().setTab('problems')
    return
  }
  const warn = get().updateWarn
  set({ updateWarn: null })
  if (!warn) {
    return
  }
  if (isGameId(warn.game)) {
    useNav.getState().openGame(warn.game)
  }
  useProfiles.getState().open(warn.profile)
  useTab.getState().setTab('problems')
}

export type { GraphicsPrompt, PlayCheck, SaveWarn, UpdateContext, UpdateRollback, UpdateWarn }
export {
  answerGraphics,
  openProblems,
  playAnyway,
  setPendingPreset,
  startProfile,
  startWithWarning,
  updateAndPlay,
}
