import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { SHORTCUTS } from '../shortcuts.ts'

export function Shortcuts() {
  const { t } = useLingui()
  const labels: Record<(typeof SHORTCUTS)[number]['id'], string> = {
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
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '10px', fontSize: 14 }}>
      {SHORTCUTS.map((row) => (
        <Box
          key={row.id}
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            gap: '16px',
            whiteSpace: 'nowrap',
          }}
        >
          <Box component="span">{labels[row.id]}</Box>
          <Box component="kbd" sx={{ color: 'rgba(225,225,230,0.95)' }}>
            {row.keys}
          </Box>
        </Box>
      ))}
    </Box>
  )
}
