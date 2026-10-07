import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel } from '@mui/material'
import { useProfiles } from '../profiles/store.ts'
import { offersFomod, type ShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

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
  const profileId = useShareDialog((s) => s.profileId)
  const fomod = useProfiles((s) => offersFomod(s.profiles.find((p) => p.id === profileId)?.entries))
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
      {fomod ? row('fomodChoices', t`Include FOMOD choices`) : null}
      {row('notes', file ? t`Include notes` : t`Include mod notes`)}
      {file ? row('configFiles', t`Include config files`) : null}
    </Box>
  )
}
