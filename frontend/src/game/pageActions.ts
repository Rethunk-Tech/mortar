import { create } from 'zustand'

// The element TabHeader reserves for the open tab's own buttons; PageActions portals into it.
export const usePageActionsSlot = create<{
  slot: HTMLElement | null
  setSlot: (slot: HTMLElement | null) => void
}>((set) => ({ slot: null, setSlot: (slot) => set({ slot }) }))
