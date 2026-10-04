import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { InlineError } from '../toasts/report.ts'

export function LoadingRow({ children }: { children: ReactNode }) {
  return (
    <Box
      role="status"
      sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 1, p: 2 }}
    >
      <CircularProgress size={16} aria-hidden={true} />
      <Typography sx={{ color: 'text.secondary' }}>{children}</Typography>
    </Box>
  )
}

export function LoadErrorRow({ error, onRetry }: { error: InlineError; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ p: 3, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1 }}>
      <Typography role="alert" title={error.details}>
        {error.message}
      </Typography>
      <Button variant="contained" onClick={onRetry}>
        {t`Retry`}
      </Button>
    </Box>
  )
}
