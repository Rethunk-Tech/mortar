import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Menu,
  MenuItem,
  TextField,
} from '@mui/material'
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
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useLocked } from './useLocked.ts'

const noWrap = { whiteSpace: 'nowrap' } as const
const maxPresetName = 40

function openTarget() {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, id: openId } : null
}

function SavePresetDialog({
  open,
  draft,
  onDraft,
  onClose,
  onSave,
}: {
  open: boolean
  draft: string
  onDraft: (value: string) => void
  onClose: () => void
  onSave: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{t`Save current as…`}</DialogTitle>
      <DialogContent>
        <TextField
          size="small"
          autoFocus={true}
          fullWidth={true}
          value={draft}
          slotProps={{ htmlInput: { maxLength: maxPresetName } }}
          onChange={(e) => onDraft(e.target.value)}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button disabled={draft.trim() === ''} onClick={onSave}>
          {t`Save`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function PresetItems({
  names,
  locked,
  onApply,
  onDelete,
}: {
  names: string[]
  locked: boolean
  onApply: (name: string) => void
  onDelete: (name: string) => void
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
        <MenuItem key={`d-${name}`} onClick={() => onDelete(name)}>
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
  const [draft, setDraft] = useState('')
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
      <Menu
        anchorEl={anchor}
        open={anchor !== null}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
      >
        <MenuItem
          onClick={() => {
            setDraft('')
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
          onDelete={(name) => {
            const at = openTarget()
            if (at) {
              run(() => DeleteConfigPreset(at.game, mod.uniqueId, name))
            }
          }}
        />
      </Menu>
      <SavePresetDialog
        open={saveOpen}
        draft={draft}
        onDraft={setDraft}
        onClose={() => setSaveOpen(false)}
        onSave={() => {
          const at = openTarget()
          if (!at) {
            return
          }
          run(() =>
            ReadConfig(at.game, at.id, mod.key, mod.uniqueId).then((raw) =>
              SaveConfigPreset(at.game, mod.uniqueId, draft.trim(), raw),
            ),
          )
        }}
      />
    </>
  )
}

export { PresetsButton }
