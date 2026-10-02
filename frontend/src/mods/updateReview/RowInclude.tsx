import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, Typography } from '@mui/material'

export function RowInclude({
  name,
  caution,
  acked,
  included,
  onAck,
  onInclude,
}: {
  name: string
  caution: string
  acked: boolean
  included: boolean
  onAck: (on: boolean) => void
  onInclude: (on: boolean) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <Checkbox
        checked={included && (!caution || acked)}
        disabled={caution !== '' && !acked}
        onChange={(_, on) => onInclude(on)}
        slotProps={{ input: { 'aria-label': t`Include ${name}` } }}
        sx={{ justifySelf: 'center' }}
      />
      {caution ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Checkbox
            checked={acked}
            onChange={(_, on) => onAck(on)}
            slotProps={{ input: { 'aria-label': t`Confirm update for ${name}` } }}
          />
          <Typography
            sx={{ fontSize: 11, color: 'warning.main' }}
          >{t`Acknowledge caution to include`}</Typography>
        </Box>
      ) : null}
    </>
  )
}
