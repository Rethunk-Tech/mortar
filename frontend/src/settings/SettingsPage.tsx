import { useLingui } from '@lingui/react/macro'
import { Box, TextField, Typography } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import { type SettingsSection, useNav } from '../nav/store.ts'
import { SettingsNav } from './SettingsNav.tsx'
import { SettingsSearchProvider } from './SettingsSearch.tsx'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'
import { Data } from './sections/Data.tsx'
import { General } from './sections/General.tsx'
import { Launchers } from './sections/Launchers.tsx'
import { NexusMods } from './sections/NexusMods.tsx'
import { Shortcuts } from './sections/Shortcuts.tsx'
import { Updates } from './sections/Updates.tsx'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

export function SettingsPage({ section }: { section: SettingsSection }) {
  const { t } = useLingui()
  const closeSettings = useNav((s) => s.closeSettings)
  const [query, setQuery] = useState('')
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && query) {
        setQuery('')
        e.preventDefault()
        return
      }
      if (shouldLeavePageOnEscape(e, document.querySelector('[role="dialog"]') !== null)) {
        closeSettings()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [closeSettings, query])
  const sections: { id: SettingsSection; label: string }[] = [
    { id: 'general', label: t`General` },
    { id: 'appearance', label: t`Appearance` },
    { id: 'launchers', label: t`Launchers` },
    { id: 'data', label: t`Data` },
    { id: 'nexus', label: t`Nexus Mods` },
    { id: 'updates', label: t`Updates` },
    { id: 'shortcuts', label: t`Shortcuts` },
    { id: 'about', label: t`About` },
  ]
  const body: Record<SettingsSection, ReactNode> = {
    general: <General />,
    appearance: <Appearance />,
    launchers: <Launchers />,
    data: <Data />,
    nexus: <NexusMods />,
    updates: <Updates />,
    shortcuts: <Shortcuts />,
    about: <About />,
  }
  const current = sections.find((s) => s.id === section)
  return (
    <SettingsSearchProvider query={query}>
      <Box sx={{ height: '100%', display: 'grid', gridTemplateColumns: '200px minmax(0, 1fr)' }}>
        <SettingsNav section={section} sections={sections} />
        <Box
          sx={{
            minWidth: 0,
            overflow: 'auto',
            px: 3.5,
            pt: 2,
            pb: 1.5,
            display: 'flex',
            flexDirection: 'column',
            gap: 2,
          }}
        >
          <Typography
            component="h2"
            sx={{
              fontSize: 20,
              fontWeight: 700,
              minHeight: 36,
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {current?.label}
          </Typography>
          <TextField
            size="small"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t`Search settings`}
            slotProps={{ htmlInput: { 'aria-label': t`Search settings` } }}
            onKeyDown={(e) => {
              if (e.key === 'Escape' && query) {
                e.preventDefault()
                e.stopPropagation()
                setQuery('')
              }
            }}
            sx={{ width: '100%', maxWidth: 760 }}
          />
          <Box
            sx={{
              width: '100%',
              maxWidth: 760,
              bgcolor: 'rgba(0,0,0,0.25)',
              borderRadius: 1,
              p: 2,
            }}
          >
            {query
              ? sections.map((s) => (
                  <Box key={s.id} sx={{ display: 'flex', flexDirection: 'column', gap: 2, mb: 2 }}>
                    {body[s.id]}
                  </Box>
                ))
              : body[section]}
          </Box>
        </Box>
      </Box>
    </SettingsSearchProvider>
  )
}
