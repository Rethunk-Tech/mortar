import { create } from 'zustand'
import { useSettingsScope } from '../settings/settingsScope.ts'

type GameId = string

const SLUG = /^[a-z0-9-]+$/

const isGameId = (game: string): game is GameId => SLUG.test(game)

type SettingsSection =
  | 'general'
  | 'appearance'
  | 'mods'
  | 'downloads'
  | 'accounts'
  | 'sources'
  | 'updates'
  | 'notifications'
  | 'storage'
  | 'launchers'
  | 'shortcuts'
  | 'about'

type Route =
  | { name: 'game-select' }
  | { name: 'setup' }
  // from is the game that was open when the setup began, for the way back.
  | { name: 'game-setup'; game: GameId; from?: GameId }
  | { name: 'game'; game: GameId }
  | { name: 'profiles'; game: GameId }
  // query opens the page already searching.
  | { name: 'game-settings'; game: GameId; query?: string }
  | { name: 'settings'; section: SettingsSection; back: Route }

// The game a route belongs to, looking through Settings to the page it was opened from.
const routeGame = (route: Route): GameId | null => {
  if (route.name === 'settings') {
    return routeGame(route.back)
  }
  return route.name === 'game' || route.name === 'profiles' || route.name === 'game-settings'
    ? route.game
    : null
}

const useNav = create<{
  route: Route
  openGame: (game: GameId) => void
  openGameSelect: () => void
  openSetup: () => void
  openGameSetup: (game: GameId) => void
  openProfiles: () => void
  closeProfiles: () => void
  // profileScope opens it on the open profile's own values; otherwise it edits every profile's.
  openGameSettings: (profileScope?: boolean) => void
  closeGameSettings: () => void
  searchGameSettings: (query: string) => void
  openSettings: (section?: SettingsSection) => void
  closeSettings: () => void
}>((set) => ({
  route: { name: 'game-select' },
  openGame: (game) => set({ route: { name: 'game', game } }),
  openGameSelect: () => set({ route: { name: 'game-select' } }),
  openSetup: () => set({ route: { name: 'setup' } }),
  openGameSetup: (game) =>
    set(({ route }) => {
      const from = routeGame(route) ?? (route.name === 'game-setup' ? route.from : undefined)
      return { route: { name: 'game-setup', game, ...(from && from !== game ? { from } : {}) } }
    }),
  openProfiles: () =>
    set(({ route }) => {
      const game = routeGame(route)
      return game ? { route: { name: 'profiles', game } } : {}
    }),
  closeProfiles: () =>
    set(({ route }) =>
      route.name === 'profiles' ? { route: { name: 'game', game: route.game } } : {},
    ),
  openGameSettings: (profileScope = false) => {
    useSettingsScope.setState({ profile: profileScope })
    set(({ route }) =>
      route.name === 'game' ? { route: { name: 'game-settings', game: route.game } } : {},
    )
  },
  closeGameSettings: () =>
    set(({ route }) =>
      route.name === 'game-settings' ? { route: { name: 'game', game: route.game } } : {},
    ),
  searchGameSettings: (query) => {
    useSettingsScope.setState({ profile: false })
    set(({ route }) => {
      const game = routeGame(route)
      return game ? { route: { name: 'game-settings', game, query } } : {}
    })
  },
  openSettings: (section) =>
    set(({ route }) => {
      if (route.name !== 'settings') {
        return { route: { name: 'settings', section: section ?? 'general', back: route } }
      }
      return section && section !== route.section ? { route: { ...route, section } } : {}
    }),
  closeSettings: () => set(({ route }) => (route.name === 'settings' ? { route: route.back } : {})),
}))

const openSettings = (section?: SettingsSection) => useNav.getState().openSettings(section)

export { type GameId, isGameId, openSettings, type Route, routeGame, type SettingsSection, useNav }
