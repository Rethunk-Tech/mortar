import {
  List,
  SteamStatus,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/games/service.ts'

export type GameStatus = {
  games: Awaited<ReturnType<typeof List>>
  steam: Awaited<ReturnType<typeof SteamStatus>>
}

export async function loadGameStatus(): Promise<GameStatus> {
  const [games, steam] = await Promise.all([List(), SteamStatus()])
  return { games, steam }
}
