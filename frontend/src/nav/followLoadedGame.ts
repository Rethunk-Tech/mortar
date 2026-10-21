import { useProfiles } from '../profiles/store.ts'
import { type Route, useNav } from './store.ts'

// The game screens render the profiles store's game while the title bar names the route's, so a profile loaded for
// another game than the one open (an import, a link, a LAN receive) must carry the route along.
export function gameToFollow(route: Route, loaded: string | undefined): string | null {
  const onGame =
    route.name === 'game' || route.name === 'profiles' || route.name === 'game-settings'
  return onGame && loaded && route.game !== loaded ? loaded : null
}

export function initFollowLoadedGame() {
  useProfiles.subscribe((s, prev) => {
    const loaded = s.game?.id
    if (loaded === prev.game?.id) {
      return
    }
    const game = gameToFollow(useNav.getState().route, loaded)
    if (game) {
      useNav.getState().openGame(game)
    }
  })
}
