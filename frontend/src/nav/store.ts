import { create } from 'zustand'

export type GameId = 'stardew'

export type SettingsSection = 'appearance' | 'data' | 'nexus' | 'updates' | 'about'

export type Route =
  | { name: 'game-select' }
  | { name: 'setup' }
  | { name: 'game'; game: GameId }
  | { name: 'profiles'; game: GameId }
  | { name: 'game-settings'; game: GameId }
  | { name: 'settings'; section: SettingsSection; back: Route }

// The game a route belongs to, looking through Settings to the page it was opened from.
export const routeGame = (route: Route): GameId | null => {
  if (route.name === 'settings') {
    return routeGame(route.back)
  }
  return route.name === 'game' || route.name === 'profiles' || route.name === 'game-settings'
    ? route.game
    : null
}

export const useNav = create<{
  route: Route
  openGame: (game: GameId) => void
  openGameSelect: () => void
  openSetup: () => void
  openProfiles: () => void
  closeProfiles: () => void
  openGameSettings: () => void
  closeGameSettings: () => void
  openSettings: (section?: SettingsSection) => void
  closeSettings: () => void
}>((set) => ({
  route: { name: 'game-select' },
  openGame: (game) => set({ route: { name: 'game', game } }),
  openGameSelect: () => set({ route: { name: 'game-select' } }),
  openSetup: () => set({ route: { name: 'setup' } }),
  openProfiles: () =>
    set(({ route }) =>
      route.name === 'game' ? { route: { name: 'profiles', game: route.game } } : {},
    ),
  closeProfiles: () =>
    set(({ route }) =>
      route.name === 'profiles' ? { route: { name: 'game', game: route.game } } : {},
    ),
  openGameSettings: () =>
    set(({ route }) =>
      route.name === 'game' ? { route: { name: 'game-settings', game: route.game } } : {},
    ),
  closeGameSettings: () =>
    set(({ route }) =>
      route.name === 'game-settings' ? { route: { name: 'game', game: route.game } } : {},
    ),
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
