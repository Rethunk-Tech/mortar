import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import {
  List,
  SteamStatus,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import type { Status } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import { shareInFlight } from '../profiles/listProfiles.ts'

export interface GameStatus {
  games: GameInfo[]
  steam: Awaited<ReturnType<typeof SteamStatus>>
}

export function loaderCaption(loader: string, status: Status | null): string {
  if (status?.installed && status.version !== '') {
    return `${loader} ${status.version}`
  }
  return loader
}

const shared = shareInFlight<GameStatus>()

/** The catalog and Steam's state. Startup asks from several places at once, so callers asking while a read is under
 * way share it. */
export function loadGameStatus(): Promise<GameStatus> {
  return shared('', async () => {
    const [games, steam] = await Promise.all([List(), SteamStatus()])
    return { games: games ?? [], steam }
  })
}
