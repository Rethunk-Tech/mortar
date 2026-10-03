import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  ListItemIcon,
  ListItemText,
  MenuItem,
  TextField,
} from '@mui/material'
import { FolderTree } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { addModToGroup } from './storeEntries.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16

function AddToGroupMenuItem({ locked, onClick }: { locked: boolean; onClick: () => void }) {
  const { t } = useLingui()
  return (
    <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
      <MenuItem disabled={locked} onClick={onClick}>
        <ListItemIcon sx={{ color: 'inherit' }}>
          <FolderTree size={ICON_SIZE} />
        </ListItemIcon>
        <ListItemText>{t`Add to group…`}</ListItemText>
      </MenuItem>
    </DisabledReason>
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
      <DialogContent>
        {groups.map((g) => (
          <MenuItem key={g.name} disabled={locked} onClick={() => add(g.name ?? '')}>
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
          disabled={locked || name.trim() === ''}
          onClick={() => add(name).catch(reportUnexpected)}
        >
          {t`Add`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export { AddToGroupDialog, AddToGroupMenuItem }
