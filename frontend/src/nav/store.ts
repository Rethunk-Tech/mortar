import { create } from 'zustand'

export type GameId = 'stardew'

export type Route = { name: 'game-select' } | { name: 'game'; game: GameId }

export const useNav = create<{
  route: Route
  openGame: (game: GameId) => void
  openGameSelect: () => void
}>((set) => ({
  route: { name: 'game-select' },
  openGame: (game) => set({ route: { name: 'game', game } }),
  openGameSelect: () => set({ route: { name: 'game-select' } }),
}))
