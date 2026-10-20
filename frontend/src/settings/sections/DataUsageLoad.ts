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
// enabled says the section is on screen, and disk use is measured from then on.
export function useDataUsage(enabled: boolean) {
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
    if (!enabled) {
      return
    }
    load(false)
    return () => stopRef.current()
  }, [load, enabled])
  return { usage, restart, error }
}

/** Whether the element has been on screen at least once; a hidden ancestor (display: none) keeps it false. */
export function useShownOnce(ref: RefObject<Element | null>): boolean {
  const [shown, setShown] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (shown || !el) {
      return
    }
    const seen = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        setShown(true)
      }
    })
    seen.observe(el)
    return () => seen.disconnect()
  }, [ref, shown])
  return shown
}
