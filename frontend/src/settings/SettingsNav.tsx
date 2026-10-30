import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { useDebouncedValue } from '@rethunk/react-load-hooks'
import { ArrowLeft } from 'lucide-react'
import { memo, type ReactElement, useEffect, useState } from 'react'
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
  const settled = useDebouncedValue(text, SEARCH_SETTLE_MS)
  useEffect(() => {
    if (settled !== settingsQuery()) {
      setSettingsQuery(settled)
    }
  }, [settled])
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

interface NavItemProps<Id extends string> {
  id: Id
  label: string
  icon: ShellPage<Id>['icon']
  groupEnd: boolean
  active: boolean
  onPage: (id: Id) => void
}

// Memoised: a section change redraws only the row leaving and the row taking the highlight.
const NavItem = memo(function NavItemView<Id extends string>({
  id,
  label,
  icon: Icon,
  groupEnd,
  active,
  onPage,
}: NavItemProps<Id>) {
  return (
    <>
      <ButtonBase
        onClick={() => onPage(id)}
        aria-current={active ? 'page' : undefined}
        sx={{
          justifyContent: 'flex-start',
          gap: space.gap,
          height: space.nav,
          px: '12px',
          borderRadius: '6px',
          fontSize: 'lead.fontSize',
          fontWeight: active ? ACTIVE_WEIGHT : 'normal',
          fontFamily: 'inherit',
          whiteSpace: 'nowrap',
          bgcolor: active ? 'var(--mortar-hairline-12)' : 'transparent',
          color: active ? 'var(--mortar-ink)' : 'var(--mortar-ink-sec)',
          '&:hover': { bgcolor: active ? 'var(--mortar-hairline-12)' : 'action.hover' },
        }}
      >
        <Icon size={ICON_SIZE} aria-hidden={true} />
        {label}
      </ButtonBase>
      {groupEnd ? (
        <Box
          role="separator"
          sx={{ height: '1px', bgcolor: 'var(--mortar-hairline)', mx: 1, my: 0.75 }}
        />
      ) : null}
    </>
  )
}) as <Id extends string>(props: NavItemProps<Id>) => ReactElement

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
      {pages.map((s) => (
        <NavItem
          key={s.id}
          id={s.id}
          label={s.label}
          icon={s.icon}
          groupEnd={s.groupEnd === true}
          active={current === s.id}
          onPage={onPage}
        />
      ))}
    </Box>
  )
}
