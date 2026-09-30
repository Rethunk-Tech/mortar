import { useEffect } from 'react'
import { create } from 'zustand'
import type { Details } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/models.ts'
import {
  CachedDetails,
  Details as readDetails,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { useNexus } from '../settings/nexus.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'

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

const enqueue = (id: number) => {
  if (queued.has(id) || reads.has(id)) {
    return
  }
  queued.add(id)
  line = line.then(async () => {
    queued.delete(id)
    if (useNexus.getState().signedIn) {
      await loadDetails(id)
    }
  })
}

// Nexus page details by Nexus mod ID, shared by the sidebar, the list and the details dialog.
export const useNexusDetails = create<{ byId: Record<number, Entry | undefined> }>(() => ({
  byId: {},
}))

export const loadDetails = (id: number) => {
  let read = reads.get(id)
  if (!read) {
    read = readDetails(id).then(
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
export const primeDetails = async (ids: number[]) => {
  const { byId } = useNexusDetails.getState()
  const unknown = [...new Set(ids)].filter((id) => !byId[id])
  if (unknown.length === 0) {
    return
  }
  const cached = (await CachedDetails(unknown)) ?? {}
  useNexusDetails.setState((s) => {
    const next = { ...s.byId }
    for (const id of unknown) {
      const details = cached[`${id}`]
      if (details && !next[id]) {
        next[id] = { details }
      }
    }
    return { byId: next }
  })
  for (const id of unknown) {
    if (!cached[`${id}`]) {
      enqueue(id)
    }
  }
}

// The entry for one mod, read on first use; undefined while the first read is on its way.
export const useNexusEntry = (id: number) => {
  useEffect(() => {
    if (id) {
      loadDetails(id).catch(reportUnexpected)
    }
  }, [id])
  return useNexusDetails((s) => s.byId[id])
}
