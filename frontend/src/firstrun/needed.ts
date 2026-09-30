import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { LocalStatus } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loadersvc/service.ts'
import { List } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { shouldShowFirstRun } from './logic.ts'

export const STARDEW = 'stardew'

export async function firstRunNeeded(games: GameInfo[]): Promise<boolean> {
  const installed = games.find((g) => g.id === STARDEW)?.installed === true
  if (!installed) {
    return true
  }
  const [status, profiles] = await Promise.all([LocalStatus(STARDEW), List(STARDEW)])
  return shouldShowFirstRun({
    installed,
    smapiInstalled: status.installed,
    profileCount: profiles?.length ?? 0,
  })
}
