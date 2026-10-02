import { type GameId, isGameId } from '../../nav/store.ts'

export function lastOpenedGame(lastGame: string): GameId | null {
  return isGameId(lastGame) ? lastGame : null
}
