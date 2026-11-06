import { sameValue } from '@rethunk/react-load-hooks'

/** What a view last drew for a key: the answer, when it was read, and whether it is only a saved snapshot that the
 * live read has not yet confirmed. */
export interface LastGood<T> {
  readonly value: T
  readonly at: number
  readonly seed: boolean
}

const entries = new Map<string, LastGood<unknown>>()
const listeners = new Set<() => void>()

const notify = () => {
  for (const listener of listeners) {
    listener()
  }
}

export function subscribeLast(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function peekLast<T>(key: string): LastGood<T> | undefined {
  return entries.get(key) as LastGood<T> | undefined
}

export function getLast<T>(key: string): T | undefined {
  return peekLast<T>(key)?.value
}

/**
 * Stores `next` as the key's last good answer and returns the stored value: the previous object when the answer is
 * equal to it, so a re-read that changed nothing re-renders nothing. `keep` merges an answer into the previous one
 * (for example keepUnchanged on a list). `at` is when the answer was read, now unless the answer is a saved one. A `seed`
 * answer never replaces a live one.
 */
export function putLast<T>(
  key: string,
  next: T,
  {
    seed = false,
    keep,
    at = Date.now(),
  }: { seed?: boolean; keep?: (prev: T, next: T) => T; at?: number } = {},
): T {
  const prev = peekLast<T>(key)
  if (prev && seed) {
    return prev.value
  }
  const merged = prev && keep ? keep(prev.value, next) : next
  const value = prev && sameValue(prev.value, merged) ? prev.value : merged
  if (prev && prev.value === value && prev.seed === seed) {
    return value
  }
  entries.set(key, { value, at, seed })
  notify()
  return value
}
