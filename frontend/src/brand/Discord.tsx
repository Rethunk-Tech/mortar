import { siDiscord } from 'simple-icons'

function DiscordLogo({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill={`#${siDiscord.hex}`}
    >
      <path d={siDiscord.path} />
    </svg>
  )
}

export { DiscordLogo }
