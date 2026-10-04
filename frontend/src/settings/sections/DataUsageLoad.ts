import { useCallback, useEffect, useRef, useState } from 'react'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import {
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { beginUsageLoad } from '../usageLoad.ts'

export function useDataUsage() {
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [bytes, setBytes] = useState(0)
  const [error, setError] = useState(false)
  const stopRef = useRef<() => void>(() => undefined)
  const load = useCallback((fresh: boolean) => {
    stopRef.current()
    setUsage(null)
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
    load(false)
    return () => stopRef.current()
  }, [load])
  return { usage, bytes, restart, error }
}
