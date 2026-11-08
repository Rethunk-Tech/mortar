import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { routeGame, useNav } from './store.ts'

// The game the user is working in: the route's game, else the last one played, else the one the profiles store holds.
function pick(routeId: string | null, lastGame: string, loaded: string | undefined): string {
  return routeId || lastGame || loaded || ''
}

export function currentGame(): string {
  return pick(
    routeGame(useNav.getState().route),
    useSettings.getState().lastGame ?? '',
    useProfiles.getState().game?.id,
  )
}

export function useCurrentGame(): string {
  const routeId = useNav((s) => routeGame(s.route))
  // A route naming a game decides the answer, so the other two are read as empty while it does: a save of the last
  // game, or a game loading, would otherwise redraw every reader of this hook.
  const onRoute = () => routeId !== null
  const lastGame = useSettings((s) => (onRoute() ? '' : (s.lastGame ?? '')))
  const loaded = useProfiles((s) => (onRoute() ? undefined : s.game?.id))
  return pick(routeId, lastGame, loaded)
}
