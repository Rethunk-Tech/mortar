import { keepUnchanged, sameValue } from '@rethunk/react-load-hooks'
import { Events } from '@wailsio/runtime'
import {
  type Dispatch,
  type SetStateAction,
  useEffect,
  useEffectEvent,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useSettings } from '../settings/store.ts'
import { useLastGood } from '../shell/useLastGood.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { ALL, FIRST_PAGE } from './browseConstants.ts'
import { parseModes } from './browseModes.ts'
import { parsePicks } from './browsePicks.ts'
import { effectiveSort, sortsFor } from './browseSort.ts'
import { clampPage, PAGE_SIZE, pageKeeping, screenSpan } from './browseState.ts'
import type { BrowseFilter, BrowseItem, BrowsePageProps } from './browseTypes.ts'
import { onBrowseRefresh } from './refresh.ts'
import { hitKey, useBrowseSelection } from './selection.ts'
import { useBrowseView } from './view.ts'

const NO_EXCLUDE: string[] = []

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

// How many source pages a search has: the largest source's count across sources, else its hits in pages. A single
// source sends no count, which the window reads as 0, so 0 means unknown too.
function sourcePages(head: BrowseResult): number {
  return head.pages || Math.ceil(head.total / PAGE_SIZE)
}

// A site whose list shifts between two page requests can send one hit on both; a screen shows it once, since two cards
// with one key leave a stale card behind when the page changes.
function once(items: BrowseItem[]): BrowseItem[] {
  const seen = new Set<string>()
  return items.filter((i) => {
    const key = hitKey(i)
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

// One screen from the source pages it spans: their hits in order, the hidden counts added and the failed sources once.
function joinPages(parts: BrowseResult[], skip: number, size: number): BrowseResult {
  const [head = EMPTY_RESULT] = parts
  return {
    ...head,
    items: once(parts.flatMap((p) => p.items).slice(skip, skip + size)),
    hidden: parts.reduce((n, p) => n + (p.hidden ?? 0), 0),
    failed: [...new Set(parts.flatMap((p) => p.failed ?? []))],
  }
}

interface Fetched {
  key: string
  pages: Map<number, Promise<BrowseResult>>
  // The source pages that have answered, so a screen made only of those shows without a loading pass.
  done: Map<number, BrowseResult>
}

// The source pages each recent query read, newest last, so coming back to a query draws its last screen at once.
const REMEMBERED_QUERIES = 12
const remembered = new Map<string, Map<number, BrowseResult>>()

function remember(query: string, done: Map<number, BrowseResult>) {
  remembered.delete(query)
  remembered.set(query, done)
  const [oldest] = remembered.keys()
  if (remembered.size > REMEMBERED_QUERIES && oldest !== undefined) {
    remembered.delete(oldest)
  }
}

interface Screen {
  next: BrowseResult
  whole: boolean
}

/** next, keeping prev's object when nothing changed and each card's object when that card did not, so a refresh redraws
 * only the cards whose numbers moved. */
function keptResult(prev: BrowseResult, next: BrowseResult): BrowseResult {
  if (sameValue(prev, next)) {
    return prev
  }
  return { ...next, items: keepUnchanged(prev.items, next.items, hitKey) }
}

// The screen from source pages that have already answered, or null while one of them is still to fetch.
function cachedScreen(done: Map<number, BrowseResult>, page: number, size: number): Screen | null {
  const { first, last, skip } = screenSpan(page, size)
  const head = done.get(first)
  if (!head) {
    return null
  }
  if (head.items.length > PAGE_SIZE) {
    return { next: head, whole: true }
  }
  const known = sourcePages(head)
  const parts = [head]
  for (let p = first + 1; p <= Math.min(last, known); p += 1) {
    const part = done.get(p)
    if (!part) {
      return null
    }
    parts.push(part)
  }
  return { next: joinPages(parts, skip, size), whole: false }
}

// One screen of `size` hits on screen `page`, from the source pages it spans; pages already fetched for this search
// are reused. A source page holding more than a page of hits comes back whole.
async function loadScreen(
  { pages, done }: Fetched,
  fetchPage: (page: number) => Promise<BrowseResult>,
  page: number,
  size: number,
): Promise<Screen> {
  const sourcePage = (p: number) => {
    let got = pages.get(p)
    if (!got) {
      got = fetchPage(p)
      pages.set(p, got)
      got.then(
        (answer) => done.set(p, answer),
        () => pages.delete(p),
      )
    }
    return got
  }
  const { first, last, skip } = screenSpan(page, size)
  const head = await sourcePage(first)
  if (head.items.length > PAGE_SIZE) {
    return { next: head, whole: true }
  }
  const known = sourcePages(head)
  const rest = await Promise.all(
    Array.from({ length: Math.max(0, Math.min(last, known) - first) }, (_, i) =>
      sourcePage(first + 1 + i),
    ),
  )
  return { next: joinPages([head, ...rest], skip, size), whole: false }
}

// When the screen size changes, the page moves to keep the first hit in view, or the hit whose details are open when it is
// on the screen: opening them narrows the grid, and the mod clicked must not leave.
function useKeepScreen(
  size: number,
  items: BrowseItem[],
  setPage: Dispatch<SetStateAction<number>>,
) {
  const shown = useRef(size)
  const anchorIndex = useEffectEvent((open: BrowseItem) =>
    items.findIndex((i) => hitKey(i) === hitKey(open)),
  )
  useEffect(() => {
    const from = shown.current
    if (from !== size) {
      const open = useBrowseSelection.getState().selected?.item
      const anchor = open ? anchorIndex(open) : -1
      setPage((current) => pageKeeping(current, from, size, Math.max(anchor, 0)))
      shown.current = size
    }
  }, [size, setPage])
}

// A source's data refreshed behind the answers on screen, or the player asked for a refresh: search again for the
// current query.
function useSearchAgainOnRefresh(setRetry: Dispatch<SetStateAction<number>>) {
  useEffect(() => {
    const again = () => setRetry((n) => n + 1)
    const off = [Events.On('browse:refreshed', again), onBrowseRefresh(again)]
    return () => {
      for (const stop of off) {
        stop()
      }
    }
  }, [setRetry])
}

// The filter the player picked, or until they do the source's default excludes (the last ones read for it draw at
// once); the first search waits for the very first read of them.
function useStoredFilter(
  game: string,
  source: string,
  chosenFilter: BrowseFilter | undefined,
  defaultExcludes: BrowsePageProps['defaultExcludes'],
) {
  const defaults = useLastGood<string[]>(
    chosenFilter || source === '' ? null : `browse-excludes:${game}:${source}`,
    () => defaultExcludes(game, source),
    [defaultExcludes, chosenFilter !== undefined],
  )
  const stored = useMemo(
    () => chosenFilter ?? { ...NO_FILTER, exclude: defaults.data ?? NO_EXCLUDE },
    [chosenFilter, defaults.data],
  )
  const waiting =
    chosenFilter === undefined && source !== '' && defaults.data === null && defaults.error === null
  return { stored, waiting }
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
  const { stored, waiting } = useStoredFilter(game, source, chosenFilter, defaultExcludes)
  const sorts = sortsFor(source, sources)
  const filter = useMemo(
    () => ({ ...stored, ...modes, sort: effectiveSort(stored.sort, sorts) }),
    [stored, modes, sorts],
  )
  const [text, setText] = useState('')
  const [page, setPage] = useState(FIRST_PAGE)
  const [retry, setRetry] = useState(0)
  const hides = useBrowseView((s) => s.hides)
  useSearchAgainOnRefresh(setRetry)
  const [result, setResult] = useState<BrowseResult>(EMPTY_RESULT)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<InlineError | null>(null)
  // A search across sources with a sort none of them ranks returns every source's page at once; those pages are shown
  // as they come, since screens cut from them would skip hits.
  const [whole, setWhole] = useState(false)
  const size = whole ? PAGE_SIZE : fit
  useKeepScreen(size, result.items, setPage)
  const fetched = useRef<Fetched>({ key: '', pages: new Map(), done: new Map() })

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
    if (waiting) {
      setStatus('loading')
      return
    }
    if (source === '') {
      setResult(EMPTY_RESULT)
      setStatus('idle')
      return
    }
    let cancelled = false
    const query = JSON.stringify([game, source, text, profileID, filter, hides])
    const key = JSON.stringify([query, retry])
    // The screens an earlier visit to this query read: drawn at once, replaced when the new read answers.
    let earlier: Screen | null = null
    if (fetched.current.key !== key) {
      earlier = retry === 0 ? cachedScreen(remembered.get(query) ?? new Map(), page, size) : null
      fetched.current = { key, pages: new Map(), done: new Map() }
      remember(query, fetched.current.done)
    }
    const show = ({ next, whole: all }: Screen) => {
      setWhole(all)
      setResult((prev) => keptResult(prev, next))
      setStatus('done')
      setPage((current) => clampPage({ page: current, total: pagedTotal(next), size }))
    }
    const ready = cachedScreen(fetched.current.done, page, size)
    if (ready) {
      show(ready)
      return
    }
    if (earlier) {
      show(earlier)
    } else {
      setStatus('loading')
    }
    loadScreen(
      fetched.current,
      (p) => search({ game, source, text, page: p, profileID, filter }),
      page,
      size,
    )
      .then((screen) => {
        if (!cancelled) {
          show(screen)
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

export { type BrowseResult, cachedScreen, pagedTotal, type Status, useBrowseQuery }
