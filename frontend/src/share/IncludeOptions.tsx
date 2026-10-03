import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel } from '@mui/material'
import type { ShareInclude } from './shareDefaults.ts'

export function IncludeOptions({
  value,
  onChange,
  file,
}: {
  value: ShareInclude
  onChange: (next: ShareInclude) => void
  file: boolean
}) {
  const { t } = useLingui()
  const row = (key: keyof ShareInclude, label: string) => (
    <FormControlLabel
      key={key}
      control={
        <Checkbox checked={value[key]} onChange={(_, on) => onChange({ ...value, [key]: on })} />
      }
      label={label}
    />
  )
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
      {row('disabledMods', t`Include disabled mods`)}
      {row('fomodChoices', t`Include FOMOD choices`)}
      {row('notes', t`Include notes`)}
      {file ? row('configFiles', t`Include config files`) : null}
    </Box>
  )
}
