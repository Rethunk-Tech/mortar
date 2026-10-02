import { useLingui } from '@lingui/react/macro'
import { Box, TextField } from '@mui/material'
import { useMemo, useState } from 'react'
import { SHORTCUTS } from '../shortcuts.ts'

export function Shortcuts() {
  const { t } = useLingui()
  const [filter, setFilter] = useState('')
  const labels = useMemo(
    (): Record<(typeof SHORTCUTS)[number]['id'], string> => ({
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
  const rows = useMemo(
    () => SHORTCUTS.filter((row) => labels[row.id].toLowerCase().includes(filter.toLowerCase())),
    [filter, labels],
  )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 14 }}>
      <TextField
        size="small"
        label={t`Filter shortcuts`}
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />
      {(['General', 'Navigation', 'Profiles', 'Mods list', 'Console'] as const).map((group) => {
        const grouped = rows.filter((row) => row.group === group)
        return grouped.length > 0 ? (
          <Box key={group}>
            <Box sx={{ mb: 0.5, fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>
              {group}
            </Box>
            <Box sx={{ bgcolor: 'rgba(0,0,0,0.25)', borderRadius: 1, overflow: 'hidden' }}>
              {grouped.map((row) => (
                <Box
                  key={row.id}
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    gap: 2,
                    px: 2,
                    py: 1,
                    borderBottom: '1px solid rgba(255,255,255,0.1)',
                  }}
                >
                  <Box component="span">{labels[row.id]}</Box>
                  <Box
                    component="kbd"
                    sx={{
                      color: 'rgba(225,225,230,0.95)',
                      fontFamily: 'inherit',
                      fontSize: 12,
                      px: 0.75,
                      border: '1px solid rgba(255,255,255,0.25)',
                      borderRadius: 0.5,
                    }}
                  >
                    {row.keys}
                  </Box>
                </Box>
              ))}
            </Box>
          </Box>
        ) : null
      })}
    </Box>
  )
}
