import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { LocalStatus } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { shouldShowFirstRun } from './logic.ts'

// A game is set up when its folder is known and it has a loader or a profile; until then opening it runs its setup.
export async function gameSetupNeeded(game: GameInfo): Promise<boolean> {
  if (!game.installed) {
    return true
  }
  const [status, profiles] = await Promise.all([LocalStatus(game.id), List(game.id)])
  return shouldShowFirstRun({
    installed: true,
    smapiInstalled: status.installed,
    profileCount: profiles?.length ?? 0,
  })
}
