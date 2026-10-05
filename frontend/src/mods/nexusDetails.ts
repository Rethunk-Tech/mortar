import { useEffect, useState } from 'react'
import { create } from 'zustand'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import {
  CachedDetails,
  MarkSeen,
  Details as readDetails,
  Seen,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { currentGame } from '../nav/currentGame.ts'
import { useNexus } from '../settings/nexus.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { currentFiles, isNewer } from './nexusFormat.ts'

interface Entry {
  details?: Details
  error?: string
}

const put = (id: number, entry: Entry) =>
  useNexusDetails.setState((s) => ({ byId: { ...s.byId, [id]: entry } }))

// One read per mod per session: the backend refreshes a day-old cache on it, so a second read adds nothing. A failed
// read is forgotten so that signing in retries it, and keeps whatever the cache already gave.
const reads = new Map<number, Promise<void>>()

// One queue for every missing mod, so a large profile never bursts Nexus; the Nexus client also refuses locally once
// the rate-limit budget reaches its floor.
let line = Promise.resolve()
const queued = new Set<number>()
const waiting = new Set<number>()

const enqueue = (id: number) => {
  if (queued.has(id) || reads.has(id)) {
    return
  }
  if (!useNexus.getState().signedIn) {
    waiting.add(id)
    return
  }
  waiting.delete(id)
  queued.add(id)
  line = line.then(async () => {
    queued.delete(id)
    if (useNexus.getState().signedIn) {
      await loadDetails(id)
    } else {
      waiting.add(id)
    }
  })
}

const useNexusDetails = create<{ byId: Record<number, Entry | undefined> }>(() => ({
  byId: {},
}))

useNexus.subscribe((s, prev) => {
  if (s.signedIn && !prev.signedIn) {
    const ids = [...waiting]
    waiting.clear()
    for (const id of ids) {
      enqueue(id)
    }
    for (const [key, entry] of Object.entries(useNexusDetails.getState().byId)) {
      if (!entry?.details) {
        reads.delete(Number(key))
        enqueue(Number(key))
      }
    }
  }
})

const loadDetails = (id: number) => {
  let read = reads.get(id)
  if (!read) {
    read = readDetails(currentGame(), id).then(
      (details) => put(id, { details }),
      (e: unknown) => {
        reads.delete(id)
        put(id, { ...useNexusDetails.getState().byId[id], error: errorMessage(e) })
      },
    )
    reads.set(id, read)
  }
  return read
}

// Shows what is cached for every mod at once, then reads the missing ones one at a time.
const primeDetails = async (ids: number[]) => {
  await mergeCachedDetails(ids)
  for (const id of new Set(ids)) {
    if (id > 0 && !useNexusDetails.getState().byId[id]?.details) {
      enqueue(id)
    }
  }
}

// The entry for one mod, read on first use; undefined while the first read is on its way.
const useNexusEntry = (id: number) => {
  useEffect(() => {
    if (id) {
      enqueue(id)
    }
  }, [id])
  return useNexusDetails((s) => s.byId[id])
}

interface SeenWatermark {
  newestFileUnix: number
  newestChange: string
}

const MS_PER_SEC = 1000

const fileUnix = (iso: string) => {
  const t = Date.parse(iso)
  return Number.isNaN(t) ? 0 : Math.floor(t / MS_PER_SEC)
}

const watermarkOf = (details: Pick<Details, 'files' | 'changelogs'>): SeenWatermark => {
  let newestFileUnix = 0
  for (const f of currentFiles(details.files ?? [], 0)) {
    newestFileUnix = Math.max(newestFileUnix, fileUnix(f.uploaded))
  }
  if (newestFileUnix === 0) {
    for (const f of details.files ?? []) {
      newestFileUnix = Math.max(newestFileUnix, fileUnix(f.uploaded))
    }
  }
  let newestChange = ''
  for (const c of details.changelogs ?? []) {
    if (!newestChange || isNewer(c.version, newestChange)) {
      newestChange = c.version
    }
  }
  return { newestFileUnix, newestChange }
}

const isNewSinceLooked = (seen: SeenWatermark | undefined, now: SeenWatermark) => {
  if (!seen) {
    return false
  }
  if (now.newestFileUnix > seen.newestFileUnix) {
    return true
  }
  if (!now.newestChange || now.newestChange === seen.newestChange) {
    return false
  }
  if (!seen.newestChange) {
    return true
  }
  return isNewer(now.newestChange, seen.newestChange)
}

const fileIsNewSinceLooked = (uploaded: string, seen: SeenWatermark | undefined) =>
  Boolean(seen && fileUnix(uploaded) > seen.newestFileUnix)

const changelogIsNewSinceLooked = (version: string, seen: SeenWatermark | undefined) => {
  if (!(seen && version)) {
    return false
  }
  if (!seen.newestChange) {
    return true
  }
  return isNewer(version, seen.newestChange)
}

const useNexusSeen = create<{ byId: Record<number, SeenWatermark | undefined> }>(() => ({
  byId: {},
}))

const initNexusSeen = () =>
  Seen()
    .then((snap) => {
      const byId: Record<number, SeenWatermark | undefined> = {}
      for (const [key, entry] of Object.entries(snap ?? {})) {
        if (entry) {
          byId[Number(key)] = {
            newestFileUnix: entry.newestFileUnix,
            newestChange: entry.newestChange,
          }
        }
      }
      useNexusSeen.setState({ byId })
    })
    .catch(reportUnexpected)

const markLooked = (modId: number, details: Details) => {
  const w = watermarkOf(details)
  useNexusSeen.setState((s) => ({ byId: { ...s.byId, [modId]: w } }))
  MarkSeen(modId, w.newestFileUnix, w.newestChange).catch(reportUnexpected)
}

const useLookedSnapshot = (modId: number, details: Details | undefined) => {
  const [looked] = useState(() => useNexusSeen.getState().byId[modId])
  useEffect(() => {
    if (details && modId) {
      markLooked(modId, details)
    }
  }, [modId, details])
  return looked
}

const useNexusFresh = (nexusId: number) => {
  const details = useNexusDetails((s) => s.byId[nexusId]?.details)
  const seen = useNexusSeen((s) => s.byId[nexusId])
  if (!(nexusId && details)) {
    return false
  }
  return isNewSinceLooked(seen, watermarkOf(details))
}

export {
  changelogIsNewSinceLooked,
  fileIsNewSinceLooked,
  initNexusSeen,
  isNewSinceLooked,
  loadDetails,
  primeDetails,
  useLookedSnapshot,
  useNexusDetails,
  useNexusEntry,
  useNexusFresh,
  useNexusSeen,
  watermarkOf,
}

// Fills the details store from the on-disk Nexus cache only, so Update review never hits the network.
export async function mergeCachedDetails(ids: number[]): Promise<void> {
  const want = [...new Set(ids)].filter((id) => id > 0)
  const { byId } = useNexusDetails.getState()
  const unknown = want.filter((id) => !byId[id]?.details)
  if (unknown.length === 0) {
    return
  }
  const cached = (await CachedDetails(currentGame(), unknown)) ?? {}
  useNexusDetails.setState((s) => {
    const next = { ...s.byId }
    for (const id of unknown) {
      const details = cached[`${id}`]
      if (details && !next[id]?.details) {
        next[id] = { details }
      }
    }
    return { byId: next }
  })
}
