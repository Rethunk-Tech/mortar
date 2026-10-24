import { type DependencyList, useCallback, useEffect, useRef, useState } from 'react'
import { loader } from './loader.ts'

// Runs `load` whenever `deps` change (never while `load` is null) and drops the answer of any run that has been
// superseded or unmounted. New deps and a skipped load reset `data` to `initial`; a failed load keeps the last good
// `data` and sets `error`. `reload` runs it again with the same deps and keeps the old value until the answer
// arrives; `setData` edits the loaded value in place.
export function useLoaded<T>(
  load: (() => Promise<T>) | null,
  deps: DependencyList,
  initial: T,
  onError?: (error: unknown) => void,
) {
  const [data, setData] = useState(initial)
  const [loading, setLoading] = useState(load !== null)
  const [error, setError] = useState<unknown>(null)
  const latest = useRef({ load, initial, onError })
  latest.current = { load, initial, onError }
  const [runs] = useState(() =>
    loader<T>({
      value: (value) => setData(() => value),
      error: (e) => {
        setError(e)
        latest.current.onError?.(e)
      },
      loading: (on) => {
        setLoading(on)
        if (on) {
          setError(null)
        }
      },
      reset: () => setData(() => latest.current.initial),
    }),
  )
  // Runs after every render and starts a load only when the caller's deps changed: an effect cannot take a
  // caller-supplied dependency list.
  useEffect(() => {
    runs.deps(deps, load)
  })
  useEffect(() => () => runs.stop(), [runs])
  const reload = useCallback(() => runs.run(latest.current.load), [runs])
  return { data, setData, loading, error, reload }
}
