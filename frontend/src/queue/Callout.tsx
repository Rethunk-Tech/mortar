import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { accent } from '../mods/paper.ts'
import { LetterTile } from '../mods/parts.tsx'
import { useProfiles } from '../profiles/store.ts'
import { profileOf, tile } from './totals.ts'

// The item's name with the profile it installs into under it; the sheet is too narrow to fit both on one line.
export function Title({ item, size }: { item: Item; size: number }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => profileOf(item, s.game?.id, s.profiles))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
      <Typography
        noWrap={true}
        title={item.name || item.fileName}
        sx={{ fontSize: size, fontWeight: 600 }}
      >
        {item.name || item.fileName}
      </Typography>
      {profile === null ? null : (
        <Typography noWrap={true} title={profile} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {profile ? t`into ${profile}` : t`into a deleted profile`}
        </Typography>
      )}
    </Box>
  )
}

// A card for an item that waits for the user, with its own actions on the right and any choices under the text.
export function Callout({
  item,
  label,
  text,
  actions,
  children,
}: {
  item: Item
  label: string
  text: string
  actions: ReactNode
  children?: ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        p: '14px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: 'primary.main',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <LetterTile mod={tile(item)} size={44} />
        <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Typography
            sx={{
              fontSize: 12,
              fontWeight: 700,
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              color: 'primary.main',
            }}
          >
            {label}
          </Typography>
          <Title item={item} size={16} />
        </Box>
        <Box sx={{ flexShrink: 0 }}>{actions}</Box>
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>{text}</Typography>
      {children}
    </Box>
  )
}
