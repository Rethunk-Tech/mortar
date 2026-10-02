import { Box, type SxProps, type Theme } from '@mui/material'
import type { ReactNode } from 'react'

export function SettingsSection({
  title,
  description,
  children,
  sx,
}: {
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  sx?: SxProps<Theme>
}) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, ...sx }}>
      {title ? (
        <Box sx={{ fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>{title}</Box>
      ) : null}
      {description ? <Box sx={{ fontSize: 13, color: 'text.secondary' }}>{description}</Box> : null}
      <Box
        sx={{
          bgcolor: 'rgba(0,0,0,0.25)',
          borderRadius: 1,
          overflow: 'hidden',
          '& > * + *': { borderTop: '1px solid rgba(255,255,255,0.1)' },
        }}
      >
        {children}
      </Box>
    </Box>
  )
}

export function SettingRow({
  label,
  description,
  children,
}: {
  label: ReactNode
  description?: ReactNode
  children: ReactNode
}) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, minHeight: 58, px: 2, py: 1 }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: 14 }}>{label}</Box>
        {description ? (
          <Box
            sx={{
              fontSize: 12,
              color: 'text.secondary',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {description}
          </Box>
        ) : null}
      </Box>
      <Box sx={{ flexShrink: 0, display: 'flex', alignItems: 'center' }}>{children}</Box>
    </Box>
  )
}
