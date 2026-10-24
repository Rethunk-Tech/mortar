import { useEffect, useState } from 'react'
import type { RunSearch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { reportUnexpected } from '../toasts/report.ts'

const SEARCH_DEBOUNCE_MS = 250
const NO_HITS: RunSearch = { hits: [], truncated: false }

export function useRunSearch(
  open: boolean,
  query: string,
  search: (text: string) => Promise<RunSearch | null>,
) {
  const [result, setResult] = useState<RunSearch>(NO_HITS)
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    const trimmed = query.trim()
    if (!(open && trimmed)) {
      setResult(NO_HITS)
      setLoading(false)
      return
    }
    // An answer to a query since replaced must not overwrite the newer one's hits or end its spinner.
    let current = true
    setLoading(true)
    const timer = globalThis.setTimeout(() => {
      search(trimmed)
        .then((next) => current && setResult(next ?? NO_HITS))
        .catch((e: unknown) => current && reportUnexpected(e))
        .finally(() => current && setLoading(false))
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      current = false
      globalThis.clearTimeout(timer)
    }
  }, [open, query, search])
  return { result, loading }
}
