import { useLingui } from '@lingui/react/macro'
import { MenuItem, TextField } from '@mui/material'
import { useEffect, useId, useState } from 'react'
import type { LaunchPreset } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SMAPI } from '../brand/productNames.ts'
import { useGameInfo } from '../games/info.ts'
import { Field, FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { sameValue } from '../shell/sameValue.ts'
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
  const consoleId = useId()
  const set = (key: 'name' | 'launchOptions' | 'launchPrefix' | 'launchEnv') => (value: string) =>
    setDraft({ ...draft, [key]: value })
  let issue = ''
  if (error === 'empty') {
    issue = t`Name the preset.`
  } else if (error === 'taken') {
    issue = t`Another preset already has this name.`
  }
  return (
    <FormDialog
      open={open}
      onClose={onCancel}
      title={preset.id === '' && preset.name === '' ? t`New launch preset` : t`Edit launch preset`}
      submitLabel={t`Save`}
      issue={issue}
      dirty={!sameValue(draft, preset)}
      onSubmit={() => onSave({ ...draft, name: draft.name.trim() })}
    >
      <TextInput label={t`Name`} value={draft.name} onChange={set('name')} autoFocus={true} />
      <TextInput
        label={t`Launch options`}
        value={draft.launchOptions ?? ''}
        onChange={set('launchOptions')}
      />
      <TextInput
        label={t`Launch prefix`}
        value={draft.launchPrefix ?? ''}
        onChange={set('launchPrefix')}
      />
      <TextInput
        label={t`Launch environment`}
        value={draft.launchEnv ?? ''}
        onChange={set('launchEnv')}
        multiline={true}
      />
      {smapi ? (
        <Field label={t`${{ loader: SMAPI }} console window`} htmlFor={consoleId}>
          <TextField
            id={consoleId}
            select={true}
            fullWidth={true}
            size="small"
            value={draft.showConsole || FOLLOW}
            onChange={(e) =>
              setDraft({ ...draft, showConsole: e.target.value === FOLLOW ? '' : e.target.value })
            }
          >
            <MenuItem value={FOLLOW}>{t`Follow the setting`}</MenuItem>
            <MenuItem value="true">{t`Show`}</MenuItem>
            <MenuItem value="false">{t`Hide`}</MenuItem>
          </TextField>
        </Field>
      ) : null}
    </FormDialog>
  )
}
