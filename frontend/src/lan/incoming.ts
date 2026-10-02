import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type {
  Arrival,
  TransferProgress,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/models.ts'
import { Inbox } from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'

interface IncomingState {
  items: Arrival[]
  progress: Record<number, TransferProgress>
  add: (arrival: Arrival) => void
  removeFirst: () => void
  setProgress: (progress: TransferProgress) => void
}

let initialized = false

async function initIncoming(): Promise<void> {
  if (initialized) {
    return
  }
  initialized = true
  Events.On('lan:arrived', (event) => {
    useIncomingShares.getState().add(event.data)
  })
  Events.On('lan:transfer', (event) => {
    useIncomingShares.getState().setProgress(event.data)
  })
  for (const arrival of (await Inbox()) ?? []) {
    useIncomingShares.getState().add(arrival)
  }
}

const useIncomingShares = create<IncomingState>((set) => ({
  items: [],
  progress: {},
  add: (arrival) => set((state) => ({ items: [...state.items, arrival] })),
  removeFirst: () => set((state) => ({ items: state.items.slice(1) })),
  setProgress: (progress) =>
    set((state) => ({ progress: { ...state.progress, [progress.id]: progress } })),
}))

export { initIncoming, useIncomingShares }
