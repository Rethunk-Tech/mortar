import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { PackagePlus } from 'lucide-react'
import { memo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { SavedGroup } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import {
  Delete,
  Rename,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { idKey } from '../mods/dependents.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { ApplySavedGroupToProfile } from './ApplySavedGroupToProfile.tsx'
import { SavedGroupNameDialog } from './dialogs.tsx'
import { SavedGroupRow } from './SavedGroupRow.tsx'
import { useSavedGroups } from './useSavedGroups.ts'

function hasSavedGroup(profile: Profile, group: SavedGroup) {
  const installed = new Set(
    (profile.entries ?? []).flatMap((entry) => (entry.mods ?? []).map((mod) => idKey(mod.id))),
  )
  // Optional files have no UniqueID and travel with their main mod.
  return (group.mods ?? []).every(
    (mod) => (mod.overlayOf ?? '') !== '' || installed.has(idKey(mod.id)),
  )
}

function holderNamesOf(profiles: Profile[], group: SavedGroup) {
  return profiles
    .filter((profile) => hasSavedGroup(profile, group))
    .map((profile) => profile.name)
    .join(', ')
}

function NoSavedGroups() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<PackagePlus size={28} />} title={t`No saved groups yet.`}>
      {t`Save a group from the Mods tab to reuse it in other profiles.`}
    </EmptyState>
  )
}

export const SavedGroupsSection = memo(function SavedGroupsSectionView({
  game,
  profiles,
}: {
  game: string
  profiles: Profile[]
}) {
  const { t } = useLingui()
  const { savedGroups, setSavedGroups, loading } = useSavedGroups(game)
  const [renaming, setRenaming] = useState<SavedGroup | null>(null)
  const [deleting, setDeleting] = useState<SavedGroup | null>(null)
  const [applying, setApplying] = useState<SavedGroup | null>(null)
  const [busy, run] = usePending()
  return (
    <>
      <Box
        component="aside"
        aria-label={t`Saved groups`}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
          p: space.pad,
          bgcolor: 'var(--mortar-panel)',
          borderRadius: '8px',
          // The panel's own padding is the empty state's room below.
          '& > .empty-state': { pb: 0 },
        }}
      >
        <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700 }}>
          {t`Saved groups`}
        </Typography>
        {loading ? <LoadingRow>{t`Loading saved groups…`}</LoadingRow> : null}
        {!loading && savedGroups.length === 0 ? <NoSavedGroups /> : null}
        {savedGroups.map((group) => (
          <SavedGroupRow
            key={group.id}
            game={game}
            group={group}
            holderNames={holderNamesOf(profiles, group) || t`None`.toLowerCase()}
            onApply={() => setApplying(group)}
            onRename={() => setRenaming(group)}
            onDelete={() => setDeleting(group)}
            onChanged={(updated) =>
              setSavedGroups((current) => current.map((g) => (g.id === updated.id ? updated : g)))
            }
          />
        ))}
      </Box>
      <ApplySavedGroupToProfile
        group={applying}
        game={game}
        profiles={profiles}
        onClose={() => setApplying(null)}
      />
      <SavedGroupNameDialog
        open={renaming !== null}
        title={t`Rename saved group`}
        initialName={renaming?.name ?? ''}
        submitLabel={t`Save`}
        errorTitle={t`Could not rename the saved group`}
        onClose={() => setRenaming(null)}
        onSubmit={async (name) => {
          if (!renaming) {
            return
          }
          const updated = await Rename(game, renaming.id, name)
          setSavedGroups((current) => current.map((g) => (g.id === updated.id ? updated : g)))
          setRenaming(null)
        }}
      />
      <ConfirmDialog
        open={deleting !== null}
        title={t`Delete ${deleting?.name ?? ''}?`}
        body={t`This saved group will be deleted. Groups already added to profiles stay.`}
        confirmLabel={t`Delete`}
        color="error"
        busy={busy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting === null) {
            return
          }
          const target = deleting
          run(
            () =>
              Delete(game, target.id).then(() => {
                setSavedGroups((current) => current.filter((item) => item.id !== target.id))
                setDeleting(null)
                useToasts.getState().push({ kind: 'success', title: t`Saved group deleted` })
              }),
            { errorTitle: t`Could not delete the saved group` },
          )
        }}
      />
    </>
  )
})
