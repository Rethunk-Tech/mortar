import type { GameId } from '../../nav/store.ts'

export function lastOpenedGame(lastGame: string): GameId | null {
  return lastGame === 'stardew' ? lastGame : null
}
