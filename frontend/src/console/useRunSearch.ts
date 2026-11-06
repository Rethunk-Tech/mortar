import { useDebouncedLoad } from '@rethunk/react-load-hooks'
import type { RunSearch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { peekLast, putLast } from '../shell/lastGood.ts'
import { reportUnexpected } from '../toasts/report.ts'

const SEARCH_DEBOUNCE_MS = 250
const NO_HITS: RunSearch = { hits: [], truncated: false }

/**
 * Searches the stored runs of `scope` (a game and profile). The hits of the previous query stay on screen while the
 * next one runs, and a query answered before draws at once; `loading` is true only when there is nothing to show.
 */
export function useRunSearch(
  open: boolean,
  scope: string,
  query: string,
  search: (text: string) => Promise<RunSearch | null>,
) {
  const trimmed = query.trim()
  const key = `run-search:${scope}:${trimmed}`
  const { data, loading: running } = useDebouncedLoad(
    open && trimmed
      ? () =>
          search(trimmed).then((next) => {
            // Stored before it lands, so the cache already holds it when the hook re-renders.
            putLast(key, next ?? NO_HITS)
            return next ?? NO_HITS
          })
      : null,
    [open, trimmed, search, key],
    { initial: NO_HITS, delayMs: SEARCH_DEBOUNCE_MS, onError: reportUnexpected },
  )
  const cached = trimmed ? peekLast<RunSearch>(key)?.value : undefined
  const result = cached ?? data
  const loading = running && cached === undefined && (result.hits ?? []).length === 0
  return { result, loading, refreshing: running && !loading }
}
