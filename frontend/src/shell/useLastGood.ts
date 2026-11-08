import { type Load, loader } from '@rethunk/react-load-hooks'
import {
  type DependencyList,
  useCallback,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from 'react'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { peekLast, putLast, subscribeLast } from './lastGood.ts'

export interface LastGoodOptions<T> {
  /** A cheap read of a saved snapshot, shown while the first live read of a key runs. */
  seed?: () => Promise<T | null | undefined>
  /** Merges a fresh answer into the shown one so unchanged parts keep their identity. */
  keep?: (prev: T, next: T) => T
  onError?: (error: unknown) => void
}

/**
 * Loads `load` for `key` on mount and whenever `key` or `deps` change, drawing the key's last good answer at once
 * (module memory, then `seed`) and refreshing behind it. An answer equal to the shown one changes nothing; a null
 * key or load (a closed dialog) clears nothing. A failed refresh keeps the shown answer and sets `error`. Show a
 * skeleton only when `data === null && error === null`.
 */
export function useLastGood<T>(
  key: string | null,
  load: Load<T> | null,
  deps: DependencyList,
  options?: LastGoodOptions<T>,
) {
  const entry = useSyncExternalStore(subscribeLast, () =>
    key === null ? undefined : peekLast<T>(key),
  )
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  const latest = useRef({ key, load, options })
  latest.current = { key, load, options }
  const [runs] = useState(() =>
    loader<T>(
      {
        value: (value) => {
          const { key: now, options: opts } = latest.current
          if (now !== null) {
            putLast(now, value, opts?.keep ? { keep: opts.keep } : {})
          }
        },
        error: (e) => {
          setError(inlineError(e))
          latest.current.options?.onError?.(e)
        },
        loading: (on) => {
          // Nothing shown yet means refreshing reads false whatever this says, so starting a read redraws nothing.
          const { key: now } = latest.current
          const visible = on && now !== null && peekLast(now) !== undefined
          setRefreshing((prev) => (prev === visible ? prev : visible))
          if (on) {
            setError(null)
          }
        },
        reset: () => undefined,
      },
      { keepAcrossKeys: true },
    ),
  )
  const active = key !== null && load !== null
  // Runs after every render and starts a load only when key or deps changed: an effect cannot take a
  // caller-supplied dependency list.
  useEffect(() => {
    runs.deps([key, ...deps], active ? load : null)
  })
  useEffect(() => () => runs.stop(), [runs])
  const shown = entry !== undefined
  useEffect(() => {
    const { key: now, options: opts } = latest.current
    if (!active || shown || now === null || !opts?.seed) {
      return
    }
    let live = true
    opts.seed().then(
      (value) => {
        if (live && value !== null && value !== undefined) {
          putLast(now, value, { seed: true })
        }
      },
      () => undefined,
    )
    return () => {
      live = false
    }
  }, [active, shown])
  const reload = useCallback(() => runs.run(latest.current.load), [runs])
  const set = useCallback((value: T) => {
    if (latest.current.key !== null) {
      putLast(latest.current.key, value)
    }
  }, [])
  return {
    data: entry?.value ?? null,
    /** The shown answer is a saved snapshot the live read has not confirmed yet. */
    fromSeed: entry?.seed ?? false,
    at: entry?.at ?? null,
    refreshing: refreshing && shown,
    error,
    reload,
    set,
  }
}
