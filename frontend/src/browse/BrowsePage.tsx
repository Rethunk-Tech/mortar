import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { useCallback, useEffect, useMemo, useRef } from 'react'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { persist } from '../settings/persist.ts'
import { useOfflineEmpty } from '../shell/offlineText.ts'
import { space } from '../theme/density.ts'
import { BrowseBody, ResultFooter } from './BrowseBody.tsx'
import { BrowseDetails } from './BrowseDetails.tsx'
import { BrowseFilters } from './BrowseFilters.tsx'
import { BrowseMenu } from './BrowseMenu.tsx'
import { BrowseToolbar } from './BrowseToolbar.tsx'
import { ALL, FIRST_PAGE, GITHUB, searchHint } from './browseConstants.ts'
import { type BrowseModes, formatModes } from './browseModes.ts'
import { PAGE_SIZE } from './browseState.ts'
import type { BrowseFilter, BrowseItem, BrowsePageProps } from './browseTypes.ts'
import { hitKey, stepSelection, useBrowseSelection } from './selection.ts'
import { pagedTotal, useBrowseQuery } from './useBrowseQuery.ts'
import { useCategoryNames } from './useCategoryNames.ts'
import { useBrowseView } from './view.ts'

// Up and Down move the details panel to the next hit and Escape closes it, unless a field, dialog or menu has the key.
function useSelectionKeys(items: BrowseItem[]) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const owned =
        e.target instanceof Element &&
        e.target.closest(
          'input, textarea, select, [role="dialog"], [role="menu"], [contenteditable]',
        )
      const { selected, select } = useBrowseSelection.getState()
      if (owned || e.defaultPrevented) {
        return
      }
      if (e.key === 'Escape' && selected) {
        select(null)
        return
      }
      if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') {
        return
      }
      const next = stepSelection(items, selected?.item ?? null, e.key === 'ArrowDown' ? 1 : -1)
      if (next) {
        e.preventDefault()
        select({ item: next, source: next.source })
        document
          .querySelector(`[data-hit="${CSS.escape(hitKey(next))}"]`)
          ?.scrollIntoView({ block: 'nearest' })
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [items])
  // A panel left open belongs to this visit to Browse only.
  useEffect(() => () => useBrowseSelection.getState().select(null), [])
}

// ALL searches every source the game has; it is the default so where a mod is published never matters to the player.

// A new page of results starts at its top rather than where the last page was scrolled to.
function useTopOnPage(page: number) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (page >= FIRST_PAGE) {
      ref.current?.scrollTo({ top: 0 })
    }
  }, [page])
  return ref
}
// The toolbar's and filters' handlers, stable so those memoised parts skip a render when results arrive.
const clearSearch = () => useBrowseView.getState().setDraft('')

function useBrowseHandlers(
  game: string,
  {
    setSource,
    setText,
    setPage,
  }: Pick<ReturnType<typeof useBrowseQuery>, 'setSource' | 'setText' | 'setPage'>,
) {
  const { t } = useLingui()
  const setFilter = useBrowseView((s) => s.setFilter)
  const onSource = useCallback(
    (next: string) => {
      setSource(next)
      setPage(FIRST_PAGE)
    },
    [setSource, setPage],
  )
  const onText = useCallback(
    (next: string) => {
      setText(next)
      setPage(FIRST_PAGE)
    },
    [setText, setPage],
  )
  const onFilter = useCallback(
    (next: BrowseFilter) => {
      setFilter(game, next)
      setPage(FIRST_PAGE)
    },
    [setFilter, game, setPage],
  )
  const onModes = useCallback(
    (next: BrowseModes) => {
      persist(
        () => SetByKey('browseFilters', formatModes(next), game),
        t`Could not save that setting`,
      )
      setPage(FIRST_PAGE)
    },
    [game, t, setPage],
  )
  return { onSource, onText, onFilter, onModes }
}

function BrowsePage({
  game,
  profileID,
  premium,
  sources: searchable,
  search,
  categories,
  defaultExcludes,
  openUrl,
  downloadNexus,
  addGitHub,
  addPackage,
  addDirect,
  hasCompat,
}: BrowsePageProps) {
  const { t } = useLingui()
  const view = useBrowseView((s) => s.view)
  const query = useBrowseQuery({
    game,
    profileID,
    search,
    sources: searchable,
    defaultExcludes,
  })
  const { filter, sorts, modes, source, page, setPage, result } = query
  const offline = useOfflineEmpty(source === ALL ? searchable.map((s) => s.id) : [source])
  const categoryNames = useCategoryNames({ categories, game, source, skip: source === GITHUB })
  const pageCount = Math.max(FIRST_PAGE, Math.ceil(pagedTotal(result) / PAGE_SIZE) || FIRST_PAGE)
  // Stable, so the memoised toolbar skips a render when results arrive.
  const sources = useMemo(
    () => [
      ...(searchable.length > 1 ? [{ value: ALL, label: t`All sources` }] : []),
      ...searchable.map((s) => ({ value: s.id, label: s.name, unavailable: s.unavailable })),
    ],
    [searchable, t],
  )
  const { onSource, onText, onFilter, onModes } = useBrowseHandlers(game, query)
  const { setRetry } = query
  const retry = useCallback(() => setRetry((n) => n + 1), [setRetry])
  const sourceName = sources.find((s) => s.value === source)?.label ?? ''
  useSelectionKeys(result.items)
  const results = useTopOnPage(page)
  const sourceNames = useMemo(() => new Map(searchable.map((s) => [s.id, s.name])), [searchable])
  // One object for every card, kept while its parts are, so the memoised cards skip a render when results arrive.
  const card = useMemo(
    () => ({
      premium,
      openUrl,
      downloadNexus,
      addGitHub,
      addPackage,
      addDirect,
      modes,
      profileID,
      sourceNames,
    }),
    [
      premium,
      openUrl,
      downloadNexus,
      addGitHub,
      addPackage,
      addDirect,
      modes,
      profileID,
      sourceNames,
    ],
  )
  let placeholder = t`Search ${sourceName}`
  if (source === ALL) {
    placeholder = t`Search all mod sites`
  } else if (source === GITHUB) {
    placeholder = t`Search GitHub releases`
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <BrowseToolbar
        sources={sources}
        source={source}
        onSource={onSource}
        onText={onText}
        placeholder={placeholder}
      />
      <BrowseFilters
        categories={categoryNames}
        filter={filter}
        onFilter={onFilter}
        modes={modes}
        hasCompat={hasCompat}
        source={source}
        sorts={sorts}
        onModes={onModes}
      />
      <Box
        sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}
      >
        <Box sx={{ minHeight: 0, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          <Box
            ref={results}
            sx={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              px: space.gutter,
              py: space.gap,
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <BrowseBody
              status={query.status}
              result={result}
              text={query.text}
              hint={searchHint(source, premium)}
              error={query.error}
              offline={offline}
              view={view}
              page={page}
              pageCount={pageCount}
              card={card}
              onRetry={retry}
              onClear={clearSearch}
              onPage={setPage}
            />
          </Box>
          {query.status !== 'idle' && query.status !== 'error' && result.items.length > 0 ? (
            <ResultFooter result={result} />
          ) : null}
        </Box>
        <BrowseDetails game={game} card={card} sourceNames={sourceNames} />
      </Box>
      <BrowseMenu card={card} sourceNames={sourceNames} />
    </Box>
  )
}

export { BrowsePage }
