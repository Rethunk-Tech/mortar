import { Trans, useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Dialog, DialogContent, DialogTitle } from '@mui/material'
import { useEffect, useId } from 'react'
import { type SettingsSection, useSettingsDialog } from './SettingsDialogStore.ts'
import { About } from './sections/About.tsx'
import { Appearance } from './sections/Appearance.tsx'

export function SettingsDialog() {
  const { t } = useLingui()
  const titleId = useId()
  const { open, section, close, setSection, openSettings } = useSettingsDialog()
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === ',') {
        e.preventDefault()
        openSettings()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [openSettings])
  const sections: { id: SettingsSection; label: string }[] = [
    { id: 'appearance', label: t`Appearance` },
    { id: 'about', label: t`About` },
  ]
  return (
    <Dialog open={open} onClose={close} fullWidth={true} maxWidth="md" aria-labelledby={titleId}>
      <DialogTitle id={titleId}>
        <Trans>Settings</Trans>
      </DialogTitle>
      <DialogContent sx={{ display: 'flex', gap: 3, minHeight: 360 }}>
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
        <Box sx={{ flexGrow: 1, minWidth: 0, pt: 1 }}>
          {section === 'appearance' ? <Appearance /> : <About />}
        </Box>
      </DialogContent>
    </Dialog>
  )
}
