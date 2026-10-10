import { siDiscord } from 'simple-icons'
import { BrandMark } from './BrandMark.tsx'

function DiscordLogo({ size }: { size: number }) {
  return <BrandMark path={siDiscord.path} size={size} fill={`#${siDiscord.hex}`} />
}

export { DiscordLogo }
