import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { PackagePlus, Pencil, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Bundle } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import {
  Delete,
  List as ListBundles,
  Rename,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { BundleNameDialog } from './dialogs.tsx'

function hasBundle(profile: Profile, bundle: Bundle) {
  const installed = new Set(
    (profile.entries ?? []).flatMap((entry) =>
      (entry.mods ?? []).map((mod) => mod.uniqueId.toLowerCase()),
    ),
  )
  return (bundle.mods ?? []).every((mod) => installed.has(mod.uniqueId.toLowerCase()))
}

function NoBundles() {
  const { t } = useLingui()
  return (
    <EmptyState compact={true} icon={<PackagePlus size={28} />} title={t`No bundles yet.`}>
      {t`Create a bundle to reuse a set of mods.`}
    </EmptyState>
  )
}

function useBundleList(game: string) {
  const { t } = useLingui()
  const [bundles, setBundles] = useState<Bundle[]>([])
  useEffect(() => {
    let active = true
    ListBundles(game)
      .then((listed) => {
        if (active) {
          setBundles(listed ?? [])
        }
      })
      .catch((error: unknown) => {
        if (active) {
          reportError(t`Could not read bundles`)(error)
        }
      })
    return () => {
      active = false
    }
  }, [game, t])
  return [bundles, setBundles] as const
}

export function BundlesSection({ game, profiles }: { game: string; profiles: Profile[] }) {
  const { t } = useLingui()
  const [bundles, setBundles] = useBundleList(game)
  const [renaming, setRenaming] = useState<Bundle | null>(null)
  const [deleting, setDeleting] = useState<Bundle | null>(null)
  const [busy, run] = usePending()
  return (
    <>
      <Box
        component="aside"
        aria-label={t`Bundles`}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1.25,
          p: 2,
          bgcolor: 'var(--mortar-panel)',
          borderRadius: '8px',
        }}
      >
        <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700 }}>
          {t`Bundles`}
        </Typography>
        {bundles.length === 0 ? (
          <NoBundles />
        ) : (
          bundles.map((bundle) => {
            const holders = profiles.filter((profile) => hasBundle(profile, bundle))
            const modCount = plural(bundle.mods?.length ?? 0, { one: '# mod', other: '# mods' })
            const holderNames =
              holders.length === 0 ? t`none` : holders.map((profile) => profile.name).join(', ')
            return (
              <Box
                key={bundle.id}
                sx={{ p: 1.25, bgcolor: 'var(--mortar-raised)', borderRadius: '6px' }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <Typography
                    noWrap={true}
                    title={bundle.name}
                    sx={{ flex: 1, minWidth: 0, fontWeight: 600 }}
                  >
                    {bundle.name}
                  </Typography>
                  <TipIconButton
                    label={t`Rename ${bundle.name}`}
                    onClick={() => setRenaming(bundle)}
                  >
                    <Pencil size={15} />
                  </TipIconButton>
                  <TipIconButton
                    label={t`Delete ${bundle.name}`}
                    color="error"
                    onClick={() => setDeleting(bundle)}
                  >
                    <Trash2 size={15} />
                  </TipIconButton>
                </Box>
                <Typography sx={{ color: 'text.secondary', fontSize: 12 }}>{modCount}</Typography>
                <Typography
                  sx={{ color: 'text.secondary', fontSize: 12 }}
                  noWrap={true}
                  title={holderNames}
                >
                  {t`Profiles: ${holderNames}`}
                </Typography>
              </Box>
            )
          })
        )}
      </Box>
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
        body={t`This bundle will be removed from Mortar.`}
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
