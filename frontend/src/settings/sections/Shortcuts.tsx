import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { SetShortcuts } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { shortcutLabels } from '../shortcutLabels.ts'
import {
  conflictFor,
  defaultBindings,
  formatChord,
  mergeBindings,
  SHORTCUTS,
  type Shortcut,
  type ShortcutId,
  setShortcutCapturing,
  takeableFrom,
} from '../shortcuts.ts'
import { useSettings } from '../store.ts'
import { useSettingsSearch } from '../useSettingsSearch.ts'

type Labels = Record<ShortcutId, string>

function useShortcutLabels(): Labels {
  const { i18n } = useLingui()
  return useMemo(() => shortcutLabels(i18n), [i18n])
}

function ShortcutRow({
  id,
  label,
  keys,
  recording,
  conflictName,
  onTake,
  onRecord,
  onReset,
}: {
  id: ShortcutId
  label: string
  keys: string
  recording: boolean
  conflictName: string | null
  onTake: (() => void) | null
  onRecord: () => void
  onReset: () => void
}) {
  const { t } = useLingui()
  const shown = keys || t`Not set`
  const chip = recording ? t`Press a key` : shown
  return (
    <SettingRow
      label={label}
      description={
        conflictName ? (
          <Box component="span" sx={{ color: 'error.main' }}>
            {t`Already used by ${conflictName}`}
            {onTake ? (
              <Button size="small" onClick={onTake} sx={{ ml: 1 }}>
                {t`Use it anyway; ${conflictName} becomes unbound`}
              </Button>
            ) : null}
          </Box>
        ) : undefined
      }
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexShrink: 0 }}>
        <ButtonBase
          aria-label={t`Change shortcut for ${label}, currently ${shown}`}
          onClick={onRecord}
          sx={{
            color: 'var(--mortar-ink-sec)',
            fontSize: 12,
            px: 0.75,
            py: 0.25,
            border: '1px solid var(--mortar-hairline-25)',
            borderRadius: 0.5,
            bgcolor: recording ? 'var(--mortar-hairline-12)' : 'transparent',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
          }}
        >
          {chip}
        </ButtonBase>
        <Button
          size="small"
          disabled={keys === defaultBindings()[id]}
          onClick={onReset}
          aria-label={t`Reset ${label}`}
          sx={{ minWidth: 0 }}
        >
          {t`Reset`}
        </Button>
      </Box>
    </SettingRow>
  )
}

export function ResetAllShortcuts() {
  const { t } = useLingui()
  const [confirmReset, setConfirmReset] = useState(false)
  return (
    <>
      <Button variant="outlined" onClick={() => setConfirmReset(true)}>
        {t`Reset all`}
      </Button>
      <ConfirmDialog
        open={confirmReset}
        title={t`Reset every shortcut?`}
        body={t`Every key returns to its default.`}
        confirmLabel={t`Reset all`}
        color="warning"
        onCancel={() => setConfirmReset(false)}
        onConfirm={() => {
          setConfirmReset(false)
          SetShortcuts(defaultBindings()).catch(reportUnexpected)
        }}
      />
    </>
  )
}

export function Shortcuts() {
  const { t } = useLingui()
  const query = useSettingsSearch()
  const [recording, setRecording] = useState<ShortcutId | null>(null)
  // keys is set when the chord is another action's default, which the user may take from it.
  const [conflict, setConflict] = useState<{
    id: ShortcutId
    other: ShortcutId
    keys: string | null
  } | null>(null)
  const stored = useSettings((s) => s.shortcuts)
  const bindings = useMemo(() => mergeBindings(stored), [stored])
  const labels = useShortcutLabels()
  useEffect(() => {
    setShortcutCapturing(recording !== null)
    if (!recording) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault()
      e.stopPropagation()
      if (e.key === 'Escape') {
        setRecording(null)
        setConflict(null)
        return
      }
      const keys = formatChord(e)
      if (!keys) {
        return
      }
      const other = conflictFor(recording, keys, bindings)
      if (other) {
        const takeable = takeableFrom(recording, keys, bindings) !== null
        setConflict({ id: recording, other, keys: takeable ? keys : null })
        // Recording stops at a takeable chord so Tab reaches the offer instead of being recorded.
        if (takeable) {
          setRecording(null)
        }
        return
      }
      setConflict(null)
      setRecording(null)
      SetShortcuts({ ...bindings, [recording]: keys }).catch(reportUnexpected)
    }
    globalThis.addEventListener('keydown', onKey, true)
    return () => {
      globalThis.removeEventListener('keydown', onKey, true)
      setShortcutCapturing(false)
    }
  }, [recording, bindings])
  const q = query.toLowerCase()
  const groups: [Shortcut['group'], string][] = [
    ['General', t`General`],
    ['Navigation', t`Navigation`],
    ['Profiles', t`Profiles`],
    ['Mods list', t`Mods list`],
    ['Tabs', t`Tabs`],
  ]
  const rows = SHORTCUTS.filter((row) => labels[row.id].toLowerCase().includes(q))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad, fontSize: 14 }}>
      {groups.map(([group, name]) => {
        const grouped = rows.filter((row) => row.group === group)
        return grouped.length > 0 ? (
          <SettingsSection key={group} title={name}>
            {grouped.map((row) => (
              <ShortcutRow
                key={row.id}
                id={row.id}
                label={labels[row.id]}
                keys={bindings[row.id]}
                recording={recording === row.id}
                conflictName={conflict?.id === row.id ? labels[conflict.other] : null}
                onTake={
                  conflict?.id === row.id && conflict.keys !== null
                    ? () => {
                        // The settings store unbinds the default this chord belonged to.
                        SetShortcuts({ ...bindings, [row.id]: conflict.keys }).catch(
                          reportUnexpected,
                        )
                        setConflict(null)
                      }
                    : null
                }
                onRecord={() => {
                  setConflict(null)
                  setRecording(row.id)
                }}
                onReset={() => {
                  setConflict(null)
                  SetShortcuts({ ...bindings, [row.id]: defaultBindings()[row.id] }).catch(
                    reportUnexpected,
                  )
                }}
              />
            ))}
          </SettingsSection>
        ) : null
      })}
    </Box>
  )
}
