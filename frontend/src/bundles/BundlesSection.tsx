import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { PackagePlus } from 'lucide-react'
import { useState } from 'react'
import type { Bundle } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/models.ts'
import {
  Delete,
  Rename,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { idKey } from '../mods/dependents.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { ApplyBundleToProfile } from './ApplyBundleToProfile.tsx'
import { BundleRow } from './BundleRow.tsx'
import { BundleNameDialog } from './dialogs.tsx'
import { useBundles } from './useBundles.ts'

function hasBundle(profile: Profile, bundle: Bundle) {
  const installed = new Set(
    (profile.entries ?? []).flatMap((entry) => (entry.mods ?? []).map((mod) => idKey(mod.id))),
  )
  // Optional files have no UniqueID and travel with their main mod.
  return (bundle.mods ?? []).every(
    (mod) => (mod.overlayOf ?? '') !== '' || installed.has(idKey(mod.id)),
  )
}

function holderNamesOf(profiles: Profile[], bundle: Bundle) {
  return profiles
    .filter((profile) => hasBundle(profile, bundle))
    .map((profile) => profile.name)
    .join(', ')
}

function NoBundles() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<PackagePlus size={28} />} title={t`No bundles yet.`}>
      {t`Create a bundle to reuse a set of mods.`}
    </EmptyState>
  )
}

export function BundlesSection({ game, profiles }: { game: string; profiles: Profile[] }) {
  const { t } = useLingui()
  const { bundles, setBundles, loading } = useBundles(game)
  const [renaming, setRenaming] = useState<Bundle | null>(null)
  const [deleting, setDeleting] = useState<Bundle | null>(null)
  const [applying, setApplying] = useState<Bundle | null>(null)
  const [busy, run] = usePending()
  return (
    <>
      <Box
        component="aside"
        aria-label={t`Bundles`}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
          p: space.pad,
          bgcolor: 'var(--mortar-panel)',
          borderRadius: '8px',
        }}
      >
        <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700 }}>
          {t`Bundles`}
        </Typography>
        {loading ? <LoadingRow>{t`Loading bundles…`}</LoadingRow> : null}
        {!loading && bundles.length === 0 ? <NoBundles /> : null}
        {bundles.map((bundle) => (
          <BundleRow
            key={bundle.id}
            game={game}
            bundle={bundle}
            holderNames={holderNamesOf(profiles, bundle) || t`None`.toLowerCase()}
            onApply={() => setApplying(bundle)}
            onRename={() => setRenaming(bundle)}
            onDelete={() => setDeleting(bundle)}
            onChanged={(updated) =>
              setBundles((current) => current.map((b) => (b.id === updated.id ? updated : b)))
            }
          />
        ))}
      </Box>
      <ApplyBundleToProfile
        bundle={applying}
        game={game}
        profiles={profiles}
        onClose={() => setApplying(null)}
      />
      <BundleNameDialog
        open={renaming !== null}
        title={t`Rename bundle`}
        initialName={renaming?.name ?? ''}
        submitLabel={t`Save`}
        errorTitle={t`Could not rename the bundle`}
        onClose={() => setRenaming(null)}
        onSubmit={async (name) => {
          if (!renaming) {
            return
          }
          const updated = await Rename(game, renaming.id, name)
          setBundles((current) =>
            current.map((bundle) => (bundle.id === updated.id ? updated : bundle)),
          )
          setRenaming(null)
        }}
      />
      <ConfirmDialog
        open={deleting !== null}
        title={t`Delete ${deleting?.name ?? ''}?`}
        body={t`This bundle will be deleted.`}
        confirmLabel={t`Delete`}
        color="error"
        busy={busy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting === null) {
            return
          }
          const bundle = deleting
          run(
            () =>
              Delete(game, bundle.id).then(() => {
                setBundles((current) => current.filter((item) => item.id !== bundle.id))
                setDeleting(null)
                useToasts.getState().push({ kind: 'success', title: t`Bundle deleted` })
              }),
            { errorTitle: t`Could not delete the bundle` },
          )
        }}
      />
    </>
  )
}
