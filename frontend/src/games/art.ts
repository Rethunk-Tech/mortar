import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'

// The game's hero art: Steam's local cache when present, else the same image from Steam's public CDN, so a game
// shows its art before any launcher is found. Nothing is bundled or re-hosted.
export const gameArt = (game: Pick<GameInfo, 'artUrl' | 'appId'> | undefined): string => {
  if (!game) {
    return ''
  }
  if (game.artUrl) {
    return game.artUrl
  }
  return game.appId
    ? `https://cdn.cloudflare.steamstatic.com/steam/apps/${game.appId}/library_hero.jpg`
    : ''
}
