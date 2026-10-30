import { useDebouncedLoad } from '@rethunk/react-load-hooks'
import type { RunSearch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { reportUnexpected } from '../toasts/report.ts'

const SEARCH_DEBOUNCE_MS = 250
const NO_HITS: RunSearch = { hits: [], truncated: false }

export function useRunSearch(
  open: boolean,
  query: string,
  search: (text: string) => Promise<RunSearch | null>,
) {
  const trimmed = query.trim()
  const { data: result, loading } = useDebouncedLoad(
    open && trimmed ? () => search(trimmed).then((next) => next ?? NO_HITS) : null,
    [open, trimmed, search],
    { initial: NO_HITS, delayMs: SEARCH_DEBOUNCE_MS, onError: reportUnexpected },
  )
  return { result, loading }
}
