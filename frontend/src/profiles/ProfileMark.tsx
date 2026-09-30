import { Box } from '@mui/material'
import {
  Fish,
  Hammer,
  Heart,
  Leaf,
  type LucideIcon,
  Moon,
  Mountain,
  Pickaxe,
  Sparkles,
  Sprout,
  Star,
  Sun,
  Wheat,
} from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { colorHex, isProfileIcon, type ProfileIcon } from './appearance.ts'

const ICONS: Record<ProfileIcon, LucideIcon> = {
  sprout: Sprout,
  leaf: Leaf,
  wheat: Wheat,
  fish: Fish,
  hammer: Hammer,
  pickaxe: Pickaxe,
  star: Star,
  heart: Heart,
  mountain: Mountain,
  sun: Sun,
  moon: Moon,
  sparkles: Sparkles,
}

const ICON_SIZE = 0.55

export function ProfileMark({ profile, size = 28 }: { profile: Profile; size?: number }) {
  const hex = colorHex(profile.color)
  const Icon = isProfileIcon(profile.icon) ? ICONS[profile.icon] : undefined
  if (!(hex || Icon)) {
    return null
  }
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{
        width: size,
        height: size,
        borderRadius: '6px',
        flexShrink: 0,
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: hex ?? 'rgba(255,255,255,0.12)',
        color: hex ? '#1b1a17' : 'rgba(255,255,255,0.88)',
      }}
    >
      {Icon ? <Icon size={Math.round(size * ICON_SIZE)} /> : null}
    </Box>
  )
}
