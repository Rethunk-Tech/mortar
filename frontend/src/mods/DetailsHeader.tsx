import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { IconAction } from '../shell/IconAction.tsx'

// The details panel's header: picture, name, a line under it, any controls, and Close.
export function DetailsHeader({
  picture,
  title,
  subtitle,
  controls,
  onClose,
}: {
  picture: ReactNode
  title: string
  subtitle: ReactNode
  controls?: ReactNode
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
      {picture}
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 16, fontWeight: 700, overflowWrap: 'anywhere' }}>
          {title}
        </Typography>
        <Typography
          component="div"
          sx={{ fontSize: 13, color: 'text.secondary', overflowWrap: 'anywhere' }}
        >
          {subtitle}
        </Typography>
      </Box>
      {controls}
      <IconAction label={t`Close details`} icon={<X size={18} />} onClick={onClose} />
    </Box>
  )
}
