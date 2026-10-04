import { siGithub, siThunderstore } from 'simple-icons'
import nexusmods from '../vendor/nexusmods.svg'

// GitHub's brand mark is black, so it uses the ink colour (its own dark-mode treatment); the others keep their colour.
const ICONS: Record<string, { path: string; fill: string }> = {
  GitHub: { path: siGithub.path, fill: 'var(--mortar-ink)' },
  Thunderstore: { path: siThunderstore.path, fill: `#${siThunderstore.hex}` },
}

// The Nexus artwork leaves a margin inside its square that the vector marks do not, so it is drawn larger to match.
const NEXUS_SCALE = 1.2

export function SourceLogo({ name, size }: { name: string; size: number }) {
  if (name === 'Nexus') {
    const drawn = Math.round(size * NEXUS_SCALE)
    return (
      <img
        src={nexusmods}
        alt=""
        width={drawn}
        height={drawn}
        style={{ margin: (size - drawn) / 2 }}
      />
    )
  }
  const icon = ICONS[name]
  if (!icon) {
    return null
  }
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill={icon.fill}>
      <path d={icon.path} />
    </svg>
  )
}
