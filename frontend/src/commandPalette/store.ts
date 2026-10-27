import { create } from 'zustand'

export const useCommandPalette = create<{
  open: boolean
  creating: boolean
  bisect: { id: string; game: string; profile: string } | null
  // A crash check waiting for the player to confirm it.
  confirmingBisect: boolean
  setOpen: (open: boolean) => void
  setCreating: (creating: boolean) => void
  setBisect: (bisect: { id: string; game: string; profile: string } | null) => void
}>((set) => ({
  open: false,
  creating: false,
  bisect: null,
  confirmingBisect: false,
  setOpen: (open) => set({ open }),
  setCreating: (creating) => set({ creating }),
  setBisect: (bisect) => set({ bisect }),
}))
