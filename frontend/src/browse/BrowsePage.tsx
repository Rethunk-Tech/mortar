import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { useEffect } from 'react'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { persist } from '../settings/persist.ts'
import { useOfflineReason } from '../shell/offlineText.ts'
import { useToasts } from '../toasts/store.ts'
import { BrowseBody } from './BrowseBody.tsx'
import { BrowseDetails } from './BrowseDetails.tsx'
import { BrowseFilters } from './BrowseFilters.tsx'
import { BrowseMenu } from './BrowseMenu.tsx'
import { BrowseToolbar } from './BrowseToolbar.tsx'
import { ALL, FIRST_PAGE, GITHUB, searchHint } from './browseConstants.ts'
import { formatModes } from './browseModes.ts'
import { PAGE_SIZE } from './browseState.ts'
import type { BrowseItem, BrowsePageProps } from './browseTypes.ts'
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
function BrowsePage({
  game,
  profileID,
  premium,
  sources: searchable,
  search,
  categories,
  openUrl,
  downloadNexus,
  addGitHub,
  addPackage,
  addDirect,
  hasCompat,
}: BrowsePageProps) {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const view = useBrowseView((s) => s.view)
  const setFilter = useBrowseView((s) => s.setFilter)
  const query = useBrowseQuery({ game, profileID, search, sources: searchable })
  const { filter, modes, source, setSource, draft, setDraft, page, setPage, result } = query
  const offline = useOfflineReason(source === ALL ? searchable.map((s) => s.id) : [source])
  const categoryNames = useCategoryNames({ categories, game, source, skip: source === GITHUB })
  const pageCount = Math.max(FIRST_PAGE, Math.ceil(pagedTotal(result) / PAGE_SIZE) || FIRST_PAGE)
  const sources = [
    ...(searchable.length > 1 ? [{ value: ALL, label: t`All sources` }] : []),
    ...searchable.map((s) => ({ value: s.id, label: s.name, unavailable: s.unavailable })),
  ]
  const sourceName = sources.find((s) => s.value === source)?.label ?? ''
  useSelectionKeys(result.items)
  const sourceNames = new Map(searchable.map((s) => [s.id, s.name]))
  const card = {
    premium,
    openUrl,
    downloadNexus,
    addGitHub,
    addPackage,
    addDirect,
    modes,
    profileID,
    sourceNames,
  }
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
        onSource={(next) => {
          setSource(next)
          setPage(FIRST_PAGE)
        }}
        draft={draft}
        onDraft={setDraft}
        placeholder={placeholder}
      />
      <BrowseFilters
        categories={categoryNames}
        filter={filter}
        onFilter={(next) => {
          setFilter(game, next)
          setPage(FIRST_PAGE)
        }}
        modes={modes}
        hasCompat={hasCompat}
        onModes={(next) => {
          persist(
            () => SetByKey('browseFilters', formatModes(next), game),
            push,
            t`Could not save that setting`,
          )
          setPage(FIRST_PAGE)
        }}
      />
      <Box
        sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}
      >
        <Box
          sx={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            px: 2,
            py: 1,
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
            onRetry={() => query.setRetry((n) => n + 1)}
            onClear={() => setDraft('')}
            onPage={setPage}
          />
        </Box>
        <BrowseDetails game={game} card={card} sourceNames={sourceNames} />
      </Box>
      <BrowseMenu card={card} sourceNames={sourceNames} />
    </Box>
  )
}

export { BrowsePage }
