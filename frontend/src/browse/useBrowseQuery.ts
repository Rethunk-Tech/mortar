import { useEffect, useMemo, useRef, useState } from 'react'
import { useSettings } from '../settings/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { ALL, FIRST_PAGE } from './browseConstants.ts'
import { parseModes } from './browseModes.ts'
import { parsePicks } from './browsePicks.ts'
import { effectiveSort, sortsFor } from './browseSort.ts'
import { clampPage, PAGE_SIZE, pageKeeping, screenSpan } from './browseState.ts'
import type { BrowseFilter, BrowseItem, BrowsePageProps } from './browseTypes.ts'
import { useBrowseView } from './view.ts'

type Status = 'idle' | 'loading' | 'done' | 'error'

interface BrowseResult {
  total: number
  items: BrowseItem[]
  pages?: number
  failed?: string[]
  unsorted?: string[]
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

// One screen from the source pages it spans: their hits in order, the hidden counts added and the failed sources once.
function joinPages(parts: BrowseResult[], skip: number, size: number): BrowseResult {
  const [head = EMPTY_RESULT] = parts
  return {
    ...head,
    items: parts.flatMap((p) => p.items).slice(skip, skip + size),
    hidden: parts.reduce((n, p) => n + (p.hidden ?? 0), 0),
    failed: [...new Set(parts.flatMap((p) => p.failed ?? []))],
  }
}

// One screen of `size` hits on screen `page`, from the source pages it spans; pages already fetched for this search
// are reused. A source page holding more than a page of hits comes back whole.
async function loadScreen(
  pages: Map<number, Promise<BrowseResult>>,
  fetchPage: (page: number) => Promise<BrowseResult>,
  page: number,
  size: number,
): Promise<{ next: BrowseResult; whole: boolean }> {
  const sourcePage = (p: number) => {
    let got = pages.get(p)
    if (!got) {
      got = fetchPage(p)
      pages.set(p, got)
      got.catch(() => pages.delete(p))
    }
    return got
  }
  const { first, last, skip } = screenSpan(page, size)
  const head = await sourcePage(first)
  if (head.items.length > PAGE_SIZE) {
    return { next: head, whole: true }
  }
  const known = head.pages ?? Math.ceil(head.total / PAGE_SIZE)
  const rest = await Promise.all(
    Array.from({ length: Math.max(0, Math.min(last, known) - first) }, (_, i) =>
      sourcePage(first + 1 + i),
    ),
  )
  return { next: joinPages([head, ...rest], skip, size), whole: false }
}

function useBrowseQuery({
  game,
  profileID,
  search,
  sources,
  defaultExcludes,
  fit,
}: Pick<BrowsePageProps, 'game' | 'profileID' | 'search' | 'sources' | 'defaultExcludes'> & {
  // How many results fill the window.
  fit: number
}) {
  const savedPicks = useSettings((s) => s.games?.[game]?.browsePicks ?? '')
  const chosenFilter = useMemo(() => {
    const picks = parsePicks(savedPicks)
    return picks ? { ...NO_FILTER, ...picks } : undefined
  }, [savedPicks])
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
  // A search across sources with a sort none of them ranks returns every source's page at once; those pages are shown
  // as they come, since screens cut from them would skip hits.
  const [whole, setWhole] = useState(false)
  const size = whole ? PAGE_SIZE : fit
  const shown = useRef(size)
  useEffect(() => {
    const from = shown.current
    if (from !== size) {
      setPage((current) => pageKeeping(current, from, size))
      shown.current = size
    }
  }, [size])
  const fetched = useRef<{ key: string; pages: Map<number, Promise<BrowseResult>> }>({
    key: '',
    pages: new Map(),
  })

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
    const key = JSON.stringify([game, source, text, profileID, filter, retry, hides])
    if (fetched.current.key !== key) {
      fetched.current = { key, pages: new Map() }
    }
    loadScreen(
      fetched.current.pages,
      (p) => search({ game, source, text, page: p, profileID, filter }),
      page,
      size,
    )
      .then(({ next, whole: all }) => {
        if (!cancelled) {
          setWhole(all)
          setResult(next)
          setStatus('done')
          setPage((current) => clampPage({ page: current, total: pagedTotal(next), size }))
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
  }, [game, source, text, page, size, profileID, search, retry, hides, filter, waiting])

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
    size,
    setRetry,
    result,
    status,
    error,
  }
}

export { type BrowseResult, pagedTotal, type Status, useBrowseQuery }
