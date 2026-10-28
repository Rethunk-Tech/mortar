import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Arrival,
  Expired,
  OutgoingTransfer,
  TransferProgress,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/models.ts'
import {
  Dismiss,
  OutgoingTransfers,
  Pending,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

interface IncomingState {
  items: Arrival[]
  progress: Record<number, TransferProgress>
  add: (arrival: Arrival) => void
  removeById: (id: number) => void
  remove: (id: number) => boolean
  setProgress: (progress: TransferProgress) => void
  // What paired computers are pulling from this one, by transfer id.
  outgoing: Record<number, OutgoingTransfer>
  setOutgoing: (transfer: OutgoingTransfer) => void
}

let initialized = false

// A share the person already answered is gone from the list, so only one still waiting is worth a notification.
function notifyExpired(expired: Expired): void {
  if (!useIncomingShares.getState().remove(expired.id)) {
    return
  }
  useToasts.getState().push({
    kind: 'warning',
    title: i18n._(
      msg`A share from ${{ sender: expired.sender }} expired. Ask them to send it again.`,
    ),
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
  Events.On('lan:outgoing', (event) => {
    useIncomingShares.getState().setOutgoing(event.data)
  })
  for (const transfer of (await OutgoingTransfers()) ?? []) {
    useIncomingShares.getState().setOutgoing(transfer)
  }
  // The service keeps a share until it is answered (Dismiss), so the window reads without taking.
  for (const arrival of (await Pending()) ?? []) {
    useIncomingShares.getState().add(arrival)
  }
}

const useIncomingShares = create<IncomingState>((set, get) => ({
  items: [],
  progress: {},
  outgoing: {},
  add: (arrival) =>
    set((state) =>
      state.items.some((item) => item.id === arrival.id)
        ? state
        : { items: [...state.items, arrival] },
    ),
  // Answering a share also drops it from the service, which keeps it until then and would hand it to the window
  // again after a reload.
  removeById: (id) => {
    if (!get().items.some((item) => item.id === id)) {
      return
    }
    Dismiss(id).catch(reportUnexpected)
    set((state) => ({ items: state.items.filter((item) => item.id !== id) }))
  },
  remove: (id) => {
    const present = get().items.some((item) => item.id === id)
    set((state) => ({ items: state.items.filter((item) => item.id !== id) }))
    return present
  },
  setProgress: (progress) =>
    set((state) => ({ progress: { ...state.progress, [progress.id]: progress } })),
  setOutgoing: (transfer) =>
    set((state) => ({ outgoing: { ...state.outgoing, [transfer.id]: transfer } })),
}))

export { initIncoming, useIncomingShares }
