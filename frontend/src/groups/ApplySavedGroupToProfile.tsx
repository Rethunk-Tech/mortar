import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  TextField,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { SavedGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { Apply } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { useProfiles } from '../profiles/store.ts'
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
      setProfileId(useProfiles.getState().openId || (profiles[0]?.id ?? ''))
    }
  }, [open, profiles])
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
    <Dialog open={open} onClose={busy ? undefined : onClose}>
      <DialogTitle>{t`Add ${{ name: group?.name ?? '' }} to a profile`}</DialogTitle>
      <DialogContent sx={{ minWidth: 340 }}>
        <TextField
          select={true}
          fullWidth={true}
          margin="dense"
          label={t`Profile`}
          value={profileId}
          onChange={(e) => setProfileId(e.target.value)}
        >
          {profiles.map((profile) => (
            <MenuItem key={profile.id} value={profile.id}>
              {profile.name}
            </MenuItem>
          ))}
        </TextField>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button variant="contained" onClick={apply} disabled={busy || profileId === ''}>
          {t`Add to profile`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
