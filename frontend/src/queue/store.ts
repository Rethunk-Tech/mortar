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
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { follow } from '../shell/follow.ts'
import { useToasts } from '../toasts/store.ts'

// The binding types a Go slice as nullable; the store keeps it a list.
type Snapshot = Omit<State, 'items'> & { items: Item[] }

const snapshot = (s: State): Snapshot => ({ ...s, items: s.items ?? [] })

const empty: Snapshot = { items: [], paused: false, limitedUntil: 0 }

const show = () => useQueue.getState().setOpen(true)

const shouldRollBack = (item: Pick<Item, 'kind'>) => item.kind === 'update'

function entryForItem(profile: Profile | undefined, item: Pick<Item, 'modId' | 'name' | 'repo'>) {
  return (profile?.entries ?? []).find((e) => matchesItem(e, item))
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
  return isLocked(useLaunch.getState().status, profileId, useLaunch.getState().starting)
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

async function announce(prev: Snapshot, next: Snapshot) {
  const before = new Map(prev.items.map((i) => [i.id, i.state]))
  const changed = (state: string) =>
    next.items.filter((i) => i.state === state && before.get(i.id) !== state)
  const done = changed('done')
  const failed = changed('failed')
  const waiting = [...changed('needs-choice'), ...changed('needs-confirm')]
  const { game, refresh, load } = useProfiles.getState()
  const games = [...new Set(done.map((i) => i.game))]
  for (const id of games) {
    await (id === game?.id ? refresh() : load(id))
      .then(() => useMods.getState().load())
      .catch(() => undefined)
  }
  if (done.length === 1) {
    const [item] = done
    if (item) {
      const profile = useProfiles.getState().profiles.find((p) => p.id === item.profileId)
      const entry = entryForItem(profile, item)
      const extra = installUndo(item, entry)
      const first = item.name
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`${first} installed`),
        ...(extra
          ? {
              picture: extra.picture,
              action: {
                label: i18n._(msg`Undo`),
                run: () => undoInstall(item, extra.entry),
                profileId: extra.profileId,
              },
            }
          : {}),
      })
    }
  } else if (done.length > 1) {
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(
        msg`${plural(done.length, { one: '# mod installed', other: '# mods installed' })}`,
      ),
    })
  }
  if (waiting.length > 0) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(
        msg`${plural(waiting.length, { one: '# download needs your decision', other: '# downloads need your decision' })}`,
      ),
      action: { label: i18n._(msg`Show`), run: show },
    })
  }
  const nexusFail = singleNexusFailure(failed)
  if (nexusFail) {
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Couldn't reach Nexus`),
      body: nexusFail.error ?? '',
      action: {
        label: i18n._(msg`Retry now`),
        run: () => Retry(nexusFail.id),
      },
    })
  } else if (failed.length > 0) {
    const [firstFail] = failed
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(
        msg`${plural(failed.length, { one: '# download failed', other: '# downloads failed' })}`,
      ),
      body: firstFail?.error ?? '',
      action: { label: i18n._(msg`Show`), run: show },
    })
  }
}

export const useQueue = create<{
  state: Snapshot
  open: boolean
  setOpen: (open: boolean) => void
}>((set) => ({
  state: empty,
  open: false,
  setOpen: (open) => set({ open }),
}))

// The first state seen, fetched or evented, is the silent baseline: items finished before startup are not news.
export const initQueue = () =>
  follow('queue:changed', fetchState, (state, first) => {
    const prev = useQueue.getState().state
    const next = snapshot(state)
    useQueue.setState({ state: next })
    if (!first) {
      announce(prev, next).catch(() => undefined)
    }
  })

export { announce, entryForItem, installUndo, shouldRollBack, singleNexusFailure, undoInstall }
