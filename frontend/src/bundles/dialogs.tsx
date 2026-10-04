import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  TextField,
  Typography,
} from '@mui/material'
import { PackagePlus } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type {
  ApplyResult,
  Bundle,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/models.ts'
import {
  AddMods,
  Apply,
  Create,
  List as ListBundles,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/bundles/service.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

interface BundleNameDialogProps {
  open: boolean
  title: string
  initialName?: string
  submitLabel: string
  errorTitle: string
  onClose: () => void
  onSubmit: (name: string) => Promise<void>
}

function BundleNameDialog({
  open,
  title,
  initialName = '',
  submitLabel,
  errorTitle,
  onClose,
  onSubmit,
}: BundleNameDialogProps) {
  const { t } = useLingui()
  const [busy, run] = usePending()
  return (
    <PromptDialog
      open={open}
      title={title}
      label={t`Name`}
      initial={initialName}
      confirmLabel={submitLabel}
      busy={busy}
      onCancel={onClose}
      onSubmit={(name) => {
        run(() => onSubmit(name).then(() => onClose()), { errorTitle })
      }}
    />
  )
}

function useListedBundles(open: boolean, game: string) {
  const { t } = useLingui()
  const [bundles, setBundles] = useState<Bundle[]>([])
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    if (!open) {
      return
    }
    let active = true
    setLoading(true)
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
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [game, open, t])
  return { bundles, loading }
}

function BundlePickList({
  loading,
  bundles,
  busy,
  empty,
  onPick,
}: {
  loading: boolean
  bundles: Bundle[]
  busy: boolean
  empty: ReactNode
  onPick: (bundle: Bundle) => void
}) {
  const { t } = useLingui()
  if (loading) {
    return <LoadingRow>{t`Loading bundles…`}</LoadingRow>
  }
  if (bundles.length === 0) {
    return empty
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      {bundles.map((bundle) => (
        <Button
          key={bundle.id}
          type="button"
          variant="outlined"
          disabled={busy}
          onClick={() => onPick(bundle)}
          sx={{ justifyContent: 'space-between', textTransform: 'none' }}
        >
          <span>{bundle.name}</span>
          <Typography component="span" sx={{ color: 'text.secondary', fontSize: 12 }}>
            {plural(bundle.mods?.length ?? 0, { one: '# mod', other: '# mods' })}
          </Typography>
        </Button>
      ))}
    </Box>
  )
}

interface AddToBundleDialogProps {
  open: boolean
  game: string
  profileId: string
  uniqueIds: string[]
  onClose: () => void
}

function AddToBundleDialog({ open, game, profileId, uniqueIds, onClose }: AddToBundleDialogProps) {
  const { t } = useLingui()
  const { bundles, loading } = useListedBundles(open, game)
  const [newName, setNewName] = useState('')
  const [busy, run] = usePending()
  useEffect(() => {
    if (open) {
      setNewName('')
    }
  }, [open])
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <form
        onSubmit={(event) => {
          event.preventDefault()
          const name = newName.trim()
          if (!name) {
            return
          }
          run(
            () =>
              Create(game, name, profileId, uniqueIds).then((created) => {
                useToasts.getState().push({ kind: 'success', title: t`Created ${created.name}` })
                onClose()
              }),
            { errorTitle: t`Could not create the bundle` },
          )
        }}
      >
        <DialogTitle>{t`Add to bundle`}</DialogTitle>
        <DialogContent sx={{ minWidth: 420, maxWidth: 'calc(100vw - 64px)' }}>
          <Typography sx={{ mb: 1.25, color: 'text.secondary', fontSize: 13 }}>
            {t`Choose an existing bundle or create one.`}
          </Typography>
          <BundlePickList
            loading={loading}
            bundles={bundles}
            busy={busy}
            empty={
              <EmptyState
                compact={true}
                icon={<PackagePlus size={28} />}
                title={t`No bundles yet.`}
              >{t`Create a bundle to reuse a set of mods.`}</EmptyState>
            }
            onPick={(bundle) => {
              run(
                () =>
                  AddMods(game, bundle.id, profileId, uniqueIds).then((updated) => {
                    useToasts
                      .getState()
                      .push({ kind: 'success', title: t`Added mods to ${updated.name}` })
                    onClose()
                  }),
                { errorTitle: t`Could not add mods to the bundle` },
              )
            }}
          />
          <Divider sx={{ my: 2 }} />
          <TextField
            fullWidth={true}
            label={t`New bundle name`}
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            disabled={busy}
          />
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={onClose} disabled={busy}>
            {t`Cancel`}
          </Button>
          <Button
            type="submit"
            variant="contained"
            startIcon={<PackagePlus size={16} />}
            disabled={!newName.trim() || busy}
          >
            {t`Create and add`}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}

interface ApplyBundleDialogProps {
  open: boolean
  game: string
  profileName: string
  profileId: string
  onClose: () => void
  onApplied: (result: ApplyResult) => Promise<void> | void
}

function ApplyBundleDialog({
  open,
  game,
  profileName,
  profileId,
  onClose,
  onApplied,
}: ApplyBundleDialogProps) {
  const { t } = useLingui()
  const { bundles, loading } = useListedBundles(open, game)
  const [busy, run] = usePending()
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{t`Add a bundle to ${profileName}`}</DialogTitle>
      <DialogContent sx={{ minWidth: 420, maxWidth: 'calc(100vw - 64px)' }}>
        <BundlePickList
          loading={loading}
          bundles={bundles}
          busy={busy}
          empty={<Typography sx={{ color: 'text.secondary' }}>{t`No bundles yet.`}</Typography>}
          onPick={(bundle) => {
            run(
              () =>
                Apply(game, bundle.id, profileId)
                  .then((result) => onApplied(result))
                  .then(() => onClose()),
              { errorTitle: t`Could not add the bundle` },
            )
          }}
        />
      </DialogContent>
      <DialogActions>
        <Button type="button" onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export { AddToBundleDialog, ApplyBundleDialog, BundleNameDialog }
