import { useLingui } from '@lingui/react/macro'
import { Button, Menu } from '@mui/material'
import { FileCheck, Save, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ApplyConfigPreset,
  DeleteConfigPreset,
  ListConfigPresets,
  ReadConfig,
  SaveConfigPreset,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import { ICON_SIZE } from './menu.ts'
import { openTarget } from './storeView.ts'
import { useLocked } from './useLocked.ts'

const maxPresetName = 40

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
        <LockedReason key={`a-${name}`} locked={locked}>
          <MenuAction
            icon={<FileCheck size={ICON_SIZE} />}
            label={t`Apply ${name}`}
            disabled={locked}
            onClick={() => onApply(name)}
          />
        </LockedReason>
      ))}
      {names.map((name) => (
        <MenuAction
          key={`d-${name}`}
          icon={<Trash2 size={ICON_SIZE} />}
          label={t`Delete ${name}`}
          tone="error"
          onClick={() => onAskDelete(name)}
        />
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
  const [replacing, setReplacing] = useState<string | null>(null)
  const load = () => {
    const at = openTarget()
    if (!at) {
      return
    }
    ListConfigPresets(at.game, mod.id)
      .then((list) => setNames(list ?? []))
      .catch(reportUnexpected)
  }
  const run = (fn: () => Promise<unknown>) => {
    fn()
      .then(() => {
        load()
        setSaveOpen(false)
        setPendingDelete(null)
        setReplacing(null)
        setAnchor(null)
      })
      .catch(reportUnexpected)
  }
  const save = (name: string) => {
    const at = openTarget()
    if (!at) {
      return
    }
    run(() =>
      ReadConfig(at.game, at.id, mod.key, mod.id).then((raw) =>
        SaveConfigPreset(at.game, mod.id, name, raw),
      ),
    )
  }
  return (
    <>
      <Button
        size="small"
        variant="outlined"
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => {
          setAnchor(e.currentTarget)
          load()
        }}
      >
        {t`Presets`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        <MenuAction
          icon={<Save size={ICON_SIZE} />}
          label={t`Save current as…`}
          onClick={() => {
            setSaveOpen(true)
            setAnchor(null)
          }}
        />
        {names.length > 0 ? <MenuRule /> : null}
        <PresetItems
          names={names}
          locked={locked}
          onApply={(name) => {
            const at = openTarget()
            if (at) {
              run(() => ApplyConfigPreset(at.game, at.id, mod.key, mod.id, name))
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
        title={t`Save current settings`}
        label={t`Preset name`}
        maxLength={maxPresetName}
        confirmLabel={t`Save`}
        onCancel={() => setSaveOpen(false)}
        onSubmit={(value) => {
          if (names.some((n) => n.toLowerCase() === value.toLowerCase())) {
            setSaveOpen(false)
            setReplacing(value)
            return
          }
          save(value)
        }}
      />
      <ConfirmationDialog
        open={replacing !== null}
        title={t`Replace ${{ name: replacing ?? '' }}?`}
        body={t`A preset with this name already exists.`}
        confirmLabel={t`Replace`}
        onCancel={() => setReplacing(null)}
        onConfirm={() => save(replacing ?? '')}
      />
      <ConfirmationDialog
        open={pendingDelete !== null}
        title={t`Delete ${{ name: pendingDelete ?? '' }}?`}
        confirmLabel={t`Delete preset`}
        tone="danger"
        onCancel={() => setPendingDelete(null)}
        onConfirm={() => {
          const name = pendingDelete
          const at = openTarget()
          if (!(name && at)) {
            setPendingDelete(null)
            return
          }
          run(() => DeleteConfigPreset(at.game, mod.id, name))
        }}
      />
    </>
  )
}

export { PresetsButton }
