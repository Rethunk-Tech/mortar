import { type DependencyList, useEffect, useRef, useState } from 'react'

const sameDeps = (a: DependencyList | null, b: DependencyList) =>
  a !== null && a.length === b.length && b.every((dep, i) => Object.is(dep, a[i]))

// Runs `load` whenever `deps` change (never while `load` is null) and drops the answer of any run that has been
// superseded or unmounted. A skipped or failed load leaves `initial` in `data`. `reload` runs it again with the same deps; `setData` edits the loaded value in place.
export function useLoaded<T>(
  load: (() => Promise<T>) | null,
  deps: DependencyList,
  initial: T,
  onError?: (error: unknown) => void,
) {
  const [data, setData] = useState(initial)
  const [loading, setLoading] = useState(load !== null)
  const [error, setError] = useState<unknown>(null)
  const generation = useRef(0)
  const seen = useRef<DependencyList | null>(null)
  const start = useRef<() => void>(() => undefined)
  start.current = () => {
    generation.current += 1
    if (load === null) {
      setData(initial)
      setLoading(false)
      return
    }
    const mine = generation.current
    const current = () => generation.current === mine
    setLoading(true)
    setError(null)
    load()
      .then((value) => {
        if (current()) {
          setData(value)
        }
      })
      .catch((e: unknown) => {
        if (current()) {
          setData(initial)
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
  // Runs after every render and starts a load only when the caller's deps changed: an effect cannot take a
  // caller-supplied dependency list.
  useEffect(() => {
    if (!sameDeps(seen.current, deps)) {
      seen.current = deps
      start.current()
    }
  })
  useEffect(
    () => () => {
      generation.current += 1
      seen.current = null
    },
    [],
  )
  return { data, setData, loading, error, reload: () => start.current() }
}
