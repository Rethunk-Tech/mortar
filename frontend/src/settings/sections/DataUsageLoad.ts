import {
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from 'react'
import { create } from 'zustand'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import {
  LastUsage,
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { peekLast, putLast, subscribeLast } from '../../shell/lastGood.ts'
import { type InlineError, inlineError, reportUnexpected } from '../../toasts/report.ts'
import { beginUsageLoad } from '../usageLoad.ts'

// Bytes counted so far while disk use is measured. Only the bar's Measuring line reads it, so a progress tick
// re-renders that line and not the whole Storage page.
export const useUsageBytes = create<{ bytes: number }>(() => ({ bytes: 0 }))
const setBytes = (bytes: number) => useUsageBytes.setState({ bytes })

// Whether a measurement is under way. Its own store for the same reason: only the lines that say so read it, so a
// start or an end redraws them and not the page's rows.
export const useUsageProgress = create<{ measuring: boolean }>(() => ({ measuring: false }))

// Settings search renders every page and hides the ones without a hit, so a mounted Storage section is not a shown one;
// disk use is measured once the sentinel is on screen (a hidden ancestor, display: none, keeps it off). Starting from
// the observer rather than a "shown" state spares the whole section a render that changes nothing it draws.
const USAGE_KEY = 'data-usage'

// The last measurement stays on screen while the next one runs: this session's, or the one the previous run saved.
export function useDataUsage(sentinel: RefObject<Element | null>) {
  const shown = useSyncExternalStore(subscribeLast, () => peekLast<DiskUse>(USAGE_KEY))
  const [error, setError] = useState<InlineError | null>(null)
  const stopRef = useRef<() => void>(() => undefined)
  const load = useCallback((fresh: boolean) => {
    stopRef.current()
    setBytes(0)
    setError(null)
    useUsageProgress.setState({ measuring: true })
    stopRef.current = beginUsageLoad({
      usage: () => Usage(fresh),
      progress: UsageProgress,
      setBytes,
      setUsage: (u) => {
        putLast(USAGE_KEY, u)
        useUsageProgress.setState({ measuring: false })
      },
      onError: (e: unknown) => {
        setError(inlineError(e))
        useUsageProgress.setState({ measuring: false })
        reportUnexpected(e)
      },
    })
  }, [])
  const restart = useCallback(() => load(true), [load])
  useEffect(() => {
    const el = sentinel.current
    if (!el) {
      return
    }
    const seen = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        seen.disconnect()
        if (!peekLast(USAGE_KEY)) {
          LastUsage()
            .then((last) => {
              if (last) {
                putLast(USAGE_KEY, last.usage, { seed: true, at: last.at })
              }
            })
            .catch(() => undefined)
        }
        load(false)
      }
    })
    seen.observe(el)
    return () => {
      seen.disconnect()
      stopRef.current()
    }
  }, [load, sentinel])
  return {
    usage: shown?.value ?? null,
    measuredAt: shown?.at ?? null,
    restart,
    error,
  }
}
