import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { type LucideIcon, SearchX } from 'lucide-react'
import { type ReactNode, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SEARCH_HIT } from './prefFilter.ts'
import { SettingsNav } from './SettingsNav.tsx'
import { SettingsSearchProvider } from './SettingsSearch.tsx'
import { dialogOpen } from './shortcuts.ts'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

// One readable column: wider rows push controls too far from their labels, so the content stops growing here.
const CONTENT_MAX = 880
const NAV_WIDTH = 224
const NAV_WIDTH_NARROW = 196
const NARROW_WINDOW = 999

export interface ShellPage<Id extends string> {
  id: Id
  label: string
  icon: LucideIcon
  // A divider follows this page in the nav.
  groupEnd?: boolean
}

// The layout every settings screen shares: a nav of pages with search on the left, one page (or every page's
// matches while searching) on the right.
export function SettingsShell<Id extends string>({
  title,
  backLabel,
  onBack,
  pages,
  current,
  onPage,
  render,
  actions = {},
}: {
  title: string
  backLabel: string
  onBack: () => void
  pages: ShellPage<Id>[]
  current: Id
  onPage: (id: Id) => void
  render: (id: Id) => ReactNode
  // Page-wide actions sit at the right of the page title.
  actions?: Partial<Record<Id, ReactNode>>
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const pane = useRef<HTMLDivElement>(null)
  const results = useRef<HTMLDivElement>(null)
  const [noneMatch, setNoneMatch] = useState(false)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && query) {
        setQuery('')
        e.preventDefault()
        return
      }
      if (shouldLeavePageOnEscape(e, dialogOpen())) {
        onBack()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [onBack, query])
  useLayoutEffect(() => {
    if (!query) {
      setNoneMatch(false)
      return
    }
    const root = results.current
    setNoneMatch(root !== null && root.querySelector(SEARCH_HIT) === null)
  }, [query])
  const pick = (id: Id) => {
    onPage(id)
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
          title={title}
          backLabel={backLabel}
          onBack={onBack}
          pages={pages}
          current={current}
          onPage={pick}
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
          <Box
            sx={{
              width: '100%',
              maxWidth: CONTENT_MAX,
              display: 'flex',
              alignItems: 'center',
              gap: 2,
            }}
          >
            <Typography
              component="h2"
              sx={{
                flex: 1,
                fontSize: 20,
                fontWeight: 700,
                minHeight: 36,
                display: 'flex',
                alignItems: 'center',
              }}
            >
              {query ? t`Search results` : pages.find((p) => p.id === current)?.label}
            </Typography>
            {query ? null : actions[current]}
          </Box>
          <Box
            ref={results}
            sx={{
              width: '100%',
              maxWidth: CONTENT_MAX,
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
            }}
          >
            {query && noneMatch ? (
              <EmptyState icon={<SearchX />} title={t`No settings match`} compact={true}>
                {t`Try another word, or clear the search.`}
              </EmptyState>
            ) : null}
            {query
              ? pages.map((p) => (
                  <Box
                    key={p.id}
                    id={`settings-section-${p.id}`}
                    sx={{
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 2,
                      mb: 2,
                      [`&:not(:has(${SEARCH_HIT}))`]: {
                        display: 'none',
                      },
                    }}
                  >
                    {render(p.id)}
                  </Box>
                ))
              : render(current)}
          </Box>
        </Box>
      </Box>
    </SettingsSearchProvider>
  )
}
