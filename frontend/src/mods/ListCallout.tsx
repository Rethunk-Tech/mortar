import { Alert, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

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
    <Alert
      variant="compact"
      severity="info"
      role="region"
      aria-label={text}
      action={actions}
      className="list-notice"
      sx={{
        mx: space.gutter,
        mt: 1.25,
        mb: space.gap,
        // Notices stacked above the list sit one gap apart, like the list's rows.
        '.list-notice + &': { mt: 0 },
        flexShrink: 0,
        flexWrap: 'wrap',
        '& .MuiAlert-message': { flex: 1, minWidth: 200 },
        '& .MuiAlert-action': { gap: space.gap, flexWrap: 'wrap', ml: 'auto' },
      }}
    >
      <Typography sx={{ fontSize: 14 }}>{text}</Typography>
      {children}
    </Alert>
  )
}
