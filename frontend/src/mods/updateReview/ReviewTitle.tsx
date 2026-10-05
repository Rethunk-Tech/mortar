import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, DialogTitle, IconButton, Typography } from '@mui/material'
import { ArrowUp, X } from 'lucide-react'
import { ICON_FILL, ICON_TILE } from './constants.ts'

export function ReviewTitle({
  count,
  profileName,
  checkedLabel,
  onClose,
}: {
  count: number
  profileName: string
  checkedLabel: string
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogTitle component="div" sx={{ display: 'flex', alignItems: 'center', gap: 1.75, p: 3 }}>
      <Box
        sx={{
          width: ICON_TILE,
          height: ICON_TILE,
          display: 'grid',
          placeItems: 'center',
          borderRadius: '10px',
          color: 'var(--mortar-accent-ink)',
          bgcolor: (th) => alpha(th.palette.primary.main, ICON_FILL),
        }}
      >
        <ArrowUp size={22} aria-hidden={true} />
      </Box>
      <Box sx={{ flexGrow: 1 }}>
        <Typography component="h2" sx={{ fontSize: 22, fontWeight: 700 }}>
          {t`${plural(count, { one: '# update', other: '# updates' })} for ${profileName}`}
        </Typography>
        <Typography sx={{ fontSize: 13 }}>{checkedLabel}</Typography>
      </Box>
      <IconButton aria-label={t`Close`} onClick={onClose}>
        <X size={16} />
      </IconButton>
    </DialogTitle>
  )
}
