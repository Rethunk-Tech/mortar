import { siGithub, siThunderstore } from 'simple-icons'
import nexusmods from './nexusmods.svg'

export function SourceLogo({ name, size }: { name: string; size: number }) {
  if (name === 'Nexus') {
    return <img src={nexusmods} alt="" width={size} height={size} />
  }
  const icon = name === 'GitHub' ? siGithub : name === 'Thunderstore' ? siThunderstore : null
  if (!icon) {
    return null
  }
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" fill="#fff">
      <path d={icon.path} />
    </svg>
  )
}
