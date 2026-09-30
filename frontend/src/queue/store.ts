import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Item,
  State,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { State as fetchState } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { i18n } from '../i18n/index.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useToasts } from '../toasts/store.ts'

// The binding types a Go slice as nullable; the store keeps it a list.
type Snapshot = Omit<State, 'items'> & { items: Item[] }

const snapshot = (s: State): Snapshot => ({ ...s, items: s.items ?? [] })

const empty: Snapshot = { items: [], paused: false, limitedUntil: 0 }

const show = () => useQueue.getState().setOpen(true)

// Says what changed since the last state: installs land in the open profile's list, and failures are worth a nudge.
function announce(prev: Snapshot, next: Snapshot) {
  const before = new Map(prev.items.map((i) => [i.id, i.state]))
  const changed = (state: string) =>
    next.items.filter((i) => i.state === state && before.get(i.id) !== state)
  const done = changed('done')
  const failed = changed('failed')
  const { game, refresh } = useProfiles.getState()
  if (done.some((i) => i.game === game?.id)) {
    refresh()
      .then(() => useMods.getState().load())
      .catch(() => undefined)
  }
  if (done.length > 0) {
    const first = done[0]?.name ?? ''
    useToasts.getState().push({
      kind: 'success',
      title:
        done.length === 1
          ? i18n._(msg`${first} installed`)
          : i18n._(
              msg`${plural(done.length, { one: '# mod installed', other: '# mods installed' })}`,
            ),
    })
  }
  if (failed.length > 0) {
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(
        msg`${plural(failed.length, { one: '# download failed', other: '# downloads failed' })}`,
      ),
      body: failed[0]?.error ?? '',
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

export async function initQueue(): Promise<void> {
  Events.On('queue:changed', (event) => {
    const prev = useQueue.getState().state
    const next = snapshot(event.data)
    useQueue.setState({ state: next })
    announce(prev, next)
  })
  useQueue.setState({ state: snapshot(await fetchState()) })
}
