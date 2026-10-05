import { Box } from '@mui/material'
import { siFlatpak, siGogdotcom, siHeroicgameslauncher, siLutris, siSteam } from 'simple-icons'
import minigalaxy from '../vendor/minigalaxy.png'

// simple-icons has no Bottles mark, so the tile carries a plain bottle.
const BOTTLE_PATH = 'M10 2h4v3l1.5 2.5V20a2 2 0 0 1-2 2h-3a2 2 0 0 1-2-2V7.5L10 5z'

// Each launcher sits on a tile of its brand colour so every logo is the same size and reads in colour on dark and
// light surfaces. Steam's brand mark is black, so it takes the Steam client's blue instead.
const TILES: Record<string, { path: string; bg: string }> = {
  steam: { path: siSteam.path, bg: '#1A9FFF' },
  'flatpak-steam': { path: siSteam.path, bg: '#1A9FFF' },
  heroic: { path: siHeroicgameslauncher.path, bg: `#${siHeroicgameslauncher.hex}` },
  lutris: { path: siLutris.path, bg: `#${siLutris.hex}` },
  bottles: { path: BOTTLE_PATH, bg: '#C0392B' },
  gog: { path: siGogdotcom.path, bg: `#${siGogdotcom.hex}` },
}

// Launchers that run as a Flatpak carry its mark, so they read apart from the same launcher installed natively.
const FLATPAK = new Set(['flatpak-steam'])
const BADGE_SCALE = 0.5
// The badge overhangs the logo's corner by a quarter of its size, inside a dark ring this many pixels wide.
const BADGE_OVERHANG = 0.25
const BADGE_RING = 4
const MARK_SCALE = 0.62
// Minigalaxy's artwork carries a transparent margin, so it draws larger to match the other marks.
const ART_SCALE = 0.86
const TILE_RADIUS = 0.22

function Tile({ size, bg, children }: { size: number; bg: string; children: React.ReactNode }) {
  return (
    <Box
      sx={{
        width: size,
        height: size,
        borderRadius: `${Math.round(size * TILE_RADIUS)}px`,
        bgcolor: bg,
        display: 'grid',
        placeItems: 'center',
        flexShrink: 0,
      }}
    >
      {children}
    </Box>
  )
}

function Mark({ path, size, fill }: { path: string; size: number; fill: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill={fill}>
      <path d={path} />
    </svg>
  )
}

export function LauncherLogo({ id, size }: { id: string; size: number }) {
  if (id === 'minigalaxy') {
    return (
      <Tile size={size} bg="#3B4A8C">
        <Box
          component="img"
          src={minigalaxy}
          alt=""
          width={size * ART_SCALE}
          height={size * ART_SCALE}
        />
      </Tile>
    )
  }
  const tile = TILES[id]
  if (!tile) {
    return null
  }
  const logo = (
    <Tile size={size} bg={tile.bg}>
      <Mark path={tile.path} size={Math.round(size * MARK_SCALE)} fill="#FFFFFF" />
    </Tile>
  )
  if (!FLATPAK.has(id)) {
    return logo
  }
  const badge = Math.round(size * BADGE_SCALE)
  return (
    <Box sx={{ position: 'relative', width: size, height: size }}>
      {logo}
      <Box
        sx={{
          position: 'absolute',
          right: -badge * BADGE_OVERHANG,
          bottom: -badge * BADGE_OVERHANG,
          width: badge + BADGE_RING,
          height: badge + BADGE_RING,
          display: 'grid',
          placeItems: 'center',
          borderRadius: '50%',
          bgcolor: '#1E1E26',
        }}
      >
        <Mark path={siFlatpak.path} size={badge} fill={`#${siFlatpak.hex}`} />
      </Box>
    </Box>
  )
}
