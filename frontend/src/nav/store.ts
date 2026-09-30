import { create } from 'zustand'

export type GameId = 'stardew'

export type SettingsSection = 'appearance' | 'about'

export type Route =
  | { name: 'game-select' }
  | { name: 'game'; game: GameId }
  | { name: 'settings'; section: SettingsSection; back: Route }

export const useNav = create<{
  route: Route
  openGame: (game: GameId) => void
  openGameSelect: () => void
  openSettings: (section?: SettingsSection) => void
  closeSettings: () => void
}>((set) => ({
  route: { name: 'game-select' },
  openGame: (game) => set({ route: { name: 'game', game } }),
  openGameSelect: () => set({ route: { name: 'game-select' } }),
  openSettings: (section) =>
    set(({ route }) => ({
      route:
        route.name === 'settings'
          ? { ...route, section: section ?? route.section }
          : { name: 'settings', section: section ?? 'appearance', back: route },
    })),
  closeSettings: () => set(({ route }) => (route.name === 'settings' ? { route: route.back } : {})),
}))

export const openSettings = (section?: SettingsSection) => useNav.getState().openSettings(section)
