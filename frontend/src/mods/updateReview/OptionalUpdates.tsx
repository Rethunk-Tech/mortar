import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel } from '@mui/material'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { optionalUpdateWants, useOptionalSkips } from '../optionalFiles.ts'

/** "Also update N optional files" on an update row whose optional files have newer versions on Nexus. */
export function OptionalUpdates({ update, profileId }: { update: Update; profileId: string }) {
  const { i18n } = useLingui()
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const files = useNexusDetails((s) => s.byId[update.nexusId]?.details?.files)
  const skipped = useOptionalSkips((s) => s.skipped[update.key] === true)
  const setSkipped = useOptionalSkips((s) => s.setSkipped)
  const n = update.githubRepo ? 0 : optionalUpdateWants(profile, update, files ?? []).length
  if (n === 0) {
    return null
  }
  return (
    <FormControlLabel
      control={
        <Checkbox
          size="small"
          checked={!skipped}
          onChange={(_, on) => setSkipped(update.key, !on)}
        />
      }
      label={i18n._(
        plural(n, { one: 'Also update # optional file', other: 'Also update # optional files' }),
      )}
      sx={{ alignSelf: 'flex-start', '& .MuiFormControlLabel-label': { fontSize: 13 } }}
    />
  )
}
