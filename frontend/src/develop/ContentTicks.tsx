import { Box, Checkbox, FormControlLabel } from '@mui/material'
import type { Param } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'

// The content folders a project carries, as ticks: few enough to show every one.
export function ContentTicks({
  param,
  value,
  set,
}: {
  param: Param
  value: string
  set: (v: string) => void
}) {
  const choices = param.choices ?? []
  const picked = value.split('|').filter(Boolean)
  const tick = (id: string, on: boolean) =>
    set(
      choices
        .map((c) => c.value)
        .filter((v) => (v === id ? on : picked.includes(v)))
        .join('|'),
    )
  return (
    <Field label={param.label} hint={param.description}>
      <Box
        role="group"
        aria-label={param.label}
        sx={{ display: 'flex', flexWrap: 'wrap', columnGap: space.gap }}
      >
        {choices.map((c) => (
          <FormControlLabel
            key={c.value}
            label={c.label}
            control={
              <Checkbox
                size="small"
                checked={picked.includes(c.value)}
                onChange={(_, on) => tick(c.value, on)}
              />
            }
          />
        ))}
      </Box>
    </Field>
  )
}
