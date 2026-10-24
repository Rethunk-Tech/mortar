import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { PackagePlus } from 'lucide-react'
import type { ReactNode } from 'react'
import type {
  ApplyResult,
  SavedGroup,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { Apply } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/service.ts'
import { modsLabel } from '../i18n/counts.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { usePending } from '../toasts/usePending.ts'
import { useSavedGroups } from './useSavedGroups.ts'

interface SavedGroupNameDialogProps {
  open: boolean
  title: string
  initialName?: string
  submitLabel: string
  errorTitle: string
  onClose: () => void
  onSubmit: (name: string) => Promise<void>
}

function SavedGroupNameDialog({
  open,
  title,
  initialName = '',
  submitLabel,
  errorTitle,
  onClose,
  onSubmit,
}: SavedGroupNameDialogProps) {
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

function SavedGroupPickList({
  loading,
  savedGroups,
  busy,
  empty,
  onPick,
}: {
  loading: boolean
  savedGroups: SavedGroup[]
  busy: boolean
  empty: ReactNode
  onPick: (group: SavedGroup) => void
}) {
  const { t } = useLingui()
  if (loading) {
    return <LoadingRow>{t`Loading saved groups…`}</LoadingRow>
  }
  if (savedGroups.length === 0) {
    return empty
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      {savedGroups.map((group) => (
        <Button
          key={group.id}
          type="button"
          variant="outlined"
          disabled={busy}
          onClick={() => onPick(group)}
          sx={{ justifyContent: 'space-between' }}
        >
          <span>{group.name}</span>
          <Typography component="span" sx={{ color: 'text.secondary', fontSize: 'small.fontSize' }}>
            {modsLabel(group.mods?.length ?? 0)}
          </Typography>
        </Button>
      ))}
    </Box>
  )
}

interface ApplySavedGroupDialogProps {
  open: boolean
  game: string
  profileName: string
  profileId: string
  onClose: () => void
  onApplied: (result: ApplyResult) => Promise<void> | void
}

function ApplySavedGroupDialog({
  open,
  game,
  profileName,
  profileId,
  onClose,
  onApplied,
}: ApplySavedGroupDialogProps) {
  const { t } = useLingui()
  const { savedGroups, loading } = useSavedGroups(game, open)
  const [busy, run] = usePending()
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} maxWidth="sm" fullWidth={true}>
      <DialogTitle>{t`Add a saved group to ${{ profile: profileName }}`}</DialogTitle>
      <DialogContent>
        <SavedGroupPickList
          loading={loading}
          savedGroups={savedGroups}
          busy={busy}
          empty={
            <EmptyState
              compact={true}
              icon={<PackagePlus size={28} />}
              title={t`No saved groups yet.`}
            >
              {t`Open a group's menu on the Mods tab and choose Save to library.`}
            </EmptyState>
          }
          onPick={(group) => {
            run(
              () =>
                Apply(game, group.id, profileId)
                  .then((result) => onApplied(result))
                  .then(() => onClose()),
              { errorTitle: t`Could not add the saved group` },
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

export { ApplySavedGroupDialog, SavedGroupNameDialog }
