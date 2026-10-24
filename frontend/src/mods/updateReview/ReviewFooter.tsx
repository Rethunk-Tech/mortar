import { useLingui } from '@lingui/react/macro'
import { Box, Button, DialogActions, Typography } from '@mui/material'
import { ShieldCheck } from 'lucide-react'
import { space } from '../../theme/density.ts'

export function ReviewFooter({
  wantedCount,
  onClose,
  onUpdate,
  onEverywhere,
}: {
  wantedCount: number
  onClose: () => void
  onUpdate: () => void
  onEverywhere: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogActions
      sx={{
        px: space.pad,
        py: space.pad,
        gap: space.gap,
        bgcolor: 'var(--mortar-overlay-20)',
        '& > :not(style) ~ :not(style)': { ml: 0 },
      }}
    >
      <Button variant="text" color="inherit" onClick={onClose}>
        {t`Close`}
      </Button>
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          alignItems: 'center',
          gap: 0.75,
          color: 'text.secondary',
        }}
      >
        <ShieldCheck size={16} aria-hidden={true} style={{ flexShrink: 0 }} />
        <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'inherit' }}>
          {t`Settings are kept and saves backed up.`}
        </Typography>
      </Box>
      {wantedCount > 0 ? (
        <Button variant="text" color="inherit" onClick={onEverywhere}>
          {t`Update in every profile…`}
        </Button>
      ) : null}
      {wantedCount > 0 ? (
        <Button variant="contained" onClick={onUpdate}>
          {t`Update all (${wantedCount})`}
        </Button>
      ) : null}
    </DialogActions>
  )
}
