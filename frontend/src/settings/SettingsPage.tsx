import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import { type SettingsSection, useNav } from '../nav/store.ts'
import { SettingsNav } from './SettingsNav.tsx'
import { SettingsSearchProvider } from './SettingsSearch.tsx'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'
import { Downloads } from './sections/Downloads.tsx'
import { General } from './sections/General.tsx'
import { Launchers } from './sections/Launchers.tsx'
import { ModsProfiles } from './sections/ModsProfiles.tsx'
import { NexusMods } from './sections/NexusMods.tsx'
import { Notifications } from './sections/Notifications.tsx'
import { Shortcuts } from './sections/Shortcuts.tsx'
import { Storage } from './sections/Storage.tsx'
import { Updates } from './sections/Updates.tsx'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

// One readable column: wider rows push controls too far from their labels, so the content stops growing here.
const CONTENT_MAX = 880
const NAV_WIDTH = 224
const NAV_WIDTH_NARROW = 196
const NARROW_WINDOW = 999

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
    { id: 'mods', label: t`Mods and profiles` },
    { id: 'downloads', label: t`Downloads` },
    { id: 'nexus', label: t`Nexus account` },
    { id: 'updates', label: t`Updates` },
    { id: 'notifications', label: t`Notifications` },
    { id: 'storage', label: t`Storage` },
    { id: 'launchers', label: t`Launchers` },
    { id: 'shortcuts', label: t`Shortcuts` },
    { id: 'about', label: t`About` },
  ]
  const body: Record<SettingsSection, ReactNode> = {
    general: <General />,
    appearance: <Appearance />,
    mods: <ModsProfiles />,
    downloads: <Downloads />,
    nexus: <NexusMods />,
    updates: <Updates />,
    notifications: <Notifications />,
    storage: <Storage />,
    launchers: <Launchers />,
    shortcuts: <Shortcuts />,
    about: <About />,
  }
  const current = sections.find((s) => s.id === section)
  const pane = useRef<HTMLDivElement>(null)
  const pickSection = (id: SettingsSection) => {
    useNav.getState().openSettings(id)
    if (query) {
      document.getElementById(`settings-section-${id}`)?.scrollIntoView({ block: 'start' })
      return
    }
    pane.current?.scrollTo(0, 0)
  }
  return (
    <SettingsSearchProvider query={query}>
      <Box
        sx={{
          height: '100%',
          display: 'grid',
          gridTemplateColumns: `${NAV_WIDTH}px minmax(0, 1fr)`,
          [`@media (max-width: ${NARROW_WINDOW}px)`]: {
            gridTemplateColumns: `${NAV_WIDTH_NARROW}px minmax(0, 1fr)`,
          },
        }}
      >
        <SettingsNav
          section={section}
          sections={sections}
          onSection={pickSection}
          query={query}
          setQuery={setQuery}
        />
        <Box
          ref={pane}
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
            {query ? t`Search results` : current?.label}
          </Typography>
          <Box
            sx={{
              width: '100%',
              maxWidth: CONTENT_MAX,
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
            }}
          >
            {query
              ? sections.map((s) => (
                  <Box
                    key={s.id}
                    id={`settings-section-${s.id}`}
                    sx={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 2,
                      mb: 2,
                      '&:not(:has(.settings-tiles:not(:empty), .settings-match))': {
                        display: 'none',
                      },
                    }}
                  >
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
