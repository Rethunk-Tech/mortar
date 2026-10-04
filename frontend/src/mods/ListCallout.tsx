import { Box, Typography } from '@mui/material'
import { Info } from 'lucide-react'
import type { ReactNode } from 'react'
import { calloutFill, calloutLine } from '../theme/callout.ts'

// A box above the mod list that asks the user about the library: what happened, the choices, and optional detail.
export function ListCallout({
  text,
  actions,
  children,
}: {
  text: string
  actions: ReactNode
  children?: ReactNode
}) {
  return (
    <Box
      sx={{
        mx: 2,
        mt: 1.25,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        p: 1.5,
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flexWrap: 'wrap' }}>
        <Box component="span" sx={{ display: 'flex', flexShrink: 0, color: 'info.main' }}>
          <Info size={16} aria-hidden={true} />
        </Box>
        <Typography sx={{ flex: 1, minWidth: 200, fontSize: 14 }}>{text}</Typography>
        <Box sx={{ display: 'flex', gap: 1, flexShrink: 0 }}>{actions}</Box>
      </Box>
      {children}
    </Box>
  )
}
