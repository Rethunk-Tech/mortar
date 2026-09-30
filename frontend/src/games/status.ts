import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import {
  List,
  SteamStatus,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import type { Status } from '../../bindings/github.com/Rethunk-AI/mortar/internal/loader/models.ts'

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

export async function loadGameStatus(): Promise<GameStatus> {
  const [games, steam] = await Promise.all([List(), SteamStatus()])
  return { games: games ?? [], steam }
}
