import { create } from 'zustand'

// The game whose Mods folder the "From the game's Mods folder" import reads; empty while the dialog is closed.
export const useGameModsDialog = create<{
  game: string
  open: (game: string) => void
  close: () => void
}>((set) => ({
  game: '',
  open: (game) => set({ game }),
  close: () => set({ game: '' }),
}))
