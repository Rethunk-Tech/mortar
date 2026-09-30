import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import { useLocked } from './useLocked.ts'

export function LockedNote() {
  const { t } = useLingui()
  return useLocked() ? (
    <Typography sx={{ px: 2, fontSize: 13, color: 'text.secondary' }}>
      {t`Stop the game to change mods.`}
    </Typography>
  ) : null
}
