import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'

// Steam's library hero is about 3.1 times as wide as it is tall.
export const HERO_ASPECT = 3.1

const ART_STEP_PX = 320
const ART_MAX_PX = 2560

// The width the art is asked for, in device pixels on 320-pixel steps so rows of one width share the cached copy.
const artPx = (cssPx: number) =>
  Math.min(
    ART_MAX_PX,
    Math.max(
      ART_STEP_PX,
      Math.ceil((cssPx * (globalThis.devicePixelRatio || 1)) / ART_STEP_PX) * ART_STEP_PX,
    ),
  )

// The game's hero art, served by the app from Steam's local cache or Steam's public copy (see ArtMiddleware), so a
// game shows its art before any launcher is found. Nothing is bundled or re-hosted. cssPx is the width the art fills:
// the webview decodes a picture at its full size, and Steam's hero is 3840 pixels wide.
export const gameArt = (game: Pick<GameInfo, 'artUrl'> | undefined, cssPx: number): string => {
  const url = game?.artUrl ?? ''
  return url ? `${url}${url.includes('?') ? '&' : '?'}w=${artPx(cssPx)}` : ''
}
