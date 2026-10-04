import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'

export function HistoryToolbar({
  busy,
  onMark,
  onRestore,
}: {
  busy: boolean
  onMark: () => void
  onRestore: () => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', gap: 1, mb: 1, flexWrap: 'wrap' }}>
      <Button size="small" disabled={busy} onClick={onMark} sx={{ whiteSpace: 'nowrap' }}>
        {t`Mark known good`}
      </Button>
      <Button size="small" disabled={busy} onClick={onRestore} sx={{ whiteSpace: 'nowrap' }}>
        {t`Restore last known good`}
      </Button>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', alignSelf: 'center' }}>
        {t`Pick two to compare`}
      </Typography>
    </Box>
  )
}
