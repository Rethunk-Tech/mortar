import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import { create } from 'zustand'
import { States } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import {
  CachedDetails,
  MarkSeen,
  PrimeDetails,
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
    primeDetails(primed).catch(reportUnexpected)
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

let primed: number[] = []
const primeAsked = new Set<number>()
const primeInFlight = new Set<number>()

// Stores the details that are new; when there are none the state object stays as it was, so subscribers do not wake.
const addDetails = (ids: number[], found: Record<string, Details | undefined>) =>
  useNexusDetails.setState((s) => {
    let next: Record<number, Entry | undefined> | undefined
    for (const id of ids) {
      const details = found[`${id}`]
      if (details && !s.byId[id]?.details) {
        next ??= { ...s.byId }
        next[id] = { details }
      }
    }
    return next ? { byId: next } : s
  })

// Shows what is cached for every mod at once, then fills the rest with the page data of all of them in a few batched
// requests. A mod's files and changelogs are read when it is opened, not for the whole list.
const primeDetails = async (ids: number[]) => {
  primed = ids
  await mergeCachedDetails(ids)
  if (!useNexus.getState().signedIn) {
    return
  }
  // A mod Nexus does not return (hidden, removed, a wrong id) stays unknown, so it is asked for once per session.
  const unknown = ids.filter(
    (id) =>
      id > 0 &&
      !useNexusDetails.getState().byId[id]?.details &&
      !primeAsked.has(id) &&
      !primeInFlight.has(id),
  )
  if (unknown.length === 0) {
    return
  }
  for (const id of unknown) {
    primeInFlight.add(id)
  }
  watchNexusReachable()
  // Only an answer settles an id: one the request could not fetch (offline, rate limited) is asked again, here or
  // when Nexus is reachable again. What did arrive is kept either way.
  const got = await PrimeDetails(currentGame(), unknown).catch(() => null)
  const found = got?.details ?? {}
  for (const id of unknown) {
    primeInFlight.delete(id)
    if (got && (!got.error || found[`${id}`])) {
      primeAsked.add(id)
    }
  }
  addDetails(unknown, found)
}

let watching = false

// The mods a failed prime left out are asked for again the moment Nexus answers again.
function watchNexusReachable() {
  if (!watching) {
    watching = true
    Events.On('netstate:changed', () => {
      nexusChanged().catch(reportUnexpected)
    })
  }
}

async function nexusChanged() {
  const states = await States()
  if (states?.some((st) => st.id === 'nexus' && !st.unreachable)) {
    await primeDetails(primed)
  }
}

// Whether the entry holds a mod's whole details, not just the batched page data.
const isFull = (entry: Entry | undefined) => Boolean(entry?.details && !entry.details.partial)

// The entry for one mod, read on first use; undefined while the first read is on its way. Page data from the batch is
// not enough for a caller that asks for one mod, so it reads as missing until the whole details arrive.
const useNexusEntry = (id: number) => {
  useEffect(() => {
    if (id) {
      enqueue(id)
    }
  }, [id])
  const entry = useNexusDetails((s) => s.byId[id])
  return isFull(entry) ? entry : { ...entry, details: undefined }
}

// A mod's page data (status, version, endorsements) for a list row: whatever the batch or a full read gave, and no
// request of its own.
const useNexusPage = (id: number) => useNexusDetails((s) => s.byId[id]?.details?.page)

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
    if (details && !details.partial && modId) {
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
  isFull,
  isNewSinceLooked,
  loadDetails,
  nexusChanged,
  primeDetails,
  useLookedSnapshot,
  useNexusDetails,
  useNexusEntry,
  useNexusFresh,
  useNexusPage,
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
  addDetails(unknown, (await CachedDetails(currentGame(), unknown)) ?? {})
}
