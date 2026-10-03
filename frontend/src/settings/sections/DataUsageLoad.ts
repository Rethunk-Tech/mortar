import { useCallback, useEffect, useRef, useState } from 'react'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { beginUsageLoad } from '../usageLoad.ts'

export function useDataUsage() {
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [bytes, setBytes] = useState(0)
  const stopRef = useRef<() => void>(() => undefined)
  const restart = useCallback(() => {
    stopRef.current()
    setUsage(null)
    stopRef.current = beginUsageLoad({
      usage: Usage,
      progress: UsageProgress,
      setBytes,
      setUsage,
      onError: reportUnexpected,
    })
  }, [])
  useEffect(() => {
    restart()
    return () => stopRef.current()
  }, [restart])
  return { usage, bytes, restart }
}
