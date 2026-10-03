import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { PackagePlus, Pencil, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Bundle } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import {
  Delete,
  List as ListBundles,
  Rename,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
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

export function BundlesSection({ game, profiles }: { game: string; profiles: Profile[] }) {
  const { t } = useLingui()
  const [bundles, setBundles] = useState<Bundle[]>([])
  const [renaming, setRenaming] = useState<Bundle | null>(null)
  const [deleting, setDeleting] = useState<Bundle | null>(null)
  const [busy, setBusy] = useState(false)
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
          useToasts.getState().push({
            kind: 'error',
            title: t`Could not read bundles`,
            body: errorMessage(error),
            detail: errorDetails(error),
          })
        }
      })
    return () => {
      active = false
    }
  }, [game, t])
  const confirmDelete = async () => {
    if (!deleting || busy) {
      return
    }
    setBusy(true)
    try {
      await Delete(game, deleting.id)
      setBundles((current) => current.filter((bundle) => bundle.id !== deleting.id))
      setDeleting(null)
      useToasts.getState().push({ kind: 'success', title: t`Bundle deleted` })
    } catch (error) {
      useToasts.getState().push({
        kind: 'error',
        title: t`Could not delete the bundle`,
        body: errorMessage(error),
        detail: errorDetails(error),
      })
    } finally {
      setBusy(false)
    }
  }
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
          bgcolor: 'rgba(40,40,48,0.78)',
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
                sx={{ p: 1.25, bgcolor: 'rgba(55,55,65,0.9)', borderRadius: '6px' }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <Typography
                    noWrap={true}
                    title={bundle.name}
                    sx={{ flex: 1, minWidth: 0, fontWeight: 600 }}
                  >
                    {bundle.name}
                  </Typography>
                  <Tooltip title={t`Rename`}>
                    <IconButton
                      aria-label={t`Rename ${bundle.name}`}
                      size="small"
                      onClick={() => setRenaming(bundle)}
                    >
                      <Pencil size={15} />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title={t`Delete`}>
                    <IconButton
                      aria-label={t`Delete ${bundle.name}`}
                      size="small"
                      color="error"
                      onClick={() => setDeleting(bundle)}
                    >
                      <Trash2 size={15} />
                    </IconButton>
                  </Tooltip>
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
      <Dialog
        open={deleting !== null}
        onClose={busy ? undefined : () => setDeleting(null)}
        transitionDuration={0}
      >
        <DialogTitle>{t`Delete ${deleting?.name ?? ''}?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>{t`This bundle will be removed from Mortar.`}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleting(null)} disabled={busy}>
            {t`Cancel`}
          </Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => confirmDelete().catch(() => undefined)}
            disabled={busy}
          >
            {t`Delete`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
