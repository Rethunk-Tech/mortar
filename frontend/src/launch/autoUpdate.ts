import { msg } from '@lingui/core/macro'
import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { Updates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  History,
  List,
  Mods,
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
import { i18n } from '../i18n/index.ts'
import { installableUpdate, sameId, visibleUpdates } from '../mods/lookup.ts'
import { useProfiles } from '../profiles/store.ts'
import type { Want } from '../queue/actions.ts'
import { useNexus } from '../settings/nexus.ts'

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
const QUEUE_TIMEOUT_MS = 300_000

const textOf = (error: unknown) => (error instanceof Error ? error.message : String(error))

const acknowledgedCautions = new Set<string>()

const cautionKey = (profileId: string, update: Pick<Update, 'key' | 'version'>) =>
  `${profileId}/${update.key}/${update.version}`

function acknowledgeUpdateCaution(
  profileId: string,
  update: Pick<Update, 'key' | 'version'>,
  acknowledged: boolean,
) {
  const key = cautionKey(profileId, update)
  if (acknowledged) {
    acknowledgedCautions.add(key)
  } else {
    acknowledgedCautions.delete(key)
  }
}

function cautionAcknowledged(profileId: string, update: Pick<Update, 'key' | 'version'>) {
  return acknowledgedCautions.has(cautionKey(profileId, update))
}

function itemFailure(item: Item): Error {
  return new Error(
    item.error || i18n._(msg`The update for ${item.name || i18n._(msg`the mod`)} did not finish`),
  )
}

const delay = (ms: number) => new Promise<void>((resolve) => globalThis.setTimeout(resolve, ms))

function queueItemSucceeded(item: Item | undefined, premium: boolean): boolean {
  return item === undefined || item.state === 'done' || (!premium && item.state === 'waiting-click')
}

async function waitForUpdates(ids: string[], premium: boolean): Promise<void> {
  const deadline = Date.now() + QUEUE_TIMEOUT_MS
  for (;;) {
    const state = await QueueState()
    const items = state.items ?? []
    const found = ids.map((id) => items.find((item) => item.id === id))
    const unfinished = found.find(
      (item) =>
        item !== undefined && !FINISHED.has(item.state) && !queueItemSucceeded(item, premium),
    )
    if (!unfinished) {
      const failed = found.find((item) => item !== undefined && !queueItemSucceeded(item, premium))
      if (failed) {
        throw itemFailure(failed)
      }
      return
    }
    if (Date.now() >= deadline) {
      throw new Error(i18n._(msg`Timed out waiting for mod updates`))
    }
    await delay(Math.min(QUEUE_POLL_MS, Math.max(0, deadline - Date.now())))
  }
}

function requests(game: string, profileId: string, wants: Want[], batchId: string): Request[] {
  return wants.map((want) => ({
    kind: want.kind,
    batchId,
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

async function updateBeforePlay(
  game: string,
  profileId: string,
  onProgress?: (count: number) => void,
  force = false,
): Promise<AutoUpdateResult> {
  const profile = useProfiles.getState().profiles.find((candidate) => candidate.id === profileId)
  if (!(force || profile?.updateBeforePlay)) {
    return { restorePoint: null, previousRunId: '', previousErrors: null }
  }
  if (!profile) {
    return { restorePoint: null, previousRunId: '', previousErrors: null }
  }

  const before = await previousRun(game, profileId)
  const point: AutoUpdateRestorePoint = { game, profileId, historyId: '', updates: [] }
  try {
    const history = await History(game, profileId)
    point.historyId = history?.[0]?.id ?? ''
    const result = await Updates(game, profileId)
    const mods = (await Mods(game, profileId)) ?? []
    const plan = planAutoUpdates(visibleUpdates(result, profile), pinnedKeys(profile))
    const updates = plan.updates.filter((update) => {
      const mod = mods.find(
        (candidate) => candidate.key === update.key && sameId(candidate.uniqueId, update.uniqueId),
      )
      return !mod?.updateCautionMessage?.trim() || cautionAcknowledged(profileId, update)
    })
    if (updates.length === 0) {
      return { restorePoint: null, previousRunId: before.id, previousErrors: before.errors }
    }
    const wants = updates.map((update) => ({
      kind: 'update' as const,
      ...(update.githubRepo ? { repo: update.githubRepo } : { modId: update.nexusId }),
      name: update.name,
      version: update.version,
      currentKey: update.key,
    }))
    point.updates = updates
    const batchId = wants.length > 1 ? crypto.randomUUID() : ''
    if (useProfiles.getState().openId !== profileId) {
      useProfiles.getState().open(profileId)
    }
    onProgress?.(updates.length)
    const added = await Add(requests(game, profileId, wants, batchId))
    if (!added || added.length !== wants.length) {
      throw new Error(i18n._(msg`Could not add the update downloads`))
    }
    // Waiting-click is a successful handoff for free Nexus accounts; the user must finish that click on Nexus.
    await waitForUpdates(
      added.map((item) => item.id),
      useNexus.getState().premium,
    )
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
  return point.updates.flatMap((update) =>
    (profile.entries ?? [])
      .filter((candidate) => candidate.previousKey === update.key)
      .map((entry) => entry.key)
      .filter((key) => key !== ''),
  )
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
export {
  acknowledgeUpdateCaution,
  cautionAcknowledged,
  isAutoUpdateError,
  planAutoUpdates,
  queueItemSucceeded,
  rollbackAutoUpdate,
  updateBeforePlay,
}
