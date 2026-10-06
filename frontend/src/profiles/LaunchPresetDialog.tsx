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
import type { LaunchPreset } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useGameInfo } from '../games/info.ts'
import { presetNameError } from './profilePresets.ts'

const FOLLOW = 'follow'

/** Edits one launch preset; Save hands back the preset, the parent persists it. */
export function LaunchPresetDialog({
  open,
  preset,
  all,
  onCancel,
  onSave,
}: {
  open: boolean
  preset: LaunchPreset
  all: LaunchPreset[]
  onCancel: () => void
  onSave: (preset: LaunchPreset) => void
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(preset)
  const smapi = useGameInfo()?.loaderId === 'smapi'
  useEffect(() => {
    if (open) {
      setDraft(preset)
    }
  }, [open, preset])
  const error = presetNameError(draft.name, all, preset.id)
  const field = (key: 'launchOptions' | 'launchPrefix' | 'launchEnv') => ({
    value: draft[key] ?? '',
    onChange: (e: { target: { value: string } }) => setDraft({ ...draft, [key]: e.target.value }),
    fullWidth: true,
    margin: 'dense' as const,
    slotProps: { root: { sx: { userSelect: 'text' } } },
  })
  return (
    <Dialog open={open} onClose={onCancel} fullWidth={true} maxWidth="sm">
      <DialogTitle>
        {preset.id === '' && preset.name === '' ? t`New launch preset` : t`Edit launch preset`}
      </DialogTitle>
      <DialogContent>
        <TextField
          value={draft.name}
          onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          autoFocus={true}
          fullWidth={true}
          margin="dense"
          label={t`Name`}
          error={error !== null && draft.name !== ''}
          helperText={error === 'taken' ? t`Another preset already has this name.` : undefined}
          slotProps={{ root: { sx: { userSelect: 'text' } } }}
        />
        <TextField {...field('launchOptions')} label={t`Launch options`} />
        <TextField {...field('launchPrefix')} label={t`Launch prefix`} />
        <TextField
          {...field('launchEnv')}
          multiline={true}
          minRows={2}
          label={t`Launch environment`}
        />
        {smapi ? (
          <TextField
            select={true}
            fullWidth={true}
            margin="dense"
            label={t`SMAPI console window`}
            value={draft.showConsole || FOLLOW}
            onChange={(e) =>
              setDraft({ ...draft, showConsole: e.target.value === FOLLOW ? '' : e.target.value })
            }
          >
            <MenuItem value={FOLLOW}>{t`Follow the setting`}</MenuItem>
            <MenuItem value="true">{t`Show`}</MenuItem>
            <MenuItem value="false">{t`Hide`}</MenuItem>
          </TextField>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          disabled={error !== null}
          onClick={() => onSave({ ...draft, name: draft.name.trim() })}
        >{t`Save`}</Button>
      </DialogActions>
    </Dialog>
  )
}
