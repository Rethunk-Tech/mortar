import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, TextField, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import type { SettingsSection } from '../nav/store.ts'
import { useNav } from '../nav/store.ts'

const ACTIVE_WEIGHT = 600

export function SettingsNav({
  section,
  sections,
  onSection,
  query,
  setQuery,
}: {
  section: SettingsSection
  sections: { id: SettingsSection; label: string }[]
  onSection?: (id: SettingsSection) => void
  query: string
  setQuery: (query: string) => void
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
        bgcolor: 'var(--mortar-nav)',
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
        sx={{ width: '100%', mb: 0.5 }}
      />
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
              bgcolor: active ? 'var(--mortar-hairline-12)' : 'transparent',
              color: active ? 'var(--mortar-ink)' : 'var(--mortar-ink-sec)',
              '&:hover': { bgcolor: active ? 'var(--mortar-hairline-12)' : 'action.hover' },
            }}
          >
            {s.label}
          </ButtonBase>
        )
      })}
    </Box>
  )
}
