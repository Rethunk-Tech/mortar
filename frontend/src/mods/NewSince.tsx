import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'

const DOT = 7

export function NewDot({ show }: { show: boolean }) {
  const { t } = useLingui()
  if (!show) {
    return null
  }
  return (
    <Box
      aria-label={t`New`}
      sx={{
        position: 'absolute',
        top: 2,
        right: 2,
        width: DOT,
        height: DOT,
        borderRadius: '50%',
        bgcolor: 'primary.main',
        pointerEvents: 'none',
      }}
    />
  )
}

export function NewSinceLooked({ show }: { show: boolean }) {
  const { t } = useLingui()
  if (!show) {
    return null
  }
  return (
    <Typography
      component="span"
      sx={{ fontSize: 11, color: 'primary.main', whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {t`New since you last looked`}
    </Typography>
  )
}
