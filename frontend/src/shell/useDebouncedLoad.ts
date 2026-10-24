import { type DependencyList, useEffect, useRef, useState } from 'react'
import { loader } from './loader.ts'

// useLoaded for input that changes on every key: `load` runs `delayMs` after `deps` last changed, and `data` keeps the
// last answer meanwhile instead of going back to `initial`. Only the newest run lands: the answer or error of one that
// new deps, a later `run` or unmounting superseded is dropped, and `onError` hears only the newest failure. A null
// `load` resets `data` to `initial`. `run` starts another load at once, such as the next page, under the same rule.
export function useDebouncedLoad<T>(
  load: (() => Promise<T>) | null,
  deps: DependencyList,
  {
    initial,
    delayMs,
    onError,
  }: { initial: T; delayMs: number; onError?: (error: unknown) => void },
) {
  const [data, setData] = useState(initial)
  const [loading, setLoading] = useState(load !== null)
  const [error, setError] = useState<unknown>(null)
  const latest = useRef({ initial, onError })
  latest.current = { initial, onError }
  const [runs] = useState(() =>
    loader<T>(
      {
        value: (value) => {
          setData(() => value)
          setError(null)
        },
        error: (e) => {
          setError(e)
          latest.current.onError?.(e)
        },
        loading: setLoading,
        reset: () => setData(() => latest.current.initial),
      },
      {
        keepAcrossKeys: true,
        defer: (go) => {
          const timer = globalThis.setTimeout(go, delayMs)
          return () => globalThis.clearTimeout(timer)
        },
      },
    ),
  )
  // Runs after every render and acts only when the caller's deps changed: an effect cannot take a caller-supplied
  // dependency list.
  useEffect(() => {
    runs.deps(deps, load)
  })
  useEffect(() => () => runs.stop(), [runs])
  return {
    data,
    loading,
    error,
    run: (next: () => Promise<T>) => runs.run(next),
  }
}
