import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import type { SettingsSection } from '../nav/store.ts'
import { useNav } from '../nav/store.ts'

const ACTIVE_WEIGHT = 600

export function SettingsNav({
  section,
  sections,
  onSection,
}: {
  section: SettingsSection
  sections: { id: SettingsSection; label: string }[]
  onSection?: (id: SettingsSection) => void
}) {
  const { t } = useLingui()
  const closeSettings = useNav((s) => s.closeSettings)
  const setSection = useNav((s) => s.openSettings)
  return (
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
            onClick={() => (onSection ?? setSection)(s.id)}
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
  )
}
