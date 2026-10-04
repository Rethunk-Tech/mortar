import { Alert, Typography } from '@mui/material'
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
    <Alert
      severity="info"
      role="region"
      aria-label={text}
      action={actions}
      sx={{
        mx: 2,
        mt: 1.25,
        flexShrink: 0,
        alignItems: 'center',
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        '& .MuiAlert-message': { flex: 1, minWidth: 200 },
        '& .MuiAlert-action': { gap: 1, flexShrink: 0, alignItems: 'center' },
      }}
    >
      <Typography sx={{ fontSize: 14 }}>{text}</Typography>
      {children}
    </Alert>
  )
}
