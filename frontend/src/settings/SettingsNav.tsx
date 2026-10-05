import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import { SearchField } from '../shell/SearchField.tsx'
import type { ShellPage } from './SettingsShell.tsx'

const ACTIVE_WEIGHT = 600
const ICON_SIZE = 18

export function SettingsNav<Id extends string>({
  title,
  backLabel,
  onBack,
  pages,
  current,
  onPage,
  query,
  setQuery,
}: {
  title: string
  backLabel: string
  onBack: () => void
  pages: ShellPage<Id>[]
  current: Id
  onPage: (id: Id) => void
  query: string
  setQuery: (query: string) => void
}) {
  const { t } = useLingui()
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
          aria-label={backLabel}
          onClick={onBack}
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
        <Typography
          component="h1"
          sx={{
            fontSize: 20,
            fontWeight: 700,
            minWidth: 0,
            lineHeight: 1.2,
            overflowWrap: 'anywhere',
          }}
        >
          {title}
        </Typography>
      </Box>
      <SearchField
        label={t`Search settings`}
        value={query}
        onChange={setQuery}
        onKeyDown={(e) => {
          if (e.key === 'Escape' && query) {
            e.preventDefault()
            e.stopPropagation()
            setQuery('')
          }
        }}
        sx={{ width: '100%', mb: 0.5 }}
      />
      {pages.map((s) => {
        const active = current === s.id
        const Icon = s.icon
        return (
          <Box key={s.id} sx={{ display: 'contents' }}>
            <ButtonBase
              onClick={() => onPage(s.id)}
              aria-current={active ? 'page' : undefined}
              sx={{
                justifyContent: 'flex-start',
                gap: 1.5,
                height: 42,
                px: '12px',
                borderRadius: '6px',
                fontSize: 15,
                fontWeight: active ? ACTIVE_WEIGHT : 'normal',
                fontFamily: 'inherit',
                whiteSpace: 'nowrap',
                bgcolor: active ? 'var(--mortar-hairline-12)' : 'transparent',
                color: active ? 'var(--mortar-ink)' : 'var(--mortar-ink-sec)',
                '&:hover': { bgcolor: active ? 'var(--mortar-hairline-12)' : 'action.hover' },
              }}
            >
              <Icon size={ICON_SIZE} aria-hidden={true} />
              {s.label}
            </ButtonBase>
            {s.groupEnd ? (
              <Box
                role="separator"
                sx={{ height: '1px', bgcolor: 'var(--mortar-hairline)', mx: 1, my: 0.75 }}
              />
            ) : null}
          </Box>
        )
      })}
    </Box>
  )
}
