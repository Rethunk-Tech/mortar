import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Updates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  History,
  List,
  Revert,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type {
  Item,
  Request,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  Add,
  State as QueueState,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { installableUpdate, visibleUpdates } from '../mods/lookup.ts'
import { useProfiles } from '../profiles/store.ts'
import type { Want } from '../queue/actions.ts'

interface AutoUpdatePlan {
  updates: Update[]
  wants: Want[]
}

function planAutoUpdates(updates: Update[], pinned: ReadonlySet<string>): AutoUpdatePlan {
  const selected = updates.filter((update) => !pinned.has(update.key) && installableUpdate(update))
  return {
    updates: selected,
    wants: selected.map((update) => ({
      kind: 'update',
      ...(update.githubRepo ? { repo: update.githubRepo } : { modId: update.nexusId }),
      name: update.name,
      version: update.version,
      currentKey: update.key,
    })),
  }
}

interface AutoUpdateRestorePoint {
  game: string
  profileId: string
  historyId: string
  updates: Update[]
}

interface AutoUpdateError extends Error {
  restorePoint: AutoUpdateRestorePoint
}

function isAutoUpdateError(error: unknown): error is AutoUpdateError {
  return error instanceof Error && 'restorePoint' in error
}

function withRestorePoint(error: unknown, restorePoint: AutoUpdateRestorePoint): AutoUpdateError {
  const failure = error instanceof Error ? error : new Error(textOf(error), { cause: error })
  Object.defineProperty(failure, 'restorePoint', { value: restorePoint })
  return failure as AutoUpdateError
}

interface AutoUpdateResult {
  restorePoint: AutoUpdateRestorePoint | null
  previousRunId: string
  previousErrors: number | null
}

const FINISHED = new Set([
  'done',
  'failed',
  'skipped',
  'cancelled',
  'waiting-click',
  'needs-choice',
  'needs-confirm',
  'needs-fomod',
  'needs-root',
  'needs-merge',
])

const QUEUE_POLL_MS = 250

const textOf = (error: unknown) => (error instanceof Error ? error.message : String(error))

function itemFailure(item: Item): Error {
  return new Error(item.error || `${item.name || 'A mod'} update did not finish`)
}

const delay = (ms: number) => new Promise<void>((resolve) => globalThis.setTimeout(resolve, ms))

async function waitForUpdates(ids: string[]): Promise<void> {
  for (;;) {
    const state = await QueueState()
    const items = state.items ?? []
    const found = ids.map((id) => items.find((item) => item.id === id))
    if (found.every((item) => item !== undefined)) {
      const unfinished = found.find((item) => item !== undefined && !FINISHED.has(item.state))
      if (!unfinished) {
        const failed = found.find((item) => item !== undefined && item.state !== 'done')
        if (failed) {
          throw itemFailure(failed)
        }
        return
      }
    }
    await delay(QUEUE_POLL_MS)
  }
}

function requests(game: string, profileId: string, wants: Want[]): Request[] {
  return wants.map((want) => ({
    kind: want.kind,
    game,
    profileId,
    modId: want.modId ?? 0,
    fileId: want.fileId ?? 0,
    name: want.name ?? '',
    fileName: want.fileName ?? '',
    version: want.version ?? '',
    currentKey: want.currentKey ?? '',
    repo: want.repo ?? '',
    tag: '',
    asset: '',
    latest: want.latest ?? false,
  }))
}

async function previousRun(game: string, profileId: string) {
  try {
    const runs = await Runs(game, profileId)
    const run = runs?.[0]
    return { id: run?.id ?? '', errors: run?.errors ?? 0 }
  } catch {
    return { id: '', errors: null }
  }
}

function pinnedKeys(profile: Profile): Set<string> {
  return new Set((profile.entries ?? []).filter((entry) => entry.pinned).map((entry) => entry.key))
}

async function updateBeforePlay(game: string, profileId: string): Promise<AutoUpdateResult> {
  const profile = useProfiles.getState().profiles.find((candidate) => candidate.id === profileId)
  if (!profile?.updateBeforePlay) {
    return { restorePoint: null, previousRunId: '', previousErrors: null }
  }

  const before = await previousRun(game, profileId)
  const point: AutoUpdateRestorePoint = { game, profileId, historyId: '', updates: [] }
  try {
    const history = await History(game, profileId)
    point.historyId = history?.[0]?.id ?? ''
    const result = await Updates(game, profileId)
    const plan = planAutoUpdates(visibleUpdates(result, profile), pinnedKeys(profile))
    if (plan.updates.length === 0) {
      return { restorePoint: null, previousRunId: before.id, previousErrors: before.errors }
    }
    point.updates = plan.updates
    if (useProfiles.getState().openId !== profileId) {
      useProfiles.getState().open(profileId)
    }
    const added = await Add(requests(game, profileId, plan.wants))
    if (!added || added.length !== plan.wants.length) {
      throw new Error('the update downloads could not be added')
    }
    await waitForUpdates(added.map((item) => item.id))
    return { restorePoint: point, previousRunId: before.id, previousErrors: before.errors }
  } catch (error) {
    if (isAutoUpdateError(error)) {
      throw error
    }
    throw withRestorePoint(error, point)
  }
}

async function currentUpdateKeys(point: AutoUpdateRestorePoint): Promise<string[]> {
  const profiles = await List(point.game)
  const profile = profiles?.find((candidate) => candidate.id === point.profileId)
  if (!profile) {
    return []
  }
  return point.updates.flatMap((update) => {
    return (profile.entries ?? [])
      .filter((candidate) => candidate.previousKey === update.key)
      .map((entry) => entry.key)
      .filter((key) => key !== '')
  })
}

async function rollbackAutoUpdate(point: AutoUpdateRestorePoint): Promise<void> {
  if (point.historyId) {
    const next = await Revert(point.game, point.profileId, point.historyId)
    useProfiles.getState().replace(next)
  } else {
    const keys = await currentUpdateKeys(point)
    for (const key of keys.reverse()) {
      const next = await RollBack(point.game, point.profileId, key)
      useProfiles.getState().replace(next)
    }
  }
  const { useMods } = await import('../mods/store.ts')
  await useMods.getState().load()
}

export type { AutoUpdatePlan, AutoUpdateRestorePoint, AutoUpdateResult }
export { isAutoUpdateError, planAutoUpdates, rollbackAutoUpdate, updateBeforePlay }
