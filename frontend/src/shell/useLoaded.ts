import { type DependencyList, useEffect, useRef, useState } from 'react'

const sameDeps = (a: DependencyList | null, b: DependencyList) =>
  a !== null && a.length === b.length && b.every((dep, i) => Object.is(dep, a[i]))

// Runs `load` whenever `deps` change (never while `load` is null) and drops the answer of any run that has been
// superseded or unmounted. New deps, a skipped load and a failed load leave `initial` in `data`. `reload` runs it again with the same deps
// and keeps the old value until the answer arrives; `setData` edits the loaded value in place.
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
  const start = useRef<(fresh: boolean) => void>(() => undefined)
  start.current = (fresh) => {
    generation.current += 1
    if (fresh || load === null) {
      setData(initial)
    }
    if (load === null) {
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
      start.current(seen.current !== null)
      seen.current = deps
    }
  })
  useEffect(
    () => () => {
      generation.current += 1
      seen.current = null
    },
    [],
  )
  return { data, setData, loading, error, reload: () => start.current(false) }
}
