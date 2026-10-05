import { useEffect, useMemo, useState } from 'react'
import { useSettings } from '../settings/store.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { ALL, FIRST_PAGE } from './browseConstants.ts'
import { parseModes } from './browseModes.ts'
import { clampPage, DEBOUNCE_MS, PAGE_SIZE } from './browseState.ts'
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
  sort: '',
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
}: Pick<BrowsePageProps, 'game' | 'profileID' | 'search' | 'sources'>) {
  const stored = useBrowseView((s) => s.filters[game]) ?? NO_FILTER
  const saved = useSettings((s) => s.games?.[game]?.browseFilters ?? '')
  const modes = useMemo(() => parseModes(saved), [saved])
  const filter = useMemo(() => ({ ...stored, ...modes }), [stored, modes])
  const [chosen, setSource] = useState(ALL)
  const merged = sources.length > 1
  const known = sources.some((s) => s.id === chosen) || (chosen === ALL && merged)
  const fallback = merged ? ALL : (sources[0]?.id ?? '')
  const source = known ? chosen : fallback
  const [draft, setDraft] = useState('')
  const [text, setText] = useState('')
  const [page, setPage] = useState(FIRST_PAGE)
  const [retry, setRetry] = useState(0)
  const [result, setResult] = useState<BrowseResult>(EMPTY_RESULT)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<InlineError | null>(null)

  const pendingQuery = useBrowseView((s) => s.pendingQuery)
  useEffect(() => {
    if (pendingQuery !== '') {
      setSource(ALL)
      setDraft(pendingQuery)
      useBrowseView.getState().setPendingQuery('')
    }
  }, [pendingQuery])

  useEffect(() => {
    const timer = setTimeout(() => {
      setText(draft)
      setPage(FIRST_PAGE)
    }, DEBOUNCE_MS)
    return () => {
      clearTimeout(timer)
    }
  }, [draft])

  useEffect(() => {
    if (source === '' || retry < 0) {
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
  }, [game, source, text, page, profileID, search, retry, filter])

  return {
    filter,
    modes,
    source,
    setSource,
    draft,
    setDraft,
    text,
    page,
    setPage,
    setRetry,
    result,
    status,
    error,
  }
}

export { type BrowseResult, pagedTotal, type Status, useBrowseQuery }
