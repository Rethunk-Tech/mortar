import { create } from 'zustand'

export const useCommandPalette = create<{
  open: boolean
  creating: boolean
  setOpen: (open: boolean) => void
  setCreating: (creating: boolean) => void
}>((set) => ({
  open: false,
  creating: false,
  setOpen: (open) => set({ open }),
  setCreating: (creating) => set({ creating }),
}))
