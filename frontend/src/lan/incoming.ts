import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { Arrival } from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/models.ts'
import { Inbox } from '../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'

interface IncomingState {
  items: Arrival[]
  add: (arrival: Arrival) => void
  removeFirst: () => void
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
  for (const arrival of (await Inbox()) ?? []) {
    useIncomingShares.getState().add(arrival)
  }
}

const useIncomingShares = create<IncomingState>((set) => ({
  items: [],
  add: (arrival) => set((state) => ({ items: [...state.items, arrival] })),
  removeFirst: () => set((state) => ({ items: state.items.slice(1) })),
}))

export { initIncoming, useIncomingShares }
