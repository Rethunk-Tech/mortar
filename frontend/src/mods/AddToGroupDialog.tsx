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
import { FolderPlus, FolderTree } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import { addModToGroup } from './storeEntries.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16

function AddToGroupMenuItem({ locked, onClick }: { locked: boolean; onClick: () => void }) {
  const { t } = useLingui()
  return (
    <LockedReason locked={locked}>
      <MenuAction
        disabled={locked}
        icon={<FolderTree size={ICON_SIZE} />}
        label={t`Add to group…`}
        onClick={onClick}
      />
    </LockedReason>
  )
}

function AddToGroupDialog({
  open,
  onClose,
  entryKey,
  profile,
}: {
  open: boolean
  onClose: () => void
  entryKey: string
  profile: Profile | undefined
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [name, setName] = useState('')
  const groups = profile?.groups ?? []
  const add = async (groupName: string) => {
    const trimmed = groupName.trim()
    if (trimmed === '' || locked) {
      return
    }
    await addModToGroup(entryKey, trimmed)
    setName('')
    onClose()
  }
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Add to group…`}</DialogTitle>
      <LockedReason locked={locked}>
        <DialogContent>
          {groups.length === 0 ? (
            <EmptyState compact={true} icon={<FolderPlus size={28} />} title={t`No groups yet`}>
              {t`Name a group below to create it and add this mod.`}
            </EmptyState>
          ) : null}
          {groups.map((g) => (
            <MenuItem
              key={g.name}
              disabled={locked}
              onClick={() => {
                add(g.name ?? '').catch(reportUnexpected)
              }}
            >
              {g.name}
            </MenuItem>
          ))}
          <TextField
            autoFocus={true}
            margin="dense"
            label={t`New group`}
            fullWidth={true}
            value={name}
            disabled={locked}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                add(name).catch(reportUnexpected)
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{t`Cancel`}</Button>
          <Button
            variant="contained"
            disabled={locked || name.trim() === ''}
            onClick={() => add(name).catch(reportUnexpected)}
          >
            {t`Add`}
          </Button>
        </DialogActions>
      </LockedReason>
    </Dialog>
  )
}

export { AddToGroupDialog, AddToGroupMenuItem }
