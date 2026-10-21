import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { listProfiles } from '../profiles/listProfiles.ts'
import { shouldShowFirstRun } from './logic.ts'

// A game is set up when its folder is known and it has a profile; until then opening it runs its setup.
export const setupNeeded = (game: GameInfo, profileCount: number): boolean =>
  shouldShowFirstRun({ installed: game.installed, profileCount })

export async function gameSetupNeeded(game: GameInfo): Promise<boolean> {
  if (!game.installed) {
    return true
  }
  return setupNeeded(game, (await listProfiles(game.id))?.length ?? 0)
}
