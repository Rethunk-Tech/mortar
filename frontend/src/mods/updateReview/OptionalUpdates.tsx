import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel } from '@mui/material'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { profileById, useProfiles } from '../../profiles/store.ts'
import { nexusEntryOf, useNexusDetails } from '../nexusDetails.ts'
import { optionalUpdateWants, useOptionalSkips } from '../optionalFiles.ts'

/** "Also update N optional files" on an update row whose optional files have newer versions on Nexus. */
export function OptionalUpdates({ update, profileId }: { update: Update; profileId: string }) {
  const { i18n } = useLingui()
  const profile = useProfiles(profileById(profileId))
  const game = useProfiles((s) => s.game?.id ?? '')
  const files = useNexusDetails((s) => nexusEntryOf(s.byKey, game, update.nexusId)?.details?.files)
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
        msg`${plural(
          { count: n },
          { one: 'Also update # optional file', other: 'Also update # optional files' },
        )}`,
      )}
      sx={{
        alignSelf: 'flex-start',
        '& .MuiFormControlLabel-label': { fontSize: 'base.fontSize' },
      }}
    />
  )
}
