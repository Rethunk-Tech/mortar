import { msg, plural } from '@lingui/core/macro'
import { create } from 'zustand'
import type {
  Entry,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  RemoveEntry,
  RollBack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type {
  Item,
  State,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import {
  State as fetchState,
  Retry,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { i18n } from '../i18n/index.ts'
import { considerMissing } from '../install/store.ts'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { useMods } from '../mods/store.ts'
import { openSettings } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { isTrackedImportBatch, observeImportState } from '../share/importCompletion.ts'
import { follow } from '../shell/follow.ts'
import { changeStillLatest, type HistoryActionState } from '../toasts/history.ts'
import { useToasts } from '../toasts/store.ts'

// The binding types a Go slice as nullable; the store keeps it a list.
type Snapshot = Omit<State, 'items'> & { items: Item[] }

const snapshot = (s: State): Snapshot => ({ ...s, items: s.items ?? [] })

const empty: Snapshot = { items: [], paused: false, limitedUntil: 0 }

const show = () => useQueue.getState().setOpen(true)

function showLive(): HistoryActionState {
  const {
    state: { items },
  } = useQueue.getState()
  if (
    items.some(
      (i) =>
        i.state === 'needs-choice' ||
        i.state === 'needs-confirm' ||
        i.state === 'needs-merge' ||
        i.state === 'failed',
    )
  ) {
    return { disabled: false }
  }
  return { disabled: true, reason: i18n._(msg`Nothing in the queue needs a decision.`) }
}

function retryLive(id: string): HistoryActionState {
  const item = useQueue.getState().state.items.find((i) => i.id === id)
  if (item?.state !== 'failed') {
    return { disabled: true, reason: i18n._(msg`That download is no longer waiting to retry.`) }
  }
  return { disabled: false }
}

const shouldRollBack = (item: Pick<Item, 'kind'>) => item.kind === 'update'

function entryForItem(profile: Profile | undefined, item: Pick<Item, 'modId' | 'name' | 'repo'>) {
  return (profile?.entries ?? []).find((e) => matchesItem(e, item))
}

function unblockedDependent(
  missing: { uniqueId: string; dependentName: string }[] | null | undefined,
  installedIds: string[],
): string | undefined {
  if (!missing || installedIds.length === 0) {
    return
  }
  const ids = new Set(installedIds.map((id) => id.toLowerCase()))
  return missing.find((m) => ids.has(m.uniqueId.toLowerCase()))?.dependentName
}

const MS_PER_SEC = 1000
const REFRESH_DEBOUNCE_MS = 1000
let refreshTimer: ReturnType<typeof setTimeout> | undefined
let installToast: number | undefined

function retryWaitSeconds(until: number, now = Date.now()): number {
  return Math.max(1, until - Math.floor(now / MS_PER_SEC))
}

function queueErrorDetail(error: string): string | undefined {
  return error === '' ? undefined : error
}

function downloadFailCopy(error: string): { body: string; detail?: string } {
  const detail = queueErrorDetail(error)
  return {
    body: i18n._(msg`The download could not finish. Retry or skip it from the queue.`),
    ...(detail === undefined ? {} : { detail }),
  }
}

function failureToast(item: Item) {
  const error = item.error ?? ''
  if (error.includes('API key')) {
    return {
      title: i18n._(msg`Nexus rejected your API key`),
      action: { label: i18n._(msg`Open Nexus settings`), run: () => openSettings('nexus') },
    }
  }
  if (error.includes('Not enough disk space')) {
    return {
      title: i18n._(msg`Not enough disk space`),
      action: { label: i18n._(msg`Open storage settings`), run: () => openSettings('storage') },
    }
  }
  if (error.includes('quarantined')) {
    return { title: i18n._(msg`File quarantined`) }
  }
  return {
    title: i18n._(msg`Could not reach Nexus`),
    action: {
      label: i18n._(msg`Retry now`),
      run: () => Retry(item.id),
      live: () => retryLive(item.id),
    },
  }
}

function matchesItem(e: Entry, item: Pick<Item, 'modId' | 'name' | 'repo'>) {
  if (item.modId && e.source.modId === item.modId) {
    return true
  }
  if (item.repo && e.source.repo === item.repo) {
    return true
  }
  return (e.mods ?? []).some((m) => m.name === item.name)
}

function profileLocked(profileId: string) {
  const { status, starting, startingProfile } = useLaunch.getState()
  return isLocked(status, profileId, starting ? startingProfile : '')
}

function singleNexusFailure(failed: Item[]) {
  if (failed.length !== 1) {
    return
  }
  const [item] = failed
  return item && item.repo === '' ? item : undefined
}

async function undoInstall(item: Item, entry: Entry | undefined) {
  if (!entry || profileLocked(item.profileId)) {
    return false
  }
  try {
    const next = shouldRollBack(item)
      ? await RollBack(item.game, item.profileId, entry.key)
      : await RemoveEntry(item.game, item.profileId, entry.key)
    useProfiles.getState().replace(next)
  } catch {
    return false
  }
  await useMods.getState().load()
  return true
}

// Says what changed since the last state: installs land in the open profile's list, and failures are worth a nudge.
function installUndo(item: Item, entry: Entry | undefined) {
  if (!entry) {
    return
  }
  return {
    picture: entry.source.picture,
    profileId: item.profileId,
    entry,
  }
}

function pushDownloadFailures(failed: Item[]) {
  if (useSettings.getState().notifyDownloadFailed === false) {
    return
  }
  const nexusFail = singleNexusFailure(failed)
  if (nexusFail) {
    const cause = failureToast(nexusFail)
    useToasts.getState().push({
      kind: 'error',
      title: cause.title,
      ...downloadFailCopy(nexusFail.error ?? ''),
      ...(cause.action === undefined ? {} : { action: cause.action }),
    })
    return
  }
  if (failed.length === 0) {
    return
  }
  const [firstFail] = failed
  useToasts.getState().push({
    kind: 'error',
    title: i18n._(
      msg`${plural(failed.length, { one: '# download failed', other: '# downloads failed' })}`,
    ),
    ...downloadFailCopy(firstFail?.error ?? ''),
    action: { label: i18n._(msg`Show`), run: show, live: showLive },
  })
}

function pushRateLimitPause(prev: Snapshot, next: Snapshot) {
  if (next.limitedUntil > 0 && next.limitedUntil !== prev.limitedUntil) {
    const n = retryWaitSeconds(next.limitedUntil)
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Downloads paused`),
      body: i18n._(msg`Retrying in ${n} s.`),
    })
  }
}

function debounceProfileRefresh(games: string[]) {
  clearTimeout(refreshTimer)
  refreshTimer = setTimeout(async () => {
    refreshTimer = undefined
    const { game, refresh, load } = useProfiles.getState()
    for (const id of games) {
      await (id === game?.id ? refresh() : load(id))
        .then(() => useMods.getState().load())
        .catch(() => undefined)
    }
  }, REFRESH_DEBOUNCE_MS)
}

function toastInstalls(shownDone: Item[], unblocked: string | undefined) {
  if (useSettings.getState().notifyDownloadFinished === false) {
    return
  }
  if (shownDone.length === 1) {
    const [item] = shownDone
    if (!item) {
      return
    }
    const profile = useProfiles.getState().profiles.find((p) => p.id === item.profileId)
    const entry = entryForItem(profile, item)
    const extra = installUndo(item, entry)
    const first = item.name
    installToast = useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`${first} installed into ${profile?.name ?? 'profile'}`),
      ...(unblocked ? { body: i18n._(msg`${unblocked} can load now.`) } : {}),
      ...(extra
        ? {
            picture: extra.picture,
            action: {
              label: i18n._(msg`Undo`),
              run: () => undoInstall(item, extra.entry),
              profileId: extra.profileId,
              live: () =>
                changeStillLatest(
                  useProfiles.getState().profiles.find((p) => p.id === extra.profileId),
                  extra.entry.key,
                  (extra.entry.mods ?? []).map((m) => m.uniqueId),
                ),
            },
          }
        : {}),
    })
    return
  }
  if (shownDone.length > 1) {
    const title = i18n._(
      msg`${plural(shownDone.length, { one: '# mod installed', other: '# mods installed' })}`,
    )
    if (installToast === undefined) {
      installToast = useToasts.getState().push({ kind: 'success', title })
    } else {
      useToasts.getState().update(installToast, { title })
    }
  }
}

function announce(prev: Snapshot, next: Snapshot) {
  const before = new Map(prev.items.map((i) => [i.id, i.state]))
  const changed = (state: string) =>
    next.items.filter((i) => i.state === state && before.get(i.id) !== state)
  const done = changed('done')
  const failed = changed('failed')
  const waiting = [
    ...changed('needs-choice'),
    ...changed('needs-confirm'),
    ...changed('needs-merge'),
  ]
  const blocked = useMods.getState().problems?.missing
  const games = [...new Set(done.map((i) => i.game))]
  const shown = (items: Item[]) => items.filter((item) => !isTrackedImportBatch(item.batchId ?? ''))
  const shownDone = shown(done)
  const shownFailed = shown(failed)
  const shownWaiting = shown(waiting)
  if (games.length > 0) {
    debounceProfileRefresh(games)
  }
  const dependentIds: string[] = []
  for (const item of done) {
    const profile = useProfiles.getState().profiles.find((p) => p.id === item.profileId)
    for (const mod of entryForItem(profile, item)?.mods ?? []) {
      if (mod.uniqueId) {
        dependentIds.push(mod.uniqueId)
      }
    }
  }
  considerMissing(dependentIds)
  const unblocked = unblockedDependent(blocked, dependentIds)
  toastInstalls(shownDone, unblocked)
  if (shownWaiting.length > 0) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(
        msg`${plural(shownWaiting.length, { one: '# download needs your decision', other: '# downloads need your decision' })}`,
      ),
      action: { label: i18n._(msg`Show`), run: show, live: showLive },
    })
  }
  pushDownloadFailures(shownFailed)
  pushRateLimitPause(prev, next)
}

export const useQueue = create<{
  state: Snapshot
  open: boolean
  historyBatchId: string
  setOpen: (open: boolean) => void
  openHistory: (batchId: string) => void
  consumeHistoryBatch: () => void
}>((set) => ({
  state: empty,
  open: false,
  historyBatchId: '',
  setOpen: (open) => set({ open, ...(open ? {} : { historyBatchId: '' }) }),
  openHistory: (batchId) => set({ open: true, historyBatchId: batchId }),
  consumeHistoryBatch: () => set({ historyBatchId: '' }),
}))

// The first state seen, fetched or evented, is the silent baseline: items finished before startup are not news.
export const initQueue = () =>
  follow('queue:changed', fetchState, (state, first) => {
    const prev = useQueue.getState().state
    const next = snapshot(state)
    useQueue.setState({ state: next })
    observeImportState(next.items)
    if (!first) {
      announce(prev, next)
    }
  })

export {
  entryForItem,
  installUndo,
  queueErrorDetail,
  retryWaitSeconds,
  shouldRollBack,
  singleNexusFailure,
  unblockedDependent,
  undoInstall,
}
