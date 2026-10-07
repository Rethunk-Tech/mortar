import type { HistoryDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

// Each profile edit changes the key, so the cache keeps only the newest few answers; an older one is never asked for again.
const CAP = 8
const entries = new Map<string, HistoryDiff | null>()

export const changesSinceCache = {
  has: (key: string) => entries.has(key),
  get: (key: string) => entries.get(key),
  set(key: string, diff: HistoryDiff | null) {
    entries.delete(key)
    entries.set(key, diff)
    for (const old of entries.keys()) {
      if (entries.size <= CAP) {
        break
      }
      entries.delete(old)
    }
  },
}
