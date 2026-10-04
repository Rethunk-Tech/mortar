import { plural } from '@lingui/core/macro'
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
import type { Bundle } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/models.ts'
import { Apply } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { bundleWants } from './missingWants.ts'

// Adds a saved bundle to a chosen profile from the Bundles panel, the other way round from a profile's menu.
export function ApplyBundleToProfile({
  bundle,
  game,
  profiles,
  onClose,
}: {
  bundle: Bundle | null
  game: string
  profiles: Profile[]
  onClose: () => void
}) {
  const { t } = useLingui()
  const [profileId, setProfileId] = useState('')
  const [busy, run] = usePending()
  const open = bundle !== null
  useEffect(() => {
    if (open) {
      setProfileId(useProfiles.getState().openId || (profiles[0]?.id ?? ''))
    }
  }, [open, profiles])
  const apply = () => {
    if (!bundle) {
      return
    }
    run(
      async () => {
        const result = await Apply(game, bundle.id, profileId)
        useProfiles.getState().replace(result.profile)
        if (useProfiles.getState().openId === profileId) {
          useMods.getState().load().catch(reportUnexpected)
        }
        onClose()
        const missing = result.missing ?? []
        const added = plural(result.added, { one: '# mod added', other: '# mods added' })
        const wants = bundleWants(result.missingMods)
        useToasts.getState().push({
          kind: missing.length > 0 ? 'warning' : 'success',
          title: t`Bundle added`,
          body:
            missing.length > 0
              ? `${added}\n${t`Not downloaded yet: ${missing.join(', ')}`}`
              : added,
          ...(wants.length > 0
            ? {
                action: {
                  label: t`Download them`,
                  run: () => download(wants, true).catch(reportUnexpected),
                },
              }
            : {}),
        })
      },
      { errorTitle: t`Could not add the bundle` },
    )
  }
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose}>
      <DialogTitle>{t`Add ${bundle?.name ?? ''} to a profile`}</DialogTitle>
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
