import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Typography } from '@mui/material'
import { useState } from 'react'
import type {
  LaunchPreset,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  SetDefaultLaunchPreset,
  SetLaunchPresets,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportError } from '../toasts/report.ts'
import { LaunchPresetDialog } from './LaunchPresetDialog.tsx'
import { duplicatePreset } from './profilePresets.ts'
import { useProfiles } from './store.ts'

const NEW_PRESET: LaunchPreset = { id: '', name: '' }

function PresetRow({
  name,
  isDefault,
  onDefault,
  actions,
}: {
  name: string
  isDefault: boolean
  onDefault: () => void
  actions?: { edit: () => void; copy: () => void; remove: () => void }
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5 }}>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0 }}>
        {name}
      </Typography>
      {isDefault ? (
        <Chip size="small" label={t`Default`} />
      ) : (
        <Button onClick={onDefault}>{t`Set as default`}</Button>
      )}
      {actions ? (
        <>
          <Button onClick={actions.edit}>{t`Edit`}</Button>
          <Button onClick={actions.copy}>{t`Duplicate`}</Button>
          <Button color="error" onClick={actions.remove}>{t`Delete`}</Button>
        </>
      ) : null}
    </Box>
  )
}

/** Named launch presets of one profile; every change is saved at once, since Play lists them. */
function LaunchPresetsBlock({ gameId, profileId }: { gameId: string; profileId: string }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const replace = useProfiles((s) => s.replace)
  const [editing, setEditing] = useState<LaunchPreset | null>(null)
  const [removing, setRemoving] = useState<LaunchPreset | null>(null)
  const presets = profile?.launchPresets ?? []
  const defaultId = profile?.defaultLaunchPreset ?? ''
  const save = (next: LaunchPreset[], nextDefault: string) =>
    SetLaunchPresets(gameId, profileId, next, nextDefault)
      .then((p: Profile) => replace(p))
      .catch(reportError(t`Could not save the launch presets`))
  const setDefault = (id: string) =>
    SetDefaultLaunchPreset(gameId, profileId, id)
      .then((p: Profile) => replace(p))
      .catch(reportError(t`Could not set the default launch preset`))
  const saveOne = (preset: LaunchPreset) => {
    setEditing(null)
    const exists = presets.some((p) => p.id === preset.id)
    const next = exists
      ? presets.map((p) => (p.id === preset.id ? preset : p))
      : [...presets, preset]
    return save(next, defaultId)
  }
  return (
    <Box sx={{ mt: 1, mb: 1 }}>
      <Typography
        sx={{ fontSize: 13, color: 'text.secondary', mb: 0.5 }}
      >{t`Launch presets`}</Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`The fields above are the Standard preset. Play uses the default preset; the Play menu can launch with another one once.`}
      </Typography>
      <PresetRow
        name={t`Standard`}
        isDefault={!presets.some((p) => p.id === defaultId)}
        onDefault={() => setDefault('')}
      />
      {presets.map((preset) => (
        <PresetRow
          key={preset.id}
          name={preset.name}
          isDefault={preset.id === defaultId}
          onDefault={() => setDefault(preset.id)}
          actions={{
            edit: () => setEditing(preset),
            copy: () => save([...presets, duplicatePreset(preset, presets)], defaultId),
            remove: () => setRemoving(preset),
          }}
        />
      ))}
      <Button onClick={() => setEditing(NEW_PRESET)}>{t`Add preset`}</Button>
      <LaunchPresetDialog
        open={editing !== null}
        preset={editing ?? NEW_PRESET}
        all={presets}
        onCancel={() => setEditing(null)}
        onSave={saveOne}
      />
      <ConfirmDialog
        open={removing !== null}
        title={t`Delete ${removing?.name ?? ''}?`}
        body={t`This launch preset will be deleted.`}
        confirmLabel={t`Delete preset`}
        color="error"
        onCancel={() => setRemoving(null)}
        onConfirm={() => {
          const gone = removing
          setRemoving(null)
          if (gone) {
            save(
              presets.filter((p) => p.id !== gone.id),
              defaultId === gone.id ? '' : defaultId,
            )
          }
        }}
      />
    </Box>
  )
}

export { LaunchPresetsBlock }
