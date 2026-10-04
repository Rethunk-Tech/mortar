import { msg } from '@lingui/core/macro'
import { StartPreset } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Broken } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { UpdateWarning } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { foldedOverrides, resolveOverride } from '../profiles/overrideValue.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { errorMessage, reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  type AutoUpdateRestorePoint,
  type AutoUpdateResult,
  isAutoUpdateError,
  updateBeforePlay,
} from './autoUpdate.ts'
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

type LaunchSet = (p: {
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
}

function updateContext(result: AutoUpdateResult): UpdateContext | undefined {
  return result.restorePoint ? { ...result, restorePoint: result.restorePoint } : undefined
}

function updateFailure(error: unknown, resume: () => Promise<void>) {
  const point = isAutoUpdateError(error) ? error.restorePoint : undefined
  useToasts.getState().push({
    kind: 'error',
    title: i18n._(msg`Could not update mods before Play`),
    body: errorMessage(error),
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
    await StartPreset(opts.game, opts.profile, pendingPreset, opts.direct)
  } catch (error) {
    pendingPreset = ''
    opts.set({ starting: false, startingProfile: '', updateRollback: null })
    reportError(i18n._(msg`Could not launch the game`))(error)
  }
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
  if (!opts.skipPrePlay) {
    const listed = useProfiles.getState().profiles.find((p) => p.id === opts.profile)
    const skipCheck =
      resolveOverride(
        'skipPlayCheck',
        String(
          (useSettings.getState().games?.stardew as { skipPlayCheck?: boolean } | undefined)
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
  if (warn.game === 'stardew') {
    useNav.getState().openGame('stardew')
  }
  useProfiles.getState().open(warn.profile)
  useTab.getState().setTab('problems')
}

export type { PlayCheck, SaveWarn, UpdateContext, UpdateRollback, UpdateWarn }
export { openProblems, playAnyway, setPendingPreset, startProfile, startWithWarning, updateAndPlay }
