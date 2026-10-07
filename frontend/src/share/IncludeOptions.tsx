import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { useProfiles } from '../profiles/store.ts'
import { type IncludeTarget, includeAvailability } from './methods.ts'
import { offersFomod, type ShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

export function IncludeOptions({
  value,
  onChange,
  target,
}: {
  value: ShareInclude
  onChange: (next: ShareInclude) => void
  target: IncludeTarget
}) {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const fomod = useProfiles((s) => offersFomod(s.profiles.find((p) => p.id === profileId)?.entries))
  const can = includeAvailability(target)
  const reasons = {
    'file-only': t`A link has no room for it. Use a file or a nearby computer.`,
    'paired-only': t`Only a paired computer gets it.`,
  }
  const row = (key: keyof ShareInclude, label: string) => {
    const why = can[key]
    return (
      <FormControlLabel
        key={key}
        disabled={why !== null}
        control={
          <Checkbox
            checked={why === null && value[key]}
            onChange={(_, on) => onChange({ ...value, [key]: on })}
          />
        }
        label={
          <Box>
            <Typography component="span" sx={{ fontSize: 'inherit' }}>
              {label}
            </Typography>
            {why ? (
              <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{reasons[why]}</Typography>
            ) : null}
          </Box>
        }
      />
    )
  }
  const file = target !== 'link'
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
      {row('disabledMods', t`Include disabled mods`)}
      {fomod ? row('fomodChoices', t`Include FOMOD choices`) : null}
      {row('notes', file ? t`Include notes` : t`Include mod notes`)}
      {row('configFiles', t`Include config files`)}
      {row('problemChoices', t`Include problem choices`)}
    </Box>
  )
}
