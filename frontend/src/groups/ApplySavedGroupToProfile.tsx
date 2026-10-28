import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { SavedGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { Apply } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { FormDialog, SelectField } from '../shell/dialog/FormDialog.tsx'
import { usePending } from '../toasts/usePending.ts'
import { savedGroupApplied } from './applied.ts'

// Adds a saved group to a chosen profile from the Saved groups panel, the other way round from a profile's menu.
export function ApplySavedGroupToProfile({
  group,
  game,
  profiles,
  onClose,
}: {
  group: SavedGroup | null
  game: string
  profiles: Profile[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const [profileId, setProfileId] = useState('')
  const [busy, run] = usePending()
  const open = group !== null
  useEffect(() => {
    if (open) {
      const state = useProfiles.getState()
      setProfileId(state.openId || (state.profiles[0]?.id ?? ''))
    }
  }, [open])
  const apply = () => {
    if (!group) {
      return
    }
    run(
      async () => {
        const result = await Apply(game, group.id, profileId)
        onClose()
        savedGroupApplied(result, profileId)
      },
      { errorTitle: t`Could not add the saved group` },
    )
  }
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Add ${{ name: group?.name ?? '' }} to a profile`}
      submitLabel={t`Add to profile`}
      issue={profileId === '' ? t`Pick a profile.` : ''}
      dirty={false}
      busy={busy}
      onSubmit={apply}
    >
      <SelectField label={t`Profile`} value={profileId} onChange={setProfileId}>
        {profiles.map((profile) => (
          <MenuItem key={profile.id} value={profile.id}>
            {profile.name}
          </MenuItem>
        ))}
      </SelectField>
    </FormDialog>
  )
}
