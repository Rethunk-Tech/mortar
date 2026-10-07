import { useLingui } from '@lingui/react/macro'
import { Box, IconButton, Tooltip, Typography } from '@mui/material'
import { Check } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { colorHex, PROFILE_COLORS, PROFILE_ICONS } from './appearance.ts'
import { ProfileMark } from './ProfileMark.tsx'

export function AppearancePickers({
  profile,
  color,
  icon,
  onColor,
  onIcon,
}: {
  profile: Profile
  color: string
  icon: string
  onColor: (next: string) => void
  onIcon: (next: string) => void
}) {
  const { t } = useLingui()
  const colorName = (token: string) => {
    switch (token) {
      case 'rose':
        return t`Rose`
      case 'orange':
        return t`Orange`
      case 'gold':
        return t`Gold`
      case 'lime':
        return t`Lime`
      case 'teal':
        return t`Teal`
      case 'sky':
        return t`Sky`
      case 'violet':
        return t`Violet`
      case 'pink':
        return t`Pink`
      default:
        return token
    }
  }
  const iconName = (token: string) => {
    switch (token) {
      case 'sprout':
        return t`Sprout`
      case 'leaf':
        return t`Leaf`
      case 'wheat':
        return t`Wheat`
      case 'fish':
        return t`Fish`
      case 'hammer':
        return t`Hammer`
      case 'pickaxe':
        return t`Pickaxe`
      case 'star':
        return t`Star`
      case 'heart':
        return t`Heart`
      case 'mountain':
        return t`Mountain`
      case 'sun':
        return t`Sun`
      case 'moon':
        return t`Moon`
      case 'sparkles':
        return t`Sparkles`
      default:
        return token
    }
  }
  return (
    <>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Colour`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_COLORS.map((token) => (
          <Tooltip key={token} title={colorName(token)}>
            <IconButton
              aria-label={colorName(token)}
              aria-pressed={color === token}
              onClick={() => onColor(color === token ? '' : token)}
              sx={{
                width: 32,
                height: 32,
                bgcolor: colorHex(token),
                outline: color === token ? '2px solid var(--mortar-ink)' : '2px solid transparent',
                outlineOffset: 2,
                '&:hover': { bgcolor: colorHex(token) },
              }}
            >
              {color === token ? <Check size={16} color="#1a1a1a" aria-hidden={true} /> : null}
            </IconButton>
          </Tooltip>
        ))}
      </Box>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Icon`}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 2 }}>
        {PROFILE_ICONS.map((name) => (
          <Tooltip key={name} title={iconName(name)}>
            <IconButton
              aria-label={iconName(name)}
              aria-pressed={icon === name}
              onClick={() => onIcon(icon === name ? '' : name)}
              sx={{
                width: 36,
                height: 36,
                borderRadius: '6px',
                bgcolor: icon === name ? 'var(--mortar-hairline-12)' : 'transparent',
                outline: icon === name ? '2px solid var(--mortar-ink)' : '2px solid transparent',
                outlineOffset: 1,
              }}
            >
              <ProfileMark profile={{ ...profile, color, icon: name }} size={28} />
            </IconButton>
          </Tooltip>
        ))}
      </Box>
    </>
  )
}
