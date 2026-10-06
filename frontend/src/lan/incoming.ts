import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Arrival,
  Expired,
  TransferProgress,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import { Inbox } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

interface IncomingState {
  items: Arrival[]
  progress: Record<number, TransferProgress>
  add: (arrival: Arrival) => void
  removeFirst: () => void
  remove: (id: number) => boolean
  setProgress: (progress: TransferProgress) => void
}

let initialized = false

// A share the person already answered is gone from the list, so only one still waiting is worth a notification.
function notifyExpired(expired: Expired): void {
  if (!useIncomingShares.getState().remove(expired.id)) {
    return
  }
  useToasts.getState().push({
    kind: 'warning',
    title: i18n._(msg`A share from ${expired.sender} expired. Ask them to send it again.`),
  })
}

async function initIncoming(): Promise<void> {
  if (initialized) {
    return
  }
  initialized = true
  Events.On('lan:arrived', (event) => {
    useIncomingShares.getState().add(event.data)
  })
  Events.On('lan:expired', (event) => {
    notifyExpired(event.data)
  })
  Events.On('lan:transfer', (event) => {
    useIncomingShares.getState().setProgress(event.data)
  })
  for (const arrival of (await Inbox()) ?? []) {
    useIncomingShares.getState().add(arrival)
  }
}

const useIncomingShares = create<IncomingState>((set, get) => ({
  items: [],
  progress: {},
  add: (arrival) => set((state) => ({ items: [...state.items, arrival] })),
  removeFirst: () => set((state) => ({ items: state.items.slice(1) })),
  remove: (id) => {
    const present = get().items.some((item) => item.id === id)
    set((state) => ({ items: state.items.filter((item) => item.id !== id) }))
    return present
  },
  setProgress: (progress) =>
    set((state) => ({ progress: { ...state.progress, [progress.id]: progress } })),
}))

export { initIncoming, useIncomingShares }
