import { useEffect, useState } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { PrefSpecs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'

let cache: PrefSpec[] = []

export function usePrefSpecs(): PrefSpec[] {
  const [specs, setSpecs] = useState(cache)
  useEffect(() => {
    if (cache.length > 0) {
      return
    }
    PrefSpecs()
      .then((next) => {
        cache = next ?? []
        setSpecs(cache)
      })
      .catch(() => {
        cache = []
        setSpecs(cache)
      })
  }, [])
  return specs
}

export function specByKey(specs: PrefSpec[], key: string): PrefSpec | undefined {
  return specs.find((spec) => spec.key === key)
}
