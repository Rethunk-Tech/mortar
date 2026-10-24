import { type DependencyList, useEffect, useRef, useState } from 'react'
import { sameDeps } from './useLoaded.ts'

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
  const generation = useRef(0)
  const timer = useRef<ReturnType<typeof globalThis.setTimeout> | undefined>(undefined)
  const seen = useRef<DependencyList | null>(null)
  const start = useRef<(next: () => Promise<T>) => void>(() => undefined)
  start.current = (next) => {
    generation.current += 1
    const mine = generation.current
    const current = () => generation.current === mine
    setLoading(true)
    next()
      .then((value) => {
        if (current()) {
          setData(value)
          setError(null)
        }
      })
      .catch((e: unknown) => {
        if (current()) {
          setError(e)
          onError?.(e)
        }
      })
      .finally(() => {
        if (current()) {
          setLoading(false)
        }
      })
  }
  // Runs after every render and acts only when the caller's deps changed: an effect cannot take a caller-supplied
  // dependency list.
  useEffect(() => {
    if (sameDeps(seen.current, deps)) {
      return
    }
    seen.current = deps
    globalThis.clearTimeout(timer.current)
    generation.current += 1
    if (load === null) {
      setData(initial)
      setLoading(false)
      return
    }
    setLoading(true)
    timer.current = globalThis.setTimeout(() => start.current(load), delayMs)
  })
  useEffect(
    () => () => {
      generation.current += 1
      globalThis.clearTimeout(timer.current)
      seen.current = null
    },
    [],
  )
  return { data, loading, error, run: (next: () => Promise<T>) => start.current(next) }
}
