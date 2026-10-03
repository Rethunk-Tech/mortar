import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { SetShortcuts } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import {
  conflictFor,
  defaultBindings,
  formatChord,
  mergeBindings,
  SHORTCUTS,
  type ShortcutId,
  setShortcutCapturing,
} from '../shortcuts.ts'
import { useSettings } from '../store.ts'

type Labels = Record<ShortcutId, string>

function useShortcutLabels(): Labels {
  const { t } = useLingui()
  return useMemo(
    () => ({
      'command-palette': t`Open the command palette`,
      'filter-mods': t`Focus the search`,
      play: t`Play the open profile`,
      'check-updates': t`Check for mod updates`,
      'open-settings': t`Open Settings`,
      dismiss: t`Close dialog or clear selection`,
      'select-all-mods': t`Select all mods`,
      'mod-up': t`Focus the previous mod`,
      'mod-down': t`Focus the next mod`,
      'mod-toggle': t`Toggle the focused mod`,
      'mod-details': t`Open focused mod details`,
      'mod-remove': t`Remove the focused mod`,
      'tab-mods': t`Switch to Mods`,
      'tab-problems': t`Switch to Problems`,
      'tab-saves': t`Switch to Saves`,
      'tab-notes': t`Switch to Notes`,
      'tab-console': t`Switch to Console`,
      'tab-performance': t`Switch to Performance`,
      'new-profile': t`Create a new profile`,
      'duplicate-profile': t`Duplicate the open profile`,
      'rename-profile': t`Rename the open profile`,
      'find-all-mods': t`Find a mod in all profiles`,
      import: t`Open the Import dialog`,
      'export-profile': t`Export or share the open profile`,
      downloads: t`Toggle Downloads`,
      notifications: t`Open notification history`,
      'previous-profile': t`Open the previous profile`,
      'next-profile': t`Open the next profile`,
      'collapse-sidebar': t`Collapse or expand the profile sidebar`,
      back: t`Go back`,
      help: t`Get help`,
      'vanilla-play': t`Play the open profile without SMAPI`,
    }),
    [t],
  )
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
        px: 2,
        py: 1,
        borderBottom: '1px solid var(--mortar-hairline)',
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
          aria-label={t`Change shortcut`}
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
          sx={{ whiteSpace: 'nowrap', minWidth: 0 }}
        >
          {t`Reset`}
        </Button>
      </Box>
    </Box>
  )
}

export function Shortcuts() {
  const { t } = useLingui()
  const [filter, setFilter] = useState('')
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
  const rows = SHORTCUTS.filter((row) =>
    labels[row.id].toLowerCase().includes(filter.toLowerCase()),
  )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 14 }}>
      <TextField
        size="small"
        label={t`Filter shortcuts`}
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />
      <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          size="small"
          onClick={() => {
            setConflict(null)
            SetShortcuts(defaultBindings()).catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Reset all`}
        </Button>
      </Box>
      {(['General', 'Navigation', 'Profiles', 'Mods list', 'Console'] as const).map((group) => {
        const grouped = rows.filter((row) => row.group === group)
        return grouped.length > 0 ? (
          <Box key={group}>
            <Box sx={{ mb: 0.5, fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>
              {group}
            </Box>
            <Box sx={{ bgcolor: 'var(--mortar-overlay-25)', borderRadius: 1, overflow: 'hidden' }}>
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
            </Box>
          </Box>
        ) : null
      })}
    </Box>
  )
}
