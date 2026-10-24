import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel } from '@mui/material'

/** A caution row is left out of Update all, and its Update stays off, until the user accepts the caution. */
export function CautionAck({ acked, onAck }: { acked: boolean; onAck: (on: boolean) => void }) {
  const { t } = useLingui()
  return (
    <FormControlLabel
      control={<Checkbox size="small" checked={acked} onChange={(_, on) => onAck(on)} />}
      label={t`Update despite this`}
      sx={{
        alignSelf: 'flex-start',
        ml: -0.75,
        '& .MuiFormControlLabel-label': { fontSize: 'small.fontSize' },
      }}
    />
  )
}
