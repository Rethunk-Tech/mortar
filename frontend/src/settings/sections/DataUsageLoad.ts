import { type RefObject, useCallback, useEffect, useRef, useState } from 'react'
import { create } from 'zustand'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import {
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { beginUsageLoad } from '../usageLoad.ts'

// Bytes counted so far while disk use is measured. Only the bar's Measuring line reads it, so a progress tick
// re-renders that line and not the whole Storage page.
export const useUsageBytes = create<{ bytes: number }>(() => ({ bytes: 0 }))
const setBytes = (bytes: number) => useUsageBytes.setState({ bytes })

// Settings search renders every page and hides the ones without a hit, so a mounted Storage section is not a shown one;
// disk use is measured once the sentinel is on screen (a hidden ancestor, display: none, keeps it off). Starting from
// the observer rather than a "shown" state spares the whole section a render that changes nothing it draws.
export function useDataUsage(sentinel: RefObject<Element | null>) {
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [error, setError] = useState(false)
  const stopRef = useRef<() => void>(() => undefined)
  const load = useCallback((fresh: boolean) => {
    stopRef.current()
    setUsage(null)
    setBytes(0)
    setError(false)
    stopRef.current = beginUsageLoad({
      usage: () => Usage(fresh),
      progress: UsageProgress,
      setBytes,
      setUsage,
      onError: (e: unknown) => {
        setError(true)
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
        load(false)
      }
    })
    seen.observe(el)
    return () => {
      seen.disconnect()
      stopRef.current()
    }
  }, [load, sentinel])
  return { usage, restart, error }
}
