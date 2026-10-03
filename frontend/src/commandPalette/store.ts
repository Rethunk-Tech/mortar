import { create } from 'zustand'

export const useCommandPalette = create<{
  open: boolean
  creating: boolean
  bisect: { id: string; game: string; profile: string } | null
  setOpen: (open: boolean) => void
  setCreating: (creating: boolean) => void
  setBisect: (bisect: { id: string; game: string; profile: string } | null) => void
}>((set) => ({
  open: false,
  creating: false,
  bisect: null,
  setOpen: (open) => set({ open }),
  setCreating: (creating) => set({ creating }),
  setBisect: (bisect) => set({ bisect }),
}))
