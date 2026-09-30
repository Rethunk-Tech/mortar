import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import { useEffect } from 'react'
import { type SettingsSection, useNav } from '../nav/store.ts'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'

export function SettingsPage({ section }: { section: SettingsSection }) {
  const { t } = useLingui()
  const closeSettings = useNav((s) => s.closeSettings)
  const setSection = useNav((s) => s.openSettings)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeSettings()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [closeSettings])
  const sections: { id: SettingsSection; label: string }[] = [
    { id: 'appearance', label: t`Appearance` },
    { id: 'about', label: t`About` },
  ]
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 2, py: 1.5 }}>
        <ButtonBase
          aria-label={t`Back`}
          onClick={closeSettings}
          sx={{
            width: 36,
            height: 36,
            borderRadius: '6px',
            '&:hover': { bgcolor: 'action.hover' },
          }}
        >
          <ArrowLeft size={20} />
        </ButtonBase>
        <Typography component="h1" sx={{ fontSize: 22, fontWeight: 600 }}>
          {t`Settings`}
        </Typography>
      </Box>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex', gap: 3, px: 2, pb: 2 }}>
        <Box
          component="nav"
          aria-label={t`Settings sections`}
          sx={{ width: 160, flexShrink: 0, display: 'flex', flexDirection: 'column' }}
        >
          {sections.map((s) => (
            <ButtonBase
              key={s.id}
              onClick={() => setSection(s.id)}
              aria-current={section === s.id ? 'page' : undefined}
              sx={{
                justifyContent: 'flex-start',
                height: 38,
                px: '12px',
                fontSize: 14,
                fontFamily: 'inherit',
                whiteSpace: 'nowrap',
                borderLeft: '2px solid',
                borderColor: section === s.id ? 'primary.main' : 'transparent',
                color: section === s.id ? '#ffffff' : 'rgba(210,210,215,0.92)',
              }}
            >
              {s.label}
            </ButtonBase>
          ))}
        </Box>
        <Box sx={{ flexGrow: 1, minWidth: 0, overflow: 'auto', pt: 1 }}>
          {section === 'appearance' ? <Appearance /> : <About />}
        </Box>
      </Box>
    </Box>
  )
}
