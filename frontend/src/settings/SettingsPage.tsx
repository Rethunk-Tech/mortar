import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import { type SettingsSection, useNav } from '../nav/store.ts'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'
import { Data } from './sections/Data.tsx'
import { NexusMods } from './sections/NexusMods.tsx'
import { Shortcuts } from './sections/Shortcuts.tsx'
import { Updates } from './sections/Updates.tsx'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

const ACTIVE_WEIGHT = 600

export function SettingsPage({ section }: { section: SettingsSection }) {
  const { t } = useLingui()
  const closeSettings = useNav((s) => s.closeSettings)
  const setSection = useNav((s) => s.openSettings)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (shouldLeavePageOnEscape(e, document.querySelector('[role="dialog"]') !== null)) {
        closeSettings()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [closeSettings])
  const sections: { id: SettingsSection; label: string }[] = [
    { id: 'appearance', label: t`Appearance` },
    { id: 'data', label: t`Data` },
    { id: 'nexus', label: t`Nexus Mods` },
    { id: 'updates', label: t`Updates` },
    { id: 'shortcuts', label: t`Shortcuts` },
    { id: 'about', label: t`About` },
  ]
  const body: Record<SettingsSection, ReactNode> = {
    appearance: <Appearance />,
    data: <Data />,
    nexus: <NexusMods />,
    updates: <Updates />,
    shortcuts: <Shortcuts />,
    about: <About />,
  }
  const current = sections.find((s) => s.id === section)
  return (
    <Box sx={{ height: '100%', display: 'grid', gridTemplateColumns: '200px minmax(0, 1fr)' }}>
      <Box
        component="nav"
        aria-label={t`Settings sections`}
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: '2px',
          px: 1,
          py: 2,
          bgcolor: 'rgba(30,30,36,0.8)',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, pb: 1.25 }}>
          <ButtonBase
            aria-label={t`Back`}
            onClick={closeSettings}
            sx={{
              width: 36,
              height: 36,
              flexShrink: 0,
              borderRadius: '6px',
              '&:hover': { bgcolor: 'action.hover' },
            }}
          >
            <ArrowLeft size={20} />
          </ButtonBase>
          <Typography component="h1" sx={{ fontSize: 20, fontWeight: 700 }}>
            {t`Settings`}
          </Typography>
        </Box>
        {sections.map((s) => {
          const active = section === s.id
          return (
            <ButtonBase
              key={s.id}
              onClick={() => setSection(s.id)}
              aria-current={active ? 'page' : undefined}
              sx={{
                justifyContent: 'flex-start',
                height: 38,
                px: '12px',
                borderRadius: '6px',
                fontSize: 14,
                fontWeight: active ? ACTIVE_WEIGHT : 'normal',
                fontFamily: 'inherit',
                whiteSpace: 'nowrap',
                bgcolor: active ? 'rgba(255,255,255,0.12)' : 'transparent',
                color: active ? '#ffffff' : 'rgba(225,225,230,0.95)',
                '&:hover': { bgcolor: active ? 'rgba(255,255,255,0.12)' : 'action.hover' },
              }}
            >
              {s.label}
            </ButtonBase>
          )
        })}
      </Box>
      <Box
        sx={{
          minWidth: 0,
          overflow: 'auto',
          px: 3.5,
          py: 2,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
        }}
      >
        {/* Same 36px row and size as the Settings heading beside it, so the two titles line up. */}
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
        {body[section]}
      </Box>
    </Box>
  )
}
