import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { type LucideIcon, Search, SearchX } from 'lucide-react'
import { type ReactNode, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SEARCH_HIT } from './prefFilter.ts'
import { SettingsNav } from './SettingsNav.tsx'
import { dialogOpen } from './shortcuts.ts'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'
import { SettingsSearchContext } from './useSettingsSearch.ts'

// One readable column: wider rows push controls too far from their labels, so the content stops growing here.
const CONTENT_MAX = 880
const NAV_WIDTH = 224
const NAV_WIDTH_NARROW = 196
const NARROW_WINDOW = 999
// At most this many matching rows also offers the search on the other settings screen.
const FEW_HITS = 3

type Elsewhere = { label: string; search: (query: string) => void } | undefined
type Related = { match: (query: string) => string[]; row: (label: string) => ReactNode } | undefined

// What a search offers beside its own matches: the settings that live in another place, and the same search there.
function SearchLeads({
  query,
  hits,
  relatedLabels,
  related,
  elsewhere,
}: {
  query: string
  hits: number
  relatedLabels: string[]
  related: Related
  elsewhere: Elsewhere
}) {
  return (
    <>
      {relatedLabels.map((label) => (
        <Box key={label}>{related?.row(label)}</Box>
      ))}
      {query && elsewhere && hits <= FEW_HITS ? (
        <Button
          startIcon={<Search size={16} />}
          onClick={() => elsewhere.search(query)}
          sx={{ alignSelf: 'flex-start' }}
        >
          {elsewhere.label}
        </Button>
      ) : null}
    </>
  )
}

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
  initialQuery = '',
  elsewhere,
  related,
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
  initialQuery?: string
  // Another settings screen to run the same search on, offered when this one finds little.
  elsewhere?: Elsewhere
  // Settings that live outside this screen: `match` names the ones a search finds and `row` shows each.
  related?: Related
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState(initialQuery)
  const pane = useRef<HTMLDivElement>(null)
  const results = useRef<HTMLDivElement>(null)
  const [hits, setHits] = useState(0)
  const relatedLabels = query && related ? related.match(query) : []
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
    setHits(query ? (results.current?.querySelectorAll(SEARCH_HIT).length ?? 0) : 0)
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
    <SettingsSearchContext value={query}>
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
            {query && hits === 0 && relatedLabels.length === 0 ? (
              <EmptyState icon={<SearchX />} title={t`No settings match`} compact={true}>
                {t`Try another word, or clear the search.`}
              </EmptyState>
            ) : null}
            <SearchLeads
              query={query}
              hits={hits}
              relatedLabels={relatedLabels}
              related={related}
              elsewhere={elsewhere}
            />
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
    </SettingsSearchContext>
  )
}
