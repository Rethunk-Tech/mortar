import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { SetShortcuts } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { SettingsSection } from '../SettingsSection.tsx'
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
  onRecord,
  onReset,
}: {
  id: ShortcutId
  label: string
  keys: string
  recording: boolean
  conflictName: string | null
  onRecord: () => void
  onReset: () => void
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: 2,
        minHeight: 56,
        px: 2.5,
        py: 1,
        fontSize: 16,
      }}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25, minWidth: 0 }}>
        <Box component="span">{label}</Box>
        {conflictName ? (
          <Box component="span" sx={{ fontSize: 12, color: 'error.main' }}>
            {t`Already used by ${conflictName}`}
          </Box>
        ) : null}
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexShrink: 0 }}>
        <Box
          component="button"
          type="button"
          aria-label={t`Change shortcut for ${label}, currently ${keys}`}
          onClick={onRecord}
          sx={{
            color: 'var(--mortar-ink-sec)',
            fontFamily: 'inherit',
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
          {recording ? t`Press a key` : keys || ''}
        </Box>
        <Button
          size="small"
          disabled={keys === defaultBindings()[id]}
          onClick={onReset}
          aria-label={t`Reset ${label}`}
          sx={{ whiteSpace: 'nowrap', minWidth: 0 }}
        >
          {t`Reset`}
        </Button>
      </Box>
    </Box>
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
  const [conflict, setConflict] = useState<{ id: ShortcutId; other: ShortcutId } | null>(null)
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
        setConflict({ id: recording, other })
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
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 14 }}>
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
