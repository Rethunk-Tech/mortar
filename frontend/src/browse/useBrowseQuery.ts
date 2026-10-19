import { useEffect, useMemo, useState } from 'react'
import { useSettings } from '../settings/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { ALL, FIRST_PAGE } from './browseConstants.ts'
import { parseModes } from './browseModes.ts'
import { effectiveSort, sortsFor } from './browseSort.ts'
import { clampPage, PAGE_SIZE } from './browseState.ts'
import type { BrowseFilter, BrowseItem, BrowsePageProps } from './browseTypes.ts'
import { useBrowseView } from './view.ts'

type Status = 'idle' | 'loading' | 'done' | 'error'

interface BrowseResult {
  total: number
  items: BrowseItem[]
  pages?: number
  failed?: string[]
  hidden?: number
}

const EMPTY_RESULT: BrowseResult = { total: 0, items: [] }
const NO_FILTER: BrowseFilter = {
  include: [],
  exclude: [],
  sort: 'downloads',
  installed: '',
  obsolete: '',
  broken: '',
}

// pagedTotal is the result count paging should assume: a search across sources pages by its largest source.
function pagedTotal(r: BrowseResult): number {
  return r.pages ? r.pages * PAGE_SIZE : r.total
}

function useBrowseQuery({
  game,
  profileID,
  search,
  sources,
  defaultExcludes,
}: Pick<BrowsePageProps, 'game' | 'profileID' | 'search' | 'sources' | 'defaultExcludes'>) {
  const chosenFilter = useBrowseView((s) => s.filters[game])
  const saved = useSettings((s) => s.games?.[game]?.browseFilters ?? '')
  const modes = useMemo(() => parseModes(saved), [saved])
  const [chosen, setSource] = useState(ALL)
  const merged = sources.length > 1
  const known = sources.some((s) => s.id === chosen) || (chosen === ALL && merged)
  const fallback = merged ? ALL : (sources[0]?.id ?? '')
  const source = known ? chosen : fallback
  const defaults = useLoaded<string[]>(
    chosenFilter || source === '' ? null : () => defaultExcludes(game, source),
    [defaultExcludes, game, source, chosenFilter !== undefined],
    [],
  )
  // Until the player edits the filter, it starts with the default excludes; the first search waits for them.
  const stored = useMemo(
    () => chosenFilter ?? { ...NO_FILTER, exclude: defaults.data },
    [chosenFilter, defaults.data],
  )
  const waiting = chosenFilter === undefined && defaults.loading
  const sorts = sortsFor(source, sources)
  const filter = useMemo(
    () => ({ ...stored, ...modes, sort: effectiveSort(stored.sort, sorts) }),
    [stored, modes, sorts],
  )
  const [text, setText] = useState('')
  const [page, setPage] = useState(FIRST_PAGE)
  const [retry, setRetry] = useState(0)
  const hides = useBrowseView((s) => s.hides)
  const [result, setResult] = useState<BrowseResult>(EMPTY_RESULT)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<InlineError | null>(null)

  const pendingQuery = useBrowseView((s) => s.pendingQuery)
  useEffect(() => {
    if (pendingQuery !== '') {
      setSource(ALL)
      useBrowseView.getState().setDraft(pendingQuery)
      useBrowseView.getState().setPendingQuery('')
    }
  }, [pendingQuery])
  useEffect(() => () => useBrowseView.getState().setDraft(''), [])

  useEffect(() => {
    if (source === '' || waiting || retry + hides < 0) {
      setResult(EMPTY_RESULT)
      setStatus('idle')
      return
    }
    let cancelled = false
    setStatus('loading')
    search({ game, source, text, page, profileID, filter })
      .then((next) => {
        if (!cancelled) {
          setResult(next)
          setStatus('done')
          setPage((current) => clampPage({ page: current, total: pagedTotal(next) }))
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(inlineError(err))
          setStatus('error')
        }
      })
    return () => {
      cancelled = true
    }
  }, [game, source, text, page, profileID, search, retry, hides, filter, waiting])

  return {
    filter,
    sorts,
    modes,
    source,
    setSource,
    text,
    setText,
    page,
    setPage,
    setRetry,
    result,
    status,
    error,
  }
}

export { type BrowseResult, pagedTotal, type Status, useBrowseQuery }
