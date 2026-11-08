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

// Keys with a read in flight. Kept outside the hook so that only a view that shows a read's progress redraws for it:
// every other reader of a key would otherwise redraw twice per read for a flag it never looks at.
const inflight = new Set<string>()
const inflightListeners = new Set<() => void>()

const setInflight = (key: string, on: boolean) => {
  if (on === inflight.has(key)) {
    return
  }
  if (on) {
    inflight.add(key)
  } else {
    inflight.delete(key)
  }
  for (const listener of inflightListeners) {
    listener()
  }
}

const subscribeInflight = (listener: () => void): (() => void) => {
  inflightListeners.add(listener)
  return () => {
    inflightListeners.delete(listener)
  }
}

/** Whether a read of the key is on its way, for a view that waits on it before acting on what it shows. */
export function useLastGoodRefreshing(key: string | null): boolean {
  return useSyncExternalStore(subscribeInflight, () => key !== null && inflight.has(key))
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
  const [error, setError] = useState<InlineError | null>(null)
  const latest = useRef({ key, load, options })
  latest.current = { key, load, options }
  // The key this hook last marked as being read, cleared when the read ends or the hook unmounts.
  const marked = useRef<string | null>(null)
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
          if (marked.current !== null) {
            setInflight(marked.current, false)
            marked.current = null
          }
          const { key: now } = latest.current
          if (on && now !== null) {
            setInflight(now, true)
            marked.current = now
          }
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
  useEffect(
    () => () => {
      runs.stop()
      if (marked.current !== null) {
        setInflight(marked.current, false)
        marked.current = null
      }
    },
    [runs],
  )
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
    error,
    reload,
    set,
  }
}
