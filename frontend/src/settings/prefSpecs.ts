import { useEffect, useState } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { PrefSpecs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'

let cache: PrefSpec[] = []

export function loadPrefSpecs(): Promise<PrefSpec[]> {
  if (cache.length > 0) {
    return Promise.resolve(cache)
  }
  return PrefSpecs()
    .then((next) => {
      cache = next ?? []
      return cache
    })
    .catch(() => {
      cache = []
      return cache
    })
}

export function usePrefSpecs(): PrefSpec[] {
  const [specs, setSpecs] = useState(cache)
  useEffect(() => {
    if (cache.length > 0) {
      return
    }
    loadPrefSpecs().then(setSpecs, () => undefined)
  }, [])
  return specs
}

// A game-scoped key's default for one game; the spec's own default when the game has none.
export function specDefault(spec: PrefSpec, game: string): string {
  return spec.gameDefaults?.[game] ?? spec.default
}

export function cachedSpecDefault(key: string, game: string): string | undefined {
  const spec = specByKey(cache, key)
  return spec ? specDefault(spec, game) : undefined
}

export function specByKey(specs: PrefSpec[], key: string): PrefSpec | undefined {
  return specs.find((spec) => spec.key === key)
}
