import { useLingui } from '@lingui/react/macro'
import { Button, Divider, Menu, MenuItem } from '@mui/material'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ApplyConfigPreset,
  DeleteConfigPreset,
  ListConfigPresets,
  ReadConfig,
  SaveConfigPreset,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useLocked } from './useLocked.ts'

const noWrap = { whiteSpace: 'nowrap' } as const
const maxPresetName = 40

function openTarget() {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

function PresetItems({
  names,
  locked,
  onApply,
  onAskDelete,
}: {
  names: string[]
  locked: boolean
  onApply: (name: string) => void
  onAskDelete: (name: string) => void
}) {
  const { t } = useLingui()
  return (
    <>
      {names.map((name) => (
        <DisabledReason
          key={`a-${name}`}
          title={t`Stop the game to change mods.`}
          disabled={locked}
        >
          <MenuItem disabled={locked} onClick={() => onApply(name)}>
            {t`Apply ${name}`}
          </MenuItem>
        </DisabledReason>
      ))}
      {names.map((name) => (
        <MenuItem key={`d-${name}`} sx={{ color: 'error.main' }} onClick={() => onAskDelete(name)}>
          {t`Delete ${name}`}
        </MenuItem>
      ))}
    </>
  )
}

function PresetsButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const locked = useLocked()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [names, setNames] = useState<string[]>([])
  const [saveOpen, setSaveOpen] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<string | null>(null)
  const load = () => {
    const at = openTarget()
    if (!at) {
      return
    }
    ListConfigPresets(at.game, mod.uniqueId)
      .then((list) => setNames(list ?? []))
      .catch(reportUnexpected)
  }
  const run = (fn: () => Promise<unknown>) => {
    fn()
      .then(() => {
        load()
        setSaveOpen(false)
        setPendingDelete(null)
        setAnchor(null)
      })
      .catch(reportUnexpected)
  }
  return (
    <>
      <Button
        size="small"
        variant="outlined"
        onClick={(e) => {
          setAnchor(e.currentTarget)
          load()
        }}
        sx={noWrap}
      >
        {t`Presets`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        <MenuItem
          onClick={() => {
            setSaveOpen(true)
            setAnchor(null)
          }}
        >
          {t`Save current as…`}
        </MenuItem>
        {names.length > 0 ? <Divider /> : null}
        <PresetItems
          names={names}
          locked={locked}
          onApply={(name) => {
            const at = openTarget()
            if (at) {
              run(() => ApplyConfigPreset(at.game, at.id, mod.key, mod.uniqueId, name))
            }
          }}
          onAskDelete={(name) => {
            setAnchor(null)
            setPendingDelete(name)
          }}
        />
      </Menu>
      <PromptDialog
        open={saveOpen}
        title={t`Save current as…`}
        label={t`Preset name`}
        maxLength={maxPresetName}
        confirmLabel={t`Save`}
        onCancel={() => setSaveOpen(false)}
        onSubmit={(value) => {
          const at = openTarget()
          if (!at) {
            return
          }
          run(() =>
            ReadConfig(at.game, at.id, mod.key, mod.uniqueId).then((raw) =>
              SaveConfigPreset(at.game, mod.uniqueId, value, raw),
            ),
          )
        }}
      />
      <ConfirmDialog
        open={pendingDelete !== null}
        title={t`Delete ${pendingDelete ?? ''}?`}
        body={t`This preset will be deleted.`}
        confirmLabel={t`Delete`}
        color="error"
        onCancel={() => setPendingDelete(null)}
        onConfirm={() => {
          const name = pendingDelete
          const at = openTarget()
          if (!(name && at)) {
            setPendingDelete(null)
            return
          }
          run(() => DeleteConfigPreset(at.game, mod.uniqueId, name))
        }}
      />
    </>
  )
}

export { PresetsButton }
