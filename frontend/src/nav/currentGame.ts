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
  const lastGame = useSettings((s) => s.lastGame ?? '')
  const loaded = useProfiles((s) => s.game?.id)
  return pick(routeId, lastGame, loaded)
}
