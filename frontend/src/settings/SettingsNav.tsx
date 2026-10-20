import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import { useEffect, useState } from 'react'
import { SearchField } from '../shell/SearchField.tsx'
import { space } from '../theme/density.ts'
import type { ShellPage } from './SettingsShell.tsx'
import { setSettingsQuery, settingsQuery, useSettingsSearch } from './useSettingsSearch.ts'

const ACTIVE_WEIGHT = 600
const ICON_SIZE = 18

// How long typing pauses before the rows re-filter, so a word typed at speed filters the page once.
const SEARCH_SETTLE_MS = 150

// Its own component: only the field follows each key typed, not the nav around it. The rows follow the query once
// typing pauses; a clear from elsewhere (Escape in the shell, closing it) reaches the field at once through the store.
function NavSearch() {
  const { t } = useLingui()
  const query = useSettingsSearch()
  const [text, setText] = useState(query)
  useEffect(() => setText((typed) => (typed.trim() === query.trim() ? typed : query)), [query])
  useEffect(() => {
    if (text === settingsQuery()) {
      return
    }
    const id = setTimeout(() => setSettingsQuery(text), SEARCH_SETTLE_MS)
    return () => clearTimeout(id)
  }, [text])
  return (
    <SearchField
      label={t`Search settings`}
      value={text}
      onChange={setText}
      onKeyDown={(e) => {
        if (e.key === 'Escape' && text) {
          e.preventDefault()
          e.stopPropagation()
          setText('')
          setSettingsQuery('')
        }
      }}
      sx={{ width: '100%', mb: 0.5 }}
    />
  )
}

export function SettingsNav<Id extends string>({
  title,
  backLabel,
  onBack,
  pages,
  current,
  onPage,
}: {
  title: string
  backLabel: string
  onBack: () => void
  pages: ShellPage<Id>[]
  current: Id
  onPage: (id: Id) => void
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
        px: space.gap,
        py: space.pad,
        bgcolor: 'var(--mortar-nav)',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, pb: space.gap }}>
        <ButtonBase
          aria-label={backLabel}
          onClick={onBack}
          sx={{
            width: 36,
            height: space.row,
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
      <NavSearch />
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
                gap: space.gap,
                height: space.nav,
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
