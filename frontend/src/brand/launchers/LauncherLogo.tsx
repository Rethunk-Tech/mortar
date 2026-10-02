import { siGogdotcom, siHeroicgameslauncher, siLutris, siSteam } from 'simple-icons'

const ICONS: Record<string, { path: string }> = {
  steam: siSteam,
  'flatpak-steam': siSteam,
  heroic: siHeroicgameslauncher,
  lutris: siLutris,
  gog: siGogdotcom,
}

export function LauncherLogo({ id, size }: { id: string; size: number }) {
  const icon = ICONS[id]
  if (!icon) {
    return null
  }
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill="#fff">
      <path d={icon.path} />
    </svg>
  )
}
