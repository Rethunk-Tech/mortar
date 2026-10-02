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
import { useEffect, useState } from 'react'
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
import { errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

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
  const [name, setName] = useState(initialName)
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    if (open) {
      setName(initialName)
    }
  }, [initialName, open])
  const submit = async () => {
    const trimmed = name.trim()
    if (!trimmed || busy) {
      return
    }
    setBusy(true)
    try {
      await onSubmit(trimmed)
      onClose()
    } catch (error) {
      useToasts.getState().push({ kind: 'error', title: errorTitle, body: errorMessage(error) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent sx={{ minWidth: 360, maxWidth: 'calc(100vw - 64px)' }}>
        <TextField
          autoFocus={true}
          fullWidth={true}
          label={t`Name`}
          value={name}
          onChange={(event) => setName(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              submit().catch(() => undefined)
            }
          }}
          sx={{ mt: 1 }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          onClick={() => submit().catch(() => undefined)}
          disabled={!name.trim() || busy}
        >
          {submitLabel}
        </Button>
      </DialogActions>
    </Dialog>
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
  const [bundles, setBundles] = useState<Bundle[]>([])
  const [newName, setNewName] = useState('')
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    if (!open) {
      return
    }
    let active = true
    setNewName('')
    setLoading(true)
    ListBundles(game)
      .then((listed) => {
        if (active) {
          setBundles(listed ?? [])
        }
      })
      .catch((error: unknown) => {
        if (active) {
          useToasts
            .getState()
            .push({ kind: 'error', title: t`Could not read bundles`, body: errorMessage(error) })
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
  const addTo = async (bundle: Bundle) => {
    if (busy) {
      return
    }
    setBusy(true)
    try {
      const updated = await AddMods(game, bundle.id, profileId, uniqueIds)
      useToasts.getState().push({ kind: 'success', title: t`Added mods to ${updated.name}` })
      onClose()
    } catch (error) {
      useToasts.getState().push({
        kind: 'error',
        title: t`Could not add mods to the bundle`,
        body: errorMessage(error),
      })
    } finally {
      setBusy(false)
    }
  }
  const create = async () => {
    const name = newName.trim()
    if (!name || busy) {
      return
    }
    setBusy(true)
    try {
      const created = await Create(game, name, profileId, uniqueIds)
      useToasts.getState().push({ kind: 'success', title: t`Created ${created.name}` })
      onClose()
    } catch (error) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not create the bundle`, body: errorMessage(error) })
    } finally {
      setBusy(false)
    }
  }
  const bundleContent = (() => {
    if (loading) {
      return <LoadingRow>{t`Loading bundles…`}</LoadingRow>
    }
    if (bundles.length === 0) {
      return (
        <EmptyState
          compact={true}
          icon={<PackagePlus size={28} />}
          title={t`No bundles yet.`}
        >{t`Create a bundle to reuse a set of mods.`}</EmptyState>
      )
    }
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {bundles.map((bundle) => (
          <Button
            key={bundle.id}
            variant="outlined"
            disabled={busy}
            onClick={() => addTo(bundle).catch(() => undefined)}
            sx={{ justifyContent: 'space-between', textTransform: 'none' }}
          >
            <span>{bundle.name}</span>
            <Typography component="span" sx={{ color: 'text.secondary', fontSize: 12 }}>
              {t`${bundle.mods?.length ?? 0} mods`}
            </Typography>
          </Button>
        ))}
      </Box>
    )
  })()
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{t`Add to bundle`}</DialogTitle>
      <DialogContent sx={{ minWidth: 420, maxWidth: 'calc(100vw - 64px)' }}>
        <Typography sx={{ mb: 1.25, color: 'text.secondary', fontSize: 13 }}>
          {t`Choose an existing bundle or create one.`}
        </Typography>
        {bundleContent}
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
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          startIcon={<PackagePlus size={16} />}
          onClick={() => create().catch(() => undefined)}
          disabled={!newName.trim() || busy}
        >
          {t`Create and add`}
        </Button>
      </DialogActions>
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
  const [bundles, setBundles] = useState<Bundle[]>([])
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState(false)
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
          useToasts
            .getState()
            .push({ kind: 'error', title: t`Could not read bundles`, body: errorMessage(error) })
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
  const apply = async (bundle: Bundle) => {
    if (busy) {
      return
    }
    setBusy(true)
    try {
      await onApplied(await Apply(game, bundle.id, profileId))
      onClose()
    } catch (error) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not add the bundle`, body: errorMessage(error) })
    } finally {
      setBusy(false)
    }
  }
  const bundleContent = (() => {
    if (loading) {
      return <LoadingRow>{t`Loading bundles…`}</LoadingRow>
    }
    if (bundles.length === 0) {
      return <Typography sx={{ color: 'text.secondary' }}>{t`No bundles yet.`}</Typography>
    }
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {bundles.map((bundle) => (
          <Button
            key={bundle.id}
            variant="outlined"
            disabled={busy}
            onClick={() => apply(bundle).catch(() => undefined)}
            sx={{ justifyContent: 'space-between', textTransform: 'none' }}
          >
            <span>{bundle.name}</span>
            <Typography component="span" sx={{ color: 'text.secondary', fontSize: 12 }}>
              {t`${bundle.mods?.length ?? 0} mods`}
            </Typography>
          </Button>
        ))}
      </Box>
    )
  })()
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{t`Add a bundle to ${profileName}`}</DialogTitle>
      <DialogContent sx={{ minWidth: 420, maxWidth: 'calc(100vw - 64px)' }}>
        {bundleContent}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export { AddToBundleDialog, ApplyBundleDialog, BundleNameDialog }
