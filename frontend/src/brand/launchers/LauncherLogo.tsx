import { Box } from '@mui/material'
import { siFlatpak, siGogdotcom, siHeroicgameslauncher, siLutris, siSteam } from 'simple-icons'
import minigalaxy from '../vendor/minigalaxy.png'

const ICONS: Record<string, { path: string }> = {
  steam: siSteam,
  'flatpak-steam': siSteam,
  heroic: siHeroicgameslauncher,
  lutris: siLutris,
  gog: siGogdotcom,
}

// Launchers that run as a Flatpak carry its mark, so they read apart from the same launcher installed natively.
const FLATPAK = new Set(['flatpak-steam'])
const BADGE_SCALE = 0.5
// The badge overhangs the logo's corner by a quarter of its size, inside a dark ring this many pixels wide.
const BADGE_OVERHANG = 0.25
const BADGE_RING = 4

function Mark({ path, size, fill }: { path: string; size: number; fill: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill={fill}>
      <path d={path} />
    </svg>
  )
}

// Raster marks, drawn as a white silhouette to match the vector ones.
const IMAGES: Record<string, string> = { minigalaxy }

export function LauncherLogo({ id, size }: { id: string; size: number }) {
  const image = IMAGES[id]
  if (image) {
    return (
      <Box
        component="img"
        src={image}
        alt=""
        width={size}
        height={size}
        sx={{ filter: 'brightness(0) invert(1)' }}
      />
    )
  }
  const icon = ICONS[id]
  if (!icon) {
    return null
  }
  if (!FLATPAK.has(id)) {
    return <Mark path={icon.path} size={size} fill="var(--mortar-ink)" />
  }
  const badge = Math.round(size * BADGE_SCALE)
  return (
    <Box sx={{ position: 'relative', width: size, height: size }}>
      <Mark path={icon.path} size={size} fill="var(--mortar-ink)" />
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
